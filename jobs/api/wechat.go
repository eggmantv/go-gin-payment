package api

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go-gin-payment/config"
	"go-gin-payment/ext/logger"
	"go-gin-payment/models"

	"bitbucket.org/343_3rd/gmodels/common"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/signers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

func wclg() *logrus.Entry {
	return logger.LF("wechat")
}

//
// 商户平台文档: https://pay.weixin.qq.com/wiki/doc/apiv3/index.shtml
// 1. 注意区分`服务商文档`，和普通商户文档
// 2. 同一个功能微信有多个接口，比如小程序支付和扫码支付都有查询订单，退款的接口，实际上是通用的，我们都是用的小程序的接口
//
// 服务商文档: https://pay.weixin.qq.com/wiki/doc/apiv3_partner/apis/chapter4_5_1.shtml
//
// 普通商户支付和服务商支付在支付时部分参数和URL会有区分
//
// 以payment_account中的app_id字段判断是服务商模式支付还是用的我们的支付账号支付
// 1 如果为空表示使用的我们的支付账号，也就是对应微信的“普通商户支付”
// 2 不为空表示使用的服务商支付

type wechatPaymentOps struct {
	StoreID          string `json:"store_id"`
	PaymentAccountID string `json:"payment_account_id"`
	TransNo          string `json:"trans_no"`
	AppID            string `json:"app_id"`
	OpenID           string `json:"open_id"`
	Desp             string `json:"desp"`
	TotalPrice       int64  `json:"total_price"`
	From             string `json:"from"`

	paymentAccount *models.PaymentAccount `json:"-"`
}

func apiWechat(r *gin.Engine) {
	apiWechatNativePay(r)

	// JSAPI支付, 生成支付信息，这个接口支持小程序，公众号网页和APP支付
	// 其中小程序和公众号逻辑是完全一样的，需要指定from: mp
	// APP支付需要指定from: app
	//
	// https://pay.weixin.qq.com/wiki/doc/apiv3/apis/chapter3_5_1.shtml
	//
	// body:
	// {
	//  "store_id": "1",
	// 	"payment_account_id": "1609845740",
	// 	"trans_no": "abcssscascscds",
	// 	"app_id": "wx923152915c597acc",
	// 	"open_id": "ovgfD4hx1cAmnEuFoM9A4phM9h2Y", from==app则没有此key
	// 	"desp": "hello",
	// 	"total_price": 10
	//  "from": "app"|"mp"
	// }
	r.POST("/wechat/gen_mp_prepay", func(ctx *gin.Context) {
		var o wechatPaymentOps
		err := ctx.ShouldBindJSON(&o)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": err.Error()})
			return
		}

		pa, err := models.FindPaLoadPrivateCert(o.PaymentAccountID, true)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": fmt.Sprintf("err to find payment account with id: %s, err: %s", o.PaymentAccountID, err)})
			return
		}
		o.paymentAccount = pa

		d, err := createWechatPaymentOrder(&o)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": err.Error()})
		} else {
			prepayID := gjson.ParseBytes(d).Get("prepay_id").String()
			if len(prepayID) == 0 {
				ctx.JSON(http.StatusOK, common.M{"status": "error", "error": "prepay_id is empty"})
			} else {
				payParams, err := buildWechatPaymentParams(&o, prepayID)
				if err != nil {
					ctx.JSON(http.StatusOK, common.M{"status": "error", "error": "build pay params error:" + err.Error()})
				} else {
					ctx.JSON(http.StatusOK, common.M{
						"status": "ok",
						"data":   payParams,
					})
				}
			}
		}
	})

	// 小程序支付通知
	// https://pay.weixin.qq.com/wiki/doc/apiv3/apis/chapter3_5_5.shtml
	r.POST("/wechat/payment_notify/:transNo", func(ctx *gin.Context) {
		succRsp := common.M{
			"code":    "SUCCESS",
			"message": "成功",
		}
		transNo := ctx.Param("transNo")
		rec, err := models.FindPaymentRecordByTransNo(transNo)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": err.Error()})
			return
		}
		if rec.IsSuccess() {
			ctx.JSON(http.StatusOK, succRsp)
			return
		}

		o := struct {
			EventType    string `json:"event_type"`
			Summary      string `json:"summary"`
			ResourceType string `json:"resource_type"`
			Resource     struct {
				Cipher string `json:"ciphertext"`
				Nonce  string `json:"nonce"`
				Data   string `json:"associated_data"`
			} `json:"resource"`
		}{}
		err = ctx.ShouldBindJSON(&o)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": err.Error()})
			return
		}

		pa, err := models.FindPaLoadPrivateCert(rec.PaymentAccountID, true)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": "load pa error:" + err.Error()})
			return
		}

		cstr, err := utils.DecryptToString(
			pa.APIV3Secret,
			o.Resource.Data,
			o.Resource.Nonce,
			o.Resource.Cipher,
		)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": "decode wechat refund notify data error:" + err.Error()})
			return
		}
		doc := gjson.Parse(cstr)
		data := paymentState{
			State:         doc.Get("trade_state").String(),
			StateDesc:     doc.Get("trade_state_desc").String(),
			IsSuccess:     doc.Get("trade_state").String() == "SUCCESS",
			TransNo:       transNo,
			PaymentMethod: "wechat",
			PayNo:         doc.Get("transaction_id").String(),
			Raw:           doc.Value(),
		}
		d, _ := json.Marshal(data)
		go notifyToWeb("/api/payment/notify_state", d)
		if len(rec.AddiNotifyURL) > 0 {
			go notifyToWeb(rec.AddiNotifyURL, d)
		}

		ctx.JSON(http.StatusOK, succRsp)
	})

	// 使用我们的支付号检查订单状态
	r.POST("/wechat/payment_check", func(ctx *gin.Context) {
		o := struct {
			StoreID          string `json:"store_id"`
			PaymentAccountID string `json:"payment_account_id"`
			TransNo          string `json:"trans_no"`
		}{}
		err := ctx.ShouldBindJSON(&o)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": err.Error()})
			return
		}

		pa, err := models.FindPaLoadPrivateCert(o.PaymentAccountID, true)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": "load pa error:" + err.Error()})
			return
		}
		store, _ := models.FindStoreWithOnlyMerID(o.StoreID)
		res := getWechatPaymentStateByTransNo(store, pa, o.TransNo)
		if res.Err != "" {
			ctx.JSON(http.StatusOK, common.M{
				"status": "error",
				"error": fmt.Sprintf("check trans_no state error, payment account id: %s, trans_no: %s, err: %s",
					o.PaymentAccountID, o.TransNo, res.Err),
			})
			return
		}

		ctx.JSON(http.StatusOK, common.M{"status": "ok", "data": common.M{"payment_state": res}})
	})
}

func getWechatPaymentStateByTransNo(store *models.Store, pa *models.PaymentAccount, transNo string) *paymentState {
	res := paymentState{
		TransNo:       transNo,
		PaymentMethod: "wechat",
	}
	// 初始化客户端
	client, err := setUpWechatClient(pa, true)
	if err != nil {
		res.Err = "setup wechat client error:" + err.Error()
		return &res
	}
	var url string
	if pa.IsWechatServiceProviderAccount() {
		url = fmt.Sprintf("https://api.mch.weixin.qq.com/v3/pay/partner/transactions/out-trade-no/%s?sp_mchid=%s&sub_mchid=%s",
			transNo, pa.MerID, store.WechatPaymentMerID)
	} else {
		url = fmt.Sprintf("https://api.mch.weixin.qq.com/v3/pay/transactions/out-trade-no/%s?mchid=%s",
			transNo, pa.MerID)
	}
	// 发起请求
	rsp, err := client.Get(context.TODO(), url)
	if err != nil {
		res.Err = "wechat, get trans_no state error:" + err.Error()
		return &res
	}
	// 校验回包内容是否有逻辑错误
	body, err := validateWechatClientRsp(rsp)
	if err != nil {
		res.Err = err.Error()
		return &res
	}

	doc := gjson.ParseBytes(body)
	res.Raw = doc.Value()
	wclg().Printf("wechat, trans_no state check rsp: %s", body)
	if err != nil {
		res.Err = "wechat, validate rsp error:" + err.Error()
		return &res
	}

	res.State = doc.Get("trade_state").String()
	res.StateDesc = doc.Get("trade_state_desc").String()
	res.PayNo = doc.Get("transaction_id").String()
	res.IsSuccess = res.State == "SUCCESS"
	return &res
}

func buildWechatPaymentParams(o *wechatPaymentOps, prepayID string) (common.M, error) {
	t := cast.ToString(time.Now().In(models.ChinaTz).Unix())
	nonce := common.GenRandomStr(32)
	res := make(common.M)
	// app支付需要的返回值和小程序不一样
	if o.From == "app" {
		pack := "Sign=WXPay"
		signData := []string{
			o.AppID,
			t,
			nonce,
			prepayID,
		}
		sign, err := signers.Sha256WithRsa(strings.Join(signData, "\n")+"\n", o.paymentAccount.LoadedCertPrivate)
		if err != nil {
			return nil, err
		}
		res = common.M{
			"appId":        o.AppID,
			"partnerId":    o.paymentAccount.MerID,
			"prepayId":     prepayID,
			"packageValue": pack,
			"timeStamp":    t,
			"nonceStr":     nonce,
			"signType":     "RSA",
			"sign":         sign,
		}
	} else {
		pack := "prepay_id=" + prepayID
		signData := []string{
			o.AppID,
			t,
			nonce,
			pack,
		}
		sign, err := signers.Sha256WithRsa(strings.Join(signData, "\n")+"\n", o.paymentAccount.LoadedCertPrivate)
		if err != nil {
			return nil, err
		}
		res = common.M{
			"timeStamp": t,
			"nonceStr":  nonce,
			"package":   pack,
			"signType":  "RSA",
			"paySign":   sign,
		}
	}
	return res, nil
}

func createWechatPaymentOrder(o *wechatPaymentOps) ([]byte, error) {
	// 初始化客户端
	ctx := context.TODO()
	client, err := setUpWechatClient(o.paymentAccount, true)
	if err != nil {
		return nil, err
	}
	//设置请求信息,此处也可以使用结构体来进行请求
	mapInfo := map[string]interface{}{
		"out_trade_no": o.TransNo,
		"description":  o.Desp,
		"notify_url":   config.SelfAPIURL + "/wechat/payment_notify/" + o.TransNo,
		"amount": map[string]interface{}{
			"total":    o.TotalPrice,
			"currency": "CNY",
		},
	}
	var url string
	if o.paymentAccount.IsWechatServiceProviderAccount() {
		store, _ := models.FindStoreWithOnlyMerID(o.StoreID)
		mapInfo["sp_appid"] = o.paymentAccount.AppID
		mapInfo["sp_mchid"] = o.paymentAccount.MerID
		mapInfo["sub_appid"] = o.AppID
		mapInfo["sub_mchid"] = store.WechatPaymentMerID
		if o.From == "mp" {
			mapInfo["payer"] = map[string]interface{}{
				"sub_openid": o.OpenID,
			}
			url = "https://api.mch.weixin.qq.com/v3/pay/partner/transactions/jsapi"
		} else {
			url = "https://api.mch.weixin.qq.com/v3/pay/partner/transactions/app"
		}
	} else {
		mapInfo["mchid"] = o.paymentAccount.MerID
		mapInfo["appid"] = o.AppID
		if o.From == "mp" {
			mapInfo["payer"] = map[string]interface{}{
				"openid": o.OpenID,
			}
			url = "https://api.mch.weixin.qq.com/v3/pay/transactions/jsapi"
		} else {
			url = "https://api.mch.weixin.qq.com/v3/pay/transactions/app"
		}
	}

	// 发起请求
	response, err := client.Post(ctx, url, mapInfo)
	if err != nil {
		wclg().Warnf("client post err: %s", err)
		return nil, err
	}
	// 校验回包内容是否有逻辑错误
	body, err := validateWechatClientRsp(response)
	if err != nil {
		wclg().Warnf("check response err:%s", err)
		return nil, err
	}
	return body, nil
}

func setUpWechatClient(pa *models.PaymentAccount, needValidator bool) (*core.Client, error) {
	//设置header头中authorization信息
	var opts []option.ClientOption
	if needValidator {
		cs, err := getWechatPlatformCert(pa)
		if err != nil {
			return nil, err
		}
		if len(cs) == 0 {
			return nil, errors.New("platform cert is 0")
		}

		opts = []option.ClientOption{
			option.WithMerchant(pa.MerID, pa.CertSerialNumber, pa.LoadedCertPrivate), // 设置商户相关配置
			option.WithWechatPay(cs), // 设置微信支付平台证书，用于校验回包信息用
		}
	} else {
		opts = []option.ClientOption{
			option.WithMerchant(pa.MerID, pa.CertSerialNumber, pa.LoadedCertPrivate), // 设置商户相关配置
			option.WithoutValidator(),
		}
	}

	client, err := core.NewClient(context.TODO(), opts...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func validateWechatClientRsp(rsp *http.Response) ([]byte, error) {
	// 校验回包内容是否有逻辑错误
	err := core.CheckResponse(rsp)
	if err != nil {
		wclg().Warnf("check response err:%s", err)
		return nil, err
	}
	// 读取回包信息
	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		wclg().Warnf("read response body err:%s", err)
		return nil, err
	}
	return body, nil
}

// getWechatPlatformCert 获取微信的平台证书，注意这个和我们自己的公钥和私钥不是一回事
// 平台证书用于验证请求的合法性
func getWechatPlatformCert(pa *models.PaymentAccount) ([]*x509.Certificate, error) {
	ctx := context.TODO()
	client, err := setUpWechatClient(pa, false)
	if err != nil {
		return nil, err
	}
	rsp, err := client.Get(ctx, "https://api.mch.weixin.qq.com/v3/certificates")
	if err != nil {
		wclg().Warnf("get platform cert err:%s", err)
		return nil, err
	}

	if rsp.Body != nil {
		defer rsp.Body.Close()
	}
	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		wclg().Warnf("read rsp body err:%s", err.Error())
		return nil, err
	}
	certs := make([]*x509.Certificate, 0)
	gjson.ParseBytes(body).Get("data").ForEach(func(k, v gjson.Result) bool {
		cstr, err := utils.DecryptToString(
			pa.APIV3Secret,
			v.Get("encrypt_certificate.associated_data").String(),
			v.Get("encrypt_certificate.nonce").String(),
			v.Get("encrypt_certificate.ciphertext").String(),
		)
		if err != nil {
			wclg().Warnln("decode wechat platform cert error:", err)
			return true
		}

		c, err := utils.LoadCertificate(cstr)
		if err != nil {
			wclg().Warnln("load wechat platform cert error:", err)
		} else {
			certs = append(certs, c)
		}
		return true
	})
	return certs, nil
}

package api

import (
	"context"
	"fmt"
	"net/http"

	"go-gin-payment/config"
	"go-gin-payment/models"

	"bitbucket.org/343_3rd/gmodels/common"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// @title           Go Gin Payment API
// @version         1.0
// @description     This is a sample server celler server.
// @termsOfService  http://swagger.io/terms/
// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  service@eggman.tv
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.htm
// @host      localhost:5011
// @externalDocs.description  更多专题请查看蛋人网eggman.tv
// @externalDocs.url          https://eggman.tv

// @Summary      生成支付二维码接口
// @Description  获取二维码
// @Accept       json
// @Produce      json
// @Param        store_id formData string true "店铺ID，我们内部的模型ID，用于标识店铺"
// @Param        payment_account_id formData string true "支付账号ID，我们内部的模型ID，用于标识支付账号。"
// @Param        trans_no formData string true "商户系统内部订单号，由我们自己随机生成，要求6-32个字符内，只能是数字、大小写字母_-|* 且在同一个商户号下唯一。"
// @Param        app_id formData string true "应用ID，微信公众号、小程序的app_id。"
// @Param        desp formData string true "商品信息描述。"
// @Param        total_price formData string true "商品总金额，单位为分"
// @Success      200  {object} 	string "{"status": "ok", "data": {"code_url": "weixin://wxpay/bizpayurl?pr=YoETTdkz1"}}"
// @Failure      500  {string}  string "{"status": "error", "error": "error message"}"
// return code or default,{param type},data type,comment
// @Router       /wechat/native_pay [post]
func apiWechatNativePay(r *gin.Engine) {
	// https://pay.weixin.qq.com/wiki/doc/apiv3/apis/chapter3_4_1.shtml
	//
	// {
	//  "store_id": "1",
	// 	"payment_account_id": "2",
	// 	"trans_no": "5d45e2b694e1435993edb008cf21bf33",
	//  "app_id": "wx123151115c597abc",
	//  "desp": "343科技-Audiom软件购买"
	//  "total_price": 30,
	// }
	r.POST("/wechat/native_pay", func(ctx *gin.Context) {
		o := struct {
			StoreID          string `json:"store_id"`
			PaymentAccountID string `json:"payment_account_id"`
			AppID            string `json:"app_id"`
			TransNo          string `json:"trans_no"`
			Desp             string `json:"desp"`
			TotalPrice       int64  `json:"total_price"` // 金额单位为分
		}{}
		err := ctx.ShouldBindJSON(&o)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": err.Error()})
			return
		}

		// 查找商户号、私有证书等
		pa, err := models.FindPaLoadPrivateCert(o.PaymentAccountID, true)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": fmt.Sprintf("err to find payment account with id: %s, err: %s", o.PaymentAccountID, err)})
			return
		}
		store, _ := models.FindStoreWithOnlyMerID(o.StoreID)

		data := map[string]interface{}{
			"out_trade_no": o.TransNo,
			"description":  o.Desp,
			"notify_url":   config.SelfAPIURL + "/wechat/payment_notify/" + o.TransNo, // 支付通知回调地址
			"amount": map[string]interface{}{
				"total":    o.TotalPrice,
				"currency": "CNY",
			},
		}
		var url string
		// 服务商模式
		if pa.IsWechatServiceProviderAccount() {
			data["sp_appid"] = pa.AppID
			data["sp_mchid"] = pa.MerID
			data["sub_appid"] = o.AppID
			data["sub_mchid"] = store.WechatPaymentMerID

			url = "https://api.mch.weixin.qq.com/v3/pay/partner/transactions/native"
		} else {
			// 普通商户
			data["appid"] = o.AppID
			data["mchid"] = pa.MerID

			url = "https://api.mch.weixin.qq.com/v3/pay/transactions/native"
		}

		client, err := setUpWechatClient(pa, true)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": fmt.Sprintf("setup wechat client err: %s", err.Error())})
			return
		}
		response, err := client.Post(context.TODO(), url, data)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": fmt.Sprintf("err: %s, code: %d", err.Error(), response.StatusCode)})
			return
		}
		// 校验回包内容是否有逻辑错误
		body, err := validateWechatClientRsp(response)
		if err != nil {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": fmt.Sprintf("rsp: %s", string(body))})
			return
		}

		codeURL := gjson.ParseBytes(body).Get("code_url").String()
		if len(codeURL) > 0 {
			ctx.JSON(http.StatusOK, common.M{
				"status": "ok",
				"data": common.M{
					"code_url": codeURL,
				},
			})
		} else {
			ctx.JSON(http.StatusOK, common.M{"status": "error", "error": string(body)})
		}
	})
}

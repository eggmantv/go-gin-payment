package models

import (
	"crypto/rsa"
	"errors"
	"fmt"

	"go-gin-payment/conn"

	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

const (
	ACCOUNT_TYPE_WECHAT = "wechat"
	ACCOUNT_TYPE_ALIPAY = "alipay"
)

type PaymentAccount struct {
	BaseModel
	AccountType      string `gorm:"column:account_type"`
	Name             string `gorm:"column:name"`
	MerID            string `gorm:"column:mer_id"`             // wechat商家ID, alipay的AppID
	AppID            string `gorm:"column:app_id"`             // wechat服务商的AppID，如果不为空表示该支付账号为服务商支付账号，为空表示为普通商户账号
	APIV3Secret      string `gorm:"column:api_v3_secret"`      // wechat，API秘钥和APIv3密钥我们设置的一样
	CertSerialNumber string `gorm:"column:cert_serial_number"` // wechat
	CertPublic       string `gorm:"column:cert_public"`        // wechat, alipay
	CertPrivate      string `gorm:"column:cert_private"`       // wechat, alipay

	AlipayCertPublicKey    string `gorm:"column:alipay_cert_public_key"`     // alipay
	AlipayRootCert         string `gorm:"column:alipay_root_cert"`           // alipay
	AlipayAppCertPublicKey string `gorm:"column:alipay_app_cert_public_key"` // alipay

	LoadedCertPrivate *rsa.PrivateKey `gorm:"-" json:"-"`
}

// FindPaLoadPrivateCert 只有微信支付需要加载私有证书
func FindPaLoadPrivateCert(id interface{}, loadCert bool) (*PaymentAccount, error) {
	var pa PaymentAccount
	conn.DB().First(&pa, "id = ?", id)
	if !pa.Exists() {
		return nil, fmt.Errorf("not found with payment account id: %v", id)
	}

	// load private key 微信需要加载商户私钥
	if loadCert {
		if err := pa.LoadPrivCert(); err != nil {
			return nil, err
		}
	}

	l().Infof("loaded PA, id: %d, name: %s, app_id: %s", pa.ID, pa.Name, pa.AppID)
	return &pa, nil
}

func (pa *PaymentAccount) LoadPrivCert() error {
	cert, err := utils.LoadPrivateKey(pa.CertPrivate)
	if err != nil {
		return errors.New("load pri cert error:" + err.Error())
	}
	pa.LoadedCertPrivate = cert
	return nil
}

func (pa *PaymentAccount) IsWechatServiceProviderAccount() bool {
	return pa.AccountType == ACCOUNT_TYPE_WECHAT && len(pa.AppID) > 0
}

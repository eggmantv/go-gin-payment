package models

import "go-gin-payment/conn"

type Store struct {
	BaseModel
	Name               string `json:"name"`
	WechatPaymentMerID string `gorm:"column:wechat_payment_mer_id" json:"wechat_payment_mer_id"`
	UUID               string `gorm:"column:uuid" json:"uuid"`
}

func IsStoreExists(uuid string) bool {
	var s Store
	conn.DB().First(&s, "uuid = ?", uuid)
	return s.Exists()
}

func FindStoreWithOnlyMerID(id interface{}) (*Store, bool) {
	var s Store
	conn.DB().Select("id", "wechat_payment_mer_id").First(&s, id)
	if s.Exists() {
		return &s, true
	}
	return nil, false
}

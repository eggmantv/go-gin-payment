package models

import (
	"errors"

	"go-gin-payment/conn"
)

type PaymentRecord struct {
	BaseModel
	TransNo          string  `gorm:"column:trans_no" json:"trans_no"`
	PaymentAccountID int64   `gorm:"column:payment_account_id" json:"payment_account_id"`
	PayNo            string  `gorm:"column:pay_no" json:"pay_no"` // 微信订单号/交易号
	Status           string  `gorm:"column:status" json:"status"`
	TotalMoney       float64 `gorm:"total_money"`
	PaymentResponse  string  `gorm:"payment_response"`
	StoreID          int64   `gorm:"store_id"`
	AddiNotifyURL    string  `gorm:"addi_notify_url"`
}

func FindPaymentRecordByTransNo(transNo string) (*PaymentRecord, error) {
	var r PaymentRecord
	conn.DB().First(&r, "trans_no = ?", transNo)
	if !r.Exists() {
		return nil, errors.New("not found payment record, trans_no: " + transNo)
	}
	return &r, nil
}

func (r *PaymentRecord) IsSuccess() bool {
	return r.Status == "success"
}

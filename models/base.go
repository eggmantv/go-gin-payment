package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"go-gin-payment/conn"
	"go-gin-payment/ext/logger"

	"github.com/sirupsen/logrus"
)

func l() *logrus.Entry {
	return logger.LF("models")
}

var UTCTz *time.Location
var ChinaTz *time.Location

const (
	DateTimeFormat = "2006-01-02 15:04:05"

	TrueInt  = 1
	FalseInt = 0
)

func init() {
	var err error
	UTCTz, err = time.LoadLocation("UTC")
	if err != nil {
		panic(err)
	}
	ChinaTz, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}
}

const (
	NullJSONTime = "0001-01-01T00:00:00Z"
)

type BaseModel struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// SkipHooks bool `gorm:"-" json:"-"`
}

func (bm *BaseModel) Exists() bool {
	return bm.ID != 0
}

// func (bm *BaseModel) BeforeCreate() (err error) {
// 	if len(bm.UUID) == 0 {
// 		bm.UUID = common.GenUUID()
// 	}
// 	return err
// }

func TotalCount(m interface{}) int64 {
	var total int64
	conn.DB().Model(m).Count(&total)
	return total
}

func ToMap(v interface{}) map[string]interface{} {
	data := make(map[string]interface{})

	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("%v to json error", reflect.TypeOf(v)))
	}
	_ = json.Unmarshal(b, &data)
	return data
}

func FindStoreByUUID(uuid string) (*Store, bool) {
	var s Store
	conn.DB().Where("uuid = ?", uuid).First(&s)
	if s.Exists() {
		return &s, true
	}
	return nil, false
}

// NewTime 这里的value是中国时区，用这个方法转换后存入数据库就是UTC时区
func NewTime(value string) time.Time {
	t, _ := time.ParseInLocation(DateTimeFormat, value, ChinaTz)
	return t
}

package ext

import (
	"fmt"
	"strings"
	"time"

	"go-gin-payment/config"
	"go-gin-payment/ext/logger"

	"gopkg.in/resty.v1"
)

type M map[string]interface{}

func LoggerHookFunc(name string, payload map[string]interface{}) {
}

func SendToWeb(uri string, b []byte) ([]byte, error) {
	if !strings.HasPrefix(uri, "http") {
		uri = config.WebURL + uri
	}

	rsp, err := resty.SetTimeout(10*time.Second).R().
		SetHeader("X_API_SECRET", config.WebAPISecret).
		SetBody(b).
		Post(uri)
	if err != nil {
		logger.L.Warn("send to web err:", err)
		return nil, err
	}
	if !rsp.IsSuccess() {
		msgErr := fmt.Errorf("send to web err: %s, code: %d", string(rsp.Body()), rsp.StatusCode())
		logger.L.Warnf(msgErr.Error())
		return nil, msgErr
	}
	return rsp.Body(), nil
}

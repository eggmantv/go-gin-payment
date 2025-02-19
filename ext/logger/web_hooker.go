package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var prdLogLevels = []logrus.Level{
	logrus.PanicLevel,
	logrus.FatalLevel,
	logrus.ErrorLevel,
	logrus.WarnLevel,
}
var logFrom string

type hookFunc func(string, map[string]interface{})

type hooker struct {
	hookFn hookFunc
}

func init() {
	logFrom = fmt.Sprintf("gcs-%s", os.Getenv("DC_NAME"))
}

func newWebHooker(fn hookFunc) *hooker {
	return &hooker{fn}
}

func (h *hooker) Fire(entry *logrus.Entry) error {
	payload := make(map[string]interface{})

	payload["t"] = time.Now().UTC().Unix()
	payload["level"] = entry.Level.String()
	payload["time"] = entry.Time
	payload["_from"] = logFrom
	payload["message"] = entry.Message

	for k, v := range entry.Data {
		if errData, isError := v.(error); logrus.ErrorKey == k && v != nil && isError {
			payload[k] = errData.Error()
		} else {
			payload[k] = v
		}
	}

	go h.hookFn("log", payload)

	return nil
}

func (h *hooker) Levels() []logrus.Level {
	// if config.Env == "development" {
	// 	return logrus.AllLevels
	// }
	return prdLogLevels
}

package logger

import (
	"io"
	"log"
	"os"

	"bitbucket.org/343_3rd/gmodels"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var L *logrus.Logger

func SetLog(e string, fn hookFunc) *os.File {
	var logPath string
	if e == "development" {
		if exists, _ := IsFileExists("logs"); !exists {
			if err := os.Mkdir("logs", 0777); err != nil {
				L.Fatalln(err)
			}
		}

		logPath = "logs/go-gin-payment.log"
	} else {
		logPath = "/var/log/go-gin-payment/go-gin-payment.log"
	}
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		L.Fatal(err)
		return nil
	}
	dst := io.MultiWriter(f, os.Stdout)

	// set standard log
	log.SetOutput(dst)
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[STDLOG]")

	// set Gin log
	gin.DefaultWriter = dst
	gin.DefaultErrorWriter = dst

	// set logrus
	L = logrus.New()
	L.SetNoLock()
	webHook := newWebHooker(fn)
	L.AddHook(webHook)
	if e == "development" {
		L.SetOutput(os.Stdout)
	} else {
		L.SetOutput(dst)
		// L.SetFormatter(&logrus.JSONFormatter{})
	}
	L.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	// L.SetReportCaller(true)

	// set gmodels's logrus
	gmodels.SetupLog(L)

	return f
}

func LF(serviceName string) *logrus.Entry {
	return L.WithField("service", serviceName)
}

func IsFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, err
	}
	return true, err
}

package cmd_lib

import (
	"go-gin-payment/config"
	"go-gin-payment/conn"
	"go-gin-payment/ext"
	"go-gin-payment/ext/logger"
)

func SetupLog(e string) func() {
	// setup logrus
	file := logger.SetLog(e, ext.LoggerHookFunc)

	// set up app configs
	config.Parse(e)

	return func() {
		file.Close()
	}
}

func Prepare() func() {
	// mysql
	conn.NewConn()

	// connect redis, the connection is in async mode, need to wait to continue after connected
	if err := conn.RedisConnect(); err != nil {
		logger.L.Panicln("redis connect error:", err)
	}

	return func() {
		conn.Close()
		conn.RedisClose()
	}
}

func Setup(e string) func() {
	cleaner1 := SetupLog(e)
	cleaner2 := Prepare()
	return func() {
		cleaner1()
		cleaner2()
	}
}

package api

import (
	"fmt"
	"net/http"
	"time"

	"go-gin-payment/ext"
	"go-gin-payment/ext/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type paymentState struct {
	State         string      `json:"state,omitempty"`
	StateDesc     string      `json:"state_desc,omitempty"`
	Err           string      `json:"err,omitempty"`
	IsSuccess     bool        `json:"is_success"`
	TransNo       string      `json:"trans_no,omitempty"`  // 支付号
	RefundNo      string      `json:"refund_no,omitempty"` // 退款号，和支付号只会有一个不为空
	PaymentMethod string      `json:"payment_method"`
	PayNo         string      `json:"pay_no,omitempty"` // 第三方订单号
	Raw           interface{} `json:"raw"`
}

func l() *logrus.Entry {
	return logger.LF("api")
}

// RunAPI run http sever
func RunAPI() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	// gin.DisableConsoleColor()
	r := gin.New()

	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("GIN[%s] %s %s %s %s %d %s \"%s\" %s\"\n",
			param.TimeStamp.Format(time.RFC3339),
			param.ClientIP,
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())

	r.Use(authHeaderMiddlewareWithoutPaths(
		"/eggman/wechat/payment_notify",
		"/swagger/*any",
	))

	apiWechat(r)

	r.GET("/ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong, i am running!")
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}

// notifyToWeb 一直通知保证成功
func notifyToWeb(uri string, b []byte) {
	wait := time.After(30 * time.Second)
	for {
		select {
		case <-wait:
			l().Error("notify to web timeout:", string(b))
			return
		default:
			rsp, err := ext.SendToWeb(uri, b)
			// !!! web端必须返回`ok`表示处理成功，否则这里会一直尝试直到超时
			if err != nil || string(rsp) != "ok" {
				time.Sleep(2 * time.Second)
				continue
			}
			return
		}
	}
}

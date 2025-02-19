package conn

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"go-gin-payment/config"
	"go-gin-payment/ext/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var mutex sync.Mutex

// DB mysql connection
var connection *gorm.DB

const (
	connectRetryMaxTimes = 20
)

var connectRetryTimes int

// for test env
var tranx *gorm.DB

// DB if connection is not ready, try to reconnect
func DB() *gorm.DB {
	mutex.Lock()
	defer mutex.Unlock()

	NewConn()
	if config.Env == "test" {
		return tranx
	}
	return connection
}

// NewConn create a connection if there's none
func NewConn() {
	if connection == nil {
		if connectRetryTimes < connectRetryMaxTimes {
			connection = dbConnect()
			if connection == nil {
				logger.L.Infof("connect db failed for the %d times, retry next time", (connectRetryTimes + 1))
				connectRetryTimes++
				time.Sleep(2 * time.Second)
				NewConn()
			} else {
				tranx = connection
			}
		} else {
			// reset retry times
			connectRetryTimes = 0
			logger.L.Errorf("connect db failed, reach max retry times: %d", connectRetryMaxTimes)
		}
	}
}

func Close() {
	if connection != nil {
		if d, err := connection.DB(); err == nil {
			d.Close()
		}
	}
}

func dbConnect() *gorm.DB {
	if connection != nil {
		return connection
	}

	var dbURI string
	if config.Env == "development" || config.Env == "test" {
		dbURI = "root:@tcp(localhost:3306)/ggp_development?charset=utf8mb4&parseTime=True&loc=UTC"
		if config.Env == "test" {
			// 修改测试环境下事务等级，未提交也可读
			dbURI = fmt.Sprintf("%s&tx_isolation=%s", dbURI, url.QueryEscape("'READ-UNCOMMITTED'"))
		}
	} else {
		// 注意这里是连接的host中的mysql，需要用户不是来自localhost
		// hostIP := os.Getenv("HOST_IP")
		hostIP := "mysql" // docker host name
		logger.L.Println("host ip:", hostIP)
		dbURI = "ggp:t2Q1021285@tcp(" + hostIP + ":3306)/ggp_production?charset=utf8mb4&parseTime=True&loc=UTC"
	}

	logger.L.Println("connecting to mysql:", dbURI)
	db, err := gorm.Open(mysql.Open(dbURI), &gorm.Config{})
	if err != nil {
		// CAUTION: need to close db even not connected, otherwise memory will leak
		if d, err := db.DB(); err == nil {
			d.Close()
		}

		logger.L.Println("connect db error:", err)
		// panic(err)
		return nil
	}

	d, err := db.DB()
	if err != nil {
		logger.L.Println("get DB error:", err)
		return nil
	}
	if err = d.Ping(); err != nil {
		d.Close()
		logger.L.Println("connect db ping error:", err)
		return nil
	}

	logger.L.Println("connected to db:", dbURI)
	d.SetMaxIdleConns(5)
	d.SetMaxOpenConns(500)

	return db
}

// SetTestDBAsTx 重置测试环境下数据库连接为事务的方式
func SetTestDBAsTx() {
	if connection == nil {
		panic("need to connect db first!")
	}
	if config.Env != "test" {
		panic("only test env can set this!")
	}

	// 测试环境下所有数据库操作都在一个事物中
	tranx = connection.Begin()
}

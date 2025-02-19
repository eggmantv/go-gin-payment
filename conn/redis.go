package conn

import (
	"context"
	"os"
	"strings"

	"go-gin-payment/ext/logger"

	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8" // doc: https://redis.uptrace.dev/
)

var Redis *redis.Client

func RedisConnect() error {
	uri := os.Getenv("REDIS_URI")
	logger.L.Infoln("redis uri:", uri)

	uri2 := strings.TrimPrefix(uri, "redis://")
	p := strings.Split(uri2, "/")
	if len(p) != 2 {
		logger.L.Fatalln("redis uri invalid:", uri)
	}
	db, err := govalidator.ToInt(p[1])
	if err != nil {
		logger.L.Fatalln("redis uri invalid:", uri)
	}

	Redis = redis.NewClient(&redis.Options{
		Addr:         p[0],
		DB:           int(db),
		MaxRetries:   10,
		PoolSize:     20,
		MinIdleConns: 2,
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			logger.L.Println("redis connected")
			return nil
		},
	})
	return Redis.Ping(context.TODO()).Err()
}

func RedisClose() {
	if Redis != nil {
		Redis.Close()
	}
}

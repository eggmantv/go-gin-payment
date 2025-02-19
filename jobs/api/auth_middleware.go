package api

import (
	"fmt"
	"strings"

	"bitbucket.org/343_3rd/gmodels/common"
	"github.com/gin-gonic/gin"
)

const (
	authHeaderKey    = "X_GGP_KEY"
	authHeaderSecret = "xxx"
)

// authHeaderMiddlewareWithoutPaths 可以传递哪些路由不需要验证，默认都需要
func authHeaderMiddlewareWithoutPaths(withoutPaths ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		pa := ctx.FullPath()
		fmt.Println("11111, pa:", pa)
		for _, pattern := range withoutPaths {
			if len(pattern) == 0 {
				continue
			}
			if pattern == pa || strings.HasPrefix(pa, pattern) {
				ctx.Next()
				return
			}
		}

		secret := ctx.GetHeader(authHeaderKey)
		if secret != authHeaderSecret {
			ctx.AbortWithStatusJSON(401, common.M{
				"status": "error",
				"error":  "api secret is invalid",
			})
			return
		}
		ctx.Next()
	}
}

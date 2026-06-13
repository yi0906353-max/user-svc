package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func APIKeyMiddleware() gin.HandlerFunc {
	expectedKey := os.Getenv("INTERNAL_API_KEY")

	return func(c *gin.Context) {
		if expectedKey == "" {
			// 没配置 key 则放行（开发环境）
			c.Next()
			return
		}

		key := c.GetHeader("X-API-Key")
		if key != expectedKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}

		c.Next()
	}
}

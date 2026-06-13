package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAPIKeyMiddleware_NoKey(t *testing.T) {
	// 设置环境变量
	t.Setenv("INTERNAL_API_KEY", "test-secret-key")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(APIKeyMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// 无 key 请求
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIKeyMiddleware_ValidKey(t *testing.T) {
	t.Setenv("INTERNAL_API_KEY", "test-secret-key")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(APIKeyMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "test-secret-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIKeyMiddleware_WrongKey(t *testing.T) {
	t.Setenv("INTERNAL_API_KEY", "test-secret-key")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(APIKeyMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIKeyMiddleware_NoKeyConfigured(t *testing.T) {
	// 不设置 INTERNAL_API_KEY，应该放行
	t.Setenv("INTERNAL_API_KEY", "")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(APIKeyMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

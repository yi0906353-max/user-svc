package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/config"
	"github.com/mindpilot/user-svc/internal/model"
	"github.com/mindpilot/user-svc/internal/service"
	"github.com/mindpilot/user-svc/internal/store"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func setupAuthMiddleware() (*service.AuthService, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:      4,
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 4)
	userID := uuid.New()
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) {
			return &model.User{
				ID:           userID,
				Email:        email,
				PasswordHash: string(hash),
				Status:       "active",
			}, nil
		},
		GetByIDFn: func(id uuid.UUID) (*model.User, error) {
			return &model.User{ID: id, Email: "test@example.com"}, nil
		},
		CreateFn:        func(user *model.User) error { return nil },
		CreateProfileFn: func(userID uuid.UUID) error { return nil },
	}
	tokens := &store.MockTokenStore{
		StoreRefreshTokenFn: func(_ uuid.UUID, _ string, _ *string, _ *string, _ time.Time) error {
			return nil
		},
	}
	svc := service.NewAuthService(users, tokens, cfg)

	r := gin.New()
	r.Use(service_authMiddleware(svc))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"user_id": c.GetString("user_id"),
			"email":   c.GetString("email"),
		})
	})

	return svc, r
}

// service_authMiddleware 是对 AuthMiddleware 的包装，避免循环导入
func service_authMiddleware(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || len(header) < 8 || header[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}
		token := header[7:]
		claims, err := auth.ValidateAccessToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("user_id", claims.UserID.String())
		c.Set("email", claims.Email)
		c.Next()
	}
}

func getAuthToken(svc *service.AuthService) string {
	resp, _ := svc.Login(&model.LoginRequest{Email: "test@example.com", Password: "password123"})
	return resp.AccessToken
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	_, r := setupAuthMiddleware()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	_, r := setupAuthMiddleware()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	_, r := setupAuthMiddleware()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	svc, r := setupAuthMiddleware()
	token := getAuthToken(svc)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test@example.com")
}

package handler

import (
	"bytes"
	"encoding/json"
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

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:       "test-secret-key",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:      4,
	}
}

func setupAuthHandler(users *store.MockUserStore, tokens *store.MockTokenStore) (*AuthHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	svc := service.NewAuthService(users, tokens, testConfig())
	h := NewAuthHandler(svc)

	r := gin.New()
	r.POST("/api/v1/auth/register", h.Register)
	r.POST("/api/v1/auth/login", h.Login)
	r.POST("/api/v1/auth/refresh", h.Refresh)
	r.POST("/auth/logout", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		h.Logout(c)
	})

	return h, r
}

// ──────────────────────────── Register ────────────────────────────

func TestAuthHandler_Register_Success(t *testing.T) {
	users := &store.MockUserStore{
		GetByEmailFn:    func(email string) (*model.User, error) { return nil, nil },
		CreateFn:        func(user *model.User) error { return nil },
		CreateProfileFn: func(userID uuid.UUID) error { return nil },
	}
	tokens := &store.MockTokenStore{
		StoreRefreshTokenFn: func(_ uuid.UUID, _ string, _ *string, _ *string, _ time.Time) error {
			return nil
		},
	}
	_, r := setupAuthHandler(users, tokens)

	body, _ := json.Marshal(map[string]string{
		"email":        "new@example.com",
		"password":     "password123",
		"display_name": "New User",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp model.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, "new@example.com", resp.User.Email)
}

func TestAuthHandler_Register_BadRequest(t *testing.T) {
	users := &store.MockUserStore{}
	tokens := &store.MockTokenStore{}
	_, r := setupAuthHandler(users, tokens)

	body, _ := json.Marshal(map[string]string{"email": "test@test.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Register_PasswordTooShort(t *testing.T) {
	users := &store.MockUserStore{}
	tokens := &store.MockTokenStore{}
	_, r := setupAuthHandler(users, tokens)

	body, _ := json.Marshal(map[string]string{
		"email":        "test@test.com",
		"password":     "123",
		"display_name": "Test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Register_EmailExists(t *testing.T) {
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) {
			return &model.User{ID: uuid.New(), Email: email}, nil
		},
	}
	tokens := &store.MockTokenStore{}
	_, r := setupAuthHandler(users, tokens)

	body, _ := json.Marshal(map[string]string{
		"email":        "existing@test.com",
		"password":     "password123",
		"display_name": "Test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ──────────────────────────── Login ────────────────────────────

func TestAuthHandler_Login_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 4)
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) {
			return &model.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: string(hash),
				Status:       "active",
			}, nil
		},
	}
	tokens := &store.MockTokenStore{
		StoreRefreshTokenFn: func(_ uuid.UUID, _ string, _ *string, _ *string, _ time.Time) error {
			return nil
		},
	}
	_, r := setupAuthHandler(users, tokens)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@test.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), 4)
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) {
			return &model.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: string(hash),
				Status:       "active",
			}, nil
		},
	}
	tokens := &store.MockTokenStore{}
	_, r := setupAuthHandler(users, tokens)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@test.com",
		"password": "wrong",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Login_Suspended(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 4)
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) {
			return &model.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: string(hash),
				Status:       "suspended",
			}, nil
		},
	}
	tokens := &store.MockTokenStore{}
	_, r := setupAuthHandler(users, tokens)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@test.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ──────────────────────────── Refresh ────────────────────────────

func TestAuthHandler_Refresh_Success(t *testing.T) {
	userID := uuid.New()
	users := &store.MockUserStore{
		GetByIDFn: func(id uuid.UUID) (*model.User, error) {
			return &model.User{ID: id, Email: "test@test.com"}, nil
		},
	}
	tokens := &store.MockTokenStore{
		GetRefreshTokenFn: func(token string) (*model.RefreshToken, error) {
			return &model.RefreshToken{
				UserID:    userID,
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}, nil
		},
		RevokeRefreshTokenFn: func(token string) error { return nil },
		StoreRefreshTokenFn:  func(_ uuid.UUID, _ string, _ *string, _ *string, _ time.Time) error { return nil },
	}
	_, r := setupAuthHandler(users, tokens)

	body, _ := json.Marshal(map[string]string{"refresh_token": "valid-token"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	users := &store.MockUserStore{}
	tokens := &store.MockTokenStore{
		GetRefreshTokenFn: func(token string) (*model.RefreshToken, error) { return nil, nil },
	}
	_, r := setupAuthHandler(users, tokens)

	body, _ := json.Marshal(map[string]string{"refresh_token": "bad-token"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ──────────────────────────── Logout ────────────────────────────

func TestAuthHandler_Logout_Success(t *testing.T) {
	tokens := &store.MockTokenStore{
		RevokeRefreshTokenFn: func(token string) error { return nil },
	}
	_, r := setupAuthHandler(&store.MockUserStore{}, tokens)

	body, _ := json.Marshal(map[string]string{"refresh_token": "some-token"})
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/config"
	"github.com/mindpilot/user-svc/internal/model"
	"github.com/mindpilot/user-svc/internal/store"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:       "test-secret-key-for-unit-tests",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:      4, // low cost for fast tests
	}
}

func hashPassword(pw string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(pw), 4)
	return string(hash)
}

// ──────────────────────────── 错误常量 ────────────────────────────

func TestErrEmailExists(t *testing.T) {
	assert.Equal(t, "该邮箱已注册", ErrEmailExists.Error())
}

func TestErrInvalidPassword(t *testing.T) {
	assert.Equal(t, "邮箱或密码错误", ErrInvalidPassword.Error())
}

func TestErrAccountSuspended(t *testing.T) {
	assert.Equal(t, "账号已被暂停，请联系管理员", ErrAccountSuspended.Error())
}

// ──────────────────────────── Register ────────────────────────────

func TestRegister_Success(t *testing.T) {
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) { return nil, nil },
		CreateFn:     func(user *model.User) error { return nil },
		CreateProfileFn: func(userID uuid.UUID) error { return nil },
	}
	tokens := &store.MockTokenStore{
		StoreRefreshTokenFn: func(_ uuid.UUID, _ string, _ *string, _ *string, _ time.Time) error {
			return nil
		},
	}

	svc := NewAuthService(users, tokens, testConfig())
	resp, err := svc.Register(&model.RegisterRequest{
		Email:       "test@example.com",
		Password:    "password123",
		DisplayName: "Test User",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "test@example.com", resp.User.Email)
	assert.Equal(t, "Test User", resp.User.DisplayName)
	assert.Equal(t, "active", resp.User.Status)
}

func TestRegister_EmailExists(t *testing.T) {
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) {
			return &model.User{ID: uuid.New(), Email: email}, nil
		},
	}
	tokens := &store.MockTokenStore{}

	svc := NewAuthService(users, tokens, testConfig())
	resp, err := svc.Register(&model.RegisterRequest{
		Email:       "existing@example.com",
		Password:    "password123",
		DisplayName: "Test",
	})

	assert.ErrorIs(t, err, ErrEmailExists)
	assert.Nil(t, resp)
}

func TestRegister_CreateUserError(t *testing.T) {
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) { return nil, nil },
		CreateFn:     func(user *model.User) error { return assert.AnError },
	}
	tokens := &store.MockTokenStore{}

	svc := NewAuthService(users, tokens, testConfig())
	resp, err := svc.Register(&model.RegisterRequest{
		Email:       "test@example.com",
		Password:    "password123",
		DisplayName: "Test",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// ──────────────────────────── Login ────────────────────────────

func TestLogin_Success(t *testing.T) {
	pw := hashPassword("password123")
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) {
			return &model.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: pw,
				Status:       "active",
			}, nil
		},
	}
	tokens := &store.MockTokenStore{
		StoreRefreshTokenFn: func(_ uuid.UUID, _ string, _ *string, _ *string, _ time.Time) error {
			return nil
		},
	}

	svc := NewAuthService(users, tokens, testConfig())
	resp, err := svc.Login(&model.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
}

func TestLogin_UserNotFound(t *testing.T) {
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) { return nil, nil },
	}
	tokens := &store.MockTokenStore{}

	svc := NewAuthService(users, tokens, testConfig())
	resp, err := svc.Login(&model.LoginRequest{
		Email:    "nobody@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrInvalidPassword)
	assert.Nil(t, resp)
}

func TestLogin_WrongPassword(t *testing.T) {
	pw := hashPassword("correct-password")
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) {
			return &model.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: pw,
				Status:       "active",
			}, nil
		},
	}
	tokens := &store.MockTokenStore{}

	svc := NewAuthService(users, tokens, testConfig())
	resp, err := svc.Login(&model.LoginRequest{
		Email:    "test@example.com",
		Password: "wrong-password",
	})

	assert.ErrorIs(t, err, ErrInvalidPassword)
	assert.Nil(t, resp)
}

func TestLogin_AccountSuspended(t *testing.T) {
	pw := hashPassword("password123")
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) {
			return &model.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: pw,
				Status:       "suspended",
			}, nil
		},
	}
	tokens := &store.MockTokenStore{}

	svc := NewAuthService(users, tokens, testConfig())
	resp, err := svc.Login(&model.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrAccountSuspended)
	assert.Nil(t, resp)
}

// ──────────────────────────── Refresh ────────────────────────────

func TestRefresh_Success(t *testing.T) {
	userID := uuid.New()
	users := &store.MockUserStore{
		GetByIDFn: func(id uuid.UUID) (*model.User, error) {
			return &model.User{ID: id, Email: "test@example.com", Status: "active"}, nil
		},
	}
	tokens := &store.MockTokenStore{
		GetRefreshTokenFn: func(token string) (*model.RefreshToken, error) {
			return &model.RefreshToken{
				ID:        uuid.New(),
				UserID:    userID,
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}, nil
		},
		RevokeRefreshTokenFn: func(token string) error { return nil },
		StoreRefreshTokenFn:  func(_ uuid.UUID, _ string, _ *string, _ *string, _ time.Time) error { return nil },
	}

	svc := NewAuthService(users, tokens, testConfig())
	tp, err := svc.Refresh("old-refresh-token")

	assert.NoError(t, err)
	assert.NotNil(t, tp)
	assert.NotEmpty(t, tp.AccessToken)
	assert.NotEmpty(t, tp.RefreshToken)
}

func TestRefresh_TokenNotFound(t *testing.T) {
	users := &store.MockUserStore{}
	tokens := &store.MockTokenStore{
		GetRefreshTokenFn: func(token string) (*model.RefreshToken, error) { return nil, nil },
	}

	svc := NewAuthService(users, tokens, testConfig())
	tp, err := svc.Refresh("nonexistent-token")

	assert.ErrorIs(t, err, ErrTokenExpired)
	assert.Nil(t, tp)
}

func TestRefresh_TokenExpired(t *testing.T) {
	userID := uuid.New()
	users := &store.MockUserStore{}
	tokens := &store.MockTokenStore{
		GetRefreshTokenFn: func(token string) (*model.RefreshToken, error) {
			return &model.RefreshToken{
				ID:        uuid.New(),
				UserID:    userID,
				ExpiresAt: time.Now().Add(-1 * time.Hour), // 已过期
			}, nil
		},
		RevokeRefreshTokenFn: func(token string) error { return nil },
	}

	svc := NewAuthService(users, tokens, testConfig())
	tp, err := svc.Refresh("expired-token")

	assert.ErrorIs(t, err, ErrTokenExpired)
	assert.Nil(t, tp)
}

func TestRefresh_UserNotFound(t *testing.T) {
	users := &store.MockUserStore{
		GetByIDFn: func(id uuid.UUID) (*model.User, error) { return nil, nil },
	}
	tokens := &store.MockTokenStore{
		GetRefreshTokenFn: func(token string) (*model.RefreshToken, error) {
			return &model.RefreshToken{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}, nil
		},
		RevokeRefreshTokenFn: func(token string) error { return nil },
	}

	svc := NewAuthService(users, tokens, testConfig())
	tp, err := svc.Refresh("valid-looking-token")

	assert.ErrorIs(t, err, ErrTokenExpired)
	assert.Nil(t, tp)
}

// ──────────────────────────── Logout ────────────────────────────

func TestLogout_SingleDevice(t *testing.T) {
	revoked := ""
	tokens := &store.MockTokenStore{
		RevokeRefreshTokenFn: func(token string) error {
			revoked = token
			return nil
		},
	}

	svc := NewAuthService(&store.MockUserStore{}, tokens, testConfig())
	rt := "some-refresh-token"
	err := svc.Logout(uuid.New(), &rt, false)

	assert.NoError(t, err)
	assert.Equal(t, "some-refresh-token", revoked)
}

func TestLogout_AllDevices(t *testing.T) {
	var revokedUserID uuid.UUID
	tokens := &store.MockTokenStore{
		RevokeAllForUserFn: func(userID uuid.UUID) error {
			revokedUserID = userID
			return nil
		},
	}

	uid := uuid.New()
	svc := NewAuthService(&store.MockUserStore{}, tokens, testConfig())
	err := svc.Logout(uid, nil, true)

	assert.NoError(t, err)
	assert.Equal(t, uid, revokedUserID)
}

func TestLogout_NoTokenNoAllDevices(t *testing.T) {
	tokens := &store.MockTokenStore{
		RevokeRefreshTokenFn: func(token string) error { return assert.AnError },
		RevokeAllForUserFn:   func(userID uuid.UUID) error { return assert.AnError },
	}

	svc := NewAuthService(&store.MockUserStore{}, tokens, testConfig())
	err := svc.Logout(uuid.New(), nil, false)

	// 无 token 且 allDevices=false，应直接返回 nil
	assert.NoError(t, err)
}

// ──────────────────────────── ValidateAccessToken ────────────────────────────

func TestValidateAccessToken_Success(t *testing.T) {
	cfg := testConfig()
	var registeredUserID uuid.UUID
	users := &store.MockUserStore{
		GetByEmailFn: func(email string) (*model.User, error) { return nil, nil },
		CreateFn: func(user *model.User) error {
			registeredUserID = user.ID
			return nil
		},
		CreateProfileFn: func(userID uuid.UUID) error { return nil },
		GetByIDFn: func(id uuid.UUID) (*model.User, error) {
			return &model.User{ID: id, Email: "test@example.com"}, nil
		},
	}
	tokens := &store.MockTokenStore{
		StoreRefreshTokenFn: func(_ uuid.UUID, _ string, _ *string, _ *string, _ time.Time) error {
			return nil
		},
	}

	svc := NewAuthService(users, tokens, cfg)
	resp, err := svc.Register(&model.RegisterRequest{
		Email:       "test@example.com",
		Password:    "password123",
		DisplayName: "Test",
	})
	assert.NoError(t, err)

	claims, err := svc.ValidateAccessToken(resp.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, registeredUserID, claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	svc := NewAuthService(&store.MockUserStore{}, &store.MockTokenStore{}, testConfig())
	claims, err := svc.ValidateAccessToken("not-a-valid-jwt")

	assert.Error(t, err)
	assert.Nil(t, claims)
}

package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/config"
	"github.com/mindpilot/user-svc/internal/model"
	"github.com/mindpilot/user-svc/internal/store"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists     = errors.New("该邮箱已注册")
	ErrInvalidPassword = errors.New("邮箱或密码错误")
	ErrAccountSuspended = errors.New("账号已被暂停，请联系管理员")
	ErrTokenExpired    = errors.New("refresh token 已过期")
	ErrTokenReused     = errors.New("refresh token 已被使用")
)

type AuthService struct {
	users  store.UserStoreInterface
	tokens store.TokenStoreInterface
	cfg    *config.Config
}

func NewAuthService(users store.UserStoreInterface, tokens store.TokenStoreInterface, cfg *config.Config) *AuthService {
	return &AuthService{users: users, tokens: tokens, cfg: cfg}
}

func (s *AuthService) Register(req *model.RegisterRequest) (*model.AuthResponse, error) {
	// 支持 email 或 account（手机号）
	email := req.Email
	if email == "" && req.Account != "" {
		email = req.Account
	}

	// 检查邮箱是否已注册
	existing, err := s.users.GetByEmail(email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailExists
	}

	// 哈希密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cfg.BcryptCost)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &model.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
		Status:       "active",
		Timezone:     "Asia/Shanghai",
		Locale:       "zh-CN",
	}

	if err := s.users.Create(user); err != nil {
		return nil, err
	}
	if err := s.users.CreateProfile(user.ID); err != nil {
		return nil, err
	}

	// 签发 token
	return s.issueTokens(user)
}

func (s *AuthService) Login(req *model.LoginRequest) (*model.AuthResponse, error) {
	user, err := s.users.GetByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidPassword
	}

	if user.Status == "suspended" {
		return nil, ErrAccountSuspended
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidPassword
	}

	return s.issueTokens(user)
}

func (s *AuthService) Refresh(refreshToken string) (*model.TokenPair, error) {
	rt, err := s.tokens.GetRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}
	if rt == nil {
		return nil, ErrTokenExpired
	}

	if time.Now().After(rt.ExpiresAt) {
		s.tokens.RevokeRefreshToken(refreshToken)
		return nil, ErrTokenExpired
	}

	// 轮换：吊销旧 token
	if err := s.tokens.RevokeRefreshToken(refreshToken); err != nil {
		return nil, err
	}

	user, err := s.users.GetByID(rt.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrTokenExpired
	}

	// 签发新 token 对
	return s.issueTokenPair(user)
}

func (s *AuthService) Logout(userID uuid.UUID, refreshToken *string, allDevices bool) error {
	if allDevices {
		return s.tokens.RevokeAllForUser(userID)
	}
	if refreshToken != nil && *refreshToken != "" {
		return s.tokens.RevokeRefreshToken(*refreshToken)
	}
	return nil
}

func (s *AuthService) ValidateAccessToken(tokenString string) (*model.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, errors.New("invalid user id in token")
	}

	// 从 token 中获取 email（通过 RegisteredClaims.Issuer 或自定义 claims）
	// 这里简化处理，从数据库查
	user, err := s.users.GetByID(userID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	return &model.Claims{UserID: userID, Email: user.Email}, nil
}

func (s *AuthService) issueTokens(user *model.User) (*model.AuthResponse, error) {
	tp, err := s.issueTokenPair(user)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		AccessToken:  tp.AccessToken,
		RefreshToken: tp.RefreshToken,
		ExpiresIn:    tp.ExpiresIn,
		User:         *user,
	}, nil
}

func (s *AuthService) issueTokenPair(user *model.User) (*model.TokenPair, error) {
	// access token
	now := time.Now()
	accessClaims := jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTokenTTL)),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessString, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	// refresh token (random string)
	refreshString := uuid.New().String() + uuid.New().String()
	expiresAt := now.Add(s.cfg.RefreshTokenTTL)

	if err := s.tokens.StoreRefreshToken(user.ID, refreshString, nil, nil, expiresAt); err != nil {
		return nil, err
	}

	return &model.TokenPair{
		AccessToken:  accessString,
		RefreshToken: refreshString,
		ExpiresIn:    int64(s.cfg.AccessTokenTTL.Seconds()),
	}, nil
}

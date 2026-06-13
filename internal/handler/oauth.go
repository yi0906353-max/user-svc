package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/model"
	"github.com/mindpilot/user-svc/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// OAuthHandler 处理 OAuth2.0 登录
type OAuthHandler struct {
	users  store.UserStoreInterface
	tokens store.TokenStoreInterface
}

func NewOAuthHandler(users store.UserStoreInterface, tokens store.TokenStoreInterface) *OAuthHandler {
	return &OAuthHandler{users: users, tokens: tokens}
}

// OAuthProvider OAuth 提供商配置
type OAuthProvider struct {
	Name         string
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	RedirectURI  string
}

var providers = map[string]OAuthProvider{
	"github": {
		Name:         "GitHub",
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
		RedirectURI:  os.Getenv("OAUTH_REDIRECT_URI") + "/auth/callback/github",
	},
	"google": {
		Name:         "Google",
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
		RedirectURI:  os.Getenv("OAUTH_REDIRECT_URI") + "/auth/callback/google",
	},
}

// GetProviders 获取可用的 OAuth 提供商
func (h *OAuthHandler) GetProviders(c *gin.Context) {
	var available []map[string]string
	for name, p := range providers {
		if p.ClientID != "" {
			available = append(available, map[string]string{
				"name": name,
				"label": p.Name,
				"url":   fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&scope=%s&response_type=code",
					p.AuthURL, p.ClientID, p.RedirectURI, getScope(name)),
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": available})
}

// HandleCallback 处理 OAuth 回调
func (h *OAuthHandler) HandleCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	p, ok := providers[provider]
	if !ok || p.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	// 用 code 换 access token
	token, err := h.exchangeToken(p, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token exchange failed"})
		return
	}

	// 获取用户信息
	userInfo, err := h.getUserInfo(p, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
		return
	}

	// 查找或创建用户
	user, err := h.findOrCreateUser(provider, userInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user creation failed"})
		return
	}

	// 签发 token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	})
	accessString, _ := accessToken.SignedString([]byte(os.Getenv("JWT_SECRET")))

	refreshString := uuid.New().String() + uuid.New().String()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	h.tokens.StoreRefreshToken(user.ID, refreshString, nil, nil, expiresAt)

	c.JSON(http.StatusOK, model.AuthResponse{
		AccessToken:  accessString,
		RefreshToken: refreshString,
		ExpiresIn:    86400,
		User:         *user,
	})
}

func (h *OAuthHandler) exchangeToken(p OAuthProvider, code string) (string, error) {
	var body io.Reader
	var url string

	if p.Name == "GitHub" {
		url = p.TokenURL
		form := fmt.Sprintf("client_id=%s&client_secret=%s&code=%s", p.ClientID, p.ClientSecret, code)
		req, _ := http.NewRequest("POST", url, nil)
		req.URL.RawQuery = form
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		var result struct {
			AccessToken string `json:"access_token"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		return result.AccessToken, nil
	}

	// Google
	url = fmt.Sprintf("%s?client_id=%s&client_secret=%s&code=%s&grant_type=authorization_code&redirect_uri=%s",
		p.TokenURL, p.ClientID, p.ClientSecret, code, p.RedirectURI)
	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		AccessToken string `json:"access_token"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.AccessToken, nil
}

func (h *OAuthHandler) getUserInfo(p OAuthProvider, token string) (map[string]interface{}, error) {
	req, _ := http.NewRequest("GET", p.UserInfoURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if p.Name == "GitHub" {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&info)
	return info, nil
}

func (h *OAuthHandler) findOrCreateUser(provider string, info map[string]interface{}) (*model.User, error) {
	var email, name string

	if provider == "github" {
		email, _ = info["email"].(string)
		name, _ = info["name"].(string)
		if email == "" {
			email = fmt.Sprintf("%v@github.user", info["login"])
		}
	} else {
		email, _ = info["email"].(string)
		name, _ = info["name"].(string)
	}

	// 查找已有用户
	user, _ := h.users.GetByEmail(email)
	if user != nil {
		return user, nil
	}

	// 创建新用户
	if name == "" {
		name = email
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(uuid.New().String()), 10)
	user = &model.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  name,
		Status:       "active",
		Timezone:     "Asia/Shanghai",
		Locale:       "zh-CN",
	}

	if err := h.users.Create(user); err != nil {
		return nil, err
	}
	h.users.CreateProfile(user.ID)
	return user, nil
}

func getScope(provider string) string {
	switch provider {
	case "github":
		return "user:email"
	case "google":
		return "openid email profile"
	default:
		return ""
	}
}

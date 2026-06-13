package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/model"
	"github.com/mindpilot/user-svc/internal/store"
)

// WebAuthnHandler 处理 WebAuthn 注册和认证
type WebAuthnHandler struct {
	users       store.UserStoreInterface
	tokens      store.TokenStoreInterface
	credentials *WebAuthnStore
}

// WebAuthnStore WebAuthn 凭证存储（内存 + DB）
type WebAuthnStore struct {
	credentials map[string][]*WebAuthnCredential // userID -> credentials
}

type WebAuthnCredential struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	PublicKey   []byte    `json:"public_key"`
	SignCount   uint32    `json:"sign_count"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
	Name        string    `json:"name"`
}

// WebAuthnChallenge WebAuthn 挑战
type WebAuthnChallenge struct {
	Challenge    string `json:"challenge"`
	RPID        string `json:"rp_name"`
	UserID      string `json:"user_id"`
	CredentialID string `json:"credential_id,omitempty"`
}

func NewWebAuthnHandler(users store.UserStoreInterface, tokens store.TokenStoreInterface) *WebAuthnHandler {
	return &WebAuthnHandler{
		users:  users,
		tokens: tokens,
		credentials: &WebAuthnStore{
			credentials: make(map[string][]*WebAuthnCredential),
		},
	}
}

// GetChallenges 获取可用的认证方式
func (h *WebAuthnHandler) GetChallenges(c *gin.Context) {
	// 返回支持的认证方式
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"methods": []gin.H{
				{"type": "password", "label": "密码登录"},
				{"type": "webauthn", "label": "生物识别/安全密钥"},
				{"type": "oauth", "label": "第三方登录"},
			},
		},
	})
}

// StartRegistration 开始 WebAuthn 注册
func (h *WebAuthnHandler) StartRegistration(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 生成 challenge
	challenge := make([]byte, 32)
	rand.Read(challenge)

	// 生成 credential ID
	credID := make([]byte, 16)
	rand.Read(credID)

	resp := gin.H{
		"challenge": base64.RawURLEncoding.EncodeToString(challenge),
		"rp_name":   "MindPilot",
		"rp_id":     getEnvDefault("RP_ID", "localhost"),
		"user_id":   userID,
		"timeout":   60000,
		"authenticator_selection": gin.H{
			"authenticator_attachment": "platform",
			"user_verification":        "required",
		},
		"pub_key_cred_params": []gin.H{
			{"type": "public-key", "alg": -7},   // ES256
			{"type": "public-key", "alg": -257}, // RS256
		},
	}

	c.JSON(http.StatusOK, resp)
}

// CompleteRegistration 完成 WebAuthn 注册
func (h *WebAuthnHandler) CompleteRegistration(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		CredentialID string `json:"credential_id"`
		PublicKey    string `json:"public_key"`
		Name         string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 解码公钥
	pubKey, err := base64.RawURLEncoding.DecodeString(req.PublicKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid public key"})
		return
	}

	// 保存凭证
	cred := &WebAuthnCredential{
		ID:        req.CredentialID,
		UserID:    userID,
		PublicKey: pubKey,
		CreatedAt: time.Now(),
		Name:      req.Name,
	}

	h.credentials.credentials[userID] = append(h.credentials.credentials[userID], cred)

	c.JSON(http.StatusOK, gin.H{
		"message": "注册成功",
		"credential_id": cred.ID,
	})
}

// StartAuthentication 开始 WebAuthn 认证
func (h *WebAuthnHandler) StartAuthentication(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查找用户
	user, err := h.users.GetByEmail(req.Email)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// 获取用户凭证
	creds := h.credentials.credentials[user.ID.String()]
	if len(creds) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no webauthn credentials"})
		return
	}

	// 生成 challenge
	challenge := make([]byte, 32)
	rand.Read(challenge)

	// 返回第一个凭证（简化实现）
	cred := creds[0]
	c.JSON(http.StatusOK, gin.H{
		"challenge":     base64.RawURLEncoding.EncodeToString(challenge),
		"rp_id":         getEnvDefault("RP_ID", "localhost"),
		"timeout":       60000,
		"credential_id": cred.ID,
		"user_id":       user.ID.String(),
	})
}

// CompleteAuthentication 完成 WebAuthn 认证
func (h *WebAuthnHandler) CompleteAuthentication(c *gin.Context) {
	var req struct {
		UserID       string `json:"user_id"`
		CredentialID string `json:"credential_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查找凭证
	creds := h.credentials.credentials[req.UserID]
	var cred *WebAuthnCredential
	for _, c := range creds {
		if c.ID == req.CredentialID {
			cred = c
			break
		}
	}

	if cred == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		return
	}

	// 更新使用时间
	cred.LastUsedAt = time.Now()
	cred.SignCount++

	// 查找用户
	userID, _ := uuid.Parse(req.UserID)
	user, err := h.users.GetByID(userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
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

// ListCredentials 列出用户的 WebAuthn 凭证
func (h *WebAuthnHandler) ListCredentials(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	creds := h.credentials.credentials[userID]
	var result []map[string]interface{}
	for _, cred := range creds {
		result = append(result, map[string]interface{}{
			"id":         cred.ID,
			"name":       cred.Name,
			"created_at": cred.CreatedAt,
			"last_used":  cred.LastUsedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// DeleteCredential 删除 WebAuthn 凭证
func (h *WebAuthnHandler) DeleteCredential(c *gin.Context) {
	userID := c.GetString("user_id")
	credID := c.Param("credential_id")

	creds := h.credentials.credentials[userID]
	for i, cred := range creds {
		if cred.ID == credID {
			h.credentials.credentials[userID] = append(creds[:i], creds[i+1:]...)
			c.JSON(http.StatusOK, gin.H{"message": "deleted"})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mindpilot/user-svc/internal/store"
)

// VerifyHandler 处理验证码相关
type VerifyHandler struct {
	users store.UserStoreInterface
}

// 验证码存储（内存，生产环境应使用 Redis）
var (
	codeStore   = make(map[string]*verifyCode)
	codeStoreMu sync.RWMutex
)

type verifyCode struct {
	Code      string
	ExpiresAt time.Time
}

func NewVerifyHandler(users store.UserStoreInterface) *VerifyHandler {
	return &VerifyHandler{users: users}
}

// CheckAccount 检查账号是否存在
func (h *VerifyHandler) CheckAccount(c *gin.Context) {
	var req struct {
		Account string `json:"account" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入账号"})
		return
	}

	account := req.Account

	// 验证邮箱格式
	if isEmail(account) {
		if !isValidEmail(account) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式不正确，请输入真实邮箱"})
			return
		}
	} else {
		// 验证手机号格式
		if !isValidPhone(account) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "手机号格式不正确，请输入 11 位手机号"})
			return
		}
		account = account + "@phone.mindpilot"
	}

	user, _ := h.users.GetByEmail(account)
	c.JSON(http.StatusOK, gin.H{
		"exists": user != nil,
	})
}

// SendCode 发送验证码
func (h *VerifyHandler) SendCode(c *gin.Context) {
	var req struct {
		Account string `json:"account" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入账号"})
		return
	}

	account := req.Account
	if !isEmail(account) {
		account = account + "@phone.mindpilot"
	}

	// 生成 6 位验证码
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	// 存储验证码（5 分钟有效）
	codeStoreMu.Lock()
	codeStore[account] = &verifyCode{
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	codeStoreMu.Unlock()

	// 开发环境：返回验证码给前端显示
	// 生产环境：这里应调用短信/邮件 API
	fmt.Printf("📱 验证码 [%s] -> %s (code: %s)\n", account, req.Account, code)

	c.JSON(http.StatusOK, gin.H{
		"message": "验证码已发送",
		"expires_in": 300,
		"dev_code": code, // 开发环境：前端直接展示验证码
	})
}

// VerifyCode 验证验证码
func (h *VerifyHandler) VerifyCode(account, code string) bool {
	codeStoreMu.RLock()
	stored, ok := codeStore[account]
	codeStoreMu.RUnlock()

	if !ok {
		return false
	}
	if time.Now().After(stored.ExpiresAt) {
		codeStoreMu.Lock()
		delete(codeStore, account)
		codeStoreMu.Unlock()
		return false
	}
	if stored.Code != code {
		return false
	}

	// 验证通过，删除验证码
	codeStoreMu.Lock()
	delete(codeStore, account)
	codeStoreMu.Unlock()

	return true
}

func isEmail(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '@' {
			return true
		}
	}
	return false
}

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	// 必须包含 @ 和 .，且 @ 前有字符，@ 后有域名
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 || at >= len(email)-1 {
		return false
	}
	// 域名部分必须包含 .
	domain := email[at+1:]
	hasDot := false
	for _, c := range domain {
		if c == '.' {
			hasDot = true
			break
		}
	}
	return hasDot && len(domain) >= 5 // 至少 a.bc
}

// isValidPhone 验证手机号格式（中国大陆 11 位，1 开头）
func isValidPhone(phone string) bool {
	if len(phone) != 11 {
		return false
	}
	if phone[0] != '1' {
		return false
	}
	for _, c := range phone {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

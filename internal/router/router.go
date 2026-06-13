package router

import (
	"github.com/gin-gonic/gin"
	"github.com/mindpilot/user-svc/internal/handler"
	"github.com/mindpilot/user-svc/internal/middleware"
	"github.com/mindpilot/user-svc/internal/service"
	"github.com/mindpilot/user-svc/internal/store"
)

func Setup(
	authSvc *service.AuthService,
	userSvc *service.UserService,
	contactSvc *service.ContactService,
	userStore store.UserStoreInterface,
	tokenStore store.TokenStoreInterface,
) *gin.Engine {
	r := gin.Default()

	// CORS 支持
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	authH := handler.NewAuthHandler(authSvc)
	userH := handler.NewUserHandler(userSvc)
	contactH := handler.NewContactHandler(contactSvc)
	internalH := handler.NewInternalHandler(contactSvc)
	oauthH := handler.NewOAuthHandler(userStore, tokenStore)
	webauthnH := handler.NewWebAuthnHandler(userStore, tokenStore)
	verifyH := handler.NewVerifyHandler(userStore)

	v1 := r.Group("/api/v1")

	// 公开路由
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.POST("/refresh", authH.Refresh)

		// 验证码
		auth.POST("/check-account", verifyH.CheckAccount)
		auth.POST("/send-code", verifyH.SendCode)

		// OAuth2.0
		auth.GET("/oauth/providers", oauthH.GetProviders)
		auth.GET("/oauth/:provider/callback", oauthH.HandleCallback)

		// WebAuthn
		auth.POST("/webauthn/authenticate", webauthnH.StartAuthentication)
		auth.POST("/webauthn/authenticate/complete", webauthnH.CompleteAuthentication)
	}

	// 需要 JWT 的路由
	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(authSvc))
	{
		// 认证
		protected.POST("/auth/logout", authH.Logout)

		// 用户资料
		protected.GET("/users/me", userH.GetMe)
		protected.PATCH("/users/me", userH.UpdateMe)
		protected.GET("/users/me/profile", userH.GetProfile)
		protected.PATCH("/users/me/profile", userH.UpdateProfile)

		// 联系人
		protected.GET("/contacts", contactH.List)
		protected.POST("/contacts", contactH.Create)
		protected.GET("/contacts/frequent", contactH.GetFrequent)
		protected.GET("/contacts/:contact_id", contactH.Get)
		protected.PATCH("/contacts/:contact_id", contactH.Update)
		protected.DELETE("/contacts/:contact_id", contactH.Delete)

		// WebAuthn 注册（需登录）
		protected.POST("/auth/webauthn/register", webauthnH.StartRegistration)
		protected.POST("/auth/webauthn/register/complete", webauthnH.CompleteRegistration)
		protected.GET("/auth/webauthn/credentials", webauthnH.ListCredentials)
		protected.DELETE("/auth/webauthn/credentials/:credential_id", webauthnH.DeleteCredential)
	}

	// 内部 API（API Key 鉴权）
	internal := r.Group("/internal")
	internal.Use(middleware.APIKeyMiddleware())
	{
		internal.GET("/users", userH.ListUsers)
		internal.GET("/users/:user_id", userH.GetUserInternal)
		internal.GET("/users/:user_id/frequent-contacts", internalH.FrequentContactCheck)
		internal.POST("/briefing-run-logs", userH.CreateBriefingRunLog)
		internal.GET("/briefing-run-logs", userH.ListBriefingRunLogs)
	}

	return r
}

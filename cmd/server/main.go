package main

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/mindpilot/user-svc/internal/config"
	"github.com/mindpilot/user-svc/internal/router"
	"github.com/mindpilot/user-svc/internal/service"
	"github.com/mindpilot/user-svc/internal/store"
	"go.uber.org/zap"
)

func main() {
	// 日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	// 配置
	cfg := config.Load()

	// 数据库
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)

	// Store 层
	userStore := store.NewUserStore(db)
	tokenStore := store.NewTokenStore(db)
	contactStore := store.NewContactStore(db)

	// Service 层
	authSvc := service.NewAuthService(userStore, tokenStore, cfg)
	userSvc := service.NewUserService(userStore)
	contactSvc := service.NewContactService(contactStore)

	// 路由
	r := router.Setup(authSvc, userSvc, contactSvc, userStore, tokenStore)

	// 启动
	zap.L().Info("User service starting", zap.String("addr", cfg.ServerAddr))
	if err := r.Run(cfg.ServerAddr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

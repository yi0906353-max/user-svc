# Agent Context — User Service

## 你是谁

你是一个资深 Go 后端工程师，正在开发 MindPilot 的 User Service。
这是系统的身份认证和用户管理中心，所有其他服务都依赖它。

## 技术栈（硬约束）

- 语言：Go 1.22+
- Web 框架：gin v1.9+
- 数据库：sqlx + PostgreSQL
- 认证：JWT (golang-jwt/jwt/v5)
- 密码：golang.org/x/crypto/bcrypt
- 配置：viper
- 日志：zap
- 测试：testing + testify
- 迁移：golang-migrate

## 服务边界

你只负责：
1. 用户注册、登录、JWT 签发与刷新
2. 用户资料 CRUD（基本信息 + 扩展资料）
3. 联系人管理（增删改查 + 常用联系人维护）
4. 内部 API：供 inbox-svc 查询常用联系人
5. OAuth 第三方登录绑定（预留接口，第一版不实现）

你不负责：
- API Gateway 的路由配置（Kong 独立管理）
- 消息处理（inbox-svc 负责）
- 推送通知（notification-svc 负责）
- 前端页面（Web/iOS/Android 客户端负责）

## 核心设计要求

### JWT 策略
- access_token 有效期 15 分钟，payload 包含 {sub: user_id, email, exp}
- refresh_token 有效期 7 天，存储 SHA256 hash（不存原文）
- refresh_token 轮换：每次刷新时，旧 token 吊销，签发新 token
- 重放检测：如果已吊销的 refresh_token 被再次使用，吊销该用户所有 token

### 密码安全
- bcrypt cost=12
- 登录失败不透露是"邮箱不存在"还是"密码错误"
- 可选：登录失败 5 次后锁定 15 分钟（第一版不实现）

### 内部 API 鉴权
- /internal/* 路由不走 JWT
- 通过 X-API-Key header 鉴权，key 从环境变量读取
- 只允许内网调用（生产环境通过网络策略限制）

### 联系人常用判断
- interaction_count >= 10 → is_frequent = true
- interaction_count < 5 → is_frequent = false
- 中间值保持不变（防抖动）

### 数据库
- 见 schema.sql
- 所有表都有 created_at / updated_at 自动更新
- 使用 uuid v4 作为主键
- 迁移文件放在 migrations/ 目录

## 目录结构建议

```
user-svc/
├── cmd/
│   └── server/
│       └── main.go           # 启动入口
├── internal/
│   ├── config/
│   │   └── config.go         # 配置加载
│   ├── handler/
│   │   ├── auth.go           # 认证 handler
│   │   ├── user.go           # 用户资料 handler
│   │   ├── contact.go        # 联系人 handler
│   │   └── internal.go       # 内部 API handler
│   ├── middleware/
│   │   ├── auth.go           # JWT 中间件
│   │   └── apikey.go         # API Key 中间件
│   ├── model/
│   │   ├── user.go           # 用户数据结构
│   │   ├── contact.go        # 联系人数据结构
│   │   └── token.go          # Token 相关结构
│   ├── store/
│   │   ├── user.go           # 用户 DB 操作
│   │   ├── contact.go        # 联系人 DB 操作
│   │   └── token.go          # Token DB 操作
│   ├── service/
│   │   ├── auth.go           # 认证业务逻辑
│   │   ├── user.go           # 用户业务逻辑
│   │   └── contact.go        # 联系人业务逻辑
│   └── router/
│       └── router.go         # 路由定义
├── migrations/
│   ├── 001_create_users.up.sql
│   ├── 001_create_users.down.sql
│   └── ...
├── spec/
│   ├── api.yaml
│   ├── schema.sql
│   ├── message-flow.md
│   └── agent-context.md
├── go.mod
├── go.sum
└── Makefile
```

## 与 F1 (inbox-svc) 的接口

inbox-svc 需要调用你的内部 API：
- GET /internal/users/{user_id}/frequent-contacts?email=xxx
- 返回：{ is_frequent: bool, contact: Contact | null }

这个接口用于 inbox-svc 的优先级打分：
- 发件人是常用联系人 → priority +2

## 现在请开始

按照 spec 目录下的定义，实现完整的 User Service。
先实现认证（注册/登录/刷新），再实现用户资料，最后联系人。
每个模块写完立即写对应的单元测试。

---

## Step 1: 用户注册

### Scenario 1.1: 正常注册
GIVEN 用户提交邮箱、密码、显示名
WHEN 调用 POST /auth/register
THEN 密码使用 bcrypt(cost=12) 哈希后存储
AND 创建 users 记录 + user_profiles 空记录
AND 返回 access_token + refresh_token + 用户信息
AND access_token 有效期 15 分钟
AND refresh_token 有效期 7 天

### Scenario 1.2: 邮箱已注册
GIVEN 数据库中已存在相同 email 的用户
WHEN 注册
THEN 返回 409 Conflict
AND 错误信息："该邮箱已注册"

### Scenario 1.3: 密码强度不足
GIVEN 密码长度 < 8
WHEN 注册
THEN 返回 400 Bad Request
AND 错误信息："密码至少 8 个字符"

---

## Step 2: 用户登录

### Scenario 2.1: 正常登录
GIVEN 用户提交正确的邮箱和密码
WHEN 调用 POST /auth/login
THEN 验证密码（bcrypt compare）
AND 生成新的 access_token + refresh_token
AND 将 refresh_token 的 hash 存入 refresh_tokens 表
AND 返回 token 对 + 用户信息

### Scenario 2.2: 密码错误
GIVEN 用户提交错误的密码
WHEN 登录
THEN 返回 401 Unauthorized
AND 错误信息："邮箱或密码错误"（不透露是邮箱还是密码错）

### Scenario 2.3: 账号被暂停
GIVEN 用户 status = 'suspended'
WHEN 登录
THEN 返回 403 Forbidden
AND 错误信息："账号已被暂停，请联系管理员"

---

## Step 3: Token 刷新

### Scenario 3.1: 正常刷新
GIVEN 有效的 refresh_token
WHEN 调用 POST /auth/refresh
THEN 验证 token hash 存在且未过期
AND 生成新的 access_token + refresh_token（轮换）
AND 吊销旧的 refresh_token
AND 返回新 token 对

### Scenario 3.2: refresh_token 已过期
GIVEN refresh_token 已过期
WHEN 刷新
THEN 返回 401
AND 客户端需要重新登录

### Scenario 3.3: refresh_token 重放检测
GIVEN 一个已被使用过的 refresh_token（被盗用）
WHEN 尝试刷新
THEN 返回 401
AND 吊销该用户的所有 refresh_token（安全措施）

---

## Step 4: 登出

### Scenario 4.1: 正常登出
GIVEN 有效的 access_token + refresh_token
WHEN 调用 POST /auth/logout
THEN 吊销指定的 refresh_token
AND 返回 204

### Scenario 4.2: 全设备登出
GIVEN all_devices = true
WHEN 登出
THEN 吊销该用户的所有 refresh_token
AND 返回 204

---

## Step 5: 用户资料管理

### Scenario 5.1: 获取当前用户
GIVEN 有效的 access_token
WHEN 调用 GET /users/me
AND 返回用户基本信息 + timezone + locale

### Scenario 5.2: 更新资料
GIVEN 用户提交 display_name / avatar_url / timezone / locale 的部分更新
WHEN 调用 PATCH /users/me
THEN 只更新提交的字段
AND 返回更新后的用户信息

### Scenario 5.3: 获取扩展资料
GIVEN 有效的 access_token
WHEN 调用 GET /users/me/profile
THEN 返回 bio / company / title / preferences / onboarding_done

---

## Step 6: 联系人管理

### Scenario 6.1: 添加联系人
GIVEN 用户提交联系人信息（name 必填）
WHEN 调用 POST /contacts
THEN 创建联系人记录
AND 自动检查 is_frequent（interaction_count >= 10 自动标记）

### Scenario 6.2: 搜索联系人
GIVEN 关键词 "张"
WHEN 调用 GET /contacts?search=张
THEN 模糊匹配 name / email / company 字段
AND 按 interaction_count DESC 排序

### Scenario 6.3: 常用联系人自动维护
GIVEN 用户与某联系人的 interaction_count >= 10
WHEN 更新交互记录
THEN 自动设置 is_frequent = TRUE
AND interaction_count < 5 时自动降级为非常用

### Scenario 6.4: 按来源查询联系人
GIVEN source = "wechat"
WHEN 调用 GET /contacts?source=wechat
AND 返回该来源的所有联系人

### Scenario 6.5: 删除联系人
GIVEN 联系人 ID
WHEN 调用 DELETE /contacts/{id}
THEN 软删除或硬删除
AND 返回 204

---

## Step 7: 内部 API（服务间调用）

### Scenario 7.1: inbox-svc 查询常用联系人
GIVEN inbox-svc 需要判断发件人是否为常用联系人
WHEN 调用 GET /internal/users/{user_id}/frequent-contacts?email=zhang@example.com
THEN 返回 is_frequent: true/false + 联系人详情
AND 此 API 不需要 JWT，通过内网或 API Key 鉴权

### Scenario 7.2: 联系人不存在
GIVEN 查询的 email 在联系人表中不存在
WHEN 调用内部 API
THEN 返回 is_frequent: false, contact: null

---

## 边界场景

### 并发注册
GIVEN 两个请求同时用相同邮箱注册
WHEN 并发处理
THEN 数据库 UNIQUE 约束保证只有一个成功
AND 另一个返回 409

### Token 安全
GIVEN access_token 泄露
WHEN 用户主动登出
THEN 吊销所有 refresh_token，迫使重新登录
AND access_token 15 分钟后自然过期

### 联系人批量导入
GIVEN 用户从微信导入 500 个联系人
WHEN 批量写入
THEN 使用事务批量 INSERT（每批 100 条）
AND 去重：按 (user_id, source, source_id) 判断

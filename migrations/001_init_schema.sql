-- MindPilot 统一数据库初始化
-- 合并 user-svc + agent-svc + inbox-svc 的 schema
-- 执行方式: psql -U mindpilot -d mindpilot -f 001_init_schema.sql

-- ============================================================
-- 公共函数
-- ============================================================

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- User Service 表
-- ============================================================

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE NOT NULL,
    phone           TEXT UNIQUE,
    password_hash   TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    avatar_url      TEXT,
    timezone        TEXT DEFAULT 'Asia/Shanghai',
    locale          TEXT DEFAULT 'zh-CN',
    status          TEXT DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_phone ON users(phone) WHERE phone IS NOT NULL;

CREATE TABLE user_profiles (
    user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bio             TEXT,
    company         TEXT,
    title           TEXT,
    preferences     JSONB DEFAULT '{}',
    onboarding_done BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE contacts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    email           TEXT,
    phone           TEXT,
    company         TEXT,
    title           TEXT,
    avatar_url      TEXT,
    source          TEXT,
    source_id       TEXT,
    is_frequent     BOOLEAN DEFAULT FALSE,
    interaction_count INTEGER DEFAULT 0,
    last_interacted_at TIMESTAMPTZ,
    tags            JSONB DEFAULT '[]',
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_contacts_user ON contacts(user_id);
CREATE INDEX idx_contacts_user_frequent ON contacts(user_id) WHERE is_frequent = TRUE;
CREATE INDEX idx_contacts_source ON contacts(user_id, source, source_id);

CREATE TABLE oauth_connections (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    access_token    TEXT,
    refresh_token   TEXT,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);

CREATE INDEX idx_oauth_user ON oauth_connections(user_id);

CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      TEXT UNIQUE NOT NULL,
    device_info     TEXT,
    ip_address      TEXT,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);

-- User Service 触发器
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_profiles_updated_at
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_contacts_updated_at
    BEFORE UPDATE ON contacts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ============================================================
-- Agent Service 表
-- ============================================================

CREATE TABLE workflow_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT UNIQUE NOT NULL,
    description     TEXT,
    version         INTEGER DEFAULT 1,
    dag_definition  JSONB NOT NULL,
    default_config  JSONB DEFAULT '{}',
    status          TEXT DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wf_templates_name ON workflow_templates(name, version DESC);

CREATE TABLE workflow_executions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id     UUID NOT NULL REFERENCES workflow_templates(id),
    trigger_type    TEXT NOT NULL,
    trigger_ref     TEXT,
    input_data      JSONB NOT NULL,
    status          TEXT DEFAULT 'running',
    output_data     JSONB,
    error_message   TEXT,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    duration_ms     INTEGER
);

CREATE INDEX idx_wf_executions_template ON workflow_executions(template_id, started_at DESC);
CREATE INDEX idx_wf_executions_status ON workflow_executions(status, started_at);

CREATE TABLE node_executions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id    UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    node_name       TEXT NOT NULL,
    node_type       TEXT NOT NULL,
    input_data      JSONB,
    output_data     JSONB,
    status          TEXT DEFAULT 'pending',
    error_message   TEXT,
    model_used      TEXT,
    tokens_used     INTEGER,
    cost_usd        NUMERIC(10,6),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    duration_ms     INTEGER
);

CREATE INDEX idx_node_executions_exec ON node_executions(execution_id, node_name);

CREATE TABLE prompt_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT UNIQUE NOT NULL,
    template        TEXT NOT NULL,
    description     TEXT,
    version         INTEGER DEFAULT 1,
    model_hints     JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prompt_templates_name ON prompt_templates(name, version DESC);

CREATE TABLE model_routes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_name      TEXT UNIQUE NOT NULL,
    primary_model   TEXT NOT NULL,
    fallback_model  TEXT,
    criteria        JSONB DEFAULT '{}',
    status          TEXT DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Agent Service 触发器
CREATE TRIGGER trg_wf_templates_updated_at
    BEFORE UPDATE ON workflow_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_prompt_templates_updated_at
    BEFORE UPDATE ON prompt_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_model_routes_updated_at
    BEFORE UPDATE ON model_routes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ============================================================
-- Inbox Service 表
-- ============================================================

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    source          TEXT NOT NULL,
    external_id     TEXT,
    sender_name     TEXT,
    sender_avatar   TEXT,
    source_icon     TEXT,
    direction       TEXT NOT NULL DEFAULT 'inbound',
    content_enc     TEXT NOT NULL,
    content_preview TEXT,
    summary         TEXT,
    category        TEXT,
    priority        SMALLINT DEFAULT 0,
    action_items    JSONB DEFAULT '[]',
    llm_model_used  TEXT,
    llm_tokens      INTEGER,
    read            BOOLEAN DEFAULT FALSE,
    starred         BOOLEAN DEFAULT FALSE,
    archived        BOOLEAN DEFAULT FALSE,
    processing_status TEXT DEFAULT 'pending',
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at    TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_user_created ON messages(user_id, created_at DESC);
CREATE INDEX idx_messages_user_category ON messages(user_id, category) WHERE category = 'needs_reply';
CREATE INDEX idx_messages_user_priority ON messages(user_id, priority DESC) WHERE processing_status = 'completed';
CREATE INDEX idx_messages_external ON messages(user_id, source, external_id);
CREATE INDEX idx_messages_processing ON messages(processing_status, created_at) WHERE processing_status IN ('pending', 'processing');

CREATE TABLE message_processing_logs (
    id          BIGSERIAL PRIMARY KEY,
    message_id  UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    stage       TEXT NOT NULL,
    status      TEXT NOT NULL,
    duration_ms INTEGER,
    detail      JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_logs_message ON message_processing_logs(message_id);

-- Inbox Service 触发器
CREATE TRIGGER trg_messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

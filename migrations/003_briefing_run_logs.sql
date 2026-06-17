-- 简报运行日志表
CREATE TABLE IF NOT EXISTS briefing_run_logs (
    id              BIGSERIAL PRIMARY KEY,
    run_id          TEXT UNIQUE NOT NULL,
    run_type        TEXT NOT NULL,           -- cron / manual
    started_at      TIMESTAMPTZ NOT NULL,
    completed_at    TIMESTAMPTZ,
    users_total     INTEGER DEFAULT 0,
    users_success   INTEGER DEFAULT 0,
    users_failed    INTEGER DEFAULT 0,
    total_tokens    INTEGER DEFAULT 0,
    total_cost      NUMERIC(10,6) DEFAULT 0,
    failures        JSONB DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_briefing_run_logs_started ON briefing_run_logs(started_at DESC);

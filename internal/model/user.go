package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	Email          string          `db:"email" json:"email"`
	Phone          *string         `db:"phone" json:"phone,omitempty"`
	PasswordHash   string          `db:"password_hash" json:"-"`
	DisplayName    string          `db:"display_name" json:"display_name"`
	AvatarURL      *string         `db:"avatar_url" json:"avatar_url,omitempty"`
	Timezone       string          `db:"timezone" json:"timezone"`
	Locale         string          `db:"locale" json:"locale"`
	Status         string          `db:"status" json:"status"`
	BriefingPrefs  json.RawMessage `db:"briefing_prefs" json:"briefing_prefs,omitempty"`
	PushEnabled    bool            `db:"push_enabled" json:"push_enabled"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`
}

type UserProfile struct {
	User
	Bio            *string         `db:"bio" json:"bio,omitempty"`
	Company        *string         `db:"company" json:"company,omitempty"`
	Title          *string         `db:"title" json:"title,omitempty"`
	Preferences    json.RawMessage `db:"preferences" json:"preferences,omitempty"`
	OnboardingDone bool            `db:"onboarding_done" json:"onboarding_done"`
}

type RegisterRequest struct {
	Email       string `json:"email"`
	Account     string `json:"account"`
	Password    string `json:"password" binding:"required,min=6"`
	DisplayName string `json:"display_name"`
	Code        string `json:"code"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken *string `json:"refresh_token"`
	AllDevices   bool    `json:"all_devices"`
}

type UpdateUserRequest struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	Timezone    *string `json:"timezone"`
	Locale      *string `json:"locale"`
}

type UpdateProfileRequest struct {
	Bio            *string          `json:"bio"`
	Company        *string          `json:"company"`
	Title          *string          `json:"title"`
	Preferences    *json.RawMessage `json:"preferences"`
	OnboardingDone *bool            `json:"onboarding_done"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         User   `json:"user"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// ──────────────────────────── 简报运行日志 ────────────────────────────

type BriefingRunLog struct {
	ID           int64           `db:"id" json:"id"`
	RunID        string          `db:"run_id" json:"run_id"`
	RunType      string          `db:"run_type" json:"run_type"`
	StartedAt    time.Time       `db:"started_at" json:"started_at"`
	CompletedAt  *time.Time      `db:"completed_at" json:"completed_at,omitempty"`
	UsersTotal   int             `db:"users_total" json:"users_total"`
	UsersSuccess int             `db:"users_success" json:"users_success"`
	UsersFailed  int             `db:"users_failed" json:"users_failed"`
	TotalTokens  int             `db:"total_tokens" json:"total_tokens"`
	TotalCost    float64         `db:"total_cost" json:"total_cost"`
	Failures     json.RawMessage `db:"failures" json:"failures"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
}

type CreateBriefingRunLogRequest struct {
	RunID        string          `json:"run_id" binding:"required"`
	RunType      string          `json:"run_type"`
	StartedAt    time.Time       `json:"started_at"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	UsersTotal   int             `json:"users_total"`
	UsersSuccess int             `json:"users_success"`
	UsersFailed  int             `json:"users_failed"`
	TotalTokens  int             `json:"total_tokens"`
	TotalCost    float64         `json:"total_cost"`
	Failures     json.RawMessage `json:"failures"`
}

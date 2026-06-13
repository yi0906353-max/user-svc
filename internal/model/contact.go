package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Contact struct {
	ID               uuid.UUID       `db:"id" json:"id"`
	UserID           uuid.UUID       `db:"user_id" json:"user_id"`
	Name             string          `db:"name" json:"name"`
	Email            *string         `db:"email" json:"email,omitempty"`
	Phone            *string         `db:"phone" json:"phone,omitempty"`
	Company          *string         `db:"company" json:"company,omitempty"`
	Title            *string         `db:"title" json:"title,omitempty"`
	AvatarURL        *string         `db:"avatar_url" json:"avatar_url,omitempty"`
	Source           *string         `db:"source" json:"source,omitempty"`
	SourceID         *string         `db:"source_id" json:"source_id,omitempty"`
	IsFrequent       bool            `db:"is_frequent" json:"is_frequent"`
	InteractionCount int             `db:"interaction_count" json:"interaction_count"`
	LastInteractedAt *time.Time      `db:"last_interacted_at" json:"last_interacted_at,omitempty"`
	Tags             json.RawMessage `db:"tags" json:"tags"`
	Notes            *string         `db:"notes" json:"notes,omitempty"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at" json:"updated_at"`
}

type CreateContactRequest struct {
	Name     string   `json:"name" binding:"required"`
	Email    *string  `json:"email"`
	Phone    *string  `json:"phone"`
	Company  *string  `json:"company"`
	Title    *string  `json:"title"`
	Source   *string  `json:"source"`
	SourceID *string  `json:"source_id"`
	Tags     []string `json:"tags"`
	Notes    *string  `json:"notes"`
}

type UpdateContactRequest struct {
	Name    *string  `json:"name"`
	Email   *string  `json:"email"`
	Phone   *string  `json:"phone"`
	Company *string  `json:"company"`
	Title   *string  `json:"title"`
	Tags    []string `json:"tags"`
	Notes   *string  `json:"notes"`
}

type ListContactsQuery struct {
	Search     string `form:"search"`
	Source     string `form:"source"`
	IsFrequent *bool  `form:"is_frequent"`
	Cursor     string `form:"cursor"`
	Limit      int    `form:"limit,default=20"`
}

type FrequentContactResponse struct {
	IsFrequent bool      `json:"is_frequent"`
	Contact    *Contact  `json:"contact"`
}

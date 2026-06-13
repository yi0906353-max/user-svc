package store

import (
	"time"

	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/model"
)

// UserStoreInterface 定义用户存储操作
type UserStoreInterface interface {
	Create(user *model.User) error
	CreateProfile(userID uuid.UUID) error
	GetByEmail(email string) (*model.User, error)
	GetByID(id uuid.UUID) (*model.User, error)
	GetProfile(userID uuid.UUID) (*model.UserProfile, error)
	Update(userID uuid.UUID, req *model.UpdateUserRequest) error
	UpdateProfile(userID uuid.UUID, req *model.UpdateProfileRequest) error
	ListActive() ([]model.User, error)
	CreateBriefingRunLog(log *model.BriefingRunLog) error
	ListBriefingRunLogs(limit int) ([]model.BriefingRunLog, error)
}

// TokenStoreInterface 定义 token 存储操作
type TokenStoreInterface interface {
	StoreRefreshToken(userID uuid.UUID, token string, deviceInfo *string, ipAddress *string, expiresAt time.Time) error
	GetRefreshToken(token string) (*model.RefreshToken, error)
	RevokeRefreshToken(token string) error
	RevokeAllForUser(userID uuid.UUID) error
}

// ContactStoreInterface 定义联系人存储操作
type ContactStoreInterface interface {
	Create(contact *model.Contact) error
	GetByID(id uuid.UUID) (*model.Contact, error)
	List(userID uuid.UUID, q *model.ListContactsQuery) ([]model.Contact, int64, error)
	GetFrequent(userID uuid.UUID, limit int) ([]model.Contact, error)
	Update(id uuid.UUID, req *model.UpdateContactRequest) error
	Delete(id uuid.UUID) error
	FindByEmail(userID uuid.UUID, email string) (*model.Contact, error)
	IncrementInteraction(id uuid.UUID) error
}

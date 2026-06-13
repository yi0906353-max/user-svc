package store

import (
	"time"

	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/model"
)

// ──────────────────────────── MockUserStore ────────────────────────────

type MockUserStore struct {
	CreateFn             func(user *model.User) error
	CreateProfileFn      func(userID uuid.UUID) error
	GetByEmailFn         func(email string) (*model.User, error)
	GetByIDFn            func(id uuid.UUID) (*model.User, error)
	GetProfileFn         func(userID uuid.UUID) (*model.UserProfile, error)
	UpdateFn             func(userID uuid.UUID, req *model.UpdateUserRequest) error
	UpdateProfileFn      func(userID uuid.UUID, req *model.UpdateProfileRequest) error
	ListActiveFn         func() ([]model.User, error)
	CreateBriefingRunLogFn  func(log *model.BriefingRunLog) error
	ListBriefingRunLogsFn   func(limit int) ([]model.BriefingRunLog, error)
}

func (m *MockUserStore) Create(user *model.User) error                   { return m.CreateFn(user) }
func (m *MockUserStore) CreateProfile(userID uuid.UUID) error            { return m.CreateProfileFn(userID) }
func (m *MockUserStore) GetByEmail(email string) (*model.User, error)    { return m.GetByEmailFn(email) }
func (m *MockUserStore) GetByID(id uuid.UUID) (*model.User, error)      { return m.GetByIDFn(id) }
func (m *MockUserStore) GetProfile(userID uuid.UUID) (*model.UserProfile, error) {
	return m.GetProfileFn(userID)
}
func (m *MockUserStore) Update(userID uuid.UUID, req *model.UpdateUserRequest) error {
	return m.UpdateFn(userID, req)
}
func (m *MockUserStore) UpdateProfile(userID uuid.UUID, req *model.UpdateProfileRequest) error {
	return m.UpdateProfileFn(userID, req)
}
func (m *MockUserStore) ListActive() ([]model.User, error) { return m.ListActiveFn() }
func (m *MockUserStore) CreateBriefingRunLog(log *model.BriefingRunLog) error {
	return m.CreateBriefingRunLogFn(log)
}
func (m *MockUserStore) ListBriefingRunLogs(limit int) ([]model.BriefingRunLog, error) {
	return m.ListBriefingRunLogsFn(limit)
}

// ──────────────────────────── MockTokenStore ────────────────────────────

type MockTokenStore struct {
	StoreRefreshTokenFn  func(userID uuid.UUID, token string, deviceInfo *string, ipAddress *string, expiresAt time.Time) error
	GetRefreshTokenFn    func(token string) (*model.RefreshToken, error)
	RevokeRefreshTokenFn func(token string) error
	RevokeAllForUserFn   func(userID uuid.UUID) error
}

func (m *MockTokenStore) StoreRefreshToken(userID uuid.UUID, token string, deviceInfo *string, ipAddress *string, expiresAt time.Time) error {
	return m.StoreRefreshTokenFn(userID, token, deviceInfo, ipAddress, expiresAt)
}
func (m *MockTokenStore) GetRefreshToken(token string) (*model.RefreshToken, error) {
	return m.GetRefreshTokenFn(token)
}
func (m *MockTokenStore) RevokeRefreshToken(token string) error { return m.RevokeRefreshTokenFn(token) }
func (m *MockTokenStore) RevokeAllForUser(userID uuid.UUID) error {
	return m.RevokeAllForUserFn(userID)
}

// ──────────────────────────── MockContactStore ────────────────────────────

type MockContactStore struct {
	CreateFn             func(contact *model.Contact) error
	GetByIDFn            func(id uuid.UUID) (*model.Contact, error)
	ListFn               func(userID uuid.UUID, q *model.ListContactsQuery) ([]model.Contact, int64, error)
	GetFrequentFn        func(userID uuid.UUID, limit int) ([]model.Contact, error)
	UpdateFn             func(id uuid.UUID, req *model.UpdateContactRequest) error
	DeleteFn             func(id uuid.UUID) error
	FindByEmailFn        func(userID uuid.UUID, email string) (*model.Contact, error)
	IncrementInteractionFn func(id uuid.UUID) error
}

func (m *MockContactStore) Create(contact *model.Contact) error { return m.CreateFn(contact) }
func (m *MockContactStore) GetByID(id uuid.UUID) (*model.Contact, error) {
	return m.GetByIDFn(id)
}
func (m *MockContactStore) List(userID uuid.UUID, q *model.ListContactsQuery) ([]model.Contact, int64, error) {
	return m.ListFn(userID, q)
}
func (m *MockContactStore) GetFrequent(userID uuid.UUID, limit int) ([]model.Contact, error) {
	return m.GetFrequentFn(userID, limit)
}
func (m *MockContactStore) Update(id uuid.UUID, req *model.UpdateContactRequest) error {
	return m.UpdateFn(id, req)
}
func (m *MockContactStore) Delete(id uuid.UUID) error       { return m.DeleteFn(id) }
func (m *MockContactStore) FindByEmail(userID uuid.UUID, email string) (*model.Contact, error) {
	return m.FindByEmailFn(userID, email)
}
func (m *MockContactStore) IncrementInteraction(id uuid.UUID) error {
	return m.IncrementInteractionFn(id)
}

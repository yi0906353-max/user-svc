package service

import (
	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/model"
	"github.com/mindpilot/user-svc/internal/store"
)

type UserService struct {
	users store.UserStoreInterface
}

func NewUserService(users store.UserStoreInterface) *UserService {
	return &UserService{users: users}
}

func (s *UserService) GetByID(id uuid.UUID) (*model.User, error) {
	return s.users.GetByID(id)
}

func (s *UserService) GetProfile(userID uuid.UUID) (*model.UserProfile, error) {
	return s.users.GetProfile(userID)
}

func (s *UserService) Update(userID uuid.UUID, req *model.UpdateUserRequest) error {
	return s.users.Update(userID, req)
}

func (s *UserService) UpdateProfile(userID uuid.UUID, req *model.UpdateProfileRequest) error {
	return s.users.UpdateProfile(userID, req)
}

func (s *UserService) ListActive() ([]model.User, error) {
	return s.users.ListActive()
}

func (s *UserService) CreateBriefingRunLog(log *model.BriefingRunLog) error {
	return s.users.CreateBriefingRunLog(log)
}

func (s *UserService) ListBriefingRunLogs(limit int) ([]model.BriefingRunLog, error) {
	return s.users.ListBriefingRunLogs(limit)
}

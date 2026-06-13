package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/model"
	"github.com/mindpilot/user-svc/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestUserService_GetByID(t *testing.T) {
	uid := uuid.New()
	users := &store.MockUserStore{
		GetByIDFn: func(id uuid.UUID) (*model.User, error) {
			if id == uid {
				return &model.User{ID: id, Email: "test@example.com"}, nil
			}
			return nil, nil
		},
	}

	svc := NewUserService(users)
	user, err := svc.GetByID(uid)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	users := &store.MockUserStore{
		GetByIDFn: func(id uuid.UUID) (*model.User, error) { return nil, nil },
	}

	svc := NewUserService(users)
	user, err := svc.GetByID(uuid.New())

	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestUserService_GetProfile(t *testing.T) {
	uid := uuid.New()
	bio := "hello world"
	users := &store.MockUserStore{
		GetProfileFn: func(id uuid.UUID) (*model.UserProfile, error) {
			return &model.UserProfile{
				User: model.User{ID: id, Email: "test@example.com"},
				Bio:  &bio,
			}, nil
		},
	}

	svc := NewUserService(users)
	profile, err := svc.GetProfile(uid)

	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, "hello world", *profile.Bio)
}

func TestUserService_Update(t *testing.T) {
	var calledWith uuid.UUID
	users := &store.MockUserStore{
		UpdateFn: func(userID uuid.UUID, req *model.UpdateUserRequest) error {
			calledWith = userID
			return nil
		},
	}

	uid := uuid.New()
	svc := NewUserService(users)
	name := "New Name"
	err := svc.Update(uid, &model.UpdateUserRequest{DisplayName: &name})

	assert.NoError(t, err)
	assert.Equal(t, uid, calledWith)
}

func TestUserService_UpdateProfile(t *testing.T) {
	var calledWith uuid.UUID
	users := &store.MockUserStore{
		UpdateProfileFn: func(userID uuid.UUID, req *model.UpdateProfileRequest) error {
			calledWith = userID
			return nil
		},
	}

	uid := uuid.New()
	svc := NewUserService(users)
	bio := "new bio"
	err := svc.UpdateProfile(uid, &model.UpdateProfileRequest{Bio: &bio})

	assert.NoError(t, err)
	assert.Equal(t, uid, calledWith)
}

func TestUserService_ListActive(t *testing.T) {
	users := &store.MockUserStore{
		ListActiveFn: func() ([]model.User, error) {
			return []model.User{
				{ID: uuid.New(), Email: "a@test.com", Status: "active"},
				{ID: uuid.New(), Email: "b@test.com", Status: "active"},
			}, nil
		},
	}

	svc := NewUserService(users)
	list, err := svc.ListActive()

	assert.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestUserService_CreateBriefingRunLog(t *testing.T) {
	var saved *model.BriefingRunLog
	users := &store.MockUserStore{
		CreateBriefingRunLogFn: func(log *model.BriefingRunLog) error {
			saved = log
			log.ID = 42
			return nil
		},
	}

	svc := NewUserService(users)
	log := &model.BriefingRunLog{RunID: "run-1", RunType: "cron"}
	err := svc.CreateBriefingRunLog(log)

	assert.NoError(t, err)
	assert.Equal(t, int64(42), log.ID)
	assert.Equal(t, "run-1", saved.RunID)
}

func TestUserService_ListBriefingRunLogs(t *testing.T) {
	users := &store.MockUserStore{
		ListBriefingRunLogsFn: func(limit int) ([]model.BriefingRunLog, error) {
			return []model.BriefingRunLog{
				{ID: 1, RunID: "run-1"},
				{ID: 2, RunID: "run-2"},
			}, nil
		},
	}

	svc := NewUserService(users)
	logs, err := svc.ListBriefingRunLogs(20)

	assert.NoError(t, err)
	assert.Len(t, logs, 2)
}

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/model"
	"github.com/mindpilot/user-svc/internal/service"
	"github.com/mindpilot/user-svc/internal/store"
	"github.com/stretchr/testify/assert"
)

func setupUserHandler(users *store.MockUserStore) (*UserHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	svc := service.NewUserService(users)
	h := NewUserHandler(svc)

	r := gin.New()
	// 模拟 JWT 中间件设置 user_id
	r.GET("/users/me", func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-Test-User-ID"))
		h.GetMe(c)
	})
	r.PATCH("/users/me", func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-Test-User-ID"))
		h.UpdateMe(c)
	})
	r.GET("/users/me/profile", func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-Test-User-ID"))
		h.GetProfile(c)
	})
	r.PATCH("/users/me/profile", func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-Test-User-ID"))
		h.UpdateProfile(c)
	})
	r.GET("/internal/users", func(c *gin.Context) { h.ListUsers(c) })
	r.GET("/internal/users/:user_id", func(c *gin.Context) { h.GetUserInternal(c) })
	r.POST("/internal/briefing-run-logs", func(c *gin.Context) { h.CreateBriefingRunLog(c) })
	r.GET("/internal/briefing-run-logs", func(c *gin.Context) { h.ListBriefingRunLogs(c) })

	return h, r
}

// ──────────────────────────── GetMe ────────────────────────────

func TestUserHandler_GetMe_Success(t *testing.T) {
	uid := uuid.New()
	users := &store.MockUserStore{
		GetByIDFn: func(id uuid.UUID) (*model.User, error) {
			return &model.User{ID: id, Email: "test@test.com", Status: "active"}, nil
		},
	}
	_, r := setupUserHandler(users)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("X-Test-User-ID", uid.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test@test.com")
}

func TestUserHandler_GetMe_InvalidUUID(t *testing.T) {
	_, r := setupUserHandler(&store.MockUserStore{})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("X-Test-User-ID", "not-a-uuid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_GetMe_NotFound(t *testing.T) {
	users := &store.MockUserStore{
		GetByIDFn: func(id uuid.UUID) (*model.User, error) { return nil, nil },
	}
	_, r := setupUserHandler(users)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ──────────────────────────── UpdateMe ────────────────────────────

func TestUserHandler_UpdateMe_Success(t *testing.T) {
	uid := uuid.New()
	users := &store.MockUserStore{
		UpdateFn: func(userID uuid.UUID, req *model.UpdateUserRequest) error { return nil },
		GetByIDFn: func(id uuid.UUID) (*model.User, error) {
			return &model.User{ID: id, Email: "test@test.com", DisplayName: "Updated"}, nil
		},
	}
	_, r := setupUserHandler(users)

	body, _ := json.Marshal(map[string]string{"display_name": "Updated"})
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uid.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Updated")
}

// ──────────────────────────── GetProfile ────────────────────────────

func TestUserHandler_GetProfile_Success(t *testing.T) {
	uid := uuid.New()
	bio := "test bio"
	users := &store.MockUserStore{
		GetProfileFn: func(id uuid.UUID) (*model.UserProfile, error) {
			return &model.UserProfile{
				User: model.User{ID: id, Email: "test@test.com"},
				Bio:  &bio,
			}, nil
		},
	}
	_, r := setupUserHandler(users)

	req := httptest.NewRequest(http.MethodGet, "/users/me/profile", nil)
	req.Header.Set("X-Test-User-ID", uid.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test bio")
}

func TestUserHandler_GetProfile_NotFound(t *testing.T) {
	users := &store.MockUserStore{
		GetProfileFn: func(id uuid.UUID) (*model.UserProfile, error) { return nil, nil },
	}
	_, r := setupUserHandler(users)

	req := httptest.NewRequest(http.MethodGet, "/users/me/profile", nil)
	req.Header.Set("X-Test-User-ID", uuid.New().String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ──────────────────────────── UpdateProfile ────────────────────────────

func TestUserHandler_UpdateProfile_Success(t *testing.T) {
	uid := uuid.New()
	users := &store.MockUserStore{
		UpdateProfileFn: func(userID uuid.UUID, req *model.UpdateProfileRequest) error { return nil },
	}
	_, r := setupUserHandler(users)

	body, _ := json.Marshal(map[string]string{"bio": "new bio"})
	req := httptest.NewRequest(http.MethodPatch, "/users/me/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uid.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ──────────────────────────── ListUsers ────────────────────────────

func TestUserHandler_ListUsers_Success(t *testing.T) {
	users := &store.MockUserStore{
		ListActiveFn: func() ([]model.User, error) {
			return []model.User{
				{ID: uuid.New(), Email: "a@test.com", Status: "active"},
				{ID: uuid.New(), Email: "b@test.com", Status: "active"},
			}, nil
		},
	}
	_, r := setupUserHandler(users)

	req := httptest.NewRequest(http.MethodGet, "/internal/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "a@test.com")
}

// ──────────────────────────── GetUserInternal ────────────────────────────

func TestUserHandler_GetUserInternal_Success(t *testing.T) {
	uid := uuid.New()
	users := &store.MockUserStore{
		GetByIDFn: func(id uuid.UUID) (*model.User, error) {
			return &model.User{ID: id, Email: "internal@test.com"}, nil
		},
	}
	_, r := setupUserHandler(users)

	req := httptest.NewRequest(http.MethodGet, "/internal/users/"+uid.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "internal@test.com")
}

func TestUserHandler_GetUserInternal_InvalidUUID(t *testing.T) {
	_, r := setupUserHandler(&store.MockUserStore{})

	req := httptest.NewRequest(http.MethodGet, "/internal/users/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ──────────────────────────── BriefingRunLogs ────────────────────────────

func TestUserHandler_CreateBriefingRunLog_Success(t *testing.T) {
	users := &store.MockUserStore{
		CreateBriefingRunLogFn: func(log *model.BriefingRunLog) error {
			log.ID = 1
			return nil
		},
	}
	_, r := setupUserHandler(users)

	body, _ := json.Marshal(map[string]interface{}{
		"run_id":        "run-123",
		"run_type":      "cron",
		"users_total":   10,
		"users_success": 9,
		"users_failed":  1,
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/briefing-run-logs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "run-123")
}

func TestUserHandler_CreateBriefingRunLog_BadRequest(t *testing.T) {
	_, r := setupUserHandler(&store.MockUserStore{})

	body, _ := json.Marshal(map[string]interface{}{}) // 缺少 run_id
	req := httptest.NewRequest(http.MethodPost, "/internal/briefing-run-logs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_ListBriefingRunLogs_Success(t *testing.T) {
	users := &store.MockUserStore{
		ListBriefingRunLogsFn: func(limit int) ([]model.BriefingRunLog, error) {
			return []model.BriefingRunLog{
				{ID: 1, RunID: "run-1"},
				{ID: 2, RunID: "run-2"},
			}, nil
		},
	}
	_, r := setupUserHandler(users)

	req := httptest.NewRequest(http.MethodGet, "/internal/briefing-run-logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "run-1")
}

package handler

import (
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

func setupInternalHandler(contacts *store.MockContactStore) (*InternalHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	svc := service.NewContactService(contacts)
	h := NewInternalHandler(svc)

	r := gin.New()
	r.GET("/internal/users/:user_id/frequent-contacts", h.FrequentContactCheck)

	return h, r
}

func TestInternalHandler_FrequentContactCheck_FoundFrequent(t *testing.T) {
	uid := uuid.New()
	contacts := &store.MockContactStore{
		FindByEmailFn: func(userID uuid.UUID, email string) (*model.Contact, error) {
			return &model.Contact{
				ID:         uuid.New(),
				UserID:     userID,
				Name:       "Frequent Contact",
				IsFrequent: true,
			}, nil
		},
	}
	_, r := setupInternalHandler(contacts)

	req := httptest.NewRequest(http.MethodGet, "/internal/users/"+uid.String()+"/frequent-contacts?email=test@example.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"is_frequent":true`)
	assert.Contains(t, w.Body.String(), "Frequent Contact")
}

func TestInternalHandler_FrequentContactCheck_FoundNotFrequent(t *testing.T) {
	uid := uuid.New()
	contacts := &store.MockContactStore{
		FindByEmailFn: func(userID uuid.UUID, email string) (*model.Contact, error) {
			return &model.Contact{
				ID:         uuid.New(),
				UserID:     userID,
				Name:       "Normal Contact",
				IsFrequent: false,
			}, nil
		},
	}
	_, r := setupInternalHandler(contacts)

	req := httptest.NewRequest(http.MethodGet, "/internal/users/"+uid.String()+"/frequent-contacts?email=test@example.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"is_frequent":false`)
}

func TestInternalHandler_FrequentContactCheck_NotFound(t *testing.T) {
	uid := uuid.New()
	contacts := &store.MockContactStore{
		FindByEmailFn: func(userID uuid.UUID, email string) (*model.Contact, error) {
			return nil, nil
		},
	}
	_, r := setupInternalHandler(contacts)

	req := httptest.NewRequest(http.MethodGet, "/internal/users/"+uid.String()+"/frequent-contacts?email=nobody@example.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"is_frequent":false`)
	assert.Contains(t, w.Body.String(), `"contact":null`)
}

func TestInternalHandler_FrequentContactCheck_InvalidUserID(t *testing.T) {
	_, r := setupInternalHandler(&store.MockContactStore{})

	req := httptest.NewRequest(http.MethodGet, "/internal/users/not-a-uuid/frequent-contacts?email=test@example.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInternalHandler_FrequentContactCheck_MissingEmail(t *testing.T) {
	uid := uuid.New()
	_, r := setupInternalHandler(&store.MockContactStore{})

	req := httptest.NewRequest(http.MethodGet, "/internal/users/"+uid.String()+"/frequent-contacts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

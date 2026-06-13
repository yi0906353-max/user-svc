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

func setupContactHandler(contacts *store.MockContactStore) (*ContactHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	svc := service.NewContactService(contacts)
	h := NewContactHandler(svc)

	r := gin.New()
	r.GET("/contacts", func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-Test-User-ID"))
		h.List(c)
	})
	r.POST("/contacts", func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-Test-User-ID"))
		h.Create(c)
	})
	r.GET("/contacts/:contact_id", func(c *gin.Context) { h.Get(c) })
	r.PATCH("/contacts/:contact_id", func(c *gin.Context) { h.Update(c) })
	r.DELETE("/contacts/:contact_id", func(c *gin.Context) { h.Delete(c) })
	r.GET("/contacts/frequent", func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-Test-User-ID"))
		h.GetFrequent(c)
	})

	return h, r
}

// ──────────────────────────── List ────────────────────────────

func TestContactHandler_List_Success(t *testing.T) {
	uid := uuid.New()
	contacts := &store.MockContactStore{
		ListFn: func(userID uuid.UUID, q *model.ListContactsQuery) ([]model.Contact, int64, error) {
			return []model.Contact{
				{ID: uuid.New(), Name: "Alice"},
				{ID: uuid.New(), Name: "Bob"},
			}, 2, nil
		},
	}
	_, r := setupContactHandler(contacts)

	req := httptest.NewRequest(http.MethodGet, "/contacts?limit=10", nil)
	req.Header.Set("X-Test-User-ID", uid.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Alice")
	assert.Contains(t, w.Body.String(), "Bob")
}

func TestContactHandler_List_Unauthorized(t *testing.T) {
	_, r := setupContactHandler(&store.MockContactStore{})

	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	req.Header.Set("X-Test-User-ID", "not-a-uuid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ──────────────────────────── Create ────────────────────────────

func TestContactHandler_Create_Success(t *testing.T) {
	uid := uuid.New()
	contacts := &store.MockContactStore{
		CreateFn: func(contact *model.Contact) error { return nil },
		GetByIDFn: func(id uuid.UUID) (*model.Contact, error) {
			return &model.Contact{ID: id, Name: "New Contact"}, nil
		},
	}
	_, r := setupContactHandler(contacts)

	body, _ := json.Marshal(map[string]string{"name": "New Contact"})
	req := httptest.NewRequest(http.MethodPost, "/contacts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uid.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "New Contact")
}

func TestContactHandler_Create_BadRequest(t *testing.T) {
	uid := uuid.New()
	_, r := setupContactHandler(&store.MockContactStore{})

	body, _ := json.Marshal(map[string]string{}) // 缺少 name
	req := httptest.NewRequest(http.MethodPost, "/contacts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", uid.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ──────────────────────────── Get ────────────────────────────

func TestContactHandler_Get_Success(t *testing.T) {
	cid := uuid.New()
	contacts := &store.MockContactStore{
		GetByIDFn: func(id uuid.UUID) (*model.Contact, error) {
			return &model.Contact{ID: id, Name: "Found Contact"}, nil
		},
	}
	_, r := setupContactHandler(contacts)

	req := httptest.NewRequest(http.MethodGet, "/contacts/"+cid.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Found Contact")
}

func TestContactHandler_Get_NotFound(t *testing.T) {
	contacts := &store.MockContactStore{
		GetByIDFn: func(id uuid.UUID) (*model.Contact, error) { return nil, nil },
	}
	_, r := setupContactHandler(contacts)

	req := httptest.NewRequest(http.MethodGet, "/contacts/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestContactHandler_Get_InvalidID(t *testing.T) {
	_, r := setupContactHandler(&store.MockContactStore{})

	req := httptest.NewRequest(http.MethodGet, "/contacts/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ──────────────────────────── Update ────────────────────────────

func TestContactHandler_Update_Success(t *testing.T) {
	cid := uuid.New()
	contacts := &store.MockContactStore{
		UpdateFn: func(id uuid.UUID, req *model.UpdateContactRequest) error { return nil },
	}
	_, r := setupContactHandler(contacts)

	body, _ := json.Marshal(map[string]string{"name": "Updated Name"})
	req := httptest.NewRequest(http.MethodPatch, "/contacts/"+cid.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ──────────────────────────── Delete ────────────────────────────

func TestContactHandler_Delete_Success(t *testing.T) {
	cid := uuid.New()
	contacts := &store.MockContactStore{
		DeleteFn: func(id uuid.UUID) error { return nil },
	}
	_, r := setupContactHandler(contacts)

	req := httptest.NewRequest(http.MethodDelete, "/contacts/"+cid.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestContactHandler_Delete_InvalidID(t *testing.T) {
	_, r := setupContactHandler(&store.MockContactStore{})

	req := httptest.NewRequest(http.MethodDelete, "/contacts/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ──────────────────────────── GetFrequent ────────────────────────────

func TestContactHandler_GetFrequent_Success(t *testing.T) {
	uid := uuid.New()
	contacts := &store.MockContactStore{
		GetFrequentFn: func(userID uuid.UUID, limit int) ([]model.Contact, error) {
			return []model.Contact{
				{ID: uuid.New(), Name: "Frequent1", IsFrequent: true},
			}, nil
		},
	}
	_, r := setupContactHandler(contacts)

	req := httptest.NewRequest(http.MethodGet, "/contacts/frequent?limit=10", nil)
	req.Header.Set("X-Test-User-ID", uid.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Frequent1")
}

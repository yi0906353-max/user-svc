package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/model"
	"github.com/mindpilot/user-svc/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestContactService_Create(t *testing.T) {
	var savedContact *model.Contact
	contacts := &store.MockContactStore{
		CreateFn: func(contact *model.Contact) error {
			savedContact = contact
			return nil
		},
		GetByIDFn: func(id uuid.UUID) (*model.Contact, error) {
			return &model.Contact{
				ID:     id,
				UserID: savedContact.UserID,
				Name:   "Alice",
			}, nil
		},
	}

	svc := NewContactService(contacts)
	email := "alice@example.com"
	result, err := svc.Create(uuid.New(), &model.CreateContactRequest{
		Name:  "Alice",
		Email: &email,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Alice", result.Name)
	assert.NotNil(t, savedContact)
	assert.Equal(t, "alice@example.com", *savedContact.Email)
}

func TestContactService_Create_WithTags(t *testing.T) {
	var savedContact *model.Contact
	contacts := &store.MockContactStore{
		CreateFn: func(contact *model.Contact) error {
			savedContact = contact
			return nil
		},
		GetByIDFn: func(id uuid.UUID) (*model.Contact, error) {
			return savedContact, nil
		},
	}

	svc := NewContactService(contacts)
	result, err := svc.Create(uuid.New(), &model.CreateContactRequest{
		Name: "Bob",
		Tags: []string{"vip", "partner"},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, string(savedContact.Tags), "vip")
	assert.Contains(t, string(savedContact.Tags), "partner")
}

func TestContactService_GetByID(t *testing.T) {
	cid := uuid.New()
	contacts := &store.MockContactStore{
		GetByIDFn: func(id uuid.UUID) (*model.Contact, error) {
			return &model.Contact{ID: id, Name: "Alice"}, nil
		},
	}

	svc := NewContactService(contacts)
	contact, err := svc.GetByID(cid)

	assert.NoError(t, err)
	assert.Equal(t, "Alice", contact.Name)
}

func TestContactService_List(t *testing.T) {
	uid := uuid.New()
	contacts := &store.MockContactStore{
		ListFn: func(userID uuid.UUID, q *model.ListContactsQuery) ([]model.Contact, int64, error) {
			return []model.Contact{
				{ID: uuid.New(), Name: "A"},
				{ID: uuid.New(), Name: "B"},
			}, 2, nil
		},
	}

	svc := NewContactService(contacts)
	list, total, err := svc.List(uid, &model.ListContactsQuery{Limit: 20})

	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, list, 2)
}

func TestContactService_GetFrequent(t *testing.T) {
	uid := uuid.New()
	contacts := &store.MockContactStore{
		GetFrequentFn: func(userID uuid.UUID, limit int) ([]model.Contact, error) {
			return []model.Contact{
				{ID: uuid.New(), Name: "Frequent1", IsFrequent: true},
			}, nil
		},
	}

	svc := NewContactService(contacts)
	list, err := svc.GetFrequent(uid, 10)

	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.True(t, list[0].IsFrequent)
}

func TestContactService_Update(t *testing.T) {
	var calledID uuid.UUID
	contacts := &store.MockContactStore{
		UpdateFn: func(id uuid.UUID, req *model.UpdateContactRequest) error {
			calledID = id
			return nil
		},
	}

	cid := uuid.New()
	svc := NewContactService(contacts)
	name := "Updated Name"
	err := svc.Update(cid, &model.UpdateContactRequest{Name: &name})

	assert.NoError(t, err)
	assert.Equal(t, cid, calledID)
}

func TestContactService_Delete(t *testing.T) {
	var calledID uuid.UUID
	contacts := &store.MockContactStore{
		DeleteFn: func(id uuid.UUID) error {
			calledID = id
			return nil
		},
	}

	cid := uuid.New()
	svc := NewContactService(contacts)
	err := svc.Delete(cid)

	assert.NoError(t, err)
	assert.Equal(t, cid, calledID)
}

func TestContactService_FindByEmail(t *testing.T) {
	uid := uuid.New()
	contacts := &store.MockContactStore{
		FindByEmailFn: func(userID uuid.UUID, email string) (*model.Contact, error) {
			return &model.Contact{ID: uuid.New(), UserID: userID, Name: "Found"}, nil
		},
	}

	svc := NewContactService(contacts)
	contact, err := svc.FindByEmail(uid, "test@example.com")

	assert.NoError(t, err)
	assert.NotNil(t, contact)
	assert.Equal(t, "Found", contact.Name)
}

func TestContactService_FindByEmail_NotFound(t *testing.T) {
	contacts := &store.MockContactStore{
		FindByEmailFn: func(userID uuid.UUID, email string) (*model.Contact, error) {
			return nil, nil
		},
	}

	svc := NewContactService(contacts)
	contact, err := svc.FindByEmail(uuid.New(), "nobody@example.com")

	assert.NoError(t, err)
	assert.Nil(t, contact)
}

func TestContactService_IncrementInteraction(t *testing.T) {
	var calledID uuid.UUID
	contacts := &store.MockContactStore{
		IncrementInteractionFn: func(id uuid.UUID) error {
			calledID = id
			return nil
		},
	}

	cid := uuid.New()
	svc := NewContactService(contacts)
	err := svc.IncrementInteraction(cid)

	assert.NoError(t, err)
	assert.Equal(t, cid, calledID)
}

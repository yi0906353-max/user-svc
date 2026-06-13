package service

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/mindpilot/user-svc/internal/model"
	"github.com/mindpilot/user-svc/internal/store"
)

type ContactService struct {
	contacts store.ContactStoreInterface
}

func NewContactService(contacts store.ContactStoreInterface) *ContactService {
	return &ContactService{contacts: contacts}
}

func (s *ContactService) Create(userID uuid.UUID, req *model.CreateContactRequest) (*model.Contact, error) {
	contact := &model.Contact{
		ID:       uuid.New(),
		UserID:   userID,
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Company:  req.Company,
		Title:    req.Title,
		Source:   req.Source,
		SourceID: req.SourceID,
		Tags:     json.RawMessage("[]"),
		Notes:    req.Notes,
	}
	if req.Tags != nil {
		tagsJSON, _ := json.Marshal(req.Tags)
		contact.Tags = tagsJSON
	}

	if err := s.contacts.Create(contact); err != nil {
		return nil, err
	}
	return s.contacts.GetByID(contact.ID)
}

func (s *ContactService) GetByID(id uuid.UUID) (*model.Contact, error) {
	return s.contacts.GetByID(id)
}

func (s *ContactService) List(userID uuid.UUID, q *model.ListContactsQuery) ([]model.Contact, int64, error) {
	return s.contacts.List(userID, q)
}

func (s *ContactService) GetFrequent(userID uuid.UUID, limit int) ([]model.Contact, error) {
	return s.contacts.GetFrequent(userID, limit)
}

func (s *ContactService) Update(id uuid.UUID, req *model.UpdateContactRequest) error {
	return s.contacts.Update(id, req)
}

func (s *ContactService) Delete(id uuid.UUID) error {
	return s.contacts.Delete(id)
}

func (s *ContactService) FindByEmail(userID uuid.UUID, email string) (*model.Contact, error) {
	return s.contacts.FindByEmail(userID, email)
}

func (s *ContactService) IncrementInteraction(id uuid.UUID) error {
	return s.contacts.IncrementInteraction(id)
}

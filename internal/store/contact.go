package store

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/mindpilot/user-svc/internal/model"
)

type ContactStore struct {
	db *sqlx.DB
}

func NewContactStore(db *sqlx.DB) *ContactStore {
	return &ContactStore{db: db}
}

func (s *ContactStore) Create(contact *model.Contact) error {
	tagsJSON, _ := json.Marshal(contact.Tags)
	query := `
		INSERT INTO contacts (id, user_id, name, email, phone, company, title, avatar_url, source, source_id, tags, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := s.db.Exec(query,
		contact.ID, contact.UserID, contact.Name, contact.Email,
		contact.Phone, contact.Company, contact.Title, contact.AvatarURL,
		contact.Source, contact.SourceID, tagsJSON, contact.Notes,
	)
	return err
}

func (s *ContactStore) GetByID(id uuid.UUID) (*model.Contact, error) {
	var c model.Contact
	err := s.db.Get(&c, "SELECT * FROM contacts WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (s *ContactStore) List(userID uuid.UUID, q *model.ListContactsQuery) ([]model.Contact, int64, error) {
	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	conditions := []string{"user_id = $1"}
	args := []interface{}{userID}
	idx := 2

	if q.Search != "" {
		conditions = append(conditions, "(name ILIKE $"+strconv.Itoa(idx)+" OR email ILIKE $"+strconv.Itoa(idx)+" OR company ILIKE $"+strconv.Itoa(idx)+")")
		args = append(args, "%"+q.Search+"%")
		idx++
	}
	if q.Source != "" {
		conditions = append(conditions, "source = $"+strconv.Itoa(idx))
		args = append(args, q.Source)
		idx++
	}
	if q.IsFrequent != nil {
		conditions = append(conditions, "is_frequent = $"+strconv.Itoa(idx))
		args = append(args, *q.IsFrequent)
		idx++
	}

	where := strings.Join(conditions, " AND ")

	// count
	var total int64
	countSQL := "SELECT COUNT(*) FROM contacts WHERE " + where
	if err := s.db.Get(&total, countSQL, args...); err != nil {
		return nil, 0, err
	}

	// data
	dataSQL := "SELECT * FROM contacts WHERE " + where + " ORDER BY interaction_count DESC, created_at DESC LIMIT $" + strconv.Itoa(idx)
	args = append(args, limit+1)

	var contacts []model.Contact
	if err := s.db.Select(&contacts, dataSQL, args...); err != nil {
		return nil, 0, err
	}

	return contacts, total, nil
}

func (s *ContactStore) GetFrequent(userID uuid.UUID, limit int) ([]model.Contact, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var contacts []model.Contact
	err := s.db.Select(&contacts,
		"SELECT * FROM contacts WHERE user_id = $1 AND is_frequent = TRUE ORDER BY interaction_count DESC LIMIT $2",
		userID, limit,
	)
	return contacts, err
}

func (s *ContactStore) FindByEmail(userID uuid.UUID, email string) (*model.Contact, error) {
	var c model.Contact
	err := s.db.Get(&c,
		"SELECT * FROM contacts WHERE user_id = $1 AND email = $2",
		userID, email,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (s *ContactStore) IncrementInteraction(id uuid.UUID) error {
	query := `
		UPDATE contacts SET
			interaction_count = interaction_count + 1,
			last_interacted_at = $1,
			is_frequent = CASE WHEN interaction_count + 1 >= 10 THEN TRUE
			                   WHEN interaction_count + 1 < 5 THEN FALSE
			                   ELSE is_frequent END
		WHERE id = $2`
	_, err := s.db.Exec(query, time.Now(), id)
	return err
}

func (s *ContactStore) Update(id uuid.UUID, req *model.UpdateContactRequest) error {
	sets := []string{}
	args := []interface{}{}
	idx := 1

	if req.Name != nil {
		sets = append(sets, "name = $"+strconv.Itoa(idx))
		args = append(args, *req.Name)
		idx++
	}
	if req.Email != nil {
		sets = append(sets, "email = $"+strconv.Itoa(idx))
		args = append(args, *req.Email)
		idx++
	}
	if req.Phone != nil {
		sets = append(sets, "phone = $"+strconv.Itoa(idx))
		args = append(args, *req.Phone)
		idx++
	}
	if req.Company != nil {
		sets = append(sets, "company = $"+strconv.Itoa(idx))
		args = append(args, *req.Company)
		idx++
	}
	if req.Title != nil {
		sets = append(sets, "title = $"+strconv.Itoa(idx))
		args = append(args, *req.Title)
		idx++
	}
	if req.Tags != nil {
		tagsJSON, _ := json.Marshal(req.Tags)
		sets = append(sets, "tags = $"+strconv.Itoa(idx))
		args = append(args, tagsJSON)
		idx++
	}
	if req.Notes != nil {
		sets = append(sets, "notes = $"+strconv.Itoa(idx))
		args = append(args, *req.Notes)
		idx++
	}

	if len(sets) == 0 {
		return nil
	}

	query := "UPDATE contacts SET " + strings.Join(sets, ", ") + " WHERE id = $" + strconv.Itoa(idx)
	args = append(args, id)

	_, err := s.db.Exec(query, args...)
	return err
}

func (s *ContactStore) Delete(id uuid.UUID) error {
	_, err := s.db.Exec("DELETE FROM contacts WHERE id = $1", id)
	return err
}

func (s *ContactStore) BatchCreate(contacts []model.Contact) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO contacts (id, user_id, name, email, phone, company, title, avatar_url, source, source_id, tags, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (user_id, source, source_id) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range contacts {
		tagsJSON, _ := json.Marshal(c.Tags)
		if _, err := stmt.Exec(c.ID, c.UserID, c.Name, c.Email, c.Phone, c.Company, c.Title, c.AvatarURL, c.Source, c.SourceID, tagsJSON, c.Notes); err != nil {
			return err
		}
	}

	return tx.Commit()
}

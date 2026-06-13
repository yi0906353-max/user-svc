package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/mindpilot/user-svc/internal/model"
)

type TokenStore struct {
	db *sqlx.DB
}

func NewTokenStore(db *sqlx.DB) *TokenStore {
	return &TokenStore{db: db}
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *TokenStore) StoreRefreshToken(userID uuid.UUID, token string, deviceInfo *string, ipAddress *string, expiresAt time.Time) error {
	hash := HashToken(token)
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, device_info, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := s.db.Exec(query, uuid.New(), userID, hash, deviceInfo, ipAddress, expiresAt)
	return err
}

func (s *TokenStore) GetRefreshToken(token string) (*model.RefreshToken, error) {
	hash := HashToken(token)
	var rt model.RefreshToken
	err := s.db.Get(&rt, "SELECT * FROM refresh_tokens WHERE token_hash = $1", hash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &rt, err
}

func (s *TokenStore) RevokeRefreshToken(token string) error {
	hash := HashToken(token)
	_, err := s.db.Exec("DELETE FROM refresh_tokens WHERE token_hash = $1", hash)
	return err
}

func (s *TokenStore) RevokeAllForUser(userID uuid.UUID) error {
	_, err := s.db.Exec("DELETE FROM refresh_tokens WHERE user_id = $1", userID)
	return err
}

func (s *TokenStore) CleanupExpired() error {
	_, err := s.db.Exec("DELETE FROM refresh_tokens WHERE expires_at < NOW()")
	return err
}

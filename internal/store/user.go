package store

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/mindpilot/user-svc/internal/model"
)

type UserStore struct {
	db *sqlx.DB
}

func NewUserStore(db *sqlx.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, display_name, avatar_url, timezone, locale, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := s.db.Exec(query,
		user.ID, user.Email, user.PasswordHash, user.DisplayName,
		user.AvatarURL, user.Timezone, user.Locale, user.Status,
	)
	return err
}

func (s *UserStore) CreateProfile(userID uuid.UUID) error {
	query := `INSERT INTO user_profiles (user_id) VALUES ($1) ON CONFLICT DO NOTHING`
	_, err := s.db.Exec(query, userID)
	return err
}

func (s *UserStore) GetByEmail(email string) (*model.User, error) {
	var user model.User
	err := s.db.Get(&user, "SELECT * FROM users WHERE email = $1", email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (s *UserStore) GetByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	err := s.db.Get(&user, "SELECT * FROM users WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (s *UserStore) GetProfile(userID uuid.UUID) (*model.UserProfile, error) {
	var profile model.UserProfile
	query := `
		SELECT u.*, p.bio, p.company, p.title, p.preferences, p.onboarding_done
		FROM users u
		LEFT JOIN user_profiles p ON u.id = p.user_id
		WHERE u.id = $1`
	err := s.db.Get(&profile, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

func (s *UserStore) Update(userID uuid.UUID, req *model.UpdateUserRequest) error {
	sets := []string{}
	args := []interface{}{}
	idx := 1

	if req.DisplayName != nil {
		sets = append(sets, "display_name = $"+itoa(idx))
		args = append(args, *req.DisplayName)
		idx++
	}
	if req.AvatarURL != nil {
		sets = append(sets, "avatar_url = $"+itoa(idx))
		args = append(args, *req.AvatarURL)
		idx++
	}
	if req.Timezone != nil {
		sets = append(sets, "timezone = $"+itoa(idx))
		args = append(args, *req.Timezone)
		idx++
	}
	if req.Locale != nil {
		sets = append(sets, "locale = $"+itoa(idx))
		args = append(args, *req.Locale)
		idx++
	}

	if len(sets) == 0 {
		return nil
	}

	query := "UPDATE users SET " + joinStrings(sets, ", ") + " WHERE id = $" + itoa(idx)
	args = append(args, userID)

	_, err := s.db.Exec(query, args...)
	return err
}

func (s *UserStore) UpdateProfile(userID uuid.UUID, req *model.UpdateProfileRequest) error {
	sets := []string{}
	args := []interface{}{}
	idx := 1

	if req.Bio != nil {
		sets = append(sets, "bio = $"+itoa(idx))
		args = append(args, *req.Bio)
		idx++
	}
	if req.Company != nil {
		sets = append(sets, "company = $"+itoa(idx))
		args = append(args, *req.Company)
		idx++
	}
	if req.Title != nil {
		sets = append(sets, "title = $"+itoa(idx))
		args = append(args, *req.Title)
		idx++
	}
	if req.Preferences != nil {
		sets = append(sets, "preferences = $"+itoa(idx))
		args = append(args, *req.Preferences)
		idx++
	}
	if req.OnboardingDone != nil {
		sets = append(sets, "onboarding_done = $"+itoa(idx))
		args = append(args, *req.OnboardingDone)
		idx++
	}

	if len(sets) == 0 {
		return nil
	}

	query := "UPDATE user_profiles SET " + joinStrings(sets, ", ") + " WHERE user_id = $" + itoa(idx)
	args = append(args, userID)

	_, err := s.db.Exec(query, args...)
	return err
}

// ListActive 返回所有活跃用户
func (s *UserStore) ListActive() ([]model.User, error) {
	var users []model.User
	err := s.db.Select(&users, "SELECT * FROM users WHERE status = 'active' ORDER BY created_at")
	return users, err
}

// ──────────────────────────── 简报运行日志 ────────────────────────────

// CreateBriefingRunLog 写入一条简报运行日志
func (s *UserStore) CreateBriefingRunLog(log *model.BriefingRunLog) error {
	query := `
		INSERT INTO briefing_run_logs (run_id, run_type, started_at, completed_at,
			users_total, users_success, users_failed, total_tokens, total_cost, failures)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)
		RETURNING id`
	return s.db.QueryRow(query,
		log.RunID, log.RunType, log.StartedAt, log.CompletedAt,
		log.UsersTotal, log.UsersSuccess, log.UsersFailed,
		log.TotalTokens, log.TotalCost, string(log.Failures),
	).Scan(&log.ID)
}

// ListBriefingRunLogs 查询简报运行日志
func (s *UserStore) ListBriefingRunLogs(limit int) ([]model.BriefingRunLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var logs []model.BriefingRunLog
	err := s.db.Select(&logs, "SELECT * FROM briefing_run_logs ORDER BY started_at DESC LIMIT $1", limit)
	return logs, err
}

// helpers
func itoa(i int) string {
	return strconv.Itoa(i)
}

func joinStrings(ss []string, sep string) string {
	return strings.Join(ss, sep)
}

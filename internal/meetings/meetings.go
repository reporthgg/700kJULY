package meetings

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Service struct {
	db *sqlx.DB
}

type Meeting struct {
	ID		string		`db:"id"`
	InitiatorID	int64		`db:"initiator_id"`
	ParticipantID	int64		`db:"participant_id"`
	Title		string		`db:"title"`
	Description	string		`db:"description"`
	StartTime	time.Time	`db:"start_time"`
	EndTime		time.Time	`db:"end_time"`
	Confirmed	bool		`db:"confirmed"`
	CreatedAt	time.Time	`db:"created_at"`
}

type User struct {
	ID		int64		`db:"id"`
	Username	string		`db:"username"`
	FirstName	string		`db:"first_name"`
	CreatedAt	time.Time	`db:"created_at"`
	UpdatedAt	time.Time	`db:"updated_at"`
}

func NewService(db *sqlx.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) StoreUser(ctx context.Context, userID int64, username, firstName string) error {
	query := `
		INSERT INTO users (id, username, first_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (id) DO UPDATE 
		SET username = $2, first_name = $3, updated_at = $4
	`

	_, err := s.db.ExecContext(ctx, query, userID, username, firstName, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка при сохранении пользователя: %v", err)
	}

	return nil
}

func (s *Service) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT id, username, first_name, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var user User
	err := s.db.GetContext(ctx, &user, query, username)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении пользователя: %v", err)
	}

	return &user, nil
}

func (s *Service) CreateMeeting(ctx context.Context, initiatorID int64, participantUsername, title, description, startTimeStr, endTimeStr string) (string, error) {

	participant, err := s.GetUserByUsername(ctx, participantUsername)
	if err != nil {
		return "", fmt.Errorf("пользователь @%s не найден", participantUsername)
	}

	startTime, err := parseFlexibleTime(startTimeStr)
	if err != nil {
		return "", fmt.Errorf("неверный формат времени начала: %v", err)
	}

	endTime, err := parseFlexibleTime(endTimeStr)
	if err != nil {
		return "", fmt.Errorf("неверный формат времени окончания: %v", err)
	}

	meetingID := uuid.New().String()

	query := `
		INSERT INTO meetings (id, initiator_id, participant_id, title, description, start_time, end_time, confirmed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = s.db.ExecContext(ctx, query, meetingID, initiatorID, participant.ID, title, description, startTime, endTime, false, time.Now())
	if err != nil {
		return "", fmt.Errorf("ошибка при сохранении встречи: %v", err)
	}

	return meetingID, nil
}

func parseFlexibleTime(timeStr string) (time.Time, error) {

	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse("2006-01-02T15:04:05", timeStr)
	if err == nil {
		return t, nil
	}

	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04",
	}

	for _, format := range formats {
		t, err = time.Parse(format, timeStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("не удалось распознать формат времени: %s", timeStr)
}

func (s *Service) ConfirmMeeting(ctx context.Context, meetingID string, participantID int64) error {

	query := `
		SELECT id, participant_id
		FROM meetings
		WHERE id = $1
	`

	var meeting struct {
		ID		string	`db:"id"`
		ParticipantID	int64	`db:"participant_id"`
	}

	err := s.db.GetContext(ctx, &meeting, query, meetingID)
	if err != nil {
		return fmt.Errorf("встреча не найдена: %v", err)
	}

	if meeting.ParticipantID != participantID {
		return fmt.Errorf("вы не являетесь участником этой встречи")
	}

	updateQuery := `
		UPDATE meetings
		SET confirmed = true
		WHERE id = $1
	`

	_, err = s.db.ExecContext(ctx, updateQuery, meetingID)
	if err != nil {
		return fmt.Errorf("ошибка при подтверждении встречи: %v", err)
	}

	return nil
}

func (s *Service) GetPendingMeetings(ctx context.Context, userID int64) ([]Meeting, error) {
	query := `
		SELECT id, initiator_id, participant_id, title, description, start_time, end_time, confirmed, created_at
		FROM meetings
		WHERE participant_id = $1 AND confirmed = false
		ORDER BY start_time ASC
	`

	var meetings []Meeting
	err := s.db.SelectContext(ctx, &meetings, query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении встреч: %v", err)
	}

	return meetings, nil
}

func (s *Service) GetInitiator(ctx context.Context, initiatorID int64) (*User, error) {
	query := `
		SELECT id, username, first_name, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := s.db.GetContext(ctx, &user, query, initiatorID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении пользователя: %v", err)
	}

	return &user, nil
}

package calendar

import (
	"context"
	"fmt"
	"telegrambot/pkg/config"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type Service struct {
	db		*sqlx.DB
	cfg		*config.Config
	googleClient	*GoogleCalendarClient
}

type Event struct {
	ID		string		`db:"id"`
	UserID		int64		`db:"user_id"`
	Title		string		`db:"title"`
	Description	string		`db:"description"`
	StartTime	time.Time	`db:"start_time"`
	EndTime		time.Time	`db:"end_time"`
	CreatedAt	time.Time	`db:"created_at"`
	GoogleEventID	string		`db:"google_event_id"`
	ReminderSent	bool		`db:"reminder_sent"`
}

func NewService(db *sqlx.DB, cfg *config.Config) *Service {
	var googleClient *GoogleCalendarClient

	if cfg.GoogleCredentials != "" {
		var err error
		googleClient, err = NewGoogleCalendarClient(cfg.GoogleCredentials, db)
		if err != nil {
			logrus.Warnf("Не удалось инициализировать Google Calendar: %v", err)

		} else {
			logrus.Info("Google Calendar клиент инициализирован")
		}
	}

	return &Service{
		db:		db,
		cfg:		cfg,
		googleClient:	googleClient,
	}
}

func (s *Service) CreateEvent(ctx context.Context, userID int64, title, description, startTimeStr, endTimeStr string) (string, error) {
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return "", fmt.Errorf("некорректный формат времени начала: %v", err)
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return "", fmt.Errorf("некорректный формат времени окончания: %v", err)
	}

	eventID := uuid.New().String()

	event := &Event{
		ID:		eventID,
		UserID:		userID,
		Title:		title,
		Description:	description,
		StartTime:	startTime,
		EndTime:	endTime,
		CreatedAt:	time.Now(),
	}

	query := `
		INSERT INTO events (id, user_id, title, description, start_time, end_time, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = s.db.ExecContext(ctx, query, eventID, userID, title, description, startTime, endTime, event.CreatedAt)
	if err != nil {
		return "", fmt.Errorf("ошибка при сохранении события: %v", err)
	}

	if s.googleClient != nil {
		googleEventID, err := s.googleClient.CreateEvent(ctx, userID, event)
		if err != nil {
			logrus.Warnf("Не удалось создать событие в Google Calendar: %v", err)
		} else {

			updateQuery := `
				UPDATE events SET google_event_id = $1 WHERE id = $2
			`
			_, _ = s.db.ExecContext(ctx, updateQuery, googleEventID, eventID)
			logrus.Infof("Событие успешно создано в Google Calendar (ID: %s)", googleEventID)
		}
	}

	return eventID, nil
}

func (s *Service) GetUpcomingEvents(ctx context.Context, userID int64, period time.Duration) ([]Event, error) {
	query := `
		SELECT id, user_id, title, description, start_time, end_time, created_at
		FROM events
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
		ORDER BY start_time ASC
	`

	now := time.Now()
	end := now.Add(period)

	var events []Event
	err := s.db.SelectContext(ctx, &events, query, userID, now, end)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении предстоящих событий: %v", err)
	}

	return events, nil
}

func (s *Service) CheckReminders(ctx context.Context) ([]Event, error) {
	query := `
		SELECT id, user_id, title, description, start_time, end_time, created_at
		FROM events
		WHERE start_time BETWEEN $1 AND $2
		AND reminder_sent = false
		ORDER BY start_time ASC
	`

	now := time.Now()
	oneHourLater := now.Add(time.Hour)

	var events []Event
	err := s.db.SelectContext(ctx, &events, query, now, oneHourLater)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении событий для напоминаний: %v", err)
	}

	return events, nil
}

func (s *Service) MarkReminderSent(ctx context.Context, eventID string) error {
	query := `
		UPDATE events
		SET reminder_sent = true
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, eventID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении статуса напоминания: %v", err)
	}

	return nil
}

func (s *Service) StartReminderChecker(sendMessage func(int64, string) error) {
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			ctx := context.Background()
			events, err := s.CheckReminders(ctx)
			if err != nil {
				logrus.Errorf("Ошибка при проверке напоминаний: %v", err)
				continue
			}

			for _, event := range events {
				message := fmt.Sprintf("⏰ Напоминание: у вас через час событие '%s' в %s",
					event.Title, event.StartTime.Format("15:04"))

				if event.Description != "" {
					message += fmt.Sprintf("\nОписание: %s", event.Description)
				}

				err := sendMessage(event.UserID, message)
				if err != nil {
					logrus.Errorf("Ошибка при отправке напоминания пользователю %d: %v", event.UserID, err)
					continue
				}

				err = s.MarkReminderSent(ctx, event.ID)
				if err != nil {
					logrus.Errorf("Ошибка при обновлении статуса напоминания: %v", err)
				}
			}
		}
	}()
}

func (s *Service) GetGoogleAuthURL(userID int64, callbackType string) (string, error) {
	if s.googleClient == nil {
		return "", fmt.Errorf("google calendar не интегрирован")
	}

	state := fmt.Sprintf("%d:%s", userID, callbackType)
	return s.googleClient.GetAuthURL(state), nil
}

func (s *Service) HandleGoogleCallback(ctx context.Context, code string, userID int64) error {
	if s.googleClient == nil {
		return fmt.Errorf("google calendar не интегрирован")
	}

	return s.googleClient.HandleAuthCallback(ctx, code, userID)
}

func (s *Service) GetEventsByDate(ctx context.Context, userID int64, date time.Time) ([]Event, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT id, user_id, title, description, start_time, end_time, created_at
		FROM events
		WHERE user_id = $1 AND start_time >= $2 AND start_time < $3
		ORDER BY start_time ASC
	`

	var events []Event
	err := s.db.SelectContext(ctx, &events, query, userID, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении событий на дату: %v", err)
	}

	return events, nil
}

func (s *Service) GetEventByID(ctx context.Context, userID int64, eventID string) (*Event, error) {
	query := `
		SELECT id, user_id, title, description, start_time, end_time, created_at, google_event_id
		FROM events
		WHERE id = $1 AND user_id = $2
	`

	var event Event
	err := s.db.GetContext(ctx, &event, query, eventID, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении события по ID: %v", err)
	}

	return &event, nil
}

func (s *Service) GetEventsByDateRange(ctx context.Context, userID int64, startDate, endDate time.Time) ([]Event, error) {
	query := `
		SELECT id, user_id, title, description, start_time, end_time, created_at
		FROM events
		WHERE user_id = $1 AND start_time >= $2 AND start_time < $3
		ORDER BY start_time ASC
	`

	var events []Event
	err := s.db.SelectContext(ctx, &events, query, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении событий в диапазоне дат: %v", err)
	}

	return events, nil
}

func (s *Service) UpdateEvent(ctx context.Context, userID int64, eventID, title, description, startTimeStr, endTimeStr string) error {

	event, err := s.GetEventByID(ctx, userID, eventID)
	if err != nil {
		return fmt.Errorf("событие не найдено или не принадлежит пользователю: %v", err)
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return fmt.Errorf("некорректный формат времени начала: %v", err)
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return fmt.Errorf("некорректный формат времени окончания: %v", err)
	}

	logrus.Infof("Обновление события: ID=%s, GoogleID=%s, Старое время=%s, Новое время=%s",
		eventID, event.GoogleEventID, event.StartTime.Format(time.RFC3339), startTime.Format(time.RFC3339))

	query := `
		UPDATE events
		SET title = $1, description = $2, start_time = $3, end_time = $4
		WHERE id = $5 AND user_id = $6
	`

	_, err = s.db.ExecContext(ctx, query, title, description, startTime, endTime, eventID, userID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении события: %v", err)
	}

	if s.googleClient != nil && event.GoogleEventID != "" {
		logrus.Infof("Отправка обновления в Google Calendar: ID=%s, GoogleID=%s",
			eventID, event.GoogleEventID)

		updatedEvent := &Event{
			ID:		event.ID,
			UserID:		userID,
			Title:		title,
			Description:	description,
			StartTime:	startTime,
			EndTime:	endTime,
			GoogleEventID:	event.GoogleEventID,
		}

		err = s.googleClient.UpdateEvent(ctx, userID, updatedEvent)
		if err != nil {
			logrus.Warnf("Не удалось обновить событие в Google Calendar: %v", err)

		} else {
			logrus.Infof("Событие успешно обновлено в Google Calendar: ID=%s, GoogleID=%s",
				eventID, event.GoogleEventID)
		}
	} else if s.googleClient != nil {
		logrus.Warnf("Событие ID=%s не имеет GoogleEventID, обновление в Google Calendar пропущено", eventID)
	}

	return nil
}

func (s *Service) DeleteEvent(ctx context.Context, userID int64, eventID string) error {

	event, err := s.GetEventByID(ctx, userID, eventID)
	if err != nil {
		return fmt.Errorf("событие не найдено или не принадлежит пользователю: %v", err)
	}

	if s.googleClient != nil && event.GoogleEventID != "" {
		err = s.googleClient.DeleteEvent(ctx, userID, event.GoogleEventID)
		if err != nil {
			logrus.Warnf("Не удалось удалить событие из Google Calendar: %v", err)

		}
	}

	query := `DELETE FROM events WHERE id = $1 AND user_id = $2`
	_, err = s.db.ExecContext(ctx, query, eventID, userID)
	if err != nil {
		return fmt.Errorf("ошибка при удалении события: %v", err)
	}

	return nil
}

func (s *Service) DeleteEventsByDateRange(ctx context.Context, userID int64, startDate, endDate time.Time) (int, error) {

	events, err := s.GetEventsByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return 0, fmt.Errorf("ошибка при получении событий для удаления: %v", err)
	}

	deletedCount := 0
	for _, event := range events {
		err := s.DeleteEvent(ctx, userID, event.ID)
		if err != nil {
			logrus.Warnf("Не удалось удалить событие %s: %v", event.ID, err)
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

func (s *Service) StartGoogleCalendarSync() {
	if s.googleClient == nil {
		logrus.Warn("Google Calendar не интегрирован, синхронизация не запущена")
		return
	}

	go func() {

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		s.syncGoogleCalendarForAllUsers()

		for range ticker.C {
			s.syncGoogleCalendarForAllUsers()
		}
	}()

	logrus.Info("Запущена периодическая синхронизация с Google Calendar")
}

func (s *Service) SyncGoogleCalendarForUser(ctx context.Context, userID int64) error {
	if s.googleClient == nil {
		return fmt.Errorf("google calendar не интегрирован")
	}

	return s.googleClient.SyncEventsFromGoogleCalendar(ctx, userID)
}

func (s *Service) syncGoogleCalendarForAllUsers() {
	ctx := context.Background()

	query := `SELECT DISTINCT user_id FROM google_tokens`
	var userIDs []int64

	err := s.db.SelectContext(ctx, &userIDs, query)
	if err != nil {
		logrus.Errorf("Ошибка при получении списка пользователей для синхронизации Google Calendar: %v", err)
		return
	}

	logrus.Infof("Запуск синхронизации с Google Calendar для %d пользователей", len(userIDs))

	for _, userID := range userIDs {
		err := s.SyncGoogleCalendarForUser(ctx, userID)
		if err != nil {
			logrus.Errorf("Ошибка при синхронизации Google Calendar для пользователя %d: %v", userID, err)
		}
	}
}

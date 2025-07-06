package calendar

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type GoogleCalendarClient struct {
	config	*oauth2.Config
	db	*sqlx.DB
}

func NewGoogleCalendarClient(credentialsPath string, db *sqlx.DB) (*GoogleCalendarClient, error) {
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл с учетными данными: %v", err)
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("не удалось разобрать учетные данные: %v", err)
	}

	return &GoogleCalendarClient{
		config:	config,
		db:	db,
	}, nil
}

func (g *GoogleCalendarClient) GetAuthURL(state string) string {
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

func (g *GoogleCalendarClient) HandleAuthCallback(ctx context.Context, code string, userID int64) error {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("не удалось обменять код на токен: %v", err)
	}

	return g.saveToken(userID, token)
}

func (g *GoogleCalendarClient) CreateEvent(ctx context.Context, userID int64, event *Event) (string, error) {
	client, err := g.getClient(ctx, userID)
	if err != nil {
		return "", err
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", fmt.Errorf("не удалось создать сервис календаря: %v", err)
	}

	calendarEvent := &calendar.Event{
		Summary:	event.Title,
		Description:	event.Description,
		Start: &calendar.EventDateTime{
			DateTime:	event.StartTime.Format(time.RFC3339),
			TimeZone:	"UTC",
		},
		End: &calendar.EventDateTime{
			DateTime:	event.EndTime.Format(time.RFC3339),
			TimeZone:	"UTC",
		},
	}

	calendarID := "primary"
	createdEvent, err := srv.Events.Insert(calendarID, calendarEvent).Do()
	if err != nil {
		return "", fmt.Errorf("не удалось создать событие: %v", err)
	}

	return createdEvent.Id, nil
}

func (g *GoogleCalendarClient) getClient(ctx context.Context, userID int64) (*http.Client, error) {
	token, err := g.loadToken(userID)
	if err != nil {
		return nil, fmt.Errorf("пользователь не авторизован в Google Calendar: %v", err)
	}

	if token.Expiry.Before(time.Now()) {
		newToken, err := g.config.TokenSource(ctx, token).Token()
		if err != nil {
			return nil, fmt.Errorf("не удалось обновить токен: %v", err)
		}
		if newToken.AccessToken != token.AccessToken {
			token = newToken
			if err := g.saveToken(userID, token); err != nil {
				return nil, err
			}
		}
	}

	return g.config.Client(ctx, token), nil
}

func (g *GoogleCalendarClient) saveToken(userID int64, token *oauth2.Token) error {
	query := `
		INSERT INTO google_tokens (user_id, access_token, refresh_token, token_type, expiry, updated_at) 
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			access_token = $2,
			refresh_token = COALESCE($3, google_tokens.refresh_token),
			token_type = $4,
			expiry = $5,
			updated_at = NOW()
	`

	var refreshToken interface{} = nil
	if token.RefreshToken != "" {
		refreshToken = token.RefreshToken
	}

	_, err := g.db.Exec(query,
		userID,
		token.AccessToken,
		refreshToken,
		token.TokenType,
		token.Expiry)

	return err
}

func (g *GoogleCalendarClient) loadToken(userID int64) (*oauth2.Token, error) {
	query := `
		SELECT access_token, refresh_token, token_type, expiry 
		FROM google_tokens 
		WHERE user_id = $1
	`

	var tokenData struct {
		AccessToken	string		`db:"access_token"`
		RefreshToken	string		`db:"refresh_token"`
		TokenType	string		`db:"token_type"`
		Expiry		time.Time	`db:"expiry"`
	}

	err := g.db.Get(&tokenData, query, userID)
	if err != nil {
		return nil, fmt.Errorf("токен не найден: %v", err)
	}

	token := &oauth2.Token{
		AccessToken:	tokenData.AccessToken,
		RefreshToken:	tokenData.RefreshToken,
		TokenType:	tokenData.TokenType,
		Expiry:		tokenData.Expiry,
	}

	return token, nil
}

func adjustTimeForGoogleCalendar(originalTime time.Time, offsetHours int) time.Time {

	return originalTime.Add(time.Duration(-offsetHours) * time.Hour)
}

func (g *GoogleCalendarClient) UpdateEvent(ctx context.Context, userID int64, event *Event) error {
	if event.GoogleEventID == "" {
		return fmt.Errorf("отсутствует ID события в Google Calendar")
	}

	fmt.Printf("Обновление события: ID=%s, Title=%s, StartTime=%s\n",
		event.GoogleEventID, event.Title, event.StartTime.Format(time.RFC3339))

	client, err := g.getClient(ctx, userID)
	if err != nil {
		return fmt.Errorf("ошибка получения клиента: %v", err)
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("не удалось создать сервис календаря: %v", err)
	}

	localLoc, _ := time.LoadLocation("Local")
	_, localOffset := time.Now().In(localLoc).Zone()
	localOffsetHours := localOffset / 3600

	fmt.Printf("DEBUG: Локальное смещение часового пояса: %+d часов\n", localOffsetHours)

	existingEvent, err := srv.Events.Get("primary", event.GoogleEventID).Do()
	if err == nil && existingEvent != nil {
		fmt.Printf("DEBUG: Существующее событие: StartDateTime=%s, TimeZone=%s\n",
			existingEvent.Start.DateTime, existingEvent.Start.TimeZone)
	}

	adjustedStartTime := adjustTimeForGoogleCalendar(event.StartTime, localOffsetHours)
	adjustedEndTime := adjustTimeForGoogleCalendar(event.EndTime, localOffsetHours)

	fmt.Printf("DEBUG: Исходное время: Start=%s, End=%s\n",
		event.StartTime.Format(time.RFC3339), event.EndTime.Format(time.RFC3339))
	fmt.Printf("DEBUG: Адаптированное время: Start=%s, End=%s\n",
		adjustedStartTime.Format(time.RFC3339), adjustedEndTime.Format(time.RFC3339))

	startStr := adjustedStartTime.Format("2006-01-02T15:04:05")
	endStr := adjustedEndTime.Format("2006-01-02T15:04:05")

	fmt.Printf("DEBUG: Используем форматы - Start=%s, End=%s\n", startStr, endStr)

	calendarEvent := &calendar.Event{
		Summary:	event.Title,
		Description:	event.Description,
		Start: &calendar.EventDateTime{
			DateTime:	startStr,
			TimeZone:	"UTC",
		},
		End: &calendar.EventDateTime{
			DateTime:	endStr,
			TimeZone:	"UTC",
		},
	}

	fmt.Printf("DEBUG: Отправляем в Google Calendar: StartDateTime=%s, StartTimeZone=%s\n",
		calendarEvent.Start.DateTime, calendarEvent.Start.TimeZone)

	updatedEvent, err := srv.Events.Update("primary", event.GoogleEventID, calendarEvent).Do()
	if err != nil {
		return fmt.Errorf("не удалось обновить событие: %v", err)
	}

	fmt.Printf("DEBUG: Ответ от Google Calendar: ID=%s, StartDateTime=%s, StartTimeZone=%s\n",
		updatedEvent.Id, updatedEvent.Start.DateTime, updatedEvent.Start.TimeZone)

	fmt.Printf("Событие успешно обновлено: ID=%s, Title=%s, StartTime=%s\n",
		updatedEvent.Id, updatedEvent.Summary, updatedEvent.Start.DateTime)
	return nil
}

func (g *GoogleCalendarClient) DeleteEvent(ctx context.Context, userID int64, googleEventID string) error {
	if googleEventID == "" {
		return fmt.Errorf("отсутствует ID события в Google Calendar")
	}

	fmt.Printf("Удаление события из Google Calendar: ID=%s\n", googleEventID)

	client, err := g.getClient(ctx, userID)
	if err != nil {
		return fmt.Errorf("ошибка получения клиента: %v", err)
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("не удалось создать сервис календаря: %v", err)
	}

	calendarID := "primary"

	_, err = srv.Events.Get(calendarID, googleEventID).Do()
	if err != nil {
		fmt.Printf("Событие %s не найдено при попытке удаления: %v\n", googleEventID, err)

		return nil
	}

	err = srv.Events.Delete(calendarID, googleEventID).Do()
	if err != nil {
		fmt.Printf("Ошибка при удалении события %s: %v\n", googleEventID, err)
		return fmt.Errorf("не удалось удалить событие из Google Calendar: %v", err)
	}

	fmt.Printf("Событие успешно удалено из Google Calendar: ID=%s\n", googleEventID)
	return nil
}

func (g *GoogleCalendarClient) GetEvents(ctx context.Context, userID int64, startTime, endTime time.Time) ([]*calendar.Event, error) {
	client, err := g.getClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("не удалось создать сервис календаря: %v", err)
	}

	calendarID := "primary"
	events, err := srv.Events.List(calendarID).
		TimeMin(startTime.Format(time.RFC3339)).
		TimeMax(endTime.Format(time.RFC3339)).
		OrderBy("startTime").
		SingleEvents(true).
		Do()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить события из Google Calendar: %v", err)
	}

	return events.Items, nil
}

func (g *GoogleCalendarClient) GetEventByID(ctx context.Context, userID int64, eventID string) (*Event, error) {
	query := `
		SELECT id, user_id, title, description, start_time, end_time, created_at, google_event_id
		FROM events
		WHERE id = $1 AND user_id = $2
	`

	var event Event
	err := g.db.GetContext(ctx, &event, query, eventID, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении события по ID: %v", err)
	}

	return &event, nil
}

func (g *GoogleCalendarClient) SyncEventsFromGoogleCalendar(ctx context.Context, userID int64) error {

	lastSyncTime, err := g.getLastSyncTime(userID)
	isFirstSync := false
	if err != nil {

		lastSyncTime = time.Now().Add(-7 * 24 * time.Hour)
		isFirstSync = true
		logrus.Warnf("Не удалось получить время последней синхронизации: %v. Используем последние 7 дней", err)
	}

	client, err := g.getClient(ctx, userID)
	if err != nil {
		return fmt.Errorf("ошибка получения клиента: %v", err)
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("не удалось создать сервис календаря: %v", err)
	}

	timeMin := lastSyncTime.Format(time.RFC3339)
	timeMax := time.Now().Add(365 * 24 * time.Hour).Format(time.RFC3339)

	logrus.Infof("Синхронизация событий Google Calendar для userID=%d с %s по %s",
		userID, timeMin, timeMax)

	eventsListCall := srv.Events.List("primary").
		TimeMin(timeMin).
		TimeMax(timeMax).
		SingleEvents(true).
		OrderBy("updated")

	if !isFirstSync {
		eventsListCall = eventsListCall.UpdatedMin(timeMin)
	}

	events, err := eventsListCall.Do()
	if err != nil {
		return fmt.Errorf("не удалось получить события из Google Calendar: %v", err)
	}

	logrus.Infof("Получено %d событий из Google Calendar для синхронизации", len(events.Items))

	for _, googleEvent := range events.Items {

		if googleEvent.Status == "cancelled" {
			err = g.handleDeletedGoogleEvent(ctx, userID, googleEvent.Id)
			if err != nil {
				logrus.Warnf("Ошибка при обработке удаленного события %s: %v", googleEvent.Id, err)
			}
			continue
		}

		localEvent, err := g.findLocalEventByGoogleID(ctx, userID, googleEvent.Id)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logrus.Warnf("Ошибка при поиске локального события для Google ID %s: %v", googleEvent.Id, err)
			continue
		}

		if localEvent == nil || errors.Is(err, sql.ErrNoRows) {
			err = g.createLocalEventFromGoogle(ctx, userID, googleEvent)
			if err != nil {
				logrus.Warnf("Ошибка при создании нового события из Google: %v", err)
			}
		} else {

			err = g.updateLocalEventFromGoogle(ctx, userID, localEvent.ID, googleEvent)
			if err != nil {
				logrus.Warnf("Ошибка при обновлении события из Google: %v", err)
			}
		}
	}

	err = g.updateLastSyncTime(userID, time.Now())
	if err != nil {
		logrus.Warnf("Ошибка при обновлении времени последней синхронизации: %v", err)
	}

	return nil
}

func (g *GoogleCalendarClient) findLocalEventByGoogleID(ctx context.Context, userID int64, googleEventID string) (*Event, error) {
	query := `
		SELECT id, user_id, title, description, start_time, end_time, created_at, google_event_id
		FROM events
		WHERE google_event_id = $1 AND user_id = $2
	`

	var event Event
	err := g.db.GetContext(ctx, &event, query, googleEventID, userID)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

func (g *GoogleCalendarClient) createLocalEventFromGoogle(ctx context.Context, userID int64, googleEvent *calendar.Event) error {
	eventID := uuid.New().String()

	startTime, err := parseGoogleEventTime(googleEvent.Start)
	if err != nil {
		return fmt.Errorf("ошибка парсинга времени начала: %v", err)
	}

	endTime, err := parseGoogleEventTime(googleEvent.End)
	if err != nil {
		return fmt.Errorf("ошибка парсинга времени окончания: %v", err)
	}

	query := `
		INSERT INTO events (id, user_id, title, description, start_time, end_time, created_at, google_event_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = g.db.ExecContext(ctx, query,
		eventID,
		userID,
		googleEvent.Summary,
		googleEvent.Description,
		startTime,
		endTime,
		time.Now(),
		googleEvent.Id)

	if err != nil {
		return fmt.Errorf("ошибка при сохранении события из Google Calendar: %v", err)
	}

	logrus.Infof("Создано новое событие из Google Calendar: ID=%s, GoogleID=%s, Title=%s",
		eventID, googleEvent.Id, googleEvent.Summary)

	return nil
}

func (g *GoogleCalendarClient) updateLocalEventFromGoogle(ctx context.Context, userID int64, eventID string, googleEvent *calendar.Event) error {

	startTime, err := parseGoogleEventTime(googleEvent.Start)
	if err != nil {
		return fmt.Errorf("ошибка парсинга времени начала: %v", err)
	}

	endTime, err := parseGoogleEventTime(googleEvent.End)
	if err != nil {
		return fmt.Errorf("ошибка парсинга времени окончания: %v", err)
	}

	query := `
		UPDATE events
		SET title = $1, description = $2, start_time = $3, end_time = $4
		WHERE id = $5 AND user_id = $6
	`

	_, err = g.db.ExecContext(ctx, query,
		googleEvent.Summary,
		googleEvent.Description,
		startTime,
		endTime,
		eventID,
		userID)

	if err != nil {
		return fmt.Errorf("ошибка при обновлении события из Google Calendar: %v", err)
	}

	logrus.Infof("Обновлено событие из Google Calendar: ID=%s, GoogleID=%s, Title=%s",
		eventID, googleEvent.Id, googleEvent.Summary)

	return nil
}

func (g *GoogleCalendarClient) handleDeletedGoogleEvent(ctx context.Context, userID int64, googleEventID string) error {

	localEvent, err := g.findLocalEventByGoogleID(ctx, userID, googleEventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {

			return nil
		}
		return fmt.Errorf("ошибка при поиске локального события для удаления: %v", err)
	}

	query := `DELETE FROM events WHERE id = $1 AND user_id = $2`
	_, err = g.db.ExecContext(ctx, query, localEvent.ID, userID)
	if err != nil {
		return fmt.Errorf("ошибка при удалении события: %v", err)
	}

	logrus.Infof("Удалено событие из локальной БД по синхронизации с Google Calendar: ID=%s, GoogleID=%s",
		localEvent.ID, googleEventID)

	return nil
}

func parseGoogleEventTime(eventTime *calendar.EventDateTime) (time.Time, error) {
	if eventTime.DateTime != "" {

		return time.Parse(time.RFC3339, eventTime.DateTime)
	} else if eventTime.Date != "" {

		return time.Parse("2006-01-02", eventTime.Date)
	}

	return time.Time{}, fmt.Errorf("не удалось определить формат времени")
}

func (g *GoogleCalendarClient) getLastSyncTime(userID int64) (time.Time, error) {
	query := `
		SELECT last_sync_time 
		FROM google_sync_state 
		WHERE user_id = $1
	`

	var lastSyncTime time.Time
	err := g.db.Get(&lastSyncTime, query, userID)
	if err != nil {
		return time.Time{}, err
	}

	return lastSyncTime, nil
}

func (g *GoogleCalendarClient) updateLastSyncTime(userID int64, syncTime time.Time) error {
	query := `
		INSERT INTO google_sync_state (user_id, last_sync_time)
		VALUES ($1, $2)
		ON CONFLICT (user_id)
		DO UPDATE SET last_sync_time = $2
	`

	_, err := g.db.Exec(query, userID, syncTime)
	return err
}

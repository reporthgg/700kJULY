package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"telegrambot/internal/auth"
	"telegrambot/internal/calendar"
	"telegrambot/internal/linking"
	"telegrambot/internal/okr"
	"telegrambot/internal/users"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	calendarService	*calendar.Service
	userService	*users.Service
	linkingService	*linking.Service
	okrService	*okr.Service
	db		*sqlx.DB
	jwtSigningKey	string
	telegramBotName	string
}

func NewHandler(
	calService *calendar.Service,
	userService *users.Service,
	linkService *linking.Service,
	okrService *okr.Service,
	database *sqlx.DB,
	jwtKey string,
	tgBotName string,
) *Handler {
	return &Handler{
		calendarService:	calService,
		userService:		userService,
		linkingService:		linkService,
		okrService:		okrService,
		db:			database,
		jwtSigningKey:		jwtKey,
		telegramBotName:	tgBotName,
	}
}

type LoginRequest struct {
	Login		string	`json:"login"`
	Password	string	`json:"password"`
}

type RegisterRequest struct {
	Login		string	`json:"login"`
	Password	string	`json:"password"`
	Email		*string	`json:"email,omitempty"`
	Phone		*string	`json:"phone,omitempty"`
}

type UserResponse struct {
	ID		int64		`json:"id"`
	Login		string		`json:"login"`
	Email		*string		`json:"email,omitempty"`
	Phone		*string		`json:"phone,omitempty"`
	CreatedAt	time.Time	`json:"created_at"`
	UpdatedAt	time.Time	`json:"updated_at"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (h *Handler) RegisterWebUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}

	if req.Login == "" || req.Password == "" {
		http.Error(w, "Логин и пароль обязательны", http.StatusBadRequest)
		return
	}

	user, err := h.userService.RegisterWebUser(r.Context(), req.Login, req.Password, req.Email, req.Phone)
	if err != nil {
		if errors.Is(err, users.ErrUserAlreadyExists) {
			http.Error(w, "Пользователь с таким логином уже существует", http.StatusConflict)
		} else {
			logrus.Errorf("Ошибка регистрации пользователя '%s': %v", req.Login, err)
			http.Error(w, "Ошибка при регистрации пользователя", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(UserResponse{
		ID:		user.ID,
		Login:		user.Login,
		Email:		user.Email,
		Phone:		user.Phone,
		CreatedAt:	user.CreatedAt,
		UpdatedAt:	user.UpdatedAt,
	})
}

func (h *Handler) AuthLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}

	if req.Login == "" || req.Password == "" {
		http.Error(w, "Логин и пароль обязательны", http.StatusBadRequest)
		return
	}

	user, err := h.userService.AuthenticateWebUser(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, users.ErrInvalidCredentials) {
			http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		} else {
			logrus.Errorf("Ошибка аутентификации пользователя '%s': %v", req.Login, err)
			http.Error(w, "Ошибка аутентификации", http.StatusInternalServerError)
		}
		return
	}

	expirationTime := 24 * time.Hour
	tokenString, err := auth.GenerateJWTToken(user.ID, h.jwtSigningKey, expirationTime)
	if err != nil {
		logrus.Errorf("Ошибка генерации JWT токена: %v", err)
		http.Error(w, "Ошибка при генерации токена", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: tokenString})
}

func (h *Handler) GetCalendarEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	webUserID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logrus.Error("Не удалось извлечь webUserID из контекста в GetCalendarEvents")
		http.Error(w, "Ошибка авторизации: webUserID не найден в токене", http.StatusUnauthorized)
		return
	}

	webUser, err := h.userService.GetWebUserByID(ctx, webUserID)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			logrus.Warnf("Веб-пользователь с ID %d не найден при запросе событий календаря.", webUserID)
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
		} else {
			logrus.Errorf("Ошибка API при получении web_user %d: %v", webUserID, err)
			http.Error(w, "Ошибка при получении данных пользователя", http.StatusInternalServerError)
		}
		return
	}
	if webUser == nil {
		logrus.Warnf("Веб-пользователь с ID %d вернулся nil (без ошибки ErrUserNotFound) при запросе событий календаря.", webUserID)
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	if len(webUser.TelegramIDs) == 0 {
		logrus.Infof("У web_user_id %d нет привязанных Telegram ID. Возвращаем пустой список событий.", webUserID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]calendar.Event{})
		return
	}

	dateStr := r.URL.Query().Get("date")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var queryBuilder strings.Builder
	args := []interface{}{}

	queryBuilder.WriteString(`
		SELECT id, user_id, COALESCE(google_event_id, '') as google_event_id, title, description, start_time, end_time, created_at
		FROM events
		WHERE user_id = ANY($1)`)
	args = append(args, webUser.TelegramIDs)

	paramIndex := 2

	if dateStr != "" {
		parsedDate, parseErr := time.Parse("2006-01-02", dateStr)
		if parseErr != nil {
			http.Error(w, "Некорректный формат даты (ожидается YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		dayStart := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, time.UTC)
		dayEnd := dayStart.Add(24 * time.Hour)

		queryBuilder.WriteString(fmt.Sprintf(" AND start_time >= $%d AND start_time < $%d", paramIndex, paramIndex+1))
		args = append(args, dayStart, dayEnd)

	} else if startDateStr != "" && endDateStr != "" {
		parsedStartDate, parseErr := time.Parse("2006-01-02", startDateStr)
		if parseErr != nil {
			http.Error(w, "Некорректный формат начальной даты (ожидается YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		parsedEndDate, parseErr := time.Parse("2006-01-02", endDateStr)
		if parseErr != nil {
			http.Error(w, "Некорректный формат конечной даты (ожидается YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		rangeStart := time.Date(parsedStartDate.Year(), parsedStartDate.Month(), parsedStartDate.Day(), 0, 0, 0, 0, time.UTC)
		rangeEnd := time.Date(parsedEndDate.Year(), parsedEndDate.Month(), parsedEndDate.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)

		queryBuilder.WriteString(fmt.Sprintf(" AND start_time >= $%d AND start_time < $%d", paramIndex, paramIndex+1))
		args = append(args, rangeStart, rangeEnd)

	} else {
		http.Error(w, "Необходимо указать 'date' или 'start_date' и 'end_date'", http.StatusBadRequest)
		return
	}

	queryBuilder.WriteString(" ORDER BY start_time")

	finalQuery := queryBuilder.String()
	logrus.Debugf("Выполняется SQL-запрос для GetCalendarEvents: %s с аргументами: %v", finalQuery, args)

	var events []calendar.Event
	err = h.db.SelectContext(ctx, &events, finalQuery, args...)
	if err != nil {
		logrus.Errorf("Ошибка API при выполнении SQL-запроса для получения событий: %v", err)
		http.Error(w, "Ошибка при получении событий", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(events); err != nil {
		logrus.Errorf("Ошибка API при сериализации событий в JSON: %v", err)
		http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError)
	}
}

type GenerateTelegramLinkResponse struct {
	Link string `json:"link"`
}

func (h *Handler) GenerateTelegramLinkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	webUserID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		logrus.Error("Не удалось извлечь webUserID из контекста в GenerateTelegramLinkHandler")
		http.Error(w, "Ошибка авторизации: webUserID не найден в токене", http.StatusUnauthorized)
		return
	}

	token, err := h.linkingService.GenerateLinkToken(webUserID)
	if err != nil {
		logrus.Errorf("Ошибка генерации токена привязки для webUserID %d: %v", webUserID, err)
		http.Error(w, "Не удалось сгенерировать ссылку для привязки", http.StatusInternalServerError)
		return
	}

	if h.telegramBotName == "" {
		logrus.Error("Имя Telegram бота не сконфигурировано в API Handler")
		http.Error(w, "Сервис временно недоступен для привязки Telegram", http.StatusInternalServerError)
		return
	}

	link := fmt.Sprintf("https://t.me/%s?start=%s", h.telegramBotName, token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GenerateTelegramLinkResponse{Link: link})
}

type CreateEventRequest struct {
	Title		string	`json:"title"`
	Description	string	`json:"description"`
	StartTime	string	`json:"start_time"`
	EndTime		string	`json:"end_time"`
}

type UpdateEventRequest struct {
	EventID		string	`json:"event_id"`
	Title		*string	`json:"title,omitempty"`
	Description	*string	`json:"description,omitempty"`
	StartTime	*string	`json:"start_time,omitempty"`
	EndTime		*string	`json:"end_time,omitempty"`
}

type DeleteEventRequest struct {
	EventID string `json:"event_id"`
}

type EventResponse struct {
	ID		string		`json:"id"`
	Title		string		`json:"title"`
	Description	string		`json:"description"`
	StartTime	time.Time	`json:"start_time"`
	EndTime		time.Time	`json:"end_time"`
	CreatedAt	time.Time	`json:"created_at"`
}

func (h *Handler) CreateCalendarEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	webUserID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logrus.Error("Не удалось извлечь webUserID из контекста в CreateCalendarEventHandler")
		http.Error(w, "Ошибка авторизации: webUserID не найден в токене", http.StatusUnauthorized)
		return
	}

	webUser, err := h.userService.GetWebUserByID(ctx, webUserID)
	if err != nil {
		logrus.Errorf("Ошибка при получении web_user %d: %v", webUserID, err)
		http.Error(w, "Ошибка при получении данных пользователя", http.StatusInternalServerError)
		return
	}
	if webUser == nil || len(webUser.TelegramIDs) == 0 {
		logrus.Warnf("Пользователь с ID %d не найден или не имеет привязанных Telegram аккаунтов", webUserID)
		http.Error(w, "Для создания события требуется привязанный Telegram аккаунт", http.StatusBadRequest)
		return
	}

	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}

	if req.Title == "" || req.StartTime == "" || req.EndTime == "" {
		http.Error(w, "Название, время начала и окончания обязательны", http.StatusBadRequest)
		return
	}

	telegramID := webUser.TelegramIDs[0]

	eventID, err := h.calendarService.CreateEvent(ctx, telegramID, req.Title, req.Description, req.StartTime, req.EndTime)
	if err != nil {
		logrus.Errorf("Ошибка при создании события для пользователя %d: %v", telegramID, err)
		http.Error(w, "Ошибка при создании события", http.StatusInternalServerError)
		return
	}

	createdEvent, err := h.calendarService.GetEventByID(ctx, telegramID, eventID)
	if err != nil {
		logrus.Errorf("Событие создано, но ошибка при получении данных: %v", err)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": eventID})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(EventResponse{
		ID:		createdEvent.ID,
		Title:		createdEvent.Title,
		Description:	createdEvent.Description,
		StartTime:	createdEvent.StartTime,
		EndTime:	createdEvent.EndTime,
		CreatedAt:	createdEvent.CreatedAt,
	})
}

func (h *Handler) UpdateCalendarEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	webUserID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logrus.Error("Не удалось извлечь webUserID из контекста в UpdateCalendarEventHandler")
		http.Error(w, "Ошибка авторизации: webUserID не найден в токене", http.StatusUnauthorized)
		return
	}

	webUser, err := h.userService.GetWebUserByID(ctx, webUserID)
	if err != nil {
		logrus.Errorf("Ошибка при получении web_user %d: %v", webUserID, err)
		http.Error(w, "Ошибка при получении данных пользователя", http.StatusInternalServerError)
		return
	}
	if webUser == nil || len(webUser.TelegramIDs) == 0 {
		logrus.Warnf("Пользователь с ID %d не найден или не имеет привязанных Telegram аккаунтов", webUserID)
		http.Error(w, "Для обновления события требуется привязанный Telegram аккаунт", http.StatusBadRequest)
		return
	}

	var req UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}

	if req.EventID == "" {
		http.Error(w, "ID события обязателен", http.StatusBadRequest)
		return
	}

	if req.Title == nil && req.Description == nil && req.StartTime == nil && req.EndTime == nil {
		http.Error(w, "Требуется хотя бы одно поле для обновления", http.StatusBadRequest)
		return
	}

	var foundEvent *calendar.Event
	var telegramIDForEvent int64

	for _, telegramID := range webUser.TelegramIDs {
		event, err := h.calendarService.GetEventByID(ctx, telegramID, req.EventID)
		if err == nil && event != nil {
			foundEvent = event
			telegramIDForEvent = telegramID
			break
		}
	}

	if foundEvent == nil {
		http.Error(w, "Событие не найдено или не принадлежит пользователю", http.StatusNotFound)
		return
	}

	title := foundEvent.Title
	if req.Title != nil {
		title = *req.Title
	}

	description := foundEvent.Description
	if req.Description != nil {
		description = *req.Description
	}

	startTimeStr := foundEvent.StartTime.Format(time.RFC3339)
	if req.StartTime != nil {
		startTimeStr = *req.StartTime
	}

	endTimeStr := foundEvent.EndTime.Format(time.RFC3339)
	if req.EndTime != nil {
		endTimeStr = *req.EndTime
	}

	err = h.calendarService.UpdateEvent(ctx, telegramIDForEvent, req.EventID, title, description, startTimeStr, endTimeStr)
	if err != nil {
		logrus.Errorf("Ошибка при обновлении события %s: %v", req.EventID, err)
		http.Error(w, "Ошибка при обновлении события", http.StatusInternalServerError)
		return
	}

	updatedEvent, err := h.calendarService.GetEventByID(ctx, telegramIDForEvent, req.EventID)
	if err != nil {
		logrus.Errorf("Событие обновлено, но ошибка при получении данных: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(EventResponse{
		ID:		updatedEvent.ID,
		Title:		updatedEvent.Title,
		Description:	updatedEvent.Description,
		StartTime:	updatedEvent.StartTime,
		EndTime:	updatedEvent.EndTime,
		CreatedAt:	updatedEvent.CreatedAt,
	})
}

func (h *Handler) DeleteCalendarEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	webUserID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logrus.Error("Не удалось извлечь webUserID из контекста в DeleteCalendarEventHandler")
		http.Error(w, "Ошибка авторизации: webUserID не найден в токене", http.StatusUnauthorized)
		return
	}

	webUser, err := h.userService.GetWebUserByID(ctx, webUserID)
	if err != nil {
		logrus.Errorf("Ошибка при получении web_user %d: %v", webUserID, err)
		http.Error(w, "Ошибка при получении данных пользователя", http.StatusInternalServerError)
		return
	}
	if webUser == nil || len(webUser.TelegramIDs) == 0 {
		logrus.Warnf("Пользователь с ID %d не найден или не имеет привязанных Telegram аккаунтов", webUserID)
		http.Error(w, "Для удаления события требуется привязанный Telegram аккаунт", http.StatusBadRequest)
		return
	}

	eventID := r.URL.Query().Get("event_id")
	if eventID == "" {

		var req DeleteEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EventID == "" {
			http.Error(w, "ID события обязателен", http.StatusBadRequest)
			return
		}
		eventID = req.EventID
	}

	var eventFound bool
	var telegramIDForEvent int64

	for _, telegramID := range webUser.TelegramIDs {
		event, err := h.calendarService.GetEventByID(ctx, telegramID, eventID)
		if err == nil && event != nil {
			eventFound = true
			telegramIDForEvent = telegramID
			break
		}
	}

	if !eventFound {
		http.Error(w, "Событие не найдено или не принадлежит пользователю", http.StatusNotFound)
		return
	}

	err = h.calendarService.DeleteEvent(ctx, telegramIDForEvent, eventID)
	if err != nil {
		logrus.Errorf("Ошибка при удалении события %s: %v", eventID, err)
		http.Error(w, "Ошибка при удалении события", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

type SetOKRReportSettingsRequest struct {
	ReportPeriod	string	`json:"report_period"`
	DayOfWeek	*int	`json:"day_of_week,omitempty"`
	Hour		int	`json:"hour"`
	Minute		int	`json:"minute"`
}

type OKRReportSettingsResponse struct {
	ID		int64		`json:"id"`
	ReportPeriod	string		`json:"report_period"`
	DayOfWeek	*int		`json:"day_of_week,omitempty"`
	Hour		int		`json:"hour"`
	Minute		int		`json:"minute"`
	Enabled		bool		`json:"enabled"`
	CreatedAt	time.Time	`json:"created_at"`
	UpdatedAt	time.Time	`json:"updated_at"`
	LastReportSent	*time.Time	`json:"last_report_sent,omitempty"`
}

func (h *Handler) SetOKRReportSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	webUserID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logrus.Error("Не удалось извлечь webUserID из контекста в SetOKRReportSettingsHandler")
		http.Error(w, "Ошибка авторизации: webUserID не найден в токене", http.StatusUnauthorized)
		return
	}

	webUser, err := h.userService.GetWebUserByID(ctx, webUserID)
	if err != nil {
		logrus.Errorf("Ошибка при получении web_user %d: %v", webUserID, err)
		http.Error(w, "Ошибка при получении данных пользователя", http.StatusInternalServerError)
		return
	}
	if webUser == nil || len(webUser.TelegramIDs) == 0 {
		logrus.Warnf("Пользователь с ID %d не найден или не имеет привязанных Telegram аккаунтов", webUserID)
		http.Error(w, "Для настройки отчетов требуется привязанный Telegram аккаунт", http.StatusBadRequest)
		return
	}

	var req SetOKRReportSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}

	if req.ReportPeriod != "day" && req.ReportPeriod != "week" && req.ReportPeriod != "month" {
		http.Error(w, "Неверный период отчета. Допустимые значения: day, week, month", http.StatusBadRequest)
		return
	}

	if req.Hour < 0 || req.Hour > 23 {
		http.Error(w, "Неверное значение часа. Должно быть от 0 до 23", http.StatusBadRequest)
		return
	}

	if req.Minute < 0 || req.Minute > 59 {
		http.Error(w, "Неверное значение минуты. Должно быть от 0 до 59", http.StatusBadRequest)
		return
	}

	if req.ReportPeriod == "week" {
		if req.DayOfWeek == nil {
			http.Error(w, "Для еженедельных отчетов необходимо указать день недели", http.StatusBadRequest)
			return
		}
		if *req.DayOfWeek < 1 || *req.DayOfWeek > 7 {
			http.Error(w, "Неверный день недели. Должно быть от 1 (Пн) до 7 (Вс)", http.StatusBadRequest)
			return
		}
	}

	telegramID := webUser.TelegramIDs[0]

	settings, err := h.okrService.SetReportSettings(ctx, telegramID, req.ReportPeriod, req.DayOfWeek, req.Hour, req.Minute)
	if err != nil {
		logrus.Errorf("Ошибка при установке настроек отчетов: %v", err)
		http.Error(w, "Ошибка при сохранении настроек отчетов", http.StatusInternalServerError)
		return
	}

	response := OKRReportSettingsResponse{
		ID:		settings.ID,
		ReportPeriod:	settings.ReportPeriod,
		DayOfWeek:	settings.DayOfWeek,
		Hour:		settings.Hour,
		Minute:		settings.Minute,
		Enabled:	settings.Enabled,
		CreatedAt:	settings.CreatedAt,
		UpdatedAt:	settings.UpdatedAt,
		LastReportSent:	settings.LastReportSent,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) DisableOKRReportSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	webUserID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logrus.Error("Не удалось извлечь webUserID из контекста в DisableOKRReportSettingsHandler")
		http.Error(w, "Ошибка авторизации: webUserID не найден в токене", http.StatusUnauthorized)
		return
	}

	webUser, err := h.userService.GetWebUserByID(ctx, webUserID)
	if err != nil {
		logrus.Errorf("Ошибка при получении web_user %d: %v", webUserID, err)
		http.Error(w, "Ошибка при получении данных пользователя", http.StatusInternalServerError)
		return
	}
	if webUser == nil || len(webUser.TelegramIDs) == 0 {
		logrus.Warnf("Пользователь с ID %d не найден или не имеет привязанных Telegram аккаунтов", webUserID)
		http.Error(w, "Для отключения отчетов требуется привязанный Telegram аккаунт", http.StatusBadRequest)
		return
	}

	telegramID := webUser.TelegramIDs[0]

	err = h.okrService.DisableReportSettings(ctx, telegramID)
	if err != nil {
		logrus.Errorf("Ошибка при отключении отчетов: %v", err)
		http.Error(w, "Ошибка при отключении отчетов", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Отчеты OKR отключены"})
}

func (h *Handler) GetOKRReportSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	webUserID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logrus.Error("Не удалось извлечь webUserID из контекста в GetOKRReportSettingsHandler")
		http.Error(w, "Ошибка авторизации: webUserID не найден в токене", http.StatusUnauthorized)
		return
	}

	webUser, err := h.userService.GetWebUserByID(ctx, webUserID)
	if err != nil {
		logrus.Errorf("Ошибка при получении web_user %d: %v", webUserID, err)
		http.Error(w, "Ошибка при получении данных пользователя", http.StatusInternalServerError)
		return
	}
	if webUser == nil || len(webUser.TelegramIDs) == 0 {
		logrus.Warnf("Пользователь с ID %d не найден или не имеет привязанных Telegram аккаунтов", webUserID)
		http.Error(w, "Для получения настроек отчетов требуется привязанный Telegram аккаунт", http.StatusBadRequest)
		return
	}

	telegramID := webUser.TelegramIDs[0]

	settings, err := h.okrService.GetReportSettings(ctx, telegramID)
	if err != nil {
		logrus.Warnf("Настройки отчетов не найдены для пользователя %d: %v", telegramID, err)
		http.Error(w, "Настройки отчетов не найдены", http.StatusNotFound)
		return
	}

	response := OKRReportSettingsResponse{
		ID:		settings.ID,
		ReportPeriod:	settings.ReportPeriod,
		DayOfWeek:	settings.DayOfWeek,
		Hour:		settings.Hour,
		Minute:		settings.Minute,
		Enabled:	settings.Enabled,
		CreatedAt:	settings.CreatedAt,
		UpdatedAt:	settings.UpdatedAt,
		LastReportSent:	settings.LastReportSent,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetGoogleAuthURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	webUserID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		logrus.Error("Не удалось извлечь webUserID из контекста в GetGoogleAuthURLHandler")
		http.Error(w, "Ошибка авторизации: webUserID не найден в токене", http.StatusUnauthorized)
		return
	}

	webUser, err := h.userService.GetWebUserByID(ctx, webUserID)
	if err != nil {
		logrus.Errorf("Ошибка при получении web_user %d: %v", webUserID, err)
		http.Error(w, "Ошибка при получении данных пользователя", http.StatusInternalServerError)
		return
	}
	if webUser == nil || len(webUser.TelegramIDs) == 0 {
		logrus.Warnf("Пользователь с ID %d не найден или не имеет привязанных Telegram аккаунтов", webUserID)
		http.Error(w, "Для подключения Google Calendar требуется привязанный Telegram аккаунт", http.StatusBadRequest)
		return
	}

	telegramID := webUser.TelegramIDs[0]

	authURL, err := h.calendarService.GetGoogleAuthURL(telegramID, "web")
	if err != nil {
		logrus.Errorf("Ошибка при создании URL авторизации Google: %v", err)
		http.Error(w, "Не удалось создать URL авторизации Google", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"auth_url": authURL})
}

func (h *Handler) HandleGoogleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	code := r.URL.Query().Get("code")
	if code == "" {
		err := r.URL.Query().Get("error")
		logrus.Errorf("Google OAuth ошибка: %s", err)
		http.Error(w, "Авторизация в Google была отменена или произошла ошибка", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	parts := strings.Split(state, ":")
	if len(parts) != 2 {
		logrus.Errorf("Некорректный формат state: %s", state)
		http.Error(w, "Некорректный формат state", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		logrus.Errorf("Не удалось извлечь user_id из state: %v", err)
		http.Error(w, "Некорректный параметр state", http.StatusBadRequest)
		return
	}

	callbackType := parts[1]
	if callbackType != "web" {
		logrus.Errorf("Некорректный тип callback: %s", callbackType)
		http.Error(w, "Некорректный тип callback", http.StatusBadRequest)
		return
	}

	err = h.calendarService.HandleGoogleCallback(ctx, code, userID)
	if err != nil {
		logrus.Errorf("Ошибка при обработке Google callback: %v", err)
		http.Error(w, "Не удалось завершить авторизацию Google", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Google Calendar подключен</title>
			<style>
				body { font-family: Arial, sans-serif; text-align: center; margin-top: 50px; }
				.success { color: green; font-size: 24px; margin-bottom: 20px; }
				.info { color: #333; margin-bottom: 20px; }
				.close { background-color: #4CAF50; color: white; padding: 10px 20px; 
					border: none; border-radius: 4px; cursor: pointer; }
			</style>
		</head>
		<body>
			<div class="success">Google Calendar успешно подключен!</div>
			<div class="info">Теперь вы можете закрыть это окно и вернуться в приложение.</div>
			<button class="close" onclick="window.close();">Закрыть окно</button>
			<script>
				// Автоматически закрываем окно через 5 секунд
				setTimeout(function() {
					window.close();
				}, 5000);
			</script>
		</body>
		</html>
	`))
}

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"telegrambot/internal/api"
	"telegrambot/internal/auth"
	"telegrambot/internal/calendar"
	"telegrambot/internal/chatgpt"
	"telegrambot/internal/finance"
	"telegrambot/internal/linking"
	"telegrambot/internal/meetings"
	"telegrambot/internal/messagestore"
	"telegrambot/internal/middleware"
	"telegrambot/internal/okr"
	"telegrambot/internal/telegram"
	"telegrambot/internal/users"
	"telegrambot/pkg/config"
	"telegrambot/pkg/db"

	"github.com/sirupsen/logrus"
)

func main() {

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	cfg := config.LoadConfig()

	database, err := db.NewPostgresDB(cfg)
	if err != nil {
		logrus.Fatalf("Ошибка при подключении к базе данных: %v", err)
	}
	defer database.Close()

	chatgptService := chatgpt.NewChatGPTService(cfg, database)
	calendarService := calendar.NewService(database, cfg)
	meetingsService := meetings.NewService(database)
	financeService := finance.NewService(database)
	okrService := okr.NewService(database)
	userRepo := users.NewRepository(database)
	userService := users.NewService(userRepo)
	linkingSvc := linking.NewService()

	messageStoreRepo := messagestore.NewRepository(database)
	messageStoreService := messagestore.NewService(messageStoreRepo)

	telegramHandler, err := telegram.NewHandler(
		cfg,
		chatgptService,
		calendarService,
		meetingsService,
		financeService,
		okrService,
		messageStoreService,
		userService,
		linkingSvc,
		database,
	)
	if err != nil {
		logrus.Fatalf("Ошибка при инициализации Telegram бота: %v", err)
	}

	var botUsername string
	if telegramHandler != nil && telegramHandler.GetBotInfo() != nil {
		botUsername = telegramHandler.GetBotInfo().UserName
	} else {
		logrus.Warn("Не удалось получить имя пользователя бота для API Handler. Ссылки на привязку Telegram могут быть неполными.")
	}

	apiHandler := api.NewHandler(
		calendarService,
		userService,
		linkingSvc,
		okrService,
		database,
		cfg.JWTSigningKey,
		botUsername,
	)

	calendarService.StartReminderChecker(telegramHandler.SendMessage)
	calendarService.StartGoogleCalendarSync()

	okrService.StartReportChecker(telegramHandler.SendMessage)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", telegramHandler.HandleWebhook)

	mux.Handle("/api/auth/login", middleware.CORSMiddleware(http.HandlerFunc(apiHandler.AuthLoginHandler)))

	mux.Handle("/api/auth/register", middleware.CORSMiddleware(http.HandlerFunc(apiHandler.RegisterWebUserHandler)))

	linkTelegramHandler := http.HandlerFunc(apiHandler.GenerateTelegramLinkHandler)
	mux.Handle("/api/users/me/link-telegram", middleware.CORSMiddleware(auth.JWTMiddleware(linkTelegramHandler, cfg.JWTSigningKey)))

	calendarEventsHandler := http.HandlerFunc(apiHandler.GetCalendarEvents)
	mux.Handle("/api/calendar/events", middleware.CORSMiddleware(auth.JWTMiddleware(calendarEventsHandler, cfg.JWTSigningKey)))

	createEventHandler := http.HandlerFunc(apiHandler.CreateCalendarEventHandler)
	mux.Handle("/api/calendar/event/create", middleware.CORSMiddleware(auth.JWTMiddleware(createEventHandler, cfg.JWTSigningKey)))

	updateEventHandler := http.HandlerFunc(apiHandler.UpdateCalendarEventHandler)
	mux.Handle("/api/calendar/event/update", middleware.CORSMiddleware(auth.JWTMiddleware(updateEventHandler, cfg.JWTSigningKey)))

	deleteEventHandler := http.HandlerFunc(apiHandler.DeleteCalendarEventHandler)
	mux.Handle("/api/calendar/event/delete", middleware.CORSMiddleware(auth.JWTMiddleware(deleteEventHandler, cfg.JWTSigningKey)))

	setOKRReportSettingsHandler := http.HandlerFunc(apiHandler.SetOKRReportSettingsHandler)
	mux.Handle("/api/okr/report-settings/set", middleware.CORSMiddleware(auth.JWTMiddleware(setOKRReportSettingsHandler, cfg.JWTSigningKey)))

	disableOKRReportSettingsHandler := http.HandlerFunc(apiHandler.DisableOKRReportSettingsHandler)
	mux.Handle("/api/okr/report-settings/disable", middleware.CORSMiddleware(auth.JWTMiddleware(disableOKRReportSettingsHandler, cfg.JWTSigningKey)))

	getOKRReportSettingsHandler := http.HandlerFunc(apiHandler.GetOKRReportSettingsHandler)
	mux.Handle("/api/okr/report-settings/get", middleware.CORSMiddleware(auth.JWTMiddleware(getOKRReportSettingsHandler, cfg.JWTSigningKey)))

	getGoogleAuthURLHandler := http.HandlerFunc(apiHandler.GetGoogleAuthURLHandler)
	mux.Handle("/api/calendar/google/auth-url", middleware.CORSMiddleware(auth.JWTMiddleware(getGoogleAuthURLHandler, cfg.JWTSigningKey)))

	mux.Handle("/api/calendar/google/callback", middleware.CORSMiddleware(http.HandlerFunc(apiHandler.HandleGoogleCallbackHandler)))

	server := &http.Server{
		Addr:		":" + cfg.ServerPort,
		Handler:	mux,
	}

	go func() {
		logrus.Infof("Сервер запущен на порту %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Ошибка при запуске сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Завершение работы сервера...")

	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatalf("Ошибка при остановке сервера: %v", err)
	}

	logrus.Info("Сервер остановлен")
}

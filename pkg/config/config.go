package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	PostgresHost		string
	PostgresPort		string
	PostgresUser		string
	PostgresPassword	string
	PostgresDB		string
	TelegramToken		string
	OpenAIKey		string
	GoogleCalendarID	string
	GoogleCredentials	string
	ServerHost		string
	ServerPort		string
	JWTSigningKey		string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		logrus.Warn("Не найден файл .env")
	}

	return &Config{
		PostgresHost:		getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:		getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:		getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword:	getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:		getEnv("POSTGRES_DB", "telegrambot"),
		TelegramToken:		getEnv("TELEGRAM_TOKEN", ""),
		OpenAIKey:		getEnv("OPENAI_KEY", ""),
		GoogleCalendarID:	getEnv("GOOGLE_CALENDAR_ID", ""),
		GoogleCredentials:	getEnv("GOOGLE_CREDENTIALS", ""),
		ServerHost:		getEnv("SERVER_HOST", "0.0.0.0"),
		ServerPort:		getEnv("SERVER_PORT", "8080"),
		JWTSigningKey:		getEnv("JWT_SIGNING_KEY", "your-secret-signing-key"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

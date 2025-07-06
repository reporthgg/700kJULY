package messagestore

import (
	"context"
	"telegrambot/internal/messagestore/models"

	"github.com/sirupsen/logrus"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) StoreUserMessage(ctx context.Context, userID string, messageText string, platform string) (int, error) {
	logrus.Debugf("Сохранение сообщения пользователя %s: %s", userID, messageText)
	return s.repo.StoreUserMessage(ctx, userID, messageText, platform)
}

func (s *Service) StoreAiResponse(ctx context.Context, userMessageID int, responseText string, promptTokens, completionTokens *int) error {
	logrus.Debugf("Сохранение ответа ИИ на сообщение %d", userMessageID)
	return s.repo.StoreAiResponse(ctx, userMessageID, responseText, promptTokens, completionTokens)
}

func (s *Service) GetMessageHistory(ctx context.Context, userID string) ([]models.MessageHistoryItem, error) {
	logrus.Debugf("Получение истории сообщений пользователя %s", userID)
	return s.repo.GetMessageHistoryChronological(ctx, userID)
}

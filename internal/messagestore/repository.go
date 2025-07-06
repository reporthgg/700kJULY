package messagestore

import (
	"context"
	"fmt"
	"telegrambot/internal/messagestore/models"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) StoreUserMessage(ctx context.Context, userID string, messageText string, platform string) (int, error) {
	query := `
		INSERT INTO user_messages (user_identifier, message_text, platform, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id
	`

	var messageID int
	err := r.db.GetContext(ctx, &messageID, query, userID, messageText, platform)
	if err != nil {
		return 0, fmt.Errorf("не удалось сохранить сообщение пользователя: %w", err)
	}

	return messageID, nil
}

func (r *Repository) StoreAiResponse(ctx context.Context, userMessageID int, responseText string, promptTokens, completionTokens *int) error {
	query := `
		INSERT INTO ai_responses (user_message_id, response_text, prompt_tokens, completion_tokens, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := r.db.ExecContext(ctx, query, userMessageID, responseText, promptTokens, completionTokens)
	if err != nil {
		return fmt.Errorf("не удалось сохранить ответ ИИ: %w", err)
	}

	return nil
}

func (r *Repository) GetMessageHistory(ctx context.Context, userID string) ([]models.MessageHistoryItem, error) {

	query := `
		-- Сначала получаем сообщения пользователя
		SELECT 
			'user' as role, 
			um.message_text as content
		FROM 
			user_messages um
		WHERE 
			um.user_identifier = $1
			AND um.created_at > NOW() - INTERVAL '24 hours'
		
		UNION ALL
		
		-- Затем получаем ответы ИИ
		SELECT 
			'assistant' as role, 
			ar.response_text as content
		FROM 
			ai_responses ar
		JOIN 
			user_messages um ON ar.user_message_id = um.id
		WHERE 
			um.user_identifier = $1
			AND ar.created_at > NOW() - INTERVAL '24 hours'
		
		-- Сортируем по времени создания
		ORDER BY 
			role, content
	`

	var history []models.MessageHistoryItem
	err := r.db.SelectContext(ctx, &history, query, userID)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить историю сообщений: %w", err)
	}

	logrus.Infof("Получено %d элементов истории сообщений для пользователя %s", len(history), userID)
	return history, nil
}

func (r *Repository) GetMessageHistoryChronological(ctx context.Context, userID string) ([]models.MessageHistoryItem, error) {

	query := `
		SELECT
			'user' as role,
			um.message_text as content,
			um.created_at as created_at
		FROM
			user_messages um
		WHERE
			um.user_identifier = $1
			AND um.created_at > NOW() - INTERVAL '24 hours'

		UNION ALL

		SELECT
			'assistant' as role,
			ar.response_text as content,
			ar.created_at as created_at
		FROM
			ai_responses ar
		JOIN
			user_messages um ON ar.user_message_id = um.id
		WHERE
			um.user_identifier = $1
			AND ar.created_at > NOW() - INTERVAL '1 hours'

		ORDER BY
			created_at ASC
	`

	type messageWithTime struct {
		Role		string		`db:"role"`
		Content		string		`db:"content"`
		CreatedAt	time.Time	`db:"created_at"`
	}

	var messagesWithTime []messageWithTime
	err := r.db.SelectContext(ctx, &messagesWithTime, query, userID)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить хронологическую историю сообщений: %w", err)
	}

	history := make([]models.MessageHistoryItem, len(messagesWithTime))
	for i, msg := range messagesWithTime {
		history[i] = models.MessageHistoryItem{
			Role:		msg.Role,
			Content:	msg.Content,
		}
	}

	logrus.Infof("Получено %d элементов хронологической истории для пользователя %s", len(history), userID)
	return history, nil
}

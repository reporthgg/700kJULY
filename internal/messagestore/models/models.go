package models

import (
	"time"
)

type UserMessage struct {
	ID		int		`db:"id" json:"id"`
	UserIdentifier	string		`db:"user_identifier" json:"user_identifier"`
	MessageText	string		`db:"message_text" json:"message_text"`
	Platform	string		`db:"platform" json:"platform"`
	CreatedAt	time.Time	`db:"created_at" json:"created_at"`
}

type AiResponse struct {
	ID			int		`db:"id" json:"id"`
	UserMessageID		int		`db:"user_message_id" json:"user_message_id"`
	ResponseText		string		`db:"response_text" json:"response_text"`
	PromptTokens		*int		`db:"prompt_tokens" json:"prompt_tokens,omitempty"`
	CompletionTokens	*int		`db:"completion_tokens" json:"completion_tokens,omitempty"`
	CreatedAt		time.Time	`db:"created_at" json:"created_at"`
}

type MessageHistoryItem struct {
	Role	string	`json:"role"`
	Content	string	`json:"content"`
}

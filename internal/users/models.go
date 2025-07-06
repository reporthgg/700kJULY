package users

import (
	"time"

	"github.com/lib/pq"
)

type WebUser struct {
	ID		int64		`db:"id" json:"id"`
	Login		string		`db:"login" json:"login"`
	Email		*string		`db:"email" json:"email,omitempty"`
	Phone		*string		`db:"phone" json:"phone,omitempty"`
	PasswordHash	string		`db:"password_hash" json:"-"`
	TelegramIDs	pq.Int64Array	`db:"telegram_ids" json:"telegram_ids,omitempty"`
	CreatedAt	time.Time	`db:"created_at" json:"created_at"`
	UpdatedAt	time.Time	`db:"updated_at" json:"updated_at"`
}

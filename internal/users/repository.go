package users

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, login string, passwordHash string, email *string, phone *string) (*WebUser, error) {
	query := `
		INSERT INTO web_users (login, password_hash, email, phone, telegram_ids)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, login, email, phone, password_hash, telegram_ids, created_at, updated_at
	`

	initialTelegramIDs := pq.Int64Array{}
	var user WebUser
	err := r.db.GetContext(ctx, &user, query, login, passwordHash, email, phone, initialTelegramIDs)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании web_user: %w", err)
	}
	return &user, nil
}

func (r *Repository) GetUserByLogin(ctx context.Context, login string) (*WebUser, error) {
	query := `
		SELECT id, login, email, phone, password_hash, telegram_ids, created_at, updated_at
		FROM web_users
		WHERE login = $1
	`
	var user WebUser
	err := r.db.GetContext(ctx, &user, query, login)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка при получении web_user по логину: %w", err)
	}
	return &user, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id int64) (*WebUser, error) {
	query := `
		SELECT id, login, email, phone, password_hash, telegram_ids, created_at, updated_at
		FROM web_users
		WHERE id = $1
	`
	var user WebUser
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка при получении web_user по ID: %w", err)
	}
	return &user, nil
}

func (r *Repository) AddTelegramIDToWebUser(ctx context.Context, webUserID int64, telegramID int64) (pq.Int64Array, error) {
	query := `
		UPDATE web_users
		SET telegram_ids = array_append(COALESCE(telegram_ids, '{}'), $2)
		WHERE id = $1
		AND NOT ($2 = ANY(COALESCE(telegram_ids, '{}')))
		RETURNING telegram_ids
	`

	var updatedTelegramIDs pq.Int64Array
	err := r.db.GetContext(ctx, &updatedTelegramIDs, query, webUserID, telegramID)
	if err != nil {
		if err == sql.ErrNoRows {

			currentUser, getErr := r.GetUserByID(ctx, webUserID)
			if getErr != nil {
				return nil, fmt.Errorf("ошибка при получении пользователя %d после попытки добавления telegram_id: %w", webUserID, getErr)
			}
			if currentUser == nil {
				return nil, fmt.Errorf("web_user с ID %d не найден при попытке добавить telegram_id", webUserID)
			}
			return currentUser.TelegramIDs, nil
		}
		return nil, fmt.Errorf("ошибка при добавлении telegram_id %d к web_user %d: %w", telegramID, webUserID, err)
	}
	return updatedTelegramIDs, nil
}

func (r *Repository) GetWebUserByTelegramID(ctx context.Context, telegramID int64) (*WebUser, error) {
	query := `
		SELECT id, login, email, phone, password_hash, telegram_ids, created_at, updated_at
		FROM web_users
		WHERE $1 = ANY(telegram_ids)
		LIMIT 1 
		-- LIMIT 1 если мы ожидаем, что telegram_id уникально связан только с одним web_user
	`
	var user WebUser
	err := r.db.GetContext(ctx, &user, query, telegramID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка при получении web_user по telegram_id %d: %w", telegramID, err)
	}
	return &user, nil
}

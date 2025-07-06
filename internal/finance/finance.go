package finance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Service struct {
	db *sqlx.DB
}

type Transaction struct {
	ID		string		`db:"id"`
	UserID		int64		`db:"user_id"`
	Amount		float64		`db:"amount"`
	Details		string		`db:"details"`
	Category	string		`db:"category"`
	CreatedAt	time.Time	`db:"created_at"`
}

type Summary struct {
	Income		float64
	Expenses	float64
	Balance		float64
	Categories	map[string]float64
}

func NewService(db *sqlx.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) AddTransaction(ctx context.Context, userID int64, amount float64, details, category string) (string, error) {

	transactionID := uuid.New().String()

	if category == "" {
		if amount > 0 {
			category = "Доход"
		} else {
			category = "Расход"
		}
	}

	query := `
		INSERT INTO transactions (id, user_id, amount, details, category, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.ExecContext(ctx, query, transactionID, userID, amount, details, category, time.Now())
	if err != nil {
		return "", fmt.Errorf("ошибка при сохранении транзакции: %v", err)
	}

	return transactionID, nil
}

func (s *Service) GetTransactions(ctx context.Context, userID int64, startTime, endTime time.Time) ([]Transaction, error) {
	query := `
		SELECT id, user_id, amount, details, category, created_at
		FROM transactions
		WHERE user_id = $1 AND created_at BETWEEN $2 AND $3
		ORDER BY created_at DESC
	`

	var transactions []Transaction
	err := s.db.SelectContext(ctx, &transactions, query, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении транзакций: %v", err)
	}

	return transactions, nil
}

func (s *Service) GetSummary(ctx context.Context, userID int64, period string) (*Summary, error) {

	now := time.Now()
	var startTime time.Time

	switch period {
	case "day":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		startTime = now.AddDate(0, 0, -7)
	case "month":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case "year":
		startTime = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		return nil, fmt.Errorf("неизвестный период: %s", period)
	}

	transactions, err := s.GetTransactions(ctx, userID, startTime, now)
	if err != nil {
		return nil, err
	}

	summary := &Summary{
		Income:		0,
		Expenses:	0,
		Balance:	0,
		Categories:	make(map[string]float64),
	}

	for _, t := range transactions {
		if t.Amount > 0 {
			summary.Income += t.Amount
		} else {
			summary.Expenses += -t.Amount
		}
		summary.Balance += t.Amount
		summary.Categories[t.Category] += t.Amount
	}

	return summary, nil
}

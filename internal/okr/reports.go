package okr

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type ReportSettings struct {
	ID		int64		`db:"id"`
	UserID		int64		`db:"user_id"`
	ReportPeriod	string		`db:"report_period"`
	DayOfWeek	*int		`db:"day_of_week"`
	Hour		int		`db:"hour"`
	Minute		int		`db:"minute"`
	Enabled		bool		`db:"enabled"`
	CreatedAt	time.Time	`db:"created_at"`
	UpdatedAt	time.Time	`db:"updated_at"`
	LastReportSent	*time.Time	`db:"last_report_sent"`
}

func (s *Service) SetReportSettings(ctx context.Context, userID int64, reportPeriod string,
	dayOfWeek *int, hour, minute int) (*ReportSettings, error) {

	reportPeriod = strings.ToLower(reportPeriod)
	if reportPeriod != "day" && reportPeriod != "week" && reportPeriod != "month" {
		return nil, fmt.Errorf("неверный период отчета: %s. Допустимые значения: day, week, month", reportPeriod)
	}

	if hour < 0 || hour > 23 {
		return nil, fmt.Errorf("неверное значение часа: %d. Должно быть от 0 до 23", hour)
	}
	if minute < 0 || minute > 59 {
		return nil, fmt.Errorf("неверное значение минуты: %d. Должно быть от 0 до 59", minute)
	}

	if reportPeriod == "week" {
		if dayOfWeek == nil {
			return nil, fmt.Errorf("для еженедельных отчетов необходимо указать день недели")
		}
		if *dayOfWeek < 1 || *dayOfWeek > 7 {
			return nil, fmt.Errorf("неверный день недели: %d. Должно быть от 1 (Понедельник) до 7 (Воскресенье)", *dayOfWeek)
		}
	} else if dayOfWeek != nil {

		dayOfWeek = nil
	}

	var existingID int64
	query := `SELECT id FROM okr_report_settings WHERE user_id = $1`
	err := s.db.GetContext(ctx, &existingID, query, userID)

	now := time.Now()

	if err == nil {

		query = `
			UPDATE okr_report_settings
			SET report_period = $1, day_of_week = $2, hour = $3, minute = $4, 
				enabled = true, updated_at = $5
			WHERE id = $6
			RETURNING id, user_id, report_period, day_of_week, hour, minute, 
				enabled, created_at, updated_at, last_report_sent
		`

		var settings ReportSettings
		err = s.db.GetContext(
			ctx,
			&settings,
			query,
			reportPeriod,
			dayOfWeek,
			hour,
			minute,
			now,
			existingID,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка при обновлении настроек отчетов: %v", err)
		}

		return &settings, nil
	}

	query = `
		INSERT INTO okr_report_settings 
		(user_id, report_period, day_of_week, hour, minute, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, $6, $7)
		RETURNING id, user_id, report_period, day_of_week, hour, minute, 
			enabled, created_at, updated_at, last_report_sent
	`

	var settings ReportSettings
	err = s.db.GetContext(
		ctx,
		&settings,
		query,
		userID,
		reportPeriod,
		dayOfWeek,
		hour,
		minute,
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании настроек отчетов: %v", err)
	}

	return &settings, nil
}

func (s *Service) GetReportSettings(ctx context.Context, userID int64) (*ReportSettings, error) {
	query := `
		SELECT id, user_id, report_period, day_of_week, hour, minute, 
			enabled, created_at, updated_at, last_report_sent
		FROM okr_report_settings
		WHERE user_id = $1
	`

	var settings ReportSettings
	err := s.db.GetContext(ctx, &settings, query, userID)
	if err != nil {
		return nil, fmt.Errorf("настройки отчетов не найдены: %v", err)
	}

	return &settings, nil
}

func (s *Service) DisableReportSettings(ctx context.Context, userID int64) error {
	query := `
		UPDATE okr_report_settings
		SET enabled = false, updated_at = $1
		WHERE user_id = $2
	`

	result, err := s.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("ошибка при отключении отчетов: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("настройки отчетов для пользователя не найдены")
	}

	return nil
}

func (s *Service) GenerateReport(ctx context.Context, userID int64, period string) (string, error) {
	now := time.Now()
	var startDate time.Time

	switch period {
	case "day":

		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":

		daysFromMonday := int(now.Weekday()) - 1
		if daysFromMonday < 0 {
			daysFromMonday = 6
		}
		startDate = time.Date(now.Year(), now.Month(), now.Day()-daysFromMonday, 0, 0, 0, 0, now.Location())
	case "month":

		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		return "", fmt.Errorf("неподдерживаемый период отчета: %s", period)
	}

	objectives, err := s.GetObjectivesByDateRange(ctx, userID, startDate, now)
	if err != nil {
		return "", fmt.Errorf("ошибка при получении целей: %v", err)
	}

	if len(objectives) == 0 {
		return fmt.Sprintf("За период %s у вас нет активных целей OKR.", formatPeriodRussian(period, startDate, now)), nil
	}

	var reportBuilder strings.Builder
	reportBuilder.WriteString(fmt.Sprintf("📊 *Отчет по OKR за %s*\n\n", formatPeriodRussian(period, startDate, now)))

	for i, obj := range objectives {

		keyResults, err := s.GetKeyResultsForObjective(ctx, obj.ID)
		if err != nil {
			logrus.Errorf("Ошибка при получении ключевых результатов для цели %s: %v", obj.ID, err)
			continue
		}

		var totalProgress float64
		if len(keyResults) > 0 {
			for _, kr := range keyResults {
				totalProgress += kr.Progress
			}
			totalProgress /= float64(len(keyResults))
		}

		reportBuilder.WriteString(fmt.Sprintf("*Цель %d*: %s\n", i+1, obj.Title))
		reportBuilder.WriteString(fmt.Sprintf("Сфера: %s\n", obj.Sphere))
		reportBuilder.WriteString(fmt.Sprintf("Общий прогресс: %.0f%%\n\n", totalProgress))

		if len(keyResults) == 0 {
			reportBuilder.WriteString("Нет активных ключевых результатов\n\n")
			continue
		}

		reportBuilder.WriteString("*Ключевые результаты:*\n")
		for j, kr := range keyResults {

			tasks, err := s.GetTasksForKeyResult(ctx, kr.ID)
			if err != nil {
				logrus.Errorf("Ошибка при получении задач для ключевого результата %d: %v", kr.ID, err)
			}

			reportBuilder.WriteString(fmt.Sprintf("%d. %s: %.0f%% (%.1f/%s %s)\n",
				j+1, kr.Title, kr.Progress, kr.Progress*kr.Target/100, formatFloat(kr.Target), kr.Unit))

			if len(tasks) > 0 {
				var completedTasks int
				for _, task := range tasks {
					if task.Progress >= 99.9 {
						completedTasks++
					}
				}
				reportBuilder.WriteString(fmt.Sprintf("   ✅ Выполнено задач: %d из %d\n", completedTasks, len(tasks)))
			}
		}

		reportBuilder.WriteString("\n")
	}

	reportBuilder.WriteString("Продолжайте двигаться к своим целям! 💪")

	return reportBuilder.String(), nil
}

func (s *Service) UpdateLastReportSent(ctx context.Context, userID int64) error {
	query := `
		UPDATE okr_report_settings
		SET last_report_sent = $1, updated_at = $1
		WHERE user_id = $2
	`

	_, err := s.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении времени последнего отчета: %v", err)
	}

	return nil
}

func (s *Service) StartReportChecker(sendMessageFunc func(chatID int64, text string) error) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			s.checkAndSendReports(sendMessageFunc)
		}
	}()

	logrus.Info("Запущен механизм периодической отправки отчетов OKR")
}

func (s *Service) checkAndSendReports(sendMessageFunc func(chatID int64, text string) error) {
	ctx := context.Background()
	now := time.Now()

	query := `
		SELECT id, user_id, report_period, day_of_week, hour, minute, 
			enabled, created_at, updated_at, last_report_sent
		FROM okr_report_settings
		WHERE enabled = true
	`

	var settings []ReportSettings
	err := s.db.SelectContext(ctx, &settings, query)
	if err != nil {
		logrus.Errorf("Ошибка при получении настроек отчетов: %v", err)
		return
	}

	for _, setting := range settings {
		shouldSendReport := false

		if now.Hour() == setting.Hour && now.Minute() == setting.Minute {

			if setting.ReportPeriod == "day" {
				shouldSendReport = true
			}

			if setting.ReportPeriod == "week" && setting.DayOfWeek != nil {
				weekday := int(now.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				if weekday == *setting.DayOfWeek {
					shouldSendReport = true
				}
			}

			if setting.ReportPeriod == "month" {
				lastDayOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
				if now.Day() == lastDayOfMonth {
					shouldSendReport = true
				}
			}
		}

		if shouldSendReport {

			if setting.LastReportSent != nil {
				lastSent := *setting.LastReportSent
				if lastSent.Year() == now.Year() &&
					lastSent.Month() == now.Month() &&
					lastSent.Day() == now.Day() &&
					lastSent.Hour() == now.Hour() &&
					now.Sub(lastSent).Minutes() < 10 {

					continue
				}
			}

			report, err := s.GenerateReport(ctx, setting.UserID, setting.ReportPeriod)
			if err != nil {
				logrus.Errorf("Ошибка при генерации отчета для пользователя %d: %v", setting.UserID, err)
				continue
			}

			err = sendMessageFunc(setting.UserID, report)
			if err != nil {
				logrus.Errorf("Ошибка при отправке отчета пользователю %d: %v", setting.UserID, err)
				continue
			}

			s.UpdateLastReportSent(ctx, setting.UserID)
			logrus.Infof("Отправлен отчет OKR пользователю %d за период %s", setting.UserID, setting.ReportPeriod)
		}
	}
}

func formatPeriodRussian(period string, startDate, endDate time.Time) string {
	switch period {
	case "day":
		return fmt.Sprintf("день %02d.%02d.%d", startDate.Day(), startDate.Month(), startDate.Year())
	case "week":
		return fmt.Sprintf("неделю %02d.%02d - %02d.%02d.%d",
			startDate.Day(), startDate.Month(), endDate.Day(), endDate.Month(), endDate.Year())
	case "month":
		months := []string{
			"января", "февраля", "марта", "апреля", "мая", "июня",
			"июля", "августа", "сентября", "октября", "ноября", "декабря",
		}
		return fmt.Sprintf("месяц %s %d", months[startDate.Month()-1], startDate.Year())
	default:
		return period
	}
}

func formatFloat(value float64) string {
	if value == float64(int(value)) {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.1f", value)
}

func (s *Service) GetObjectivesByDateRange(ctx context.Context, userID int64, startDate, endDate time.Time) ([]Objective, error) {
	query := `
		SELECT id, user_id, title, sphere, period, deadline, created_at
		FROM objectives
		WHERE user_id = $1 AND (
			(deadline IS NULL) OR
			(deadline >= $2)
		)
		ORDER BY created_at DESC
	`

	var objectives []Objective
	err := s.db.SelectContext(ctx, &objectives, query, userID, startDate)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении целей за период: %v", err)
	}

	return objectives, nil
}

func (s *Service) GetTasksForKeyResult(ctx context.Context, keyResultID int64) ([]Task, error) {
	query := `
		SELECT id, key_result_id, title, target, unit, progress, deadline, created_at
		FROM tasks
		WHERE key_result_id = $1
		ORDER BY created_at DESC
	`

	var tasks []Task
	err := s.db.SelectContext(ctx, &tasks, query, keyResultID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении задач для ключевого результата: %v", err)
	}

	return tasks, nil
}

package okr

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Service struct {
	db *sqlx.DB
}

type Objective struct {
	ID		string		`db:"id"`
	UserID		int64		`db:"user_id"`
	Title		string		`db:"title"`
	Sphere		string		`db:"sphere"`
	Period		string		`db:"period"`
	Deadline	*time.Time	`db:"deadline"`
	CreatedAt	time.Time	`db:"created_at"`
}

type KeyResult struct {
	ID		int64		`db:"id"`
	ObjectiveID	string		`db:"objective_id"`
	Title		string		`db:"title"`
	Target		float64		`db:"target"`
	Unit		string		`db:"unit"`
	Progress	float64		`db:"progress"`
	Deadline	*time.Time	`db:"deadline"`
	CreatedAt	time.Time	`db:"created_at"`
}

type Task struct {
	ID		int64		`db:"id"`
	KeyResultID	int64		`db:"key_result_id"`
	Title		string		`db:"title"`
	Target		float64		`db:"target"`
	Unit		string		`db:"unit"`
	Progress	float64		`db:"progress"`
	Deadline	*time.Time	`db:"deadline"`
	CreatedAt	time.Time	`db:"created_at"`
}

func NewService(db *sqlx.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) CreateObjective(ctx context.Context, userID int64, title, sphere, period string, deadline *time.Time, keyResults []KeyResult) (string, error) {

	objectiveID := uuid.New().String()

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка при начале транзакции: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := `
		INSERT INTO objectives (id, user_id, title, sphere, period, deadline, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = tx.ExecContext(ctx, query, objectiveID, userID, title, sphere, period, deadline, time.Now())
	if err != nil {
		return "", fmt.Errorf("ошибка при сохранении цели: %v", err)
	}

	for _, kr := range keyResults {
		krQuery := `
			INSERT INTO key_results (objective_id, title, target, unit, progress, deadline, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`

		_, err = tx.ExecContext(ctx, krQuery, objectiveID, kr.Title, kr.Target, kr.Unit, kr.Progress, kr.Deadline, time.Now())
		if err != nil {
			return "", fmt.Errorf("ошибка при сохранении ключевого результата: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return "", fmt.Errorf("ошибка при подтверждении транзакции: %v", err)
	}

	return objectiveID, nil
}

func (s *Service) CreateKeyResult(ctx context.Context, userID int64, objectiveID string, title string, target float64, unit string, deadline *time.Time) (int64, error) {

	checkQuery := `
		SELECT id FROM objectives WHERE id = $1 AND user_id = $2
	`
	var id string
	err := s.db.GetContext(ctx, &id, checkQuery, objectiveID, userID)
	if err != nil {
		return 0, fmt.Errorf("цель не найдена или не принадлежит пользователю: %v", err)
	}

	query := `
		INSERT INTO key_results (objective_id, title, target, unit, progress, deadline, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var keyResultID int64
	err = s.db.GetContext(
		ctx,
		&keyResultID,
		query,
		objectiveID,
		title,
		target,
		unit,
		0.0,
		deadline,
		time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("ошибка при создании ключевого результата: %v", err)
	}

	return keyResultID, nil
}

func (s *Service) CreateTask(ctx context.Context, userID int64, keyResultID int64, title string, target float64, unit string, deadline *time.Time) (int64, error) {

	checkQuery := `
		SELECT kr.id
		FROM key_results kr
		JOIN objectives o ON kr.objective_id = o.id
		WHERE kr.id = $1 AND o.user_id = $2
	`
	var id int64
	err := s.db.GetContext(ctx, &id, checkQuery, keyResultID, userID)
	if err != nil {
		return 0, fmt.Errorf("ключевой результат не найден или не принадлежит пользователю: %v", err)
	}

	query := `
		INSERT INTO tasks (key_result_id, title, target, unit, progress, deadline, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var taskID int64
	err = s.db.GetContext(
		ctx,
		&taskID,
		query,
		keyResultID,
		title,
		target,
		unit,
		0.0,
		deadline,
		time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("ошибка при создании задачи: %v", err)
	}

	return taskID, nil
}

func (s *Service) GetObjectives(ctx context.Context, userID int64) ([]Objective, error) {
	query := `
		SELECT id, user_id, title, sphere, period, deadline, created_at
		FROM objectives
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	var objectives []Objective
	err := s.db.SelectContext(ctx, &objectives, query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении целей: %v", err)
	}

	return objectives, nil
}

func (s *Service) GetKeyResults(ctx context.Context, objectiveID string) ([]KeyResult, error) {
	query := `
		SELECT id, objective_id, title, target, unit, progress, deadline, created_at
		FROM key_results
		WHERE objective_id = $1
		ORDER BY created_at ASC
	`

	var keyResults []KeyResult
	err := s.db.SelectContext(ctx, &keyResults, query, objectiveID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении ключевых результатов: %v", err)
	}

	return keyResults, nil
}

func (s *Service) GetTasks(ctx context.Context, keyResultID int64) ([]Task, error) {
	query := `
		SELECT id, key_result_id, title, target, unit, progress, deadline, created_at
		FROM tasks
		WHERE key_result_id = $1
		ORDER BY created_at ASC
	`

	var tasks []Task
	err := s.db.SelectContext(ctx, &tasks, query, keyResultID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении задач: %v", err)
	}

	return tasks, nil
}

func (s *Service) UpdateKeyResultProgress(ctx context.Context, userID int64, keyResultID int64, progress float64) (bool, error) {

	checkQuery := `
		SELECT kr.id, kr.target
		FROM key_results kr
		JOIN objectives o ON kr.objective_id = o.id
		WHERE kr.id = $1 AND o.user_id = $2
	`

	type result struct {
		ID	int64	`db:"id"`
		Target	float64	`db:"target"`
	}

	var res result
	err := s.db.GetContext(ctx, &res, checkQuery, keyResultID, userID)
	if err != nil {
		return false, fmt.Errorf("ключевой результат не найден или не принадлежит пользователю: %v", err)
	}

	currentProgressQuery := `
		SELECT progress FROM key_results WHERE id = $1
	`
	var currentProgress float64
	err = s.db.GetContext(ctx, &currentProgress, currentProgressQuery, keyResultID)
	if err != nil {
		return false, fmt.Errorf("ошибка при получении текущего прогресса: %v", err)
	}

	newProgress := currentProgress + progress

	exceeded := false
	if newProgress > res.Target {
		exceeded = true
	}

	updateQuery := `
		UPDATE key_results
		SET progress = $1
		WHERE id = $2
	`

	_, err = s.db.ExecContext(ctx, updateQuery, newProgress, keyResultID)
	if err != nil {
		return false, fmt.Errorf("ошибка при обновлении прогресса: %v", err)
	}

	return exceeded, nil
}

func (s *Service) UpdateTaskProgress(ctx context.Context, userID int64, taskID int64, progress float64) (bool, error) {

	checkQuery := `
		SELECT t.id, t.target
		FROM tasks t
		JOIN key_results kr ON t.key_result_id = kr.id
		JOIN objectives o ON kr.objective_id = o.id
		WHERE t.id = $1 AND o.user_id = $2
	`

	type result struct {
		ID	int64	`db:"id"`
		Target	float64	`db:"target"`
	}

	var res result
	err := s.db.GetContext(ctx, &res, checkQuery, taskID, userID)
	if err != nil {
		return false, fmt.Errorf("задача не найдена или не принадлежит пользователю: %v", err)
	}

	currentProgressQuery := `
		SELECT progress FROM tasks WHERE id = $1
	`
	var currentProgress float64
	err = s.db.GetContext(ctx, &currentProgress, currentProgressQuery, taskID)
	if err != nil {
		return false, fmt.Errorf("ошибка при получении текущего прогресса: %v", err)
	}

	newProgress := currentProgress + progress

	exceeded := false
	if newProgress > res.Target {
		exceeded = true
	}

	updateQuery := `
		UPDATE tasks
		SET progress = $1
		WHERE id = $2
	`

	_, err = s.db.ExecContext(ctx, updateQuery, newProgress, taskID)
	if err != nil {
		return false, fmt.Errorf("ошибка при обновлении прогресса: %v", err)
	}

	return exceeded, nil
}

func (s *Service) GetObjectiveProgress(ctx context.Context, objectiveID string) (float64, error) {
	keyResults, err := s.GetKeyResults(ctx, objectiveID)
	if err != nil {
		return 0, err
	}

	if len(keyResults) == 0 {
		return 0, nil
	}

	var totalProgress float64
	for _, kr := range keyResults {
		progressPercent := (kr.Progress / kr.Target) * 100
		if progressPercent > 100 {
			progressPercent = 100
		}
		totalProgress += progressPercent
	}

	return totalProgress / float64(len(keyResults)), nil
}

type ObjectiveDetails struct {
	Objective	Objective
	Progress	float64
	KeyResults	[]KeyResultDetails
}

type KeyResultDetails struct {
	KeyResult	KeyResult
	Progress	float64
	Tasks		[]Task
}

func (s *Service) GetObjectiveDetails(ctx context.Context, userID int64, objectiveID string) (*ObjectiveDetails, error) {

	objectiveQuery := `
		SELECT id, user_id, title, sphere, period, deadline, created_at
		FROM objectives
		WHERE id = $1 AND user_id = $2
	`
	var objective Objective
	err := s.db.GetContext(ctx, &objective, objectiveQuery, objectiveID, userID)
	if err != nil {
		return nil, fmt.Errorf("цель не найдена или не принадлежит пользователю: %v", err)
	}

	keyResults, err := s.GetKeyResults(ctx, objectiveID)
	if err != nil {
		return nil, err
	}

	objectiveProgress, err := s.GetObjectiveProgress(ctx, objectiveID)
	if err != nil {
		return nil, err
	}

	result := &ObjectiveDetails{
		Objective:	objective,
		Progress:	objectiveProgress,
		KeyResults:	make([]KeyResultDetails, 0, len(keyResults)),
	}

	for _, kr := range keyResults {
		tasks, err := s.GetTasks(ctx, kr.ID)
		if err != nil {
			return nil, err
		}

		krProgress := 0.0
		if kr.Target > 0 {
			krProgress = (kr.Progress / kr.Target) * 100
			if krProgress > 100 {
				krProgress = 100
			}
		}

		result.KeyResults = append(result.KeyResults, KeyResultDetails{
			KeyResult:	kr,
			Progress:	krProgress,
			Tasks:		tasks,
		})
	}

	return result, nil
}

func (s *Service) DeleteObjective(ctx context.Context, userID int64, objectiveID string) error {

	checkQuery := `
		SELECT id FROM objectives WHERE id = $1 AND user_id = $2
	`
	var id string
	err := s.db.GetContext(ctx, &id, checkQuery, objectiveID, userID)
	if err != nil {
		return fmt.Errorf("цель не найдена или не принадлежит пользователю: %v", err)
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка при начале транзакции: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	deleteTasks := `
		DELETE FROM tasks
		WHERE key_result_id IN (
			SELECT id FROM key_results WHERE objective_id = $1
		)
	`
	_, err = tx.ExecContext(ctx, deleteTasks, objectiveID)
	if err != nil {
		return fmt.Errorf("ошибка при удалении задач: %v", err)
	}

	deleteKeyResults := `
		DELETE FROM key_results
		WHERE objective_id = $1
	`
	_, err = tx.ExecContext(ctx, deleteKeyResults, objectiveID)
	if err != nil {
		return fmt.Errorf("ошибка при удалении ключевых результатов: %v", err)
	}

	deleteObjective := `
		DELETE FROM objectives
		WHERE id = $1
	`
	_, err = tx.ExecContext(ctx, deleteObjective, objectiveID)
	if err != nil {
		return fmt.Errorf("ошибка при удалении цели: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка при подтверждении транзакции: %v", err)
	}

	return nil
}

func (s *Service) DeleteKeyResult(ctx context.Context, userID int64, keyResultID int64) error {

	checkQuery := `
		SELECT kr.id
		FROM key_results kr
		JOIN objectives o ON kr.objective_id = o.id
		WHERE kr.id = $1 AND o.user_id = $2
	`
	var id int64
	err := s.db.GetContext(ctx, &id, checkQuery, keyResultID, userID)
	if err != nil {
		return fmt.Errorf("ключевой результат не найден или не принадлежит пользователю: %v", err)
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка при начале транзакции: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	deleteTasks := `
		DELETE FROM tasks
		WHERE key_result_id = $1
	`
	_, err = tx.ExecContext(ctx, deleteTasks, keyResultID)
	if err != nil {
		return fmt.Errorf("ошибка при удалении задач: %v", err)
	}

	deleteKeyResult := `
		DELETE FROM key_results
		WHERE id = $1
	`
	_, err = tx.ExecContext(ctx, deleteKeyResult, keyResultID)
	if err != nil {
		return fmt.Errorf("ошибка при удалении ключевого результата: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка при подтверждении транзакции: %v", err)
	}

	return nil
}

func (s *Service) DeleteTask(ctx context.Context, userID int64, taskID int64) error {

	checkQuery := `
		SELECT t.id
		FROM tasks t
		JOIN key_results kr ON t.key_result_id = kr.id
		JOIN objectives o ON kr.objective_id = o.id
		WHERE t.id = $1 AND o.user_id = $2
	`
	var id int64
	err := s.db.GetContext(ctx, &id, checkQuery, taskID, userID)
	if err != nil {
		return fmt.Errorf("задача не найдена или не принадлежит пользователю: %v", err)
	}

	deleteTask := `
		DELETE FROM tasks
		WHERE id = $1
	`
	_, err = s.db.ExecContext(ctx, deleteTask, taskID)
	if err != nil {
		return fmt.Errorf("ошибка при удалении задачи: %v", err)
	}

	return nil
}

func (s *Service) FindObjectiveByDescription(ctx context.Context, userID int64, description string) ([]Objective, error) {

	searchPattern := "%" + strings.ToLower(description) + "%"

	query := `
		SELECT id, user_id, title, sphere, period, deadline, created_at
		FROM objectives
		WHERE user_id = $1 AND LOWER(title) LIKE $2
		ORDER BY created_at DESC
	`

	var objectives []Objective
	err := s.db.SelectContext(ctx, &objectives, query, userID, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("ошибка при поиске целей: %v", err)
	}

	return objectives, nil
}

func (s *Service) FindKeyResultByDescription(ctx context.Context, userID int64, keyResultDescription string, objectiveDescription string) ([]KeyResult, error) {
	var keyResults []KeyResult
	var query string
	var args []interface{}

	searchPattern := "%" + strings.ToLower(keyResultDescription) + "%"

	if objectiveDescription != "" {

		objSearchPattern := "%" + strings.ToLower(objectiveDescription) + "%"
		query = `
			SELECT kr.id, kr.objective_id, kr.title, kr.target, kr.unit, kr.progress, kr.deadline, kr.created_at
			FROM key_results kr
			JOIN objectives o ON kr.objective_id = o.id
			WHERE o.user_id = $1 AND LOWER(kr.title) LIKE $2 AND LOWER(o.title) LIKE $3
			ORDER BY kr.created_at DESC
		`
		args = []interface{}{userID, searchPattern, objSearchPattern}
	} else {

		query = `
			SELECT kr.id, kr.objective_id, kr.title, kr.target, kr.unit, kr.progress, kr.deadline, kr.created_at
			FROM key_results kr
			JOIN objectives o ON kr.objective_id = o.id
			WHERE o.user_id = $1 AND LOWER(kr.title) LIKE $2
			ORDER BY kr.created_at DESC
		`
		args = []interface{}{userID, searchPattern}
	}

	err := s.db.SelectContext(ctx, &keyResults, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка при поиске ключевых результатов: %v", err)
	}

	return keyResults, nil
}

func (s *Service) FindTaskByDescription(ctx context.Context, userID int64, taskDescription string, keyResultDescription string) ([]Task, error) {
	var tasks []Task
	var query string
	var args []interface{}

	searchPattern := "%" + strings.ToLower(taskDescription) + "%"

	if keyResultDescription != "" {

		krSearchPattern := "%" + strings.ToLower(keyResultDescription) + "%"
		query = `
			SELECT t.id, t.key_result_id, t.title, t.target, t.unit, t.progress, t.deadline, t.created_at
			FROM tasks t
			JOIN key_results kr ON t.key_result_id = kr.id
			JOIN objectives o ON kr.objective_id = o.id
			WHERE o.user_id = $1 AND LOWER(t.title) LIKE $2 AND LOWER(kr.title) LIKE $3
			ORDER BY t.created_at DESC
		`
		args = []interface{}{userID, searchPattern, krSearchPattern}
	} else {

		query = `
			SELECT t.id, t.key_result_id, t.title, t.target, t.unit, t.progress, t.deadline, t.created_at
			FROM tasks t
			JOIN key_results kr ON t.key_result_id = kr.id
			JOIN objectives o ON kr.objective_id = o.id
			WHERE o.user_id = $1 AND LOWER(t.title) LIKE $2
			ORDER BY t.created_at DESC
		`
		args = []interface{}{userID, searchPattern}
	}

	err := s.db.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка при поиске задач: %v", err)
	}

	return tasks, nil
}

func (s *Service) CreateRecurringTasks(ctx context.Context, userID int64, keyResultID int64,
	taskTitle string, dailyTarget float64, unit string,
	startDate time.Time, endDate time.Time) ([]int64, error) {

	checkQuery := `
        SELECT kr.id, kr.objective_id
        FROM key_results kr
        JOIN objectives o ON kr.objective_id = o.id
        WHERE kr.id = $1 AND o.user_id = $2
    `
	var krResult struct {
		ID		int64	`db:"id"`
		ObjectiveID	string	`db:"objective_id"`
	}
	err := s.db.GetContext(ctx, &krResult, checkQuery, keyResultID, userID)
	if err != nil {
		return nil, fmt.Errorf("ключевой результат не найден или не принадлежит пользователю: %v", err)
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка при начале транзакции: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	taskIDs := []int64{}
	current := startDate

	for !current.After(endDate) {

		dateStr := current.Format("02.01.2006")
		fullTitle := fmt.Sprintf("%s (%s)", taskTitle, dateStr)

		deadline := time.Date(
			current.Year(), current.Month(), current.Day(),
			23, 59, 59, 0, current.Location(),
		)

		query := `
            INSERT INTO tasks (key_result_id, title, target, unit, progress, deadline, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            RETURNING id
        `

		var taskID int64
		err = tx.GetContext(
			ctx,
			&taskID,
			query,
			keyResultID,
			fullTitle,
			dailyTarget,
			unit,
			0.0,
			deadline,
			time.Now(),
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка при создании задачи: %v", err)
		}

		taskIDs = append(taskIDs, taskID)

		current = current.AddDate(0, 0, 1)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("ошибка при подтверждении транзакции: %v", err)
	}

	return taskIDs, nil
}

func (s *Service) CreateObjectiveWithRecurringTasks(ctx context.Context, userID int64,
	objectiveTitle, sphere, period string, objectiveDeadline *time.Time,
	keyResultTitle string, keyResultTarget float64, keyResultUnit string, keyResultDeadline *time.Time,
	taskTitle string, dailyTarget float64, taskUnit string,
	startDate, endDate time.Time) (string, int64, []int64, error) {

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", 0, nil, fmt.Errorf("ошибка при начале транзакции: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	objectiveID := uuid.New().String()

	objectiveQuery := `
		INSERT INTO objectives (id, user_id, title, sphere, period, deadline, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = tx.ExecContext(ctx, objectiveQuery, objectiveID, userID, objectiveTitle,
		sphere, period, objectiveDeadline, time.Now())
	if err != nil {
		return "", 0, nil, fmt.Errorf("ошибка при сохранении цели: %v", err)
	}

	krQuery := `
		INSERT INTO key_results (objective_id, title, target, unit, progress, deadline, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	var keyResultID int64
	err = tx.GetContext(
		ctx,
		&keyResultID,
		krQuery,
		objectiveID,
		keyResultTitle,
		keyResultTarget,
		keyResultUnit,
		0.0,
		keyResultDeadline,
		time.Now(),
	)
	if err != nil {
		return "", 0, nil, fmt.Errorf("ошибка при создании ключевого результата: %v", err)
	}

	taskIDs := []int64{}
	current := startDate

	for !current.After(endDate) {

		dateStr := current.Format("02.01.2006")
		fullTitle := fmt.Sprintf("%s (%s)", taskTitle, dateStr)

		taskDeadline := time.Date(
			current.Year(), current.Month(), current.Day(),
			23, 59, 59, 0, current.Location(),
		)

		taskQuery := `
			INSERT INTO tasks (key_result_id, title, target, unit, progress, deadline, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`

		var taskID int64
		err = tx.GetContext(
			ctx,
			&taskID,
			taskQuery,
			keyResultID,
			fullTitle,
			dailyTarget,
			taskUnit,
			0.0,
			taskDeadline,
			time.Now(),
		)
		if err != nil {
			return "", 0, nil, fmt.Errorf("ошибка при создании задачи: %v", err)
		}

		taskIDs = append(taskIDs, taskID)

		current = current.AddDate(0, 0, 1)
	}

	err = tx.Commit()
	if err != nil {
		return "", 0, nil, fmt.Errorf("ошибка при подтверждении транзакции: %v", err)
	}

	return objectiveID, keyResultID, taskIDs, nil
}

func (s *Service) GetKeyResultsForObjective(ctx context.Context, objectiveID string) ([]KeyResult, error) {
	query := `
		SELECT id, objective_id, title, target, unit, progress, deadline, created_at
		FROM key_results
		WHERE objective_id = $1
		ORDER BY created_at
	`

	var keyResults []KeyResult
	err := s.db.SelectContext(ctx, &keyResults, query, objectiveID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении ключевых результатов для цели %s: %v", objectiveID, err)
	}

	return keyResults, nil
}

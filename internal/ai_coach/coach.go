package ai_coach

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type AICoachService struct {
	db			*sqlx.DB
	personalityEngine	*PersonalityService
	contextEngine		*ContextService
	motivationEngine	*MotivationService
	predictionEngine	*PredictionService
	learningEngine		*LearningService
}

type AIInsight struct {
	ID			int64			`db:"id" json:"id"`
	UserID			int64			`db:"user_id" json:"user_id"`
	InsightType		string			`db:"insight_type" json:"insight_type"`
	Category		string			`db:"category" json:"category"`
	Title			string			`db:"title" json:"title"`
	Content			string			`db:"content" json:"content"`
	ActionButtonText	string			`db:"action_button_text" json:"action_button_text,omitempty"`
	ActionData		map[string]interface{}	`db:"action_data" json:"action_data,omitempty"`
	Priority		int			`db:"priority" json:"priority"`
	ObjectiveID		*string			`db:"objective_id" json:"objective_id,omitempty"`
	KeyResultID		*int64			`db:"key_result_id" json:"key_result_id,omitempty"`
	TaskID			*int64			`db:"task_id" json:"task_id,omitempty"`
	CreatedAt		time.Time		`db:"created_at" json:"created_at"`
	EffectivenessScore	float64			`db:"effectiveness_score" json:"effectiveness_score"`
}

type GoalSuggestion struct {
	Title			string			`json:"title"`
	Category		string			`json:"category"`
	Description		string			`json:"description"`
	EstimatedDays		int			`json:"estimated_days"`
	DifficultyLevel		int			`json:"difficulty_level"`
	KeyResults		[]KeyResultSuggestion	`json:"key_results"`
	Reasoning		string			`json:"reasoning"`
	MotivationStrategy	string			`json:"motivation_strategy"`
	Priority		int			`json:"priority"`
	Tags			[]string		`json:"tags"`
}

type KeyResultSuggestion struct {
	Title		string	`json:"title"`
	Target		float64	`json:"target"`
	Unit		string	`json:"unit"`
	DifficultyLevel	int	`json:"difficulty_level"`
	EstimatedHours	float64	`json:"estimated_hours"`
}

type RiskAlert struct {
	Type		string		`json:"type"`
	Severity	int		`json:"severity"`
	Title		string		`json:"title"`
	Description	string		`json:"description"`
	Suggestions	[]string	`json:"suggestions"`
	ObjectiveID	*string		`json:"objective_id,omitempty"`
	KeyResultID	*int64		`json:"key_result_id,omitempty"`
	TaskID		*int64		`json:"task_id,omitempty"`
	Probability	float64		`json:"probability"`
	ImpactScore	float64		`json:"impact_score"`
}

type ProductivityMetrics struct {
	CompletionRate		float64			`json:"completion_rate"`
	AverageTaskTime		float64			`json:"average_task_time"`
	PeakProductivityHours	[]int			`json:"peak_productivity_hours"`
	WeeklyProductivity	map[string]float64	`json:"weekly_productivity"`
	CategoryPerformance	map[string]float64	`json:"category_performance"`
	StreakDays		int			`json:"streak_days"`
	TotalPointsEarned	int			`json:"total_points_earned"`
	Level			int			`json:"level"`
	RecentAchievements	[]Achievement		`json:"recent_achievements"`
	PersonalityInsights	map[string]interface{}	`json:"personality_insights"`
	MotivationTrends	[]MotivationPoint	`json:"motivation_trends"`
	PredictedOutcomes	[]PredictionResult	`json:"predicted_outcomes"`
	ImprovementSuggestions	[]string		`json:"improvement_suggestions"`
}

type Achievement struct {
	ID		int		`json:"id"`
	Name		string		`json:"name"`
	Description	string		`json:"description"`
	Icon		string		`json:"icon"`
	Rarity		string		`json:"rarity"`
	Points		int		`json:"points"`
	EarnedAt	time.Time	`json:"earned_at"`
}

type MotivationPoint struct {
	Date	time.Time	`json:"date"`
	Score	float64		`json:"score"`
	Mood	int		`json:"mood"`
}

type PredictionResult struct {
	Type		string		`json:"type"`
	Confidence	float64		`json:"confidence"`
	PredictedValue	float64		`json:"predicted_value,omitempty"`
	PredictedDate	time.Time	`json:"predicted_date,omitempty"`
	Description	string		`json:"description"`
}

type UserPersonality struct {
	PersonalityType		string	`db:"personality_type" json:"personality_type"`
	MotivationStyle		string	`db:"motivation_style" json:"motivation_style"`
	CommunicationStyle	string	`db:"communication_style" json:"communication_style"`
	ActivityLevel		string	`db:"activity_level" json:"activity_level"`
	PreferredReminderTime	string	`db:"preferred_reminder_time" json:"preferred_reminder_time"`
}

func NewAICoachService(db *sqlx.DB) *AICoachService {
	return &AICoachService{
		db:			db,
		personalityEngine:	NewPersonalityService(db),
		contextEngine:		NewContextService(db),
		motivationEngine:	NewMotivationService(db),
		predictionEngine:	NewPredictionService(db),
		learningEngine:		NewLearningService(db),
	}
}

func (s *AICoachService) GenerateInsights(ctx context.Context, userID int64) ([]AIInsight, error) {
	logrus.Infof("Генерация AI инсайтов для пользователя: %d", userID)

	var insights []AIInsight

	personality, err := s.personalityEngine.GetUserPersonality(ctx, userID)
	if err != nil {
		logrus.Errorf("Ошибка получения личности пользователя: %v", err)
		return nil, err
	}

	currentContext, err := s.contextEngine.GetCurrentContext(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить контекст пользователя: %v", err)
	}

	productivityInsights, err := s.generateProductivityInsights(ctx, userID, personality)
	if err != nil {
		logrus.Errorf("Ошибка генерации инсайтов продуктивности: %v", err)
	} else {
		insights = append(insights, productivityInsights...)
	}

	motivationInsights, err := s.generateMotivationInsights(ctx, userID, personality, currentContext)
	if err != nil {
		logrus.Errorf("Ошибка генерации мотивационных инсайтов: %v", err)
	} else {
		insights = append(insights, motivationInsights...)
	}

	riskInsights, err := s.generateRiskInsights(ctx, userID)
	if err != nil {
		logrus.Errorf("Ошибка генерации инсайтов рисков: %v", err)
	} else {
		insights = append(insights, riskInsights...)
	}

	celebrationInsights, err := s.generateCelebrationInsights(ctx, userID, personality)
	if err != nil {
		logrus.Errorf("Ошибка генерации праздничных инсайтов: %v", err)
	} else {
		insights = append(insights, celebrationInsights...)
	}

	tipInsights, err := s.generatePersonalizedTips(ctx, userID, personality, currentContext)
	if err != nil {
		logrus.Errorf("Ошибка генерации персональных советов: %v", err)
	} else {
		insights = append(insights, tipInsights...)
	}

	for i := range insights {
		err := s.saveInsight(ctx, &insights[i])
		if err != nil {
			logrus.Errorf("Ошибка сохранения инсайта: %v", err)
		}
	}

	sort.Slice(insights, func(i, j int) bool {
		return insights[i].Priority > insights[j].Priority
	})

	logrus.Infof("Сгенерировано %d инсайтов для пользователя %d", len(insights), userID)
	return insights, nil
}

func (s *AICoachService) SuggestGoals(ctx context.Context, userID int64) ([]GoalSuggestion, error) {
	logrus.Infof("Генерация предложений целей для пользователя: %d", userID)

	personality, err := s.personalityEngine.GetUserPersonality(ctx, userID)
	if err != nil {
		return nil, err
	}

	behaviorPatterns, err := s.learningEngine.GetUserBehaviorPatterns(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить паттерны поведения: %v", err)
	}

	behaviorData := s.convertPatternsToAnalysisFormat(behaviorPatterns)

	currentGoals, err := s.getCurrentUserGoals(ctx, userID)
	if err != nil {
		return nil, err
	}

	var suggestions []GoalSuggestion

	missingCategories := s.findMissingCategories(currentGoals)
	for _, category := range missingCategories {
		suggestion := s.createCategorySuggestion(category, personality, behaviorData)
		suggestions = append(suggestions, suggestion)
	}

	improvementSuggestions := s.suggestGoalImprovements(currentGoals, personality, behaviorData)
	suggestions = append(suggestions, improvementSuggestions...)

	seasonalSuggestions := s.generateSeasonalSuggestions(personality, currentGoals)
	suggestions = append(suggestions, seasonalSuggestions...)

	achievementBasedSuggestions, err := s.generateAchievementBasedSuggestions(ctx, userID, personality)
	if err != nil {
		logrus.Warnf("Ошибка генерации предложений на основе достижений: %v", err)
	} else {
		suggestions = append(suggestions, achievementBasedSuggestions...)
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Priority > suggestions[j].Priority
	})

	maxSuggestions := 5
	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}

	logrus.Infof("Сгенерировано %d предложений целей для пользователя %d", len(suggestions), userID)
	return suggestions, nil
}

func (s *AICoachService) AnalyzeRisks(ctx context.Context, userID int64) ([]RiskAlert, error) {
	logrus.Infof("Анализ рисков для пользователя: %d", userID)

	var risks []RiskAlert

	activeGoals, err := s.getActiveUserGoals(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, goal := range activeGoals {

		deadlineRisks := s.analyzeDeadlineRisks(goal)
		risks = append(risks, deadlineRisks...)

		progressRisks, err := s.analyzeProgressRisks(ctx, goal)
		if err != nil {
			logrus.Warnf("Ошибка анализа рисков прогресса: %v", err)
		} else {
			risks = append(risks, progressRisks...)
		}

		difficultyRisks := s.analyzeDifficultyRisks(goal)
		risks = append(risks, difficultyRisks...)

		motivationRisks, err := s.analyzeMotivationRisks(ctx, userID, goal)
		if err != nil {
			logrus.Warnf("Ошибка анализа мотивационных рисков: %v", err)
		} else {
			risks = append(risks, motivationRisks...)
		}
	}

	predictions, err := s.predictionEngine.PredictGoalOutcomes(ctx, userID, activeGoals)
	if err != nil {
		logrus.Warnf("Ошибка получения предсказаний: %v", err)
	} else {

		predictionRisks := s.convertPredictionsToRisks(predictions)
		risks = append(risks, predictionRisks...)
	}

	sort.Slice(risks, func(i, j int) bool {
		if risks[i].Severity != risks[j].Severity {
			return risks[i].Severity > risks[j].Severity
		}
		return risks[i].ImpactScore > risks[j].ImpactScore
	})

	logrus.Infof("Найдено %d рисков для пользователя %d", len(risks), userID)
	return risks, nil
}

func (s *AICoachService) GenerateMotivation(ctx context.Context, userID int64) (string, error) {
	personality, err := s.personalityEngine.GetUserPersonality(ctx, userID)
	if err != nil {
		return "", err
	}

	currentContext, err := s.contextEngine.GetCurrentContext(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить контекст: %v", err)
	}

	productivity, err := s.AnalyzeProductivity(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить данные продуктивности: %v", err)
	}

	motivation := s.motivationEngine.GeneratePersonalizedMotivation(personality, currentContext, productivity)

	err = s.motivationEngine.RecordMotivationUsage(ctx, userID, motivation)
	if err != nil {
		logrus.Warnf("Не удалось записать использование мотивации: %v", err)
	}

	return motivation, nil
}

func (s *AICoachService) AnalyzeProductivity(ctx context.Context, userID int64) (*ProductivityMetrics, error) {
	logrus.Infof("Анализ продуктивности для пользователя: %d", userID)

	metrics := &ProductivityMetrics{}

	completionStats, err := s.getCompletionStatistics(ctx, userID)
	if err != nil {
		return nil, err
	}
	metrics.CompletionRate = completionStats.Rate
	metrics.AverageTaskTime = completionStats.AverageTime

	peakHours, err := s.analyzePeakProductivityHours(ctx, userID)
	if err != nil {
		logrus.Warnf("Ошибка анализа пиковых часов: %v", err)
	} else {
		metrics.PeakProductivityHours = peakHours
	}

	weeklyStats, err := s.getWeeklyProductivity(ctx, userID)
	if err != nil {
		logrus.Warnf("Ошибка получения недельной статистики: %v", err)
	} else {
		metrics.WeeklyProductivity = weeklyStats
	}

	categoryPerformance, err := s.getCategoryPerformance(ctx, userID)
	if err != nil {
		logrus.Warnf("Ошибка анализа категорий: %v", err)
	} else {
		metrics.CategoryPerformance = categoryPerformance
	}

	streakData, err := s.getStreakData(ctx, userID)
	if err != nil {
		logrus.Warnf("Ошибка получения данных серий: %v", err)
	} else {
		metrics.StreakDays = streakData.CurrentStreak
		metrics.TotalPointsEarned = streakData.TotalPoints
		metrics.Level = streakData.Level
	}

	achievements, err := s.getRecentAchievements(ctx, userID)
	if err != nil {
		logrus.Warnf("Ошибка получения достижений: %v", err)
	} else {
		metrics.RecentAchievements = achievements
	}

	personalityInsights, err := s.personalityEngine.AnalyzePersonalityTrends(ctx, userID)
	if err != nil {
		logrus.Warnf("Ошибка анализа личности: %v", err)
	} else {
		metrics.PersonalityInsights = personalityInsights
	}

	motivationTrends, err := s.getMotivationTrends(ctx, userID)
	if err != nil {
		logrus.Warnf("Ошибка получения трендов мотивации: %v", err)
	} else {
		metrics.MotivationTrends = motivationTrends
	}

	predictions, err := s.predictionEngine.GenerateProductivityPredictions(ctx, userID)
	if err != nil {
		logrus.Warnf("Ошибка генерации предсказаний: %v", err)
	} else {
		metrics.PredictedOutcomes = predictions
	}

	improvements := s.generateImprovementSuggestions(metrics)
	metrics.ImprovementSuggestions = improvements

	logrus.Infof("Анализ продуктивности завершен для пользователя %d", userID)
	return metrics, nil
}

func (s *AICoachService) GenerateWeeklyPlan(ctx context.Context, userID int64) (map[string]interface{}, error) {
	personality, err := s.personalityEngine.GetUserPersonality(ctx, userID)
	if err != nil {
		return nil, err
	}

	activeGoals, err := s.getActiveUserGoals(ctx, userID)
	if err != nil {
		return nil, err
	}

	availableTime, err := s.analyzeAvailableTime(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось проанализировать доступное время: %v", err)
		availableTime = 2.0
	}

	weeklyPlan := s.createOptimalWeeklyPlan(activeGoals, personality, availableTime)

	return weeklyPlan, nil
}

func (s *AICoachService) saveInsight(ctx context.Context, insight *AIInsight) error {
	actionDataJSON, _ := json.Marshal(insight.ActionData)

	query := `
		INSERT INTO ai_insights 
		(user_id, insight_type, category, title, content, action_button_text, action_data, 
		 priority, objective_id, key_result_id, task_id, effectiveness_score)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`

	err := s.db.GetContext(ctx, insight, query,
		insight.UserID, insight.InsightType, insight.Category, insight.Title,
		insight.Content, insight.ActionButtonText, string(actionDataJSON),
		insight.Priority, insight.ObjectiveID, insight.KeyResultID,
		insight.TaskID, insight.EffectivenessScore)

	return err
}

func (s *AICoachService) getRecentAchievements(ctx context.Context, userID int64) ([]Achievement, error) {
	var achievements []Achievement

	query := `
		SELECT ua.id, at.name, at.description, at.icon, at.rarity, at.points, ua.earned_at
		FROM user_achievements ua
		JOIN achievement_types at ON ua.achievement_type_id = at.id
		WHERE ua.user_id = $1 
		ORDER BY ua.earned_at DESC 
		LIMIT 10
	`

	err := s.db.SelectContext(ctx, &achievements, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent achievements: %w", err)
	}

	return achievements, nil
}

func (s *AICoachService) convertPatternsToAnalysisFormat(patterns []UserBehaviorPattern) map[string]interface{} {
	if len(patterns) == 0 {
		return make(map[string]interface{})
	}

	result := make(map[string]interface{})

	for _, pattern := range patterns {
		result[pattern.PatternType] = map[string]interface{}{
			"frequency":	pattern.Frequency,
			"confidence":	pattern.Confidence,
			"data":		pattern.PatternData,
			"updated_at":	pattern.UpdatedAt,
		}
	}

	return result
}

func (s *AICoachService) GetUserPersonality(ctx context.Context, userID int64) (*PersonalityProfile, error) {
	return s.personalityEngine.GetUserPersonality(ctx, userID)
}

func (s *AICoachService) GetCurrentContext(ctx context.Context, userID int64) (map[string]interface{}, error) {
	return s.contextEngine.GetCurrentContext(ctx, userID)
}

func (s *AICoachService) UpdateConversationContext(ctx context.Context, userID int64, message string, intent string) error {
	return s.contextEngine.UpdateConversationContext(ctx, userID, message, intent)
}

func (s *AICoachService) LearnFromInteraction(ctx context.Context, userID int64, interaction map[string]interface{}) error {
	return s.learningEngine.LearnFromInteraction(ctx, userID, interaction)
}

func (s *AICoachService) GeneratePersonalizedMessage(profile *PersonalityProfile, messageType string, context map[string]interface{}) string {
	return s.personalityEngine.GeneratePersonalizedMessage(profile, messageType, context)
}

func (s *AICoachService) PredictGoalOutcomes(ctx context.Context, userID int64, goals []interface{}) ([]PredictionResult, error) {
	return s.predictionEngine.PredictGoalOutcomes(ctx, userID, goals)
}

func (s *AICoachService) PredictCompletionProbability(ctx context.Context, userID int64, objectiveID string) (*CompletionPrediction, error) {
	return s.predictionEngine.PredictCompletionProbability(ctx, userID, objectiveID)
}

func (s *AICoachService) UpdateMoodContext(ctx context.Context, userID int64, moodScore, sleepQuality int) error {
	return s.contextEngine.UpdateMoodContext(ctx, userID, moodScore, sleepQuality)
}

func (s *AICoachService) LearnFromBehavior(ctx context.Context, userID int64, behaviorData map[string]interface{}) error {
	return s.learningEngine.LearnFromInteraction(ctx, userID, behaviorData)
}

func (s *AICoachService) GetActiveUserGoals(ctx context.Context, userID int64) ([]interface{}, error) {

	query := `
		SELECT id, title, description, priority, status, created_at, updated_at 
		FROM objectives 
		WHERE user_id = $1 AND status IN ('active', 'in_progress')
		ORDER BY priority DESC, created_at DESC
	`

	var goals []interface{}
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		goal := make(map[string]interface{})
		var id, title, description, priority, status string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &title, &description, &priority, &status, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		goal["id"] = id
		goal["title"] = title
		goal["description"] = description
		goal["priority"] = priority
		goal["status"] = status
		goal["created_at"] = createdAt
		goal["updated_at"] = updatedAt

		goals = append(goals, goal)
	}

	return goals, nil
}

func (s *AICoachService) GenerateMotivationPlan(ctx context.Context, userID int64, goals []interface{}) (map[string]interface{}, error) {
	return s.motivationEngine.GenerateMotivationPlan(ctx, userID, goals)
}

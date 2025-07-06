package ai_coach

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *AICoachService) generateProductivityInsights(ctx context.Context, userID int64, personality *PersonalityProfile) ([]AIInsight, error) {
	var insights []AIInsight

	peakHours, err := s.analyzePeakProductivityHours(ctx, userID)
	if err == nil && len(peakHours) > 0 {
		insight := AIInsight{
			UserID:			userID,
			InsightType:		"productivity",
			Category:		"optimization",
			Title:			"Твое время пиковой продуктивности",
			Content:		fmt.Sprintf("Ты наиболее продуктивен в %s. Планируй важные задачи на это время!", s.formatHours(peakHours)),
			Priority:		4,
			EffectivenessScore:	0.85,
		}
		insights = append(insights, insight)
	}

	completionRate, err := s.getRecentCompletionRate(ctx, userID)
	if err == nil {
		if completionRate > 0.8 {
			insight := AIInsight{
				UserID:			userID,
				InsightType:		"celebration",
				Category:		"achievement",
				Title:			"Отличная работа!",
				Content:		fmt.Sprintf("Твой уровень завершения задач составляет %.0f%% - это превосходный результат! 🎉", completionRate*100),
				Priority:		5,
				EffectivenessScore:	0.9,
			}
			insights = append(insights, insight)
		} else if completionRate < 0.3 {
			insight := AIInsight{
				UserID:			userID,
				InsightType:		"suggestion",
				Category:		"improvement",
				Title:			"Давай улучшим результат",
				Content:		"Замечаю, что многие задачи остаются незавершенными. Попробуй разбить их на более мелкие части.",
				Priority:		4,
				ActionButtonText:	"Помочь с планированием",
				EffectivenessScore:	0.7,
			}
			insights = append(insights, insight)
		}
	}

	return insights, nil
}

func (s *AICoachService) generateMotivationInsights(ctx context.Context, userID int64, personality *PersonalityProfile, context map[string]interface{}) ([]AIInsight, error) {
	var insights []AIInsight

	if moodCtx, ok := context["mood"]; ok {
		if moodMap, ok := moodCtx.(map[string]interface{}); ok {
			if motivationLevel, ok := moodMap["motivation_level"].(float64); ok {
				if motivationLevel < 0.4 {
					message := s.personalityEngine.GeneratePersonalizedMessage(personality, "motivation", context)
					insight := AIInsight{
						UserID:			userID,
						InsightType:		"motivation",
						Category:		"support",
						Title:			"Время для мотивации!",
						Content:		message,
						Priority:		5,
						ActionButtonText:	"Получить больше мотивации",
						EffectivenessScore:	0.8,
					}
					insights = append(insights, insight)
				}
			}
		}
	}

	streakDays, err := s.getCurrentStreak(ctx, userID)
	if err == nil {
		if streakDays >= 7 {
			insight := AIInsight{
				UserID:			userID,
				InsightType:		"celebration",
				Category:		"achievement",
				Title:			"Невероятная серия!",
				Content:		fmt.Sprintf("Ты поддерживаешь серию уже %d дней! Так держать! 🔥", streakDays),
				Priority:		4,
				EffectivenessScore:	0.9,
			}
			insights = append(insights, insight)
		} else if streakDays == 0 {
			insight := AIInsight{
				UserID:			userID,
				InsightType:		"motivation",
				Category:		"encouragement",
				Title:			"Время начать новую серию!",
				Content:		"Каждый день - это новая возможность. Давай начнем с малого и создадим новую серию успехов! 💪",
				Priority:		3,
				ActionButtonText:	"Выбрать простую задачу",
				EffectivenessScore:	0.75,
			}
			insights = append(insights, insight)
		}
	}

	return insights, nil
}

func (s *AICoachService) generateRiskInsights(ctx context.Context, userID int64) ([]AIInsight, error) {
	var insights []AIInsight

	urgentDeadlines, err := s.getUrgentDeadlines(ctx, userID)
	if err == nil && len(urgentDeadlines) > 0 {
		for _, deadline := range urgentDeadlines {
			insight := AIInsight{
				UserID:			userID,
				InsightType:		"warning",
				Category:		"deadline",
				Title:			"Дедлайн приближается!",
				Content:		fmt.Sprintf("До завершения '%s' осталось %d дней. Время сосредоточиться! ⏰", deadline.Title, deadline.DaysLeft),
				Priority:		5,
				ObjectiveID:		&deadline.ID,
				ActionButtonText:	"Показать план действий",
				EffectivenessScore:	0.85,
			}
			insights = append(insights, insight)
		}
	}

	lowProgressGoals, err := s.getLowProgressGoals(ctx, userID)
	if err == nil && len(lowProgressGoals) > 0 {
		for _, goal := range lowProgressGoals {
			insight := AIInsight{
				UserID:			userID,
				InsightType:		"suggestion",
				Category:		"progress",
				Title:			"Цель нуждается в внимании",
				Content:		fmt.Sprintf("'%s' показывает низкий прогресс (%.0f%%). Может, стоит пересмотреть подход?", goal.Title, goal.Progress),
				Priority:		3,
				ObjectiveID:		&goal.ID,
				ActionButtonText:	"Помочь с планированием",
				EffectivenessScore:	0.7,
			}
			insights = append(insights, insight)
		}
	}

	return insights, nil
}

func (s *AICoachService) generateCelebrationInsights(ctx context.Context, userID int64, personality *PersonalityProfile) ([]AIInsight, error) {
	var insights []AIInsight

	recentAchievements, err := s.getRecentAchievements(ctx, userID)
	if err == nil && len(recentAchievements) > 0 {
		for _, achievement := range recentAchievements {
			celebrationMsg := s.personalityEngine.GeneratePersonalizedMessage(personality, "celebration", map[string]interface{}{
				"achievement": achievement.Name,
			})

			insight := AIInsight{
				UserID:			userID,
				InsightType:		"celebration",
				Category:		"achievement",
				Title:			fmt.Sprintf("Поздравляю с достижением: %s!", achievement.Name),
				Content:		celebrationMsg,
				Priority:		4,
				ActionButtonText:	"Выбрать награду",
				EffectivenessScore:	0.9,
			}
			insights = append(insights, insight)
		}
	}

	recentCompletions, err := s.getRecentCompletedGoals(ctx, userID)
	if err == nil && len(recentCompletions) > 0 {
		for _, completion := range recentCompletions {
			insight := AIInsight{
				UserID:			userID,
				InsightType:		"celebration",
				Category:		"completion",
				Title:			"Цель достигнута! 🎉",
				Content:		fmt.Sprintf("Поздравляю с завершением '%s'! Это был отличный результат!", completion.Title),
				Priority:		5,
				ObjectiveID:		&completion.ID,
				ActionButtonText:	"Поделиться успехом",
				EffectivenessScore:	1.0,
			}
			insights = append(insights, insight)
		}
	}

	return insights, nil
}

func (s *AICoachService) generatePersonalizedTips(ctx context.Context, userID int64, personality *PersonalityProfile, context map[string]interface{}) ([]AIInsight, error) {
	var insights []AIInsight

	personalityTip := s.generatePersonalityBasedTip(personality)
	if personalityTip != "" {
		insight := AIInsight{
			UserID:			userID,
			InsightType:		"tip",
			Category:		"personal_growth",
			Title:			"Персональный совет",
			Content:		personalityTip,
			Priority:		2,
			EffectivenessScore:	0.7,
		}
		insights = append(insights, insight)
	}

	timeBasedTip := s.generateTimeBasedTip(context)
	if timeBasedTip != "" {
		insight := AIInsight{
			UserID:			userID,
			InsightType:		"tip",
			Category:		"productivity",
			Title:			"Совет по времени",
			Content:		timeBasedTip,
			Priority:		2,
			EffectivenessScore:	0.6,
		}
		insights = append(insights, insight)
	}

	return insights, nil
}

func (s *AICoachService) analyzePeakProductivityHours(ctx context.Context, userID int64) ([]int, error) {
	query := `
		SELECT EXTRACT(hour FROM created_at) as hour, COUNT(*) as count
		FROM habit_tracking
		WHERE user_id = $1 AND completed = true AND created_at > NOW() - INTERVAL '14 days'
		GROUP BY EXTRACT(hour FROM created_at)
		ORDER BY count DESC
		LIMIT 3
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hours []int
	for rows.Next() {
		var hour int
		var count int
		if err := rows.Scan(&hour, &count); err != nil {
			continue
		}
		hours = append(hours, hour)
	}

	return hours, nil
}

func (s *AICoachService) getRecentCompletionRate(ctx context.Context, userID int64) (float64, error) {
	query := `
		SELECT 
			COUNT(CASE WHEN completed = true THEN 1 END)::float / COUNT(*)::float as completion_rate
		FROM habit_tracking
		WHERE user_id = $1 AND date > CURRENT_DATE - INTERVAL '7 days'
	`

	var rate float64
	err := s.db.GetContext(ctx, &rate, query, userID)
	return rate, err
}

func (s *AICoachService) getCurrentStreak(ctx context.Context, userID int64) (int, error) {
	query := `
		SELECT streak_days
		FROM users
		WHERE id = $1
	`

	var streak int
	err := s.db.GetContext(ctx, &streak, query, userID)
	return streak, err
}

func (s *AICoachService) getUrgentDeadlines(ctx context.Context, userID int64) ([]DeadlineInfo, error) {
	query := `
		SELECT id, title, deadline,
			EXTRACT(DAYS FROM deadline - NOW())::int as days_left,
			COALESCE((SELECT AVG(progress/target*100) FROM key_results WHERE objective_id = o.id), 0) as progress
		FROM objectives o
		WHERE user_id = $1 AND deadline IS NOT NULL 
		AND deadline > NOW() AND deadline < NOW() + INTERVAL '3 days'
		AND status = 'active'
		ORDER BY deadline ASC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deadlines []DeadlineInfo
	for rows.Next() {
		var deadline DeadlineInfo
		err := rows.Scan(&deadline.ID, &deadline.Title, &deadline.Deadline, &deadline.DaysLeft, &deadline.Progress)
		if err != nil {
			continue
		}
		deadlines = append(deadlines, deadline)
	}

	return deadlines, nil
}

func (s *AICoachService) getLowProgressGoals(ctx context.Context, userID int64) ([]struct {
	ID, Title	string
	Progress	float64
}, error) {
	query := `
		SELECT o.id, o.title,
			COALESCE((SELECT AVG(progress/target*100) FROM key_results WHERE objective_id = o.id), 0) as progress
		FROM objectives o
		WHERE o.user_id = $1 AND o.status = 'active'
		AND o.created_at < NOW() - INTERVAL '7 days'
		HAVING COALESCE((SELECT AVG(progress/target*100) FROM key_results WHERE objective_id = o.id), 0) < 20
		ORDER BY progress ASC
		LIMIT 3
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []struct {
		ID, Title	string
		Progress	float64
	}
	for rows.Next() {
		var goal struct {
			ID, Title	string
			Progress	float64
		}
		err := rows.Scan(&goal.ID, &goal.Title, &goal.Progress)
		if err != nil {
			continue
		}
		goals = append(goals, goal)
	}

	return goals, nil
}

func (s *AICoachService) getRecentCompletedGoals(ctx context.Context, userID int64) ([]struct{ ID, Title string }, error) {
	query := `
		SELECT id, title
		FROM objectives
		WHERE user_id = $1 AND completion_date IS NOT NULL
		AND completion_date > NOW() - INTERVAL '24 hours'
		ORDER BY completion_date DESC
		LIMIT 3
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var completions []struct{ ID, Title string }
	for rows.Next() {
		var completion struct{ ID, Title string }
		err := rows.Scan(&completion.ID, &completion.Title)
		if err != nil {
			continue
		}
		completions = append(completions, completion)
	}

	return completions, nil
}

func (s *AICoachService) generatePersonalityBasedTip(personality *PersonalityProfile) string {
	switch personality.PersonalityType {
	case "analytical":
		return "Используй данные и метрики для отслеживания прогресса. Создай детальный план с измеримыми результатами."
	case "creative":
		return "Попробуй визуализировать свои цели. Создай mind map или используй цветные стикеры для планирования."
	case "social":
		return "Поделись своими цеплями с друзьями. Социальная поддержка поможет тебе оставаться мотивированным."
	case "pragmatic":
		return "Фокусируйся на конкретных, выполнимых шагах. Разбивай большие цели на маленькие задачи."
	default:
		return "Экспериментируй с разными подходами к достижению целей и найди то, что работает именно для тебя."
	}
}

func (s *AICoachService) generateTimeBasedTip(context map[string]interface{}) string {
	if timeCtx, ok := context["time"]; ok {
		if timeMap, ok := timeCtx.(map[string]interface{}); ok {
			if isMorning, ok := timeMap["is_morning"].(bool); ok && isMorning {
				return "Утро - отличное время для планирования дня. Определи 3 главные задачи на сегодня."
			}
			if isEvening, ok := timeMap["is_evening"].(bool); ok && isEvening {
				return "Вечер - время для рефлексии. Подведи итоги дня и запланируй завтра."
			}
			if isWeekend, ok := timeMap["is_weekend"].(bool); ok && isWeekend {
				return "Выходные - время для долгосрочного планирования и работы над личными проектами."
			}
		}
	}
	return ""
}

func (s *AICoachService) formatHours(hours []int) string {
	if len(hours) == 0 {
		return "утренние часы"
	}

	var formatted []string
	for _, hour := range hours {
		formatted = append(formatted, fmt.Sprintf("%d:00", hour))
	}

	if len(formatted) == 1 {
		return formatted[0]
	} else if len(formatted) == 2 {
		return formatted[0] + " и " + formatted[1]
	} else {
		return strings.Join(formatted[:len(formatted)-1], ", ") + " и " + formatted[len(formatted)-1]
	}
}

func (s *AICoachService) getCurrentUserGoals(ctx context.Context, userID int64) ([]interface{}, error) {

	return []interface{}{}, nil
}

func (s *AICoachService) getActiveUserGoals(ctx context.Context, userID int64) ([]interface{}, error) {

	return []interface{}{}, nil
}

func (s *AICoachService) findMissingCategories(currentGoals []interface{}) []string {

	return []string{"Здоровье и спорт", "Личностное развитие"}
}

func (s *AICoachService) createCategorySuggestion(category string, personality *PersonalityProfile, behaviorPatterns map[string]interface{}) GoalSuggestion {
	return GoalSuggestion{
		Title:			fmt.Sprintf("Цель в категории: %s", category),
		Category:		category,
		Description:		fmt.Sprintf("Рекомендуемая цель для развития в области %s", category),
		EstimatedDays:		30,
		DifficultyLevel:	3,
		Priority:		3,
		KeyResults: []KeyResultSuggestion{
			{
				Title:			"Первый шаг",
				Target:			1,
				Unit:			"выполнено",
				DifficultyLevel:	2,
				EstimatedHours:		5,
			},
		},
		Reasoning:		fmt.Sprintf("Эта категория поможет твоему развитию в %s", category),
		MotivationStrategy:	"achievement",
		Tags:			[]string{category, "рекомендация"},
	}
}

func (s *AICoachService) suggestGoalImprovements(currentGoals []interface{}, personality *PersonalityProfile, behaviorPatterns map[string]interface{}) []GoalSuggestion {

	return []GoalSuggestion{}
}

func (s *AICoachService) generateSeasonalSuggestions(personality *PersonalityProfile, currentGoals []interface{}) []GoalSuggestion {

	return []GoalSuggestion{}
}

func (s *AICoachService) generateAchievementBasedSuggestions(ctx context.Context, userID int64, personality *PersonalityProfile) ([]GoalSuggestion, error) {

	return []GoalSuggestion{}, nil
}

func (s *AICoachService) analyzeDeadlineRisks(goal interface{}) []RiskAlert {

	return []RiskAlert{}
}

func (s *AICoachService) analyzeProgressRisks(ctx context.Context, goal interface{}) ([]RiskAlert, error) {

	return []RiskAlert{}, nil
}

func (s *AICoachService) analyzeDifficultyRisks(goal interface{}) []RiskAlert {

	return []RiskAlert{}
}

func (s *AICoachService) analyzeMotivationRisks(ctx context.Context, userID int64, goal interface{}) ([]RiskAlert, error) {

	return []RiskAlert{}, nil
}

func (s *AICoachService) convertPredictionsToRisks(predictions []PredictionResult) []RiskAlert {

	return []RiskAlert{}
}

func (s *AICoachService) getCompletionStatistics(ctx context.Context, userID int64) (struct{ Rate, AverageTime float64 }, error) {
	return struct{ Rate, AverageTime float64 }{Rate: 0.7, AverageTime: 25.5}, nil
}

func (s *AICoachService) getWeeklyProductivity(ctx context.Context, userID int64) (map[string]float64, error) {
	return map[string]float64{
		"monday":	0.8, "tuesday": 0.7, "wednesday": 0.9,
		"thursday":	0.6, "friday": 0.5, "saturday": 0.3, "sunday": 0.4,
	}, nil
}

func (s *AICoachService) getCategoryPerformance(ctx context.Context, userID int64) (map[string]float64, error) {
	return map[string]float64{
		"Работа":	0.8, "Здоровье": 0.6, "Развитие": 0.7,
	}, nil
}

func (s *AICoachService) getStreakData(ctx context.Context, userID int64) (struct{ CurrentStreak, TotalPoints, Level int }, error) {
	return struct{ CurrentStreak, TotalPoints, Level int }{CurrentStreak: 5, TotalPoints: 1250, Level: 3}, nil
}

func (s *AICoachService) getMotivationTrends(ctx context.Context, userID int64) ([]MotivationPoint, error) {
	return []MotivationPoint{
		{Date: time.Now().AddDate(0, 0, -7), Score: 0.6, Mood: 3},
		{Date: time.Now().AddDate(0, 0, -3), Score: 0.8, Mood: 4},
		{Date: time.Now(), Score: 0.7, Mood: 4},
	}, nil
}

func (s *AICoachService) generateImprovementSuggestions(metrics *ProductivityMetrics) []string {
	var suggestions []string

	if metrics.CompletionRate < 0.5 {
		suggestions = append(suggestions, "Попробуй разбивать задачи на более мелкие части")
	}

	if metrics.StreakDays < 3 {
		suggestions = append(suggestions, "Постарайся поддерживать ежедневную активность")
	}

	if len(metrics.PeakProductivityHours) > 0 {
		suggestions = append(suggestions, "Планируй важные задачи на часы пиковой продуктивности")
	}

	return suggestions
}

func (s *AICoachService) analyzeAvailableTime(ctx context.Context, userID int64) (float64, error) {

	return 2.0, nil
}

func (s *AICoachService) createOptimalWeeklyPlan(activeGoals []interface{}, personality *PersonalityProfile, availableTime float64) map[string]interface{} {
	return map[string]interface{}{
		"monday":	map[string]interface{}{"focus": "Планирование недели", "time": 1.0},
		"tuesday":	map[string]interface{}{"focus": "Основные задачи", "time": 2.0},
		"wednesday":	map[string]interface{}{"focus": "Продолжение работы", "time": 2.0},
		"thursday":	map[string]interface{}{"focus": "Завершение задач", "time": 1.5},
		"friday":	map[string]interface{}{"focus": "Подведение итогов", "time": 1.0},
		"saturday":	map[string]interface{}{"focus": "Личные проекты", "time": 1.0},
		"sunday":	map[string]interface{}{"focus": "Отдых и планирование", "time": 0.5},
	}
}

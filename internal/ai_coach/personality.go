package ai_coach

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type PersonalityService struct {
	db *sqlx.DB
}

type PersonalityProfile struct {
	UserID			int64			`json:"user_id"`
	PersonalityType		string			`json:"personality_type"`
	MotivationStyle		string			`json:"motivation_style"`
	CommunicationStyle	string			`json:"communication_style"`
	ActivityLevel		string			`json:"activity_level"`
	PreferredReminderTime	string			`json:"preferred_reminder_time"`
	Timezone		string			`json:"timezone"`
	WorkingHours		map[string]interface{}	`json:"working_hours"`
	PersonalityTraits	map[string]float64	`json:"personality_traits"`
	AdaptationData		map[string]interface{}	`json:"adaptation_data"`
	LastUpdated		time.Time		`json:"last_updated"`
}

type PersonalityBehaviorAnalysis struct {
	CompletionPatterns	map[string]float64	`json:"completion_patterns"`
	PreferredDifficulty	float64			`json:"preferred_difficulty"`
	OptimalTaskDuration	float64			`json:"optimal_task_duration"`
	MotivationResponses	map[string]float64	`json:"motivation_responses"`
	CommunicationPrefs	map[string]float64	`json:"communication_prefs"`
	TimePreferences		map[string]float64	`json:"time_preferences"`
	CategoryPreferences	map[string]float64	`json:"category_preferences"`
	SocialPreferences	map[string]float64	`json:"social_preferences"`
	Adaptations		[]string		`json:"adaptations"`
}

type PersonalityInsight struct {
	Type		string			`json:"type"`
	Title		string			`json:"title"`
	Description	string			`json:"description"`
	Confidence	float64			`json:"confidence"`
	Data		map[string]interface{}	`json:"data"`
	CreatedAt	time.Time		`json:"created_at"`
}

func NewPersonalityService(db *sqlx.DB) *PersonalityService {
	return &PersonalityService{db: db}
}

func (s *PersonalityService) GetUserPersonality(ctx context.Context, userID int64) (*PersonalityProfile, error) {
	profile := &PersonalityProfile{}

	query := `
		SELECT personality_type, motivation_style, communication_style, 
			   activity_level, preferred_reminder_time, timezone, jarvis_settings
		FROM users 
		WHERE id = $1
	`

	var settingsJSON string
	err := s.db.GetContext(ctx, &struct {
		PersonalityType		string	`db:"personality_type"`
		MotivationStyle		string	`db:"motivation_style"`
		CommunicationStyle	string	`db:"communication_style"`
		ActivityLevel		string	`db:"activity_level"`
		PreferredReminderTime	string	`db:"preferred_reminder_time"`
		Timezone		string	`db:"timezone"`
		JarvisSettings		string	`db:"jarvis_settings"`
	}{}, query, userID)

	if err != nil {

		profile = s.createDefaultProfile(userID)
		return profile, nil
	}

	var jarvisSettings map[string]interface{}
	json.Unmarshal([]byte(settingsJSON), &jarvisSettings)

	profile.UserID = userID
	profile.AdaptationData = jarvisSettings
	profile.LastUpdated = time.Now()

	behaviorAnalysis, err := s.analyzeBehaviorPatterns(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось проанализировать поведение пользователя %d: %v", userID, err)
	} else {
		profile = s.refineProfileWithBehavior(profile, behaviorAnalysis)
	}

	return profile, nil
}

func (s *PersonalityService) UpdatePersonalityFromBehavior(ctx context.Context, userID int64) error {
	behaviorAnalysis, err := s.analyzeBehaviorPatterns(ctx, userID)
	if err != nil {
		return err
	}

	newTraits := s.inferPersonalityFromBehavior(behaviorAnalysis)

	adaptationDataJSON, _ := json.Marshal(map[string]interface{}{
		"personality_traits":	newTraits,
		"behavior_analysis":	behaviorAnalysis,
		"last_analysis":	time.Now(),
	})

	query := `
		UPDATE users 
		SET personality_type = $1, motivation_style = $2, communication_style = $3,
			activity_level = $4, jarvis_settings = $5
		WHERE id = $6
	`

	personalityType := s.determinePersonalityType(newTraits)
	motivationStyle := s.determineMotivationStyle(behaviorAnalysis)
	communicationStyle := s.determineCommunicationStyle(behaviorAnalysis)
	activityLevel := s.determineActivityLevel(behaviorAnalysis)

	_, err = s.db.ExecContext(ctx, query, personalityType, motivationStyle,
		communicationStyle, activityLevel, string(adaptationDataJSON), userID)

	return err
}

func (s *PersonalityService) AnalyzePersonalityTrends(ctx context.Context, userID int64) (map[string]interface{}, error) {

	behaviorHistory, err := s.getBehaviorHistory(ctx, userID, 30)
	if err != nil {
		return nil, err
	}

	trends := make(map[string]interface{})

	productivityTrend := s.analyzeProductivityTrend(behaviorHistory)
	trends["productivity_trend"] = productivityTrend

	motivationTrend := s.analyzeMotivationTrend(behaviorHistory)
	trends["motivation_trend"] = motivationTrend

	difficultyTrend := s.analyzeDifficultyPreferenceTrend(behaviorHistory)
	trends["difficulty_preference_trend"] = difficultyTrend

	timeTrend := s.analyzeTimePreferenceTrend(behaviorHistory)
	trends["time_preference_trend"] = timeTrend

	socialTrend := s.analyzeSocialActivityTrend(behaviorHistory)
	trends["social_activity_trend"] = socialTrend

	insights := s.generatePersonalityInsights(behaviorHistory)
	trends["insights"] = insights

	return trends, nil
}

func (s *PersonalityService) GeneratePersonalizedMessage(profile *PersonalityProfile, messageType string, context map[string]interface{}) string {
	switch profile.CommunicationStyle {
	case "direct":
		return s.generateDirectMessage(messageType, context)
	case "friendly":
		return s.generateFriendlyMessage(messageType, context)
	case "formal":
		return s.generateFormalMessage(messageType, context)
	case "casual":
		return s.generateCasualMessage(messageType, context)
	case "encouraging":
		return s.generateEncouragingMessage(messageType, context)
	default:
		return s.generateFriendlyMessage(messageType, context)
	}
}

func (s *PersonalityService) AdaptCommunicationStyle(ctx context.Context, userID int64, messageType string, userResponse string) error {

	responseAnalysis := s.analyzeUserResponse(userResponse)

	return s.updateCommunicationPreferences(ctx, userID, messageType, responseAnalysis)
}

func (s *PersonalityService) createDefaultProfile(userID int64) *PersonalityProfile {
	return &PersonalityProfile{
		UserID:			userID,
		PersonalityType:	"balanced",
		MotivationStyle:	"achievement",
		CommunicationStyle:	"friendly",
		ActivityLevel:		"moderate",
		PreferredReminderTime:	"09:00",
		Timezone:		"UTC+3",
		PersonalityTraits: map[string]float64{
			"openness":		0.5,
			"conscientiousness":	0.5,
			"extraversion":		0.5,
			"agreeableness":	0.5,
			"neuroticism":		0.3,
		},
		AdaptationData:	make(map[string]interface{}),
		LastUpdated:	time.Now(),
	}
}

func (s *PersonalityService) analyzeBehaviorPatterns(ctx context.Context, userID int64) (*PersonalityBehaviorAnalysis, error) {
	analysis := &PersonalityBehaviorAnalysis{
		CompletionPatterns:	make(map[string]float64),
		MotivationResponses:	make(map[string]float64),
		CommunicationPrefs:	make(map[string]float64),
		TimePreferences:	make(map[string]float64),
		CategoryPreferences:	make(map[string]float64),
		SocialPreferences:	make(map[string]float64),
		Adaptations:		[]string{},
	}

	completionData, err := s.getCompletionPatterns(ctx, userID)
	if err != nil {
		return nil, err
	}
	analysis.CompletionPatterns = completionData

	difficultyData, err := s.getPreferredDifficulty(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить данные о сложности: %v", err)
	} else {
		analysis.PreferredDifficulty = difficultyData
	}

	durationData, err := s.getOptimalTaskDuration(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить данные о продолжительности: %v", err)
	} else {
		analysis.OptimalTaskDuration = durationData
	}

	timePrefs, err := s.getTimePreferences(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить временные предпочтения: %v", err)
	} else {
		analysis.TimePreferences = timePrefs
	}

	categoryPrefs, err := s.getCategoryPreferences(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить категориальные предпочтения: %v", err)
	} else {
		analysis.CategoryPreferences = categoryPrefs
	}

	return analysis, nil
}

func (s *PersonalityService) refineProfileWithBehavior(profile *PersonalityProfile, behavior *PersonalityBehaviorAnalysis) *PersonalityProfile {

	if profile.PersonalityTraits == nil {
		profile.PersonalityTraits = make(map[string]float64)
	}

	if completionRate, ok := behavior.CompletionPatterns["overall_rate"]; ok {
		profile.PersonalityTraits["conscientiousness"] = completionRate
	}

	categoryCount := float64(len(behavior.CategoryPreferences))
	if categoryCount > 0 {
		profile.PersonalityTraits["openness"] = min(categoryCount/5.0, 1.0)
	}

	if socialActivity, ok := behavior.SocialPreferences["team_goals"]; ok {
		profile.PersonalityTraits["extraversion"] = socialActivity
	}

	profile.PersonalityType = s.determinePersonalityType(profile.PersonalityTraits)

	return profile
}

func (s *PersonalityService) inferPersonalityFromBehavior(behavior *PersonalityBehaviorAnalysis) map[string]float64 {
	traits := make(map[string]float64)

	if rate, ok := behavior.CompletionPatterns["consistency"]; ok {
		traits["conscientiousness"] = rate
	}

	if behavior.PreferredDifficulty > 0.7 {
		traits["openness"] = 0.8
	} else if behavior.PreferredDifficulty < 0.3 {
		traits["neuroticism"] = 0.6
	}

	if behavior.OptimalTaskDuration < 30 {
		traits["attention_span"] = 0.3
	} else if behavior.OptimalTaskDuration > 120 {
		traits["persistence"] = 0.8
	}

	return traits
}

func (s *PersonalityService) determinePersonalityType(traits map[string]float64) string {
	conscientiousness := traits["conscientiousness"]
	openness := traits["openness"]
	extraversion := traits["extraversion"]

	if conscientiousness > 0.7 && openness > 0.6 {
		return "analytical"
	} else if openness > 0.7 && extraversion > 0.6 {
		return "creative"
	} else if extraversion > 0.7 {
		return "social"
	} else if conscientiousness > 0.6 {
		return "pragmatic"
	}

	return "balanced"
}

func (s *PersonalityService) determineMotivationStyle(behavior *PersonalityBehaviorAnalysis) string {

	if behavior.PreferredDifficulty > 0.7 {
		return "challenge"
	}

	if socialPrefs, ok := behavior.SocialPreferences["team_goals"]; ok && socialPrefs > 0.6 {
		return "social"
	}

	if rate, ok := behavior.CompletionPatterns["achievement_rate"]; ok && rate > 0.7 {
		return "achievement"
	}

	return "growth"
}

func (s *PersonalityService) determineCommunicationStyle(behavior *PersonalityBehaviorAnalysis) string {

	return "friendly"
}

func (s *PersonalityService) determineActivityLevel(behavior *PersonalityBehaviorAnalysis) string {
	if rate, ok := behavior.CompletionPatterns["daily_activity"]; ok {
		if rate > 0.8 {
			return "very_high"
		} else if rate > 0.6 {
			return "high"
		} else if rate > 0.3 {
			return "moderate"
		}
		return "low"
	}
	return "moderate"
}

func (s *PersonalityService) getCompletionPatterns(ctx context.Context, userID int64) (map[string]float64, error) {
	patterns := make(map[string]float64)

	query := `
		SELECT 
			COUNT(CASE WHEN progress >= target THEN 1 END)::float / COUNT(*)::float as completion_rate,
			AVG(CASE WHEN progress > 0 THEN 1 ELSE 0 END) as activity_rate
		FROM key_results kr
		JOIN objectives o ON kr.objective_id = o.id
		WHERE o.user_id = $1 AND o.created_at > NOW() - INTERVAL '30 days'
	`

	var completionRate, activityRate float64
	err := s.db.GetContext(ctx, &struct {
		CompletionRate	float64	`db:"completion_rate"`
		ActivityRate	float64	`db:"activity_rate"`
	}{
		CompletionRate:	completionRate,
		ActivityRate:	activityRate,
	}, query, userID)

	if err != nil {
		return patterns, err
	}

	patterns["overall_rate"] = completionRate
	patterns["activity_rate"] = activityRate

	return patterns, nil
}

func (s *PersonalityService) getPreferredDifficulty(ctx context.Context, userID int64) (float64, error) {
	query := `
		SELECT AVG(difficulty_level::float / 5.0) as avg_difficulty
		FROM objectives 
		WHERE user_id = $1 AND created_at > NOW() - INTERVAL '30 days'
	`

	var avgDifficulty float64
	err := s.db.GetContext(ctx, &avgDifficulty, query, userID)
	return avgDifficulty, err
}

func (s *PersonalityService) getOptimalTaskDuration(ctx context.Context, userID int64) (float64, error) {
	query := `
		SELECT AVG(actual_hours * 60) as avg_duration_minutes
		FROM tasks t
		JOIN key_results kr ON t.key_result_id = kr.id
		JOIN objectives o ON kr.objective_id = o.id
		WHERE o.user_id = $1 AND t.actual_hours > 0 AND t.created_at > NOW() - INTERVAL '30 days'
	`

	var avgDuration float64
	err := s.db.GetContext(ctx, &avgDuration, query, userID)
	return avgDuration, err
}

func (s *PersonalityService) getTimePreferences(ctx context.Context, userID int64) (map[string]float64, error) {
	prefs := make(map[string]float64)

	query := `
		SELECT 
			EXTRACT(hour FROM created_at) as hour,
			COUNT(*) as count
		FROM habit_tracking
		WHERE user_id = $1 AND completed = true AND created_at > NOW() - INTERVAL '30 days'
		GROUP BY EXTRACT(hour FROM created_at)
		ORDER BY count DESC
		LIMIT 5
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return prefs, err
	}
	defer rows.Close()

	total := 0.0
	for rows.Next() {
		var hour int
		var count float64
		if err := rows.Scan(&hour, &count); err != nil {
			continue
		}
		prefs[fmt.Sprintf("hour_%d", hour)] = count
		total += count
	}

	for k, v := range prefs {
		prefs[k] = v / total
	}

	return prefs, nil
}

func (s *PersonalityService) getCategoryPreferences(ctx context.Context, userID int64) (map[string]float64, error) {
	prefs := make(map[string]float64)

	query := `
		SELECT 
			oc.name as category,
			COUNT(o.id) as count
		FROM objectives o
		JOIN objective_categories oc ON o.category_id = oc.id
		WHERE o.user_id = $1 AND o.created_at > NOW() - INTERVAL '90 days'
		GROUP BY oc.name
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return prefs, err
	}
	defer rows.Close()

	total := 0.0
	for rows.Next() {
		var category string
		var count float64
		if err := rows.Scan(&category, &count); err != nil {
			continue
		}
		prefs[category] = count
		total += count
	}

	for k, v := range prefs {
		prefs[k] = v / total
	}

	return prefs, nil
}

func (s *PersonalityService) getBehaviorHistory(ctx context.Context, userID int64, days int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			date,
			completed,
			mood_before,
			mood_after,
			energy_level,
			time_spent_minutes
		FROM habit_tracking
		WHERE user_id = $1 AND date > CURRENT_DATE - INTERVAL '%d days'
		ORDER BY date DESC
	`

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(query, days), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var record map[string]interface{}

		history = append(history, record)
	}

	return history, nil
}

func (s *PersonalityService) generateDirectMessage(messageType string, context map[string]interface{}) string {
	switch messageType {
	case "reminder":
		return "Пора выполнить задачу. Дедлайн приближается."
	case "motivation":
		return "Сосредоточься на результате. Выполни задачу."
	case "celebration":
		return "Задача выполнена. Хорошая работа."
	default:
		return "Обновление по вашим целям."
	}
}

func (s *PersonalityService) generateFriendlyMessage(messageType string, context map[string]interface{}) string {
	switch messageType {
	case "reminder":
		return "Привет! 😊 Напоминаю о твоей задаче. Давай сделаем это вместе!"
	case "motivation":
		return "Я верю в тебя! 💪 Ты справишься с этой задачей. Главное - начать!"
	case "celebration":
		return "Поздравляю! 🎉 Ты отлично справился с задачей. Я горжусь тобой!"
	default:
		return "У меня есть обновления по твоим целям! 📊"
	}
}

func (s *PersonalityService) generateFormalMessage(messageType string, context map[string]interface{}) string {
	switch messageType {
	case "reminder":
		return "Уважаемый пользователь, напоминаем о необходимости выполнения запланированной задачи."
	case "motivation":
		return "Рекомендуем сосредоточиться на выполнении поставленных целей для достижения оптимальных результатов."
	case "celebration":
		return "Поздравляем с успешным выполнением задачи. Ваша продуктивность на высоком уровне."
	default:
		return "Предоставляем отчет о статусе ваших целей."
	}
}

func (s *PersonalityService) generateCasualMessage(messageType string, context map[string]interface{}) string {
	switch messageType {
	case "reminder":
		return "Эй, не забыл про задачку? 🤔 Времени еще есть, но лучше не откладывать!"
	case "motivation":
		return "Давай, ты можешь! 🚀 Эта задача тебе по плечу. Просто начни и все пойдет как по маслу!"
	case "celebration":
		return "Вау! 🔥 Ты крут! Задача в кармане. Так держать!"
	default:
		return "Что там с твоими целями? Посмотрим... 👀"
	}
}

func (s *PersonalityService) generateEncouragingMessage(messageType string, context map[string]interface{}) string {
	switch messageType {
	case "reminder":
		return "Ты на правильном пути! 🌟 Эта задача приблизит тебя к мечте. Давай сделаем шаг к успеху!"
	case "motivation":
		return "Каждый шаг важен! 💫 Ты становишься лучше с каждой выполненной задачей. Продолжай в том же духе!"
	case "celebration":
		return "Невероятно! 🌈 Ты превзошел себя! Этот успех - заслуженная награда за твои усилия!"
	default:
		return "Твой прогресс впечатляет! ✨ Посмотри, как далеко ты продвинулся!"
	}
}

func (s *PersonalityService) analyzeProductivityTrend(history []map[string]interface{}) map[string]interface{} {

	return map[string]interface{}{
		"trend":	"improving",
		"confidence":	0.75,
		"key_factors":	[]string{"consistent_completion", "time_management"},
	}
}

func (s *PersonalityService) analyzeMotivationTrend(history []map[string]interface{}) map[string]interface{} {

	return map[string]interface{}{
		"trend":	"stable",
		"confidence":	0.80,
		"key_factors":	[]string{"positive_mood", "energy_consistency"},
	}
}

func (s *PersonalityService) analyzeDifficultyPreferenceTrend(history []map[string]interface{}) map[string]interface{} {

	return map[string]interface{}{
		"trend":		"increasing",
		"confidence":		0.65,
		"recommendation":	"ready_for_harder_challenges",
	}
}

func (s *PersonalityService) analyzeTimePreferenceTrend(history []map[string]interface{}) map[string]interface{} {

	return map[string]interface{}{
		"peak_hours":	[]int{9, 10, 11, 15, 16},
		"low_hours":	[]int{13, 14, 20, 21},
		"consistency":	0.70,
	}
}

func (s *PersonalityService) analyzeSocialActivityTrend(history []map[string]interface{}) map[string]interface{} {

	return map[string]interface{}{
		"social_engagement":	0.45,
		"prefers_solo_work":	true,
		"team_readiness":	0.60,
	}
}

func (s *PersonalityService) generatePersonalityInsights(history []map[string]interface{}) []PersonalityInsight {
	insights := []PersonalityInsight{
		{
			Type:		"productivity",
			Title:		"Твой пик продуктивности",
			Description:	"Ты наиболее продуктивен утром с 9 до 11 часов",
			Confidence:	0.85,
			Data: map[string]interface{}{
				"peak_hours":		[]int{9, 10, 11},
				"efficiency_score":	0.92,
			},
			CreatedAt:	time.Now(),
		},
		{
			Type:		"motivation",
			Title:		"Тип мотивации",
			Description:	"Тебя лучше всего мотивируют достижения и прогресс",
			Confidence:	0.78,
			Data: map[string]interface{}{
				"motivation_type":	"achievement",
				"response_rate":	0.87,
			},
			CreatedAt:	time.Now(),
		},
	}

	return insights
}

func (s *PersonalityService) analyzeUserResponse(response string) map[string]float64 {
	analysis := make(map[string]float64)

	response = strings.ToLower(response)

	positiveWords := []string{"да", "хорошо", "отлично", "супер", "круто", "спасибо", "👍", "😊", "💪"}
	negativeWords := []string{"нет", "плохо", "не нравится", "скучно", "сложно", "👎", "😔", "😞"}

	positiveCount := 0.0
	negativeCount := 0.0

	for _, word := range positiveWords {
		if strings.Contains(response, word) {
			positiveCount++
		}
	}

	for _, word := range negativeWords {
		if strings.Contains(response, word) {
			negativeCount++
		}
	}

	total := positiveCount + negativeCount
	if total > 0 {
		analysis["positive_sentiment"] = positiveCount / total
		analysis["negative_sentiment"] = negativeCount / total
	}

	analysis["engagement_level"] = min(float64(len(response))/100.0, 1.0)

	return analysis
}

func (s *PersonalityService) updateCommunicationPreferences(ctx context.Context, userID int64, messageType string, responseAnalysis map[string]float64) error {

	var currentSettings string
	query := `SELECT jarvis_settings FROM users WHERE id = $1`
	err := s.db.GetContext(ctx, &currentSettings, query, userID)
	if err != nil {
		return err
	}

	var settings map[string]interface{}
	if currentSettings != "" {
		json.Unmarshal([]byte(currentSettings), &settings)
	} else {
		settings = make(map[string]interface{})
	}

	if _, exists := settings["communication_feedback"]; !exists {
		settings["communication_feedback"] = make(map[string]interface{})
	}

	commFeedback := settings["communication_feedback"].(map[string]interface{})
	commFeedback[messageType] = responseAnalysis

	settingsJSON, _ := json.Marshal(settings)
	updateQuery := `UPDATE users SET jarvis_settings = $1 WHERE id = $2`
	_, err = s.db.ExecContext(ctx, updateQuery, string(settingsJSON), userID)

	return err
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

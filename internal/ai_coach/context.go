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

type ContextService struct {
	db *sqlx.DB
}

type UserContext struct {
	ID		int64			`db:"id" json:"id"`
	UserID		int64			`db:"user_id" json:"user_id"`
	ContextType	string			`db:"context_type" json:"context_type"`
	ContextData	map[string]interface{}	`db:"context_data" json:"context_data"`
	CreatedAt	time.Time		`db:"created_at" json:"created_at"`
	ExpiresAt	*time.Time		`db:"expires_at" json:"expires_at,omitempty"`
	IsActive	bool			`db:"is_active" json:"is_active"`
}

type ConversationContext struct {
	CurrentTopic		string			`json:"current_topic"`
	LastMentionedGoals	[]string		`json:"last_mentioned_goals"`
	ConversationFlow	[]string		`json:"conversation_flow"`
	UserIntent		string			`json:"user_intent"`
	EmotionalState		string			`json:"emotional_state"`
	UnansweredQuestions	[]string		`json:"unanswered_questions"`
	ActiveSuggestions	[]string		`json:"active_suggestions"`
	SessionData		map[string]interface{}	`json:"session_data"`
	LastInteraction		time.Time		`json:"last_interaction"`
}

type ActivityContext struct {
	CurrentGoalsFocus	[]string		`json:"current_goals_focus"`
	RecentCompletions	[]CompletionRecord	`json:"recent_completions"`
	PendingDeadlines	[]DeadlineInfo		`json:"pending_deadlines"`
	ProductivityLevel	float64			`json:"productivity_level"`
	EnergyLevel		int			`json:"energy_level"`
	AvailableTime		float64			`json:"available_time"`
	PreferredCategories	[]string		`json:"preferred_categories"`
	CurrentChallenges	[]string		`json:"current_challenges"`
}

type MoodContext struct {
	CurrentMood	int		`json:"current_mood"`
	MoodTrend	string		`json:"mood_trend"`
	MotivationLevel	float64		`json:"motivation_level"`
	StressLevel	int		`json:"stress_level"`
	ConfidenceLevel	float64		`json:"confidence_level"`
	EnergyLevel	int		`json:"energy_level"`
	LastMoodUpdate	time.Time	`json:"last_mood_update"`
	MoodTriggers	[]string	`json:"mood_triggers"`
	PositiveFactors	[]string	`json:"positive_factors"`
}

type LocationContext struct {
	CurrentLocation	string			`json:"current_location"`
	LocationHistory	[]LocationRecord	`json:"location_history"`
	LocationPrefs	map[string]interface{}	`json:"location_preferences"`
	WorkLocations	[]string		`json:"work_locations"`
	FavoriteSpots	[]string		`json:"favorite_spots"`
}

type CompletionRecord struct {
	ID		string		`json:"id"`
	Title		string		`json:"title"`
	Type		string		`json:"type"`
	CompletedAt	time.Time	`json:"completed_at"`
	Mood		int		`json:"mood"`
	Satisfaction	int		`json:"satisfaction"`
}

type DeadlineInfo struct {
	ID		string		`json:"id"`
	Title		string		`json:"title"`
	Type		string		`json:"type"`
	Deadline	time.Time	`json:"deadline"`
	DaysLeft	int		`json:"days_left"`
	Progress	float64		`json:"progress"`
	Priority	int		`json:"priority"`
	Status		string		`json:"status"`
}

type LocationRecord struct {
	Location	string		`json:"location"`
	Timestamp	time.Time	`json:"timestamp"`
	Activity	string		`json:"activity"`
}

type ContextualInsight struct {
	Type		string			`json:"type"`
	Priority	int			`json:"priority"`
	Title		string			`json:"title"`
	Message		string			`json:"message"`
	ActionItems	[]string		`json:"action_items"`
	Context		map[string]interface{}	`json:"context"`
	Confidence	float64			`json:"confidence"`
}

func NewContextService(db *sqlx.DB) *ContextService {
	return &ContextService{db: db}
}

func (s *ContextService) GetCurrentContext(ctx context.Context, userID int64) (map[string]interface{}, error) {
	logrus.Infof("Получение контекста для пользователя: %d", userID)

	currentContext := make(map[string]interface{})

	conversationCtx, err := s.getConversationContext(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить контекст разговора: %v", err)
	} else {
		currentContext["conversation"] = conversationCtx
	}

	activityCtx, err := s.getActivityContext(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить контекст активности: %v", err)
	} else {
		currentContext["activity"] = activityCtx
	}

	moodCtx, err := s.getMoodContext(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить контекст настроения: %v", err)
	} else {
		currentContext["mood"] = moodCtx
	}

	locationCtx, err := s.getLocationContext(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить контекст местоположения: %v", err)
	} else {
		currentContext["location"] = locationCtx
	}

	timeCtx := s.getTimeContext()
	currentContext["time"] = timeCtx

	logrus.Infof("Контекст получен для пользователя %d: %d категорий", userID, len(currentContext))
	return currentContext, nil
}

func (s *ContextService) UpdateConversationContext(ctx context.Context, userID int64, message string, intent string) error {
	conversationCtx, err := s.getConversationContext(ctx, userID)
	if err != nil {

		conversationCtx = &ConversationContext{
			ConversationFlow:	[]string{},
			LastMentionedGoals:	[]string{},
			UnansweredQuestions:	[]string{},
			ActiveSuggestions:	[]string{},
			SessionData:		make(map[string]interface{}),
		}
	}

	conversationCtx.CurrentTopic = s.extractTopic(message)
	conversationCtx.UserIntent = intent
	conversationCtx.EmotionalState = s.detectEmotionalState(message)
	conversationCtx.LastInteraction = time.Now()

	conversationCtx.ConversationFlow = append(conversationCtx.ConversationFlow, fmt.Sprintf("%s: %s", intent, conversationCtx.CurrentTopic))

	if len(conversationCtx.ConversationFlow) > 10 {
		conversationCtx.ConversationFlow = conversationCtx.ConversationFlow[len(conversationCtx.ConversationFlow)-10:]
	}

	mentionedGoals := s.extractMentionedGoals(ctx, userID, message)
	if len(mentionedGoals) > 0 {
		conversationCtx.LastMentionedGoals = mentionedGoals
	}

	return s.saveContext(ctx, userID, "conversation", conversationCtx, time.Now().Add(24*time.Hour))
}

func (s *ContextService) UpdateActivityContext(ctx context.Context, userID int64) error {
	activityCtx := &ActivityContext{}

	currentGoals, err := s.getCurrentFocusGoals(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить текущие цели: %v", err)
	} else {
		activityCtx.CurrentGoalsFocus = currentGoals
	}

	recentCompletions, err := s.getRecentCompletions(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить недавние выполнения: %v", err)
	} else {
		activityCtx.RecentCompletions = recentCompletions
	}

	pendingDeadlines, err := s.getPendingDeadlines(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить дедлайны: %v", err)
	} else {
		activityCtx.PendingDeadlines = pendingDeadlines
	}

	activityCtx.ProductivityLevel = s.calculateProductivityLevel(recentCompletions)

	preferredCategories, err := s.getPreferredCategories(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить предпочтительные категории: %v", err)
	} else {
		activityCtx.PreferredCategories = preferredCategories
	}

	activityCtx.CurrentChallenges = s.identifyCurrentChallenges(pendingDeadlines, activityCtx.ProductivityLevel)

	return s.saveContext(ctx, userID, "activity", activityCtx, time.Now().Add(12*time.Hour))
}

func (s *ContextService) UpdateMoodContext(ctx context.Context, userID int64, mood int, energy int) error {
	moodCtx, err := s.getMoodContext(ctx, userID)
	if err != nil {
		moodCtx = &MoodContext{
			MoodTriggers:		[]string{},
			PositiveFactors:	[]string{},
		}
	}

	oldMood := moodCtx.CurrentMood
	moodCtx.CurrentMood = mood
	moodCtx.EnergyLevel = energy
	moodCtx.LastMoodUpdate = time.Now()

	if mood > oldMood {
		moodCtx.MoodTrend = "improving"
	} else if mood < oldMood {
		moodCtx.MoodTrend = "declining"
	} else {
		moodCtx.MoodTrend = "stable"
	}

	moodCtx.MotivationLevel = s.calculateMotivationLevel(mood, energy)

	moodCtx.StressLevel = s.estimateStressLevel(ctx, userID, mood, energy)

	moodCtx.ConfidenceLevel = s.calculateConfidenceLevel(ctx, userID, mood)

	return s.saveContext(ctx, userID, "mood", moodCtx, time.Now().Add(6*time.Hour))
}

func (s *ContextService) GenerateContextualInsights(ctx context.Context, userID int64, currentContext map[string]interface{}) ([]ContextualInsight, error) {
	var insights []ContextualInsight

	if convCtx, ok := currentContext["conversation"]; ok {
		convInsights := s.analyzeConversationContext(convCtx)
		insights = append(insights, convInsights...)
	}

	if actCtx, ok := currentContext["activity"]; ok {
		actInsights := s.analyzeActivityContext(actCtx)
		insights = append(insights, actInsights...)
	}

	if moodCtx, ok := currentContext["mood"]; ok {
		moodInsights := s.analyzeMoodContext(moodCtx)
		insights = append(insights, moodInsights...)
	}

	crossInsights := s.performCrossContextAnalysis(currentContext)
	insights = append(insights, crossInsights...)

	return insights, nil
}

func (s *ContextService) getConversationContext(ctx context.Context, userID int64) (*ConversationContext, error) {
	var contextData string
	query := `
		SELECT context_data 
		FROM user_context 
		WHERE user_id = $1 AND context_type = 'conversation' AND is_active = true
		ORDER BY created_at DESC 
		LIMIT 1
	`

	err := s.db.GetContext(ctx, &contextData, query, userID)
	if err != nil {
		return nil, err
	}

	var convCtx ConversationContext
	err = json.Unmarshal([]byte(contextData), &convCtx)
	return &convCtx, err
}

func (s *ContextService) getActivityContext(ctx context.Context, userID int64) (*ActivityContext, error) {
	var contextData string
	query := `
		SELECT context_data 
		FROM user_context 
		WHERE user_id = $1 AND context_type = 'activity' AND is_active = true
		ORDER BY created_at DESC 
		LIMIT 1
	`

	err := s.db.GetContext(ctx, &contextData, query, userID)
	if err != nil {
		return nil, err
	}

	var actCtx ActivityContext
	err = json.Unmarshal([]byte(contextData), &actCtx)
	return &actCtx, err
}

func (s *ContextService) getMoodContext(ctx context.Context, userID int64) (*MoodContext, error) {
	var contextData string
	query := `
		SELECT context_data 
		FROM user_context 
		WHERE user_id = $1 AND context_type = 'mood' AND is_active = true
		ORDER BY created_at DESC 
		LIMIT 1
	`

	err := s.db.GetContext(ctx, &contextData, query, userID)
	if err != nil {
		return nil, err
	}

	var moodCtx MoodContext
	err = json.Unmarshal([]byte(contextData), &moodCtx)
	return &moodCtx, err
}

func (s *ContextService) getLocationContext(ctx context.Context, userID int64) (*LocationContext, error) {
	var contextData string
	query := `
		SELECT context_data 
		FROM user_context 
		WHERE user_id = $1 AND context_type = 'location' AND is_active = true
		ORDER BY created_at DESC 
		LIMIT 1
	`

	err := s.db.GetContext(ctx, &contextData, query, userID)
	if err != nil {
		return nil, err
	}

	var locCtx LocationContext
	err = json.Unmarshal([]byte(contextData), &locCtx)
	return &locCtx, err
}

func (s *ContextService) getTimeContext() map[string]interface{} {
	now := time.Now()

	return map[string]interface{}{
		"current_time":		now,
		"hour":			now.Hour(),
		"day_of_week":		now.Weekday().String(),
		"is_weekend":		now.Weekday() == time.Saturday || now.Weekday() == time.Sunday,
		"is_morning":		now.Hour() >= 6 && now.Hour() < 12,
		"is_afternoon":		now.Hour() >= 12 && now.Hour() < 18,
		"is_evening":		now.Hour() >= 18 && now.Hour() < 22,
		"is_night":		now.Hour() >= 22 || now.Hour() < 6,
		"month":		now.Month().String(),
		"season":		s.getSeason(now),
		"is_work_hours":	s.isWorkHours(now),
	}
}

func (s *ContextService) saveContext(ctx context.Context, userID int64, contextType string, contextData interface{}, expiresAt time.Time) error {

	deactivateQuery := `
		UPDATE user_context 
		SET is_active = false 
		WHERE user_id = $1 AND context_type = $2
	`
	_, err := s.db.ExecContext(ctx, deactivateQuery, userID, contextType)
	if err != nil {
		logrus.Warnf("Не удалось деактивировать старый контекст: %v", err)
	}

	dataJSON, err := json.Marshal(contextData)
	if err != nil {
		return err
	}

	insertQuery := `
		INSERT INTO user_context (user_id, context_type, context_data, expires_at, is_active)
		VALUES ($1, $2, $3, $4, true)
	`

	_, err = s.db.ExecContext(ctx, insertQuery, userID, contextType, string(dataJSON), expiresAt)
	return err
}

func (s *ContextService) extractTopic(message string) string {
	message = strings.ToLower(message)

	if strings.Contains(message, "цел") || strings.Contains(message, "goal") {
		return "goals"
	} else if strings.Contains(message, "задач") || strings.Contains(message, "task") {
		return "tasks"
	} else if strings.Contains(message, "прогресс") || strings.Contains(message, "progress") {
		return "progress"
	} else if strings.Contains(message, "мотивац") || strings.Contains(message, "motivat") {
		return "motivation"
	} else if strings.Contains(message, "отчет") || strings.Contains(message, "report") {
		return "reporting"
	} else if strings.Contains(message, "план") || strings.Contains(message, "plan") {
		return "planning"
	}

	return "general"
}

func (s *ContextService) detectEmotionalState(message string) string {
	message = strings.ToLower(message)

	positiveWords := []string{"отлично", "хорошо", "супер", "круто", "рад", "счастлив", "вдохновлен", "мотивирован"}
	negativeWords := []string{"плохо", "устал", "грустно", "сложно", "не получается", "проблема", "стресс"}
	neutralWords := []string{"нормально", "обычно", "как всегда", "стандартно"}

	positiveCount := 0
	negativeCount := 0
	neutralCount := 0

	for _, word := range positiveWords {
		if strings.Contains(message, word) {
			positiveCount++
		}
	}

	for _, word := range negativeWords {
		if strings.Contains(message, word) {
			negativeCount++
		}
	}

	for _, word := range neutralWords {
		if strings.Contains(message, word) {
			neutralCount++
		}
	}

	if positiveCount > negativeCount && positiveCount > neutralCount {
		return "positive"
	} else if negativeCount > positiveCount && negativeCount > neutralCount {
		return "negative"
	} else if neutralCount > 0 {
		return "neutral"
	}

	return "unknown"
}

func (s *ContextService) extractMentionedGoals(ctx context.Context, userID int64, message string) []string {

	query := `
		SELECT id, title 
		FROM objectives 
		WHERE user_id = $1 AND status = 'active'
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	var mentionedGoals []string
	message = strings.ToLower(message)

	for rows.Next() {
		var id, title string
		if err := rows.Scan(&id, &title); err != nil {
			continue
		}

		titleLower := strings.ToLower(title)
		words := strings.Fields(titleLower)

		for _, word := range words {
			if len(word) > 3 && strings.Contains(message, word) {
				mentionedGoals = append(mentionedGoals, id)
				break
			}
		}
	}

	return mentionedGoals
}

func (s *ContextService) getCurrentFocusGoals(ctx context.Context, userID int64) ([]string, error) {
	query := `
		SELECT o.id, o.title
		FROM objectives o
		WHERE o.user_id = $1 AND o.status = 'active'
		ORDER BY o.priority DESC, o.created_at DESC
		LIMIT 3
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []string
	for rows.Next() {
		var id, title string
		if err := rows.Scan(&id, &title); err != nil {
			continue
		}
		goals = append(goals, fmt.Sprintf("%s: %s", id, title))
	}

	return goals, nil
}

func (s *ContextService) getRecentCompletions(ctx context.Context, userID int64) ([]CompletionRecord, error) {
	query := `
		SELECT 
			o.id, o.title, 'objective' as type, o.completion_date,
			COALESCE(ht.mood_after, 3) as mood, 5 as satisfaction
		FROM objectives o
		LEFT JOIN habit_tracking ht ON ht.objective_id = o.id AND ht.date = CURRENT_DATE
		WHERE o.user_id = $1 AND o.completion_date IS NOT NULL 
		AND o.completion_date > NOW() - INTERVAL '7 days'
		
		UNION ALL
		
		SELECT 
			kr.id::text, kr.title, 'key_result' as type, kr.completion_date,
			COALESCE(ht.mood_after, 3) as mood, 4 as satisfaction
		FROM key_results kr
		JOIN objectives o ON kr.objective_id = o.id
		LEFT JOIN habit_tracking ht ON ht.key_result_id = kr.id AND ht.date = CURRENT_DATE
		WHERE o.user_id = $1 AND kr.completion_date IS NOT NULL 
		AND kr.completion_date > NOW() - INTERVAL '7 days'
		
		ORDER BY completion_date DESC
		LIMIT 10
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var completions []CompletionRecord
	for rows.Next() {
		var record CompletionRecord
		var completedAt *time.Time

		err := rows.Scan(&record.ID, &record.Title, &record.Type, &completedAt, &record.Mood, &record.Satisfaction)
		if err != nil {
			continue
		}

		if completedAt != nil {
			record.CompletedAt = *completedAt
		}

		completions = append(completions, record)
	}

	return completions, nil
}

func (s *ContextService) getPendingDeadlines(ctx context.Context, userID int64) ([]DeadlineInfo, error) {
	query := `
		SELECT 
			o.id, o.title, 'objective' as type, o.deadline,
			EXTRACT(DAYS FROM o.deadline - NOW())::int as days_left,
			COALESCE((SELECT AVG(progress/target*100) FROM key_results WHERE objective_id = o.id), 0) as progress,
			o.priority, o.status
		FROM objectives o
		WHERE o.user_id = $1 AND o.deadline IS NOT NULL AND o.deadline > NOW()
		AND o.status = 'active'
		
		UNION ALL
		
		SELECT 
			kr.id::text, kr.title, 'key_result' as type, kr.deadline,
			EXTRACT(DAYS FROM kr.deadline - NOW())::int as days_left,
			(kr.progress/kr.target*100) as progress,
			kr.priority, kr.status
		FROM key_results kr
		JOIN objectives o ON kr.objective_id = o.id
		WHERE o.user_id = $1 AND kr.deadline IS NOT NULL AND kr.deadline > NOW()
		AND kr.status = 'active'
		
		ORDER BY days_left ASC
		LIMIT 10
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deadlines []DeadlineInfo
	for rows.Next() {
		var deadline DeadlineInfo

		err := rows.Scan(&deadline.ID, &deadline.Title, &deadline.Type, &deadline.Deadline,
			&deadline.DaysLeft, &deadline.Progress, &deadline.Priority, &deadline.Status)
		if err != nil {
			continue
		}

		deadlines = append(deadlines, deadline)
	}

	return deadlines, nil
}

func (s *ContextService) getPreferredCategories(ctx context.Context, userID int64) ([]string, error) {
	query := `
		SELECT oc.name, COUNT(o.id) as count
		FROM objectives o
		JOIN objective_categories oc ON o.category_id = oc.id
		WHERE o.user_id = $1 AND o.created_at > NOW() - INTERVAL '90 days'
		GROUP BY oc.name
		ORDER BY count DESC
		LIMIT 3
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			continue
		}
		categories = append(categories, category)
	}

	return categories, nil
}

func (s *ContextService) calculateProductivityLevel(completions []CompletionRecord) float64 {
	if len(completions) == 0 {
		return 0.5
	}

	totalSatisfaction := 0
	for _, completion := range completions {
		totalSatisfaction += completion.Satisfaction
	}

	avgSatisfaction := float64(totalSatisfaction) / float64(len(completions))

	return avgSatisfaction / 5.0
}

func (s *ContextService) identifyCurrentChallenges(deadlines []DeadlineInfo, productivityLevel float64) []string {
	var challenges []string

	urgentDeadlines := 0
	for _, deadline := range deadlines {
		if deadline.DaysLeft <= 3 {
			urgentDeadlines++
		}
	}

	if urgentDeadlines > 0 {
		challenges = append(challenges, "urgent_deadlines")
	}

	lowProgressCount := 0
	for _, deadline := range deadlines {
		if deadline.Progress < 30 && deadline.DaysLeft <= 7 {
			lowProgressCount++
		}
	}

	if lowProgressCount > 0 {
		challenges = append(challenges, "low_progress")
	}

	if productivityLevel < 0.3 {
		challenges = append(challenges, "low_productivity")
	}

	return challenges
}

func (s *ContextService) calculateMotivationLevel(mood int, energy int) float64 {

	return (float64(mood) + float64(energy)) / 10.0
}

func (s *ContextService) estimateStressLevel(ctx context.Context, userID int64, mood int, energy int) int {

	baseStress := 5 - mood

	if energy < 2 {
		baseStress += 2
	}

	query := `SELECT COUNT(*) FROM objectives WHERE user_id = $1 AND status = 'active'`
	var activeGoals int
	err := s.db.GetContext(ctx, &activeGoals, query, userID)
	if err == nil && activeGoals > 5 {
		baseStress += 1
	}

	if baseStress > 5 {
		baseStress = 5
	}
	if baseStress < 1 {
		baseStress = 1
	}

	return baseStress
}

func (s *ContextService) calculateConfidenceLevel(ctx context.Context, userID int64, mood int) float64 {

	baseConfidence := float64(mood) / 5.0

	query := `
		SELECT COUNT(*) 
		FROM objectives 
		WHERE user_id = $1 AND completion_date IS NOT NULL 
		AND completion_date > NOW() - INTERVAL '7 days'
	`

	var recentSuccesses int
	err := s.db.GetContext(ctx, &recentSuccesses, query, userID)
	if err == nil && recentSuccesses > 0 {
		baseConfidence += 0.1 * float64(recentSuccesses)
	}

	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	return baseConfidence
}

func (s *ContextService) getSeason(t time.Time) string {
	month := t.Month()

	switch {
	case month >= 3 && month <= 5:
		return "spring"
	case month >= 6 && month <= 8:
		return "summer"
	case month >= 9 && month <= 11:
		return "autumn"
	default:
		return "winter"
	}
}

func (s *ContextService) isWorkHours(t time.Time) bool {
	hour := t.Hour()
	weekday := t.Weekday()

	return hour >= 9 && hour < 18 && weekday >= time.Monday && weekday <= time.Friday
}

func (s *ContextService) analyzeConversationContext(convCtx interface{}) []ContextualInsight {
	var insights []ContextualInsight

	insight := ContextualInsight{
		Type:		"conversation",
		Priority:	2,
		Title:		"Контекст разговора",
		Message:	"Анализирую контекст нашего разговора для лучшего понимания",
		Context:	map[string]interface{}{"conversation_data": convCtx},
		Confidence:	0.8,
	}

	insights = append(insights, insight)
	return insights
}

func (s *ContextService) analyzeActivityContext(actCtx interface{}) []ContextualInsight {
	var insights []ContextualInsight

	insight := ContextualInsight{
		Type:		"activity",
		Priority:	3,
		Title:		"Анализ активности",
		Message:	"На основе твоей активности я вижу паттерны продуктивности",
		Context:	map[string]interface{}{"activity_data": actCtx},
		Confidence:	0.85,
	}

	insights = append(insights, insight)
	return insights
}

func (s *ContextService) analyzeMoodContext(moodCtx interface{}) []ContextualInsight {
	var insights []ContextualInsight

	insight := ContextualInsight{
		Type:		"mood",
		Priority:	4,
		Title:		"Анализ настроения",
		Message:	"Твое настроение влияет на продуктивность. Давай это учтем",
		Context:	map[string]interface{}{"mood_data": moodCtx},
		Confidence:	0.75,
	}

	insights = append(insights, insight)
	return insights
}

func (s *ContextService) performCrossContextAnalysis(currentContext map[string]interface{}) []ContextualInsight {
	var insights []ContextualInsight

	insight := ContextualInsight{
		Type:		"cross_context",
		Priority:	5,
		Title:		"Комплексный анализ",
		Message:	"Объединяя все данные, я вижу полную картину твоего состояния",
		Context:	currentContext,
		Confidence:	0.90,
	}

	insights = append(insights, insight)
	return insights
}

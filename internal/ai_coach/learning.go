package ai_coach

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type LearningService struct {
	db *sqlx.DB
}

func NewLearningService(db *sqlx.DB) *LearningService {
	return &LearningService{
		db: db,
	}
}

type UserBehaviorPattern struct {
	ID		int64			`db:"id" json:"id"`
	UserID		int64			`db:"user_id" json:"user_id"`
	PatternType	string			`db:"pattern_type" json:"pattern_type"`
	PatternData	map[string]interface{}	`db:"pattern_data" json:"pattern_data"`
	Frequency	int			`db:"frequency" json:"frequency"`
	Confidence	float64			`db:"confidence" json:"confidence"`
	CreatedAt	time.Time		`db:"created_at" json:"created_at"`
	UpdatedAt	time.Time		`db:"updated_at" json:"updated_at"`
}

type LearningInsight struct {
	Type		string			`json:"type"`
	Title		string			`json:"title"`
	Description	string			`json:"description"`
	Confidence	float64			`json:"confidence"`
	PatternData	map[string]interface{}	`json:"pattern_data"`
	Recommendations	[]string		`json:"recommendations"`
	ActionItems	[]string		`json:"action_items"`
	Priority	int			`json:"priority"`
	CreatedAt	time.Time		`json:"created_at"`
}

type BehaviorAnalysis struct {
	UserID			int64			`json:"user_id"`
	ProductivityTrends	[]ProductivityTrend	`json:"productivity_trends"`
	CompletionPatterns	[]CompletionPattern	`json:"completion_patterns"`
	MotivationTriggers	[]MotivationTrigger	`json:"motivation_triggers"`
	TimePreferences		[]TimePreference	`json:"time_preferences"`
	CategoryAffinities	[]CategoryAffinity	`json:"category_affinities"`
	RiskFactors		[]RiskFactor		`json:"risk_factors"`
	SuccessPatterns		[]SuccessPattern	`json:"success_patterns"`
	AdaptationNeeds		[]AdaptationNeed	`json:"adaptation_needs"`
	LearningInsights	[]LearningInsight	`json:"learning_insights"`
}

type ProductivityTrend struct {
	TimeFrame	string		`json:"time_frame"`
	Trend		string		`json:"trend"`
	Value		float64		`json:"value"`
	Confidence	float64		`json:"confidence"`
	Factors		[]string	`json:"factors"`
}

type CompletionPattern struct {
	Pattern		string			`json:"pattern"`
	Description	string			`json:"description"`
	Frequency	int			`json:"frequency"`
	Accuracy	float64			`json:"accuracy"`
	Context		map[string]interface{}	`json:"context"`
}

type TimePreference struct {
	TimeSlot		string	`json:"time_slot"`
	ProductivityScore	float64	`json:"productivity_score"`
	Preference		string	`json:"preference"`
	Confidence		float64	`json:"confidence"`
}

type CategoryAffinity struct {
	Category		string	`json:"category"`
	AffinityScore		float64	`json:"affinity_score"`
	CompletionRate		float64	`json:"completion_rate"`
	SatisfactionRate	float64	`json:"satisfaction_rate"`
	Recommendation		string	`json:"recommendation"`
}

type RiskFactor struct {
	Factor		string		`json:"factor"`
	RiskLevel	float64		`json:"risk_level"`
	Impact		string		`json:"impact"`
	Mitigation	[]string	`json:"mitigation"`
	Frequency	int		`json:"frequency"`
}

type SuccessPattern struct {
	Pattern		string			`json:"pattern"`
	Description	string			`json:"description"`
	SuccessRate	float64			`json:"success_rate"`
	Context		map[string]interface{}	`json:"context"`
	Replication	[]string		`json:"replication"`
}

type AdaptationNeed struct {
	Area		string		`json:"area"`
	Need		string		`json:"need"`
	Priority	int		`json:"priority"`
	Suggestions	[]string	`json:"suggestions"`
	Timeline	string		`json:"timeline"`
}

const (
	PatternTypeProductivity	= "productivity"
	PatternTypeMotivation	= "motivation"
	PatternTypeCompletion	= "completion"
	PatternTypeTime		= "time_preference"
	PatternTypeCategory	= "category_affinity"
	PatternTypeRisk		= "risk_behavior"
	PatternTypeSuccess	= "success_behavior"
)

func (s *LearningService) AnalyzeBehaviorPatterns(ctx context.Context, userID int64) (*BehaviorAnalysis, error) {
	analysis := &BehaviorAnalysis{
		UserID: userID,
	}

	productivityTrends, err := s.analyzeProductivityTrends(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze productivity trends: %w", err)
	}
	analysis.ProductivityTrends = productivityTrends

	completionPatterns, err := s.analyzeCompletionPatterns(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze completion patterns: %w", err)
	}
	analysis.CompletionPatterns = completionPatterns

	timePreferences, err := s.analyzeTimePreferences(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze time preferences: %w", err)
	}
	analysis.TimePreferences = timePreferences

	categoryAffinities, err := s.analyzeCategoryAffinities(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze category affinities: %w", err)
	}
	analysis.CategoryAffinities = categoryAffinities

	riskFactors, err := s.analyzeRiskFactors(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze risk factors: %w", err)
	}
	analysis.RiskFactors = riskFactors

	successPatterns, err := s.analyzeSuccessPatterns(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze success patterns: %w", err)
	}
	analysis.SuccessPatterns = successPatterns

	adaptationNeeds := s.identifyAdaptationNeeds(analysis)
	analysis.AdaptationNeeds = adaptationNeeds

	insights := s.generateLearningInsights(analysis)
	analysis.LearningInsights = insights

	return analysis, nil
}

func (s *LearningService) LearnFromInteraction(ctx context.Context, userID int64, interaction map[string]interface{}) error {

	patterns := s.extractPatternsFromInteraction(interaction)

	for _, pattern := range patterns {
		err := s.updateBehaviorPattern(ctx, userID, pattern)
		if err != nil {
			return fmt.Errorf("failed to update behavior pattern: %w", err)
		}
	}

	return nil
}

func (s *LearningService) AdaptToUser(ctx context.Context, userID int64) (map[string]interface{}, error) {

	patterns, err := s.getUserBehaviorPatterns(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user behavior patterns: %w", err)
	}

	adaptations := make(map[string]interface{})

	productivityAdaptations := s.adaptForProductivity(patterns)
	adaptations["productivity"] = productivityAdaptations

	motivationAdaptations := s.adaptMotivationStrategies(patterns)
	adaptations["motivation"] = motivationAdaptations

	communicationAdaptations := s.adaptCommunicationStyle(patterns)
	adaptations["communication"] = communicationAdaptations

	timeAdaptations := s.adaptTimeRecommendations(patterns)
	adaptations["time"] = timeAdaptations

	goalAdaptations := s.adaptGoalSuggestions(patterns)
	adaptations["goals"] = goalAdaptations

	return adaptations, nil
}

func (s *LearningService) GetUserBehaviorPatterns(ctx context.Context, userID int64) ([]UserBehaviorPattern, error) {
	return s.getUserBehaviorPatterns(ctx, userID)
}

func (s *LearningService) analyzeProductivityTrends(ctx context.Context, userID int64) ([]ProductivityTrend, error) {

	return []ProductivityTrend{
		{
			TimeFrame:	"daily",
			Trend:		"stable",
			Value:		0.75,
			Confidence:	0.8,
			Factors:	[]string{"consistent_schedule", "good_sleep"},
		},
		{
			TimeFrame:	"weekly",
			Trend:		"increasing",
			Value:		0.82,
			Confidence:	0.85,
			Factors:	[]string{"weekend_planning", "monday_motivation"},
		},
	}, nil
}

func (s *LearningService) analyzeCompletionPatterns(ctx context.Context, userID int64) ([]CompletionPattern, error) {
	return []CompletionPattern{
		{
			Pattern:	"morning_burst",
			Description:	"Высокая продуктивность в утренние часы",
			Frequency:	23,
			Accuracy:	0.87,
			Context: map[string]interface{}{
				"time_range":	"07:00-10:00",
				"task_types":	[]string{"creative", "analytical"},
			},
		},
		{
			Pattern:	"deadline_pressure",
			Description:	"Повышенная активность перед дедлайнами",
			Frequency:	12,
			Accuracy:	0.92,
			Context: map[string]interface{}{
				"days_before_deadline":	2,
				"completion_rate":	0.95,
			},
		},
	}, nil
}

func (s *LearningService) analyzeTimePreferences(ctx context.Context, userID int64) ([]TimePreference, error) {
	return []TimePreference{
		{
			TimeSlot:		"morning",
			ProductivityScore:	0.9,
			Preference:		"preferred",
			Confidence:		0.85,
		},
		{
			TimeSlot:		"afternoon",
			ProductivityScore:	0.6,
			Preference:		"neutral",
			Confidence:		0.7,
		},
		{
			TimeSlot:		"evening",
			ProductivityScore:	0.4,
			Preference:		"avoided",
			Confidence:		0.8,
		},
	}, nil
}

func (s *LearningService) analyzeCategoryAffinities(ctx context.Context, userID int64) ([]CategoryAffinity, error) {
	return []CategoryAffinity{
		{
			Category:		"technology",
			AffinityScore:		0.9,
			CompletionRate:		0.85,
			SatisfactionRate:	0.9,
			Recommendation:		"Отличная категория для сложных проектов",
		},
		{
			Category:		"health",
			AffinityScore:		0.7,
			CompletionRate:		0.6,
			SatisfactionRate:	0.8,
			Recommendation:		"Нужны дополнительные мотивационные триггеры",
		},
	}, nil
}

func (s *LearningService) analyzeRiskFactors(ctx context.Context, userID int64) ([]RiskFactor, error) {
	return []RiskFactor{
		{
			Factor:		"procrastination",
			RiskLevel:	0.6,
			Impact:		"medium",
			Mitigation:	[]string{"early_start_reminders", "task_breakdown"},
			Frequency:	8,
		},
		{
			Factor:		"overcommitment",
			RiskLevel:	0.4,
			Impact:		"high",
			Mitigation:	[]string{"capacity_planning", "goal_prioritization"},
			Frequency:	3,
		},
	}, nil
}

func (s *LearningService) analyzeSuccessPatterns(ctx context.Context, userID int64) ([]SuccessPattern, error) {
	return []SuccessPattern{
		{
			Pattern:	"small_wins_momentum",
			Description:	"Успех в малых задачах ведет к успеху в больших",
			SuccessRate:	0.85,
			Context: map[string]interface{}{
				"sequence_length":	3,
				"task_complexity":	"low_to_high",
			},
			Replication:	[]string{"start_with_easy_tasks", "build_momentum"},
		},
	}, nil
}

func (s *LearningService) identifyAdaptationNeeds(analysis *BehaviorAnalysis) []AdaptationNeed {
	var needs []AdaptationNeed

	for _, risk := range analysis.RiskFactors {
		if risk.RiskLevel > 0.5 {
			need := AdaptationNeed{
				Area:		"risk_mitigation",
				Need:		fmt.Sprintf("Снижение риска: %s", risk.Factor),
				Priority:	int(risk.RiskLevel * 5),
				Suggestions:	risk.Mitigation,
				Timeline:	"1-2 weeks",
			}
			needs = append(needs, need)
		}
	}

	for _, pref := range analysis.TimePreferences {
		if pref.Preference == "avoided" && pref.ProductivityScore < 0.5 {
			need := AdaptationNeed{
				Area:		"time_optimization",
				Need:		fmt.Sprintf("Избегать планирования важных задач в %s", pref.TimeSlot),
				Priority:	3,
				Suggestions:	[]string{"reschedule_tasks", "adjust_reminders"},
				Timeline:	"immediate",
			}
			needs = append(needs, need)
		}
	}

	return needs
}

func (s *LearningService) generateLearningInsights(analysis *BehaviorAnalysis) []LearningInsight {
	var insights []LearningInsight

	if len(analysis.ProductivityTrends) > 0 {
		trend := analysis.ProductivityTrends[0]
		insight := LearningInsight{
			Type:		"productivity",
			Title:		"Анализ продуктивности",
			Description:	fmt.Sprintf("Твоя продуктивность показывает %s тренд", trend.Trend),
			Confidence:	trend.Confidence,
			PatternData: map[string]interface{}{
				"trend":	trend.Trend,
				"value":	trend.Value,
			},
			Recommendations: []string{
				"Используй свои продуктивные часы для важных задач",
				"Планируй отдых в менее продуктивное время",
			},
			Priority:	4,
			CreatedAt:	time.Now(),
		}
		insights = append(insights, insight)
	}

	if len(analysis.CompletionPatterns) > 0 {
		pattern := analysis.CompletionPatterns[0]
		insight := LearningInsight{
			Type:		"completion",
			Title:		"Паттерн завершения задач",
			Description:	fmt.Sprintf("Обнаружен паттерн: %s", pattern.Description),
			Confidence:	pattern.Accuracy,
			PatternData:	pattern.Context,
			Recommendations: []string{
				"Используй этот паттерн для планирования сложных задач",
			},
			Priority:	3,
			CreatedAt:	time.Now(),
		}
		insights = append(insights, insight)
	}

	return insights
}

func (s *LearningService) extractPatternsFromInteraction(interaction map[string]interface{}) []UserBehaviorPattern {
	var patterns []UserBehaviorPattern

	if interactionType, ok := interaction["type"].(string); ok {
		switch interactionType {
		case "task_completion":
			patterns = append(patterns, s.extractCompletionPattern(interaction))
		case "goal_creation":
			patterns = append(patterns, s.extractGoalPattern(interaction))
		case "motivation_response":
			patterns = append(patterns, s.extractMotivationPattern(interaction))
		}
	}

	return patterns
}

func (s *LearningService) extractCompletionPattern(interaction map[string]interface{}) UserBehaviorPattern {
	return UserBehaviorPattern{
		PatternType:	PatternTypeCompletion,
		PatternData:	interaction,
		Frequency:	1,
		Confidence:	0.7,
		CreatedAt:	time.Now(),
		UpdatedAt:	time.Now(),
	}
}

func (s *LearningService) extractGoalPattern(interaction map[string]interface{}) UserBehaviorPattern {
	return UserBehaviorPattern{
		PatternType:	PatternTypeSuccess,
		PatternData:	interaction,
		Frequency:	1,
		Confidence:	0.6,
		CreatedAt:	time.Now(),
		UpdatedAt:	time.Now(),
	}
}

func (s *LearningService) extractMotivationPattern(interaction map[string]interface{}) UserBehaviorPattern {
	return UserBehaviorPattern{
		PatternType:	PatternTypeMotivation,
		PatternData:	interaction,
		Frequency:	1,
		Confidence:	0.8,
		CreatedAt:	time.Now(),
		UpdatedAt:	time.Now(),
	}
}

func (s *LearningService) updateBehaviorPattern(ctx context.Context, userID int64, pattern UserBehaviorPattern) error {
	pattern.UserID = userID

	query := `
		INSERT INTO user_behavior_patterns (user_id, pattern_type, pattern_data, frequency, confidence, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, pattern_type) 
		DO UPDATE SET 
			pattern_data = $3,
			frequency = user_behavior_patterns.frequency + 1,
			confidence = ($5 + user_behavior_patterns.confidence) / 2,
			updated_at = $7
	`

	patternDataJSON, err := json.Marshal(pattern.PatternData)
	if err != nil {
		return fmt.Errorf("failed to marshal pattern data: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query,
		pattern.UserID,
		pattern.PatternType,
		patternDataJSON,
		pattern.Frequency,
		pattern.Confidence,
		pattern.CreatedAt,
		pattern.UpdatedAt,
	)

	return err
}

func (s *LearningService) getUserBehaviorPatterns(ctx context.Context, userID int64) ([]UserBehaviorPattern, error) {
	var patterns []UserBehaviorPattern

	query := `
		SELECT id, user_id, pattern_type, pattern_data, frequency, confidence, created_at, updated_at
		FROM user_behavior_patterns
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`

	err := s.db.SelectContext(ctx, &patterns, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user behavior patterns: %w", err)
	}

	return patterns, nil
}

func (s *LearningService) adaptForProductivity(patterns []UserBehaviorPattern) map[string]interface{} {
	adaptations := make(map[string]interface{})

	for _, pattern := range patterns {
		if pattern.PatternType == PatternTypeProductivity {
			adaptations["optimal_schedule"] = "morning_focused"
			adaptations["task_complexity"] = "high_morning_low_evening"
			adaptations["break_intervals"] = 45
		}
	}

	return adaptations
}

func (s *LearningService) adaptMotivationStrategies(patterns []UserBehaviorPattern) map[string]interface{} {
	adaptations := make(map[string]interface{})

	for _, pattern := range patterns {
		if pattern.PatternType == PatternTypeMotivation {
			adaptations["preferred_motivation"] = "achievement_based"
			adaptations["tone"] = "encouraging"
			adaptations["frequency"] = "moderate"
		}
	}

	return adaptations
}

func (s *LearningService) adaptCommunicationStyle(patterns []UserBehaviorPattern) map[string]interface{} {
	adaptations := make(map[string]interface{})

	adaptations["tone"] = "friendly"
	adaptations["formality"] = "casual"
	adaptations["emoji_usage"] = "moderate"
	adaptations["message_length"] = "medium"

	return adaptations
}

func (s *LearningService) adaptTimeRecommendations(patterns []UserBehaviorPattern) map[string]interface{} {
	adaptations := make(map[string]interface{})

	for _, pattern := range patterns {
		if pattern.PatternType == PatternTypeTime {
			adaptations["optimal_work_hours"] = []string{"09:00-11:00", "14:00-16:00"}
			adaptations["preferred_break_time"] = "15_minutes"
			adaptations["deadline_buffer"] = "2_days"
		}
	}

	return adaptations
}

func (s *LearningService) adaptGoalSuggestions(patterns []UserBehaviorPattern) map[string]interface{} {
	adaptations := make(map[string]interface{})

	for _, pattern := range patterns {
		if pattern.PatternType == PatternTypeSuccess {
			adaptations["preferred_goal_size"] = "medium"
			adaptations["preferred_categories"] = []string{"technology", "productivity"}
			adaptations["optimal_timeline"] = "2_weeks"
			adaptations["complexity_preference"] = "moderate"
		}
	}

	return adaptations
}

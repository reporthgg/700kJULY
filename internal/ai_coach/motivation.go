package ai_coach

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type MotivationService struct {
	db *sqlx.DB
}

type MotivationStrategy struct {
	ID			int64			`db:"id" json:"id"`
	UserID			int64			`db:"user_id" json:"user_id"`
	StrategyType		string			`db:"strategy_type" json:"strategy_type"`
	StrategyData		map[string]interface{}	`db:"strategy_data" json:"strategy_data"`
	EffectivenessScore	float64			`db:"effectiveness_score" json:"effectiveness_score"`
	UsageCount		int			`db:"usage_count" json:"usage_count"`
	LastUsed		*time.Time		`db:"last_used" json:"last_used,omitempty"`
	CreatedAt		time.Time		`db:"created_at" json:"created_at"`
}

type MotivationMessage struct {
	Type		string			`json:"type"`
	Message		string			`json:"message"`
	Tone		string			`json:"tone"`
	Context		map[string]interface{}	`json:"context"`
	CallToAction	string			`json:"call_to_action,omitempty"`
	PersonalTouch	string			`json:"personal_touch,omitempty"`
	Encouragement	string			`json:"encouragement,omitempty"`
	SuccessStory	string			`json:"success_story,omitempty"`
	Challenge	string			`json:"challenge,omitempty"`
	Reward		string			`json:"reward,omitempty"`
	Visualization	string			`json:"visualization,omitempty"`
	Quote		string			`json:"quote,omitempty"`
	Emoji		string			`json:"emoji,omitempty"`
	Priority	int			`json:"priority"`
	StrategyUsed	string			`json:"strategy_used"`
	Confidence	float64			`json:"confidence"`
}

type MotivationTrigger struct {
	Type		string			`json:"type"`
	Condition	string			`json:"condition"`
	Parameters	map[string]interface{}	`json:"parameters"`
	Action		string			`json:"action"`
	Message		string			`json:"message"`
	Frequency	string			`json:"frequency"`
	Enabled		bool			`json:"enabled"`
}

type MotivationProfile struct {
	UserID			int64			`json:"user_id"`
	PrimaryMotivators	[]string		`json:"primary_motivators"`
	SecondaryMotivators	[]string		`json:"secondary_motivators"`
	MotivationTriggers	[]MotivationTrigger	`json:"motivation_triggers"`
	PreferredTones		[]string		`json:"preferred_tones"`
	AvoidedTones		[]string		`json:"avoided_tones"`
	EffectiveStrategies	map[string]float64	`json:"effective_strategies"`
	PersonalChallenges	[]string		`json:"personal_challenges"`
	SuccessPatterns		map[string]interface{}	`json:"success_patterns"`
	MotivationSchedule	map[string]interface{}	`json:"motivation_schedule"`
	LastMotivationUpdate	time.Time		`json:"last_motivation_update"`
}

type MotivationContext struct {
	CurrentGoal		string			`json:"current_goal"`
	ProgressLevel		float64			`json:"progress_level"`
	TimeToDeadline		int			`json:"time_to_deadline"`
	RecentActivity		string			`json:"recent_activity"`
	MoodState		string			`json:"mood_state"`
	EnergyLevel		int			`json:"energy_level"`
	MotivationLevel		float64			`json:"motivation_level"`
	StressLevel		int			`json:"stress_level"`
	LastSuccess		*time.Time		`json:"last_success,omitempty"`
	ConsecutiveFailures	int			`json:"consecutive_failures"`
	PersonalFactors		map[string]interface{}	`json:"personal_factors"`
	EnvironmentalFactors	map[string]interface{}	`json:"environmental_factors"`
}

const (
	MotivationTypeAchievement	= "achievement"
	MotivationTypeChallenge		= "challenge"
	MotivationTypeSocial		= "social"
	MotivationTypeReward		= "reward"
	MotivationTypeGrowth		= "growth"
	MotivationTypeFear		= "fear"
	MotivationTypeProgress		= "progress"
	MotivationTypeComparison	= "comparison"
	MotivationTypeVisualization	= "visualization"
	MotivationTypeStorytelling	= "storytelling"
)

const (
	ToneEncouraging		= "encouraging"
	ToneChallenging		= "challenging"
	ToneSupportive		= "supportive"
	ToneMotivating		= "motivating"
	ToneInspiring		= "inspiring"
	ToneEnergetic		= "energetic"
	ToneCalm		= "calm"
	ToneUrgent		= "urgent"
	ToneFriendly		= "friendly"
	ToneProfessional	= "professional"
)

func NewMotivationService(db *sqlx.DB) *MotivationService {
	return &MotivationService{db: db}
}

func (s *MotivationService) GeneratePersonalizedMotivation(personality *PersonalityProfile, context map[string]interface{}, productivity *ProductivityMetrics) string {
	motivationCtx := s.buildMotivationContext(context, productivity)
	profile := s.getMotivationProfile(personality.UserID)

	strategy := s.selectOptimalStrategy(profile, motivationCtx, personality)

	message := s.generateMotivationMessage(strategy, motivationCtx, personality)

	return s.formatFinalMessage(message, personality)
}

func (s *MotivationService) RecordMotivationUsage(ctx context.Context, userID int64, motivation string) error {

	query := `
		INSERT INTO motivation_strategies (user_id, strategy_type, strategy_data, usage_count, last_used, created_at)
		VALUES ($1, $2, $3, 1, $4, $5)
		ON CONFLICT (user_id, strategy_type) 
		DO UPDATE SET 
			usage_count = motivation_strategies.usage_count + 1,
			last_used = $4
	`

	strategyData := map[string]interface{}{
		"message":	motivation,
		"timestamp":	time.Now(),
	}

	dataJSON, _ := json.Marshal(strategyData)

	_, err := s.db.ExecContext(ctx, query, userID, "generated", string(dataJSON), time.Now(), time.Now())
	return err
}

func (s *MotivationService) UpdateMotivationEffectiveness(ctx context.Context, userID int64, strategyType string, effectiveness float64) error {
	query := `
		UPDATE motivation_strategies 
		SET effectiveness_score = (effectiveness_score * usage_count + $1) / (usage_count + 1)
		WHERE user_id = $2 AND strategy_type = $3
	`

	_, err := s.db.ExecContext(ctx, query, effectiveness, userID, strategyType)
	return err
}

func (s *MotivationService) GenerateMotivationPlan(ctx context.Context, userID int64, goals []interface{}) (map[string]interface{}, error) {
	profile := s.getMotivationProfile(userID)

	weeklyPlan := map[string]interface{}{
		"daily_motivations":		s.generateDailyMotivations(profile, goals),
		"milestone_celebrations":	s.planMilestoneCelebrations(goals),
		"challenge_boosts":		s.planChallengeBoosts(profile, goals),
		"reward_schedule":		s.createRewardSchedule(profile, goals),
		"motivation_triggers":		s.setupMotivationTriggers(profile),
	}

	return weeklyPlan, nil
}

func (s *MotivationService) AnalyzeMotivationPatterns(ctx context.Context, userID int64) (map[string]interface{}, error) {

	strategies, err := s.getUserMotivationStrategies(ctx, userID)
	if err != nil {
		return nil, err
	}

	effectiveness := s.analyzeStrategyEffectiveness(strategies)

	patterns := s.identifyMotivationPatterns(strategies)

	recommendations := s.generateMotivationRecommendations(effectiveness, patterns)

	analysis := map[string]interface{}{
		"effectiveness_scores":	effectiveness,
		"patterns":		patterns,
		"recommendations":	recommendations,
		"optimal_timing":	s.analyzeOptimalTiming(strategies),
		"preferred_tones":	s.analyzePreferredTones(strategies),
		"success_factors":	s.identifySuccessFactors(strategies),
	}

	return analysis, nil
}

func (s *MotivationService) buildMotivationContext(context map[string]interface{}, productivity *ProductivityMetrics) *MotivationContext {
	motivationCtx := &MotivationContext{
		PersonalFactors:	make(map[string]interface{}),
		EnvironmentalFactors:	make(map[string]interface{}),
	}

	if activityCtx, ok := context["activity"]; ok {
		if actMap, ok := activityCtx.(map[string]interface{}); ok {
			if progress, ok := actMap["productivity_level"].(float64); ok {
				motivationCtx.ProgressLevel = progress
			}
			if goals, ok := actMap["current_goals_focus"].([]string); ok && len(goals) > 0 {
				motivationCtx.CurrentGoal = goals[0]
			}
		}
	}

	if moodCtx, ok := context["mood"]; ok {
		if moodMap, ok := moodCtx.(map[string]interface{}); ok {
			if mood, ok := moodMap["current_mood"].(int); ok {
				motivationCtx.MoodState = s.moodToString(mood)
			}
			if energy, ok := moodMap["energy_level"].(int); ok {
				motivationCtx.EnergyLevel = energy
			}
			if motivation, ok := moodMap["motivation_level"].(float64); ok {
				motivationCtx.MotivationLevel = motivation
			}
			if stress, ok := moodMap["stress_level"].(int); ok {
				motivationCtx.StressLevel = stress
			}
		}
	}

	if productivity != nil {
		motivationCtx.PersonalFactors["completion_rate"] = productivity.CompletionRate
		motivationCtx.PersonalFactors["streak_days"] = productivity.StreakDays
		motivationCtx.PersonalFactors["level"] = productivity.Level
		motivationCtx.PersonalFactors["recent_achievements"] = len(productivity.RecentAchievements)
	}

	if timeCtx, ok := context["time"]; ok {
		motivationCtx.EnvironmentalFactors["time_context"] = timeCtx
	}

	return motivationCtx
}

func (s *MotivationService) getMotivationProfile(userID int64) *MotivationProfile {

	return &MotivationProfile{
		UserID:			userID,
		PrimaryMotivators:	[]string{MotivationTypeAchievement, MotivationTypeProgress},
		SecondaryMotivators:	[]string{MotivationTypeChallenge, MotivationTypeReward},
		PreferredTones:		[]string{ToneEncouraging, ToneMotivating},
		AvoidedTones:		[]string{ToneUrgent},
		EffectiveStrategies: map[string]float64{
			MotivationTypeAchievement:	0.85,
			MotivationTypeProgress:		0.78,
			MotivationTypeChallenge:	0.72,
		},
		PersonalChallenges:	[]string{"procrastination", "perfectionism"},
		SuccessPatterns: map[string]interface{}{
			"best_time":		"morning",
			"effective_duration":	45,
			"preferred_difficulty":	3,
		},
		LastMotivationUpdate:	time.Now(),
	}
}

func (s *MotivationService) selectOptimalStrategy(profile *MotivationProfile, motivationCtx *MotivationContext, personality *PersonalityProfile) string {

	if motivationCtx.MotivationLevel < 0.3 {
		return s.getBestStrategy(profile.EffectiveStrategies)
	}

	if motivationCtx.StressLevel > 3 {
		return MotivationTypeGrowth
	}

	if motivationCtx.TimeToDeadline <= 3 {
		return MotivationTypeChallenge
	}

	if motivationCtx.ProgressLevel > 0.7 {
		return MotivationTypeReward
	}

	if motivationCtx.ConsecutiveFailures > 2 {
		return MotivationTypeGrowth
	}

	if len(profile.PrimaryMotivators) > 0 {
		return profile.PrimaryMotivators[0]
	}

	return MotivationTypeAchievement
}

func (s *MotivationService) generateMotivationMessage(strategy string, motivationCtx *MotivationContext, personality *PersonalityProfile) *MotivationMessage {
	message := &MotivationMessage{
		Type:	strategy,
		Context: map[string]interface{}{
			"progress":	motivationCtx.ProgressLevel,
			"mood":		motivationCtx.MoodState,
			"energy":	motivationCtx.EnergyLevel,
		},
		StrategyUsed:	strategy,
		Confidence:	0.8,
		Priority:	3,
	}

	switch strategy {
	case MotivationTypeAchievement:
		message = s.generateAchievementMotivation(message, motivationCtx, personality)
	case MotivationTypeChallenge:
		message = s.generateChallengeMotivation(message, motivationCtx, personality)
	case MotivationTypeSocial:
		message = s.generateSocialMotivation(message, motivationCtx, personality)
	case MotivationTypeReward:
		message = s.generateRewardMotivation(message, motivationCtx, personality)
	case MotivationTypeGrowth:
		message = s.generateGrowthMotivation(message, motivationCtx, personality)
	case MotivationTypeProgress:
		message = s.generateProgressMotivation(message, motivationCtx, personality)
	case MotivationTypeVisualization:
		message = s.generateVisualizationMotivation(message, motivationCtx, personality)
	case MotivationTypeStorytelling:
		message = s.generateStorytellingMotivation(message, motivationCtx, personality)
	default:
		message = s.generateDefaultMotivation(message, motivationCtx, personality)
	}

	message = s.addPersonalTouches(message, personality)

	return message
}

func (s *MotivationService) generateAchievementMotivation(message *MotivationMessage, ctx *MotivationContext, personality *PersonalityProfile) *MotivationMessage {
	achievementMessages := []string{
		"Каждый шаг приближает тебя к цели! 🎯",
		"Ты уже прошел {progress}% пути. Продолжай в том же духе! 💪",
		"Твой прогресс впечатляет! Еще немного и цель будет достигнута! 🌟",
		"Каждое достижение делает тебя сильнее. Не останавливайся! 🚀",
		"Ты на правильном пути к успеху! Продолжай двигаться вперед! ⭐",
	}

	message.Message = s.selectRandomMessage(achievementMessages)
	message.Message = s.insertVariables(message.Message, map[string]interface{}{
		"progress": int(ctx.ProgressLevel * 100),
	})

	message.Tone = ToneMotivating
	message.CallToAction = "Сделай следующий шаг к своей цели прямо сейчас!"
	message.Encouragement = "Ты можешь достичь всего, что задумал!"
	message.Emoji = "🏆"

	quotes := []string{
		"Успех - это не конечная точка, а путь к ней. - Артур Эш",
		"Великие дела совершаются не силой, а упорством. - Сэмюэль Джонсон",
		"Единственная невозможная мечта - та, которую не пытаются осуществить. - Джо Димаджио",
	}
	message.Quote = s.selectRandomMessage(quotes)

	return message
}

func (s *MotivationService) generateChallengeMotivation(message *MotivationMessage, ctx *MotivationContext, personality *PersonalityProfile) *MotivationMessage {
	challengeMessages := []string{
		"Готов к новому вызову? Покажи, на что способен! 🔥",
		"Каждый вызов - это возможность стать лучше! 💎",
		"Сложности только закаляют характер. Ты справишься! ⚡",
		"Время проверить свои границы! Вперед, к новым вершинам! 🏔️",
		"Этот вызов создан специально для тебя. Принимаешь? 🎲",
	}

	message.Message = s.selectRandomMessage(challengeMessages)
	message.Tone = ToneChallenging
	message.CallToAction = "Принимай вызов и покажи свою силу!"
	message.Challenge = "Попробуй увеличить свою продуктивность на 20% сегодня!"
	message.Emoji = "🔥"

	return message
}

func (s *MotivationService) generateSocialMotivation(message *MotivationMessage, ctx *MotivationContext, personality *PersonalityProfile) *MotivationMessage {
	socialMessages := []string{
		"Твои друзья гордятся твоими достижениями! 👥",
		"Ты можешь стать примером для других! 🌟",
		"Представь, как будут восхищаться твоими результатами! 👏",
		"Твой успех вдохновляет окружающих! 💫",
		"Время показать всем, на что ты способен! 🎭",
	}

	message.Message = s.selectRandomMessage(socialMessages)
	message.Tone = ToneInspiring
	message.CallToAction = "Поделись своим прогрессом с друзьями!"
	message.PersonalTouch = "Твоя команда верит в тебя!"
	message.Emoji = "👥"

	return message
}

func (s *MotivationService) generateRewardMotivation(message *MotivationMessage, ctx *MotivationContext, personality *PersonalityProfile) *MotivationMessage {
	rewardMessages := []string{
		"После выполнения задачи ты заслужишь награду! 🎁",
		"Каждый шаг приближает тебя к заслуженной награде! 🏆",
		"Твои усилия точно окупятся! Продолжай! 💰",
		"Впереди ждет что-то особенное! Не останавливайся! 🎉",
		"Эта цель стоит всех твоих усилий! 💎",
	}

	message.Message = s.selectRandomMessage(rewardMessages)
	message.Tone = ToneEncouraging
	message.CallToAction = "Заверши задачу и получи заслуженную награду!"
	message.Reward = "Побалуй себя чем-то приятным после завершения!"
	message.Emoji = "🎁"

	return message
}

func (s *MotivationService) generateGrowthMotivation(message *MotivationMessage, ctx *MotivationContext, personality *PersonalityProfile) *MotivationMessage {
	growthMessages := []string{
		"Каждый день ты становишься лучше! 📈",
		"Твое развитие не знает границ! 🌱",
		"Ошибки - это ступени к мастерству! 🎯",
		"Ты растешь над собой с каждым шагом! 🚀",
		"Процесс обучения никогда не заканчивается! 📚",
	}

	message.Message = s.selectRandomMessage(growthMessages)
	message.Tone = ToneInspiring
	message.CallToAction = "Продолжай расти и развиваться!"
	message.Encouragement = "Твой потенциал безграничен!"
	message.Emoji = "🌱"

	return message
}

func (s *MotivationService) generateProgressMotivation(message *MotivationMessage, ctx *MotivationContext, personality *PersonalityProfile) *MotivationMessage {
	progressMessages := []string{
		"Посмотри, как далеко ты уже продвинулся! 📊",
		"Твой прогресс говорит сам за себя! 📈",
		"Каждый процент прогресса - это победа! 🎯",
		"Ты движешься в правильном направлении! 🧭",
		"Прогресс может быть медленным, но он есть! ⏳",
	}

	message.Message = s.selectRandomMessage(progressMessages)
	message.Tone = ToneSupportive
	message.CallToAction = "Продолжай двигаться вперед шаг за шагом!"
	message.Visualization = fmt.Sprintf("Представь: ты уже на %d%% пути к цели!", int(ctx.ProgressLevel*100))
	message.Emoji = "📊"

	return message
}

func (s *MotivationService) generateVisualizationMotivation(message *MotivationMessage, ctx *MotivationContext, personality *PersonalityProfile) *MotivationMessage {
	visualizationMessages := []string{
		"Закрой глаза и представь момент достижения цели! 🎭",
		"Визуализируй свой успех - это уже половина пути! 🌟",
		"Представь, как здорово будет достичь этой цели! 🎨",
		"Твое воображение - мощный инструмент мотивации! 🎪",
		"Визуализация успеха делает его реальным! 🔮",
	}

	message.Message = s.selectRandomMessage(visualizationMessages)
	message.Tone = ToneInspiring
	message.CallToAction = "Потрать 2 минуты на визуализацию своего успеха!"
	message.Visualization = "Представь себя через месяц, когда цель будет достигнута. Какие эмоции ты испытываешь?"
	message.Emoji = "🎭"

	return message
}

func (s *MotivationService) generateStorytellingMotivation(message *MotivationMessage, ctx *MotivationContext, personality *PersonalityProfile) *MotivationMessage {
	stories := []string{
		"Когда-то был человек, который тоже сомневался в себе. Но он не сдался и достиг невероятных высот! 📖",
		"История помнит тех, кто не боялся делать следующий шаг, даже когда было трудно! 📚",
		"Каждая великая история начинается с первого шага. Твоя история только начинается! ✨",
		"В каждом успешном человеке есть глава о том, как он преодолел трудности! 📝",
	}

	message.Message = s.selectRandomMessage(stories)
	message.Tone = ToneInspiring
	message.CallToAction = "Пиши свою историю успеха!"
	message.SuccessStory = "Вспомни свой последний успех - ты уже доказал, что можешь достигать целей!"
	message.Emoji = "📖"

	return message
}

func (s *MotivationService) generateDefaultMotivation(message *MotivationMessage, ctx *MotivationContext, personality *PersonalityProfile) *MotivationMessage {
	defaultMessages := []string{
		"Ты на правильном пути! Продолжай двигаться вперед! 🌟",
		"Каждый шаг приближает тебя к цели! 🚀",
		"Верь в себя и свои возможности! 💪",
		"Сегодня отличный день для достижений! ☀️",
		"Ты способен на большее, чем думаешь! ⭐",
	}

	message.Message = s.selectRandomMessage(defaultMessages)
	message.Tone = ToneEncouraging
	message.CallToAction = "Сделай что-то важное для своей цели прямо сейчас!"
	message.Emoji = "🌟"

	return message
}

func (s *MotivationService) addPersonalTouches(message *MotivationMessage, personality *PersonalityProfile) *MotivationMessage {

	switch personality.CommunicationStyle {
	case "friendly":
		message.Message = "Привет! " + message.Message
	case "formal":
		message.Message = strings.ReplaceAll(message.Message, "ты", "вы")
		message.Message = strings.ReplaceAll(message.Message, "твой", "ваш")
	case "casual":
		message.Message = message.Message + " 😎"
	case "encouraging":
		message.Message = "Я верю в тебя! " + message.Message
	}

	if personality.MotivationStyle == "achievement" {
		message.Priority = 4
	} else if personality.MotivationStyle == "social" {
		message.PersonalTouch = "Твоя команда поддерживает тебя!"
	}

	return message
}

func (s *MotivationService) formatFinalMessage(message *MotivationMessage, personality *PersonalityProfile) string {
	var finalMessage strings.Builder

	finalMessage.WriteString(message.Message)

	if message.PersonalTouch != "" {
		finalMessage.WriteString("\n\n" + message.PersonalTouch)
	}

	if message.Encouragement != "" {
		finalMessage.WriteString("\n\n" + message.Encouragement)
	}

	if message.Visualization != "" {
		finalMessage.WriteString("\n\n💭 " + message.Visualization)
	}

	if message.Challenge != "" {
		finalMessage.WriteString("\n\n🎯 Вызов: " + message.Challenge)
	}

	if message.Reward != "" {
		finalMessage.WriteString("\n\n🎁 " + message.Reward)
	}

	if message.SuccessStory != "" {
		finalMessage.WriteString("\n\n📖 " + message.SuccessStory)
	}

	if message.Quote != "" {
		finalMessage.WriteString("\n\n💬 " + message.Quote)
	}

	if message.CallToAction != "" {
		finalMessage.WriteString("\n\n" + message.CallToAction)
	}

	return finalMessage.String()
}

func (s *MotivationService) selectRandomMessage(messages []string) string {
	if len(messages) == 0 {
		return "Продолжай в том же духе!"
	}
	return messages[rand.Intn(len(messages))]
}

func (s *MotivationService) insertVariables(message string, variables map[string]interface{}) string {
	result := message
	for key, value := range variables {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

func (s *MotivationService) getBestStrategy(strategies map[string]float64) string {
	bestStrategy := ""
	bestScore := 0.0

	for strategy, score := range strategies {
		if score > bestScore {
			bestScore = score
			bestStrategy = strategy
		}
	}

	if bestStrategy == "" {
		return MotivationTypeAchievement
	}

	return bestStrategy
}

func (s *MotivationService) moodToString(mood int) string {
	switch {
	case mood >= 4:
		return "excellent"
	case mood >= 3:
		return "good"
	case mood >= 2:
		return "neutral"
	default:
		return "low"
	}
}

func (s *MotivationService) generateDailyMotivations(profile *MotivationProfile, goals []interface{}) map[string]interface{} {
	dailyMotivations := make(map[string]interface{})

	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

	for _, day := range days {
		dailyMotivations[day] = map[string]interface{}{
			"morning_motivation":	s.generateTimeSpecificMotivation("morning", profile),
			"midday_boost":		s.generateTimeSpecificMotivation("midday", profile),
			"evening_reflection":	s.generateTimeSpecificMotivation("evening", profile),
		}
	}

	return dailyMotivations
}

func (s *MotivationService) generateTimeSpecificMotivation(timeOfDay string, profile *MotivationProfile) string {
	switch timeOfDay {
	case "morning":
		return "Доброе утро! Сегодня отличный день для достижений! 🌅"
	case "midday":
		return "Полдень - время для проверки прогресса! Как дела? 🕐"
	case "evening":
		return "Вечер - время подвести итоги дня. Чем можешь гордиться? 🌆"
	default:
		return "Продолжай в том же духе! 💪"
	}
}

func (s *MotivationService) planMilestoneCelebrations(goals []interface{}) []map[string]interface{} {
	var celebrations []map[string]interface{}

	for _, goal := range goals {
		celebration := map[string]interface{}{
			"goal":			goal,
			"celebration_type":	"milestone",
			"message":		"🎉 Поздравляю с достижением важного этапа!",
			"reward_suggestion":	"Побалуй себя чем-то приятным!",
		}
		celebrations = append(celebrations, celebration)
	}

	return celebrations
}

func (s *MotivationService) planChallengeBoosts(profile *MotivationProfile, goals []interface{}) []map[string]interface{} {
	var challenges []map[string]interface{}

	challenge := map[string]interface{}{
		"type":		"productivity_challenge",
		"title":	"Вызов продуктивности",
		"description":	"Попробуй увеличить свою продуктивность на 25% на этой неделе!",
		"duration":	"1 week",
		"reward":	"Особое достижение и дополнительные очки!",
	}

	challenges = append(challenges, challenge)

	return challenges
}

func (s *MotivationService) createRewardSchedule(profile *MotivationProfile, goals []interface{}) map[string]interface{} {
	return map[string]interface{}{
		"daily_rewards": []string{
			"Чашка любимого кофе",
			"15 минут любимой музыки",
			"Короткая прогулка на свежем воздухе",
		},
		"weekly_rewards": []string{
			"Вечер любимого фильма",
			"Ужин в приятном месте",
			"Покупка чего-то желанного",
		},
		"milestone_rewards": []string{
			"Выходные в новом месте",
			"Покупка большой мечты",
			"Празднование с друзьями",
		},
	}
}

func (s *MotivationService) setupMotivationTriggers(profile *MotivationProfile) []MotivationTrigger {
	triggers := []MotivationTrigger{
		{
			Type:		"low_progress",
			Condition:	"progress < 0.3",
			Action:		"send_motivation",
			Message:	"Не сдавайся! Каждый шаг важен! 💪",
			Frequency:	"daily",
			Enabled:	true,
		},
		{
			Type:		"deadline_approaching",
			Condition:	"days_left <= 3",
			Action:		"send_urgency_motivation",
			Message:	"Время поджимает! Сосредоточься на главном! ⏰",
			Frequency:	"daily",
			Enabled:	true,
		},
		{
			Type:		"streak_breaking",
			Condition:	"days_without_activity > 2",
			Action:		"send_comeback_motivation",
			Message:	"Время вернуться! Твоя серия ждет продолжения! 🔥",
			Frequency:	"once",
			Enabled:	true,
		},
	}

	return triggers
}

func (s *MotivationService) getUserMotivationStrategies(ctx context.Context, userID int64) ([]MotivationStrategy, error) {
	query := `
		SELECT id, user_id, strategy_type, strategy_data, effectiveness_score, usage_count, last_used, created_at
		FROM motivation_strategies
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strategies []MotivationStrategy
	for rows.Next() {
		var strategy MotivationStrategy
		var strategyDataJSON string

		err := rows.Scan(&strategy.ID, &strategy.UserID, &strategy.StrategyType,
			&strategyDataJSON, &strategy.EffectivenessScore, &strategy.UsageCount,
			&strategy.LastUsed, &strategy.CreatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(strategyDataJSON), &strategy.StrategyData)
		strategies = append(strategies, strategy)
	}

	return strategies, nil
}

func (s *MotivationService) analyzeStrategyEffectiveness(strategies []MotivationStrategy) map[string]float64 {
	effectiveness := make(map[string]float64)

	for _, strategy := range strategies {
		effectiveness[strategy.StrategyType] = strategy.EffectivenessScore
	}

	return effectiveness
}

func (s *MotivationService) identifyMotivationPatterns(strategies []MotivationStrategy) map[string]interface{} {
	patterns := make(map[string]interface{})

	timePatterns := make(map[string]int)
	for _, strategy := range strategies {
		if strategy.LastUsed != nil {
			hour := strategy.LastUsed.Hour()
			timeSlot := s.getTimeSlot(hour)
			timePatterns[timeSlot]++
		}
	}
	patterns["time_patterns"] = timePatterns

	typeEffectiveness := make(map[string]float64)
	for _, strategy := range strategies {
		typeEffectiveness[strategy.StrategyType] = strategy.EffectivenessScore
	}
	patterns["type_effectiveness"] = typeEffectiveness

	return patterns
}

func (s *MotivationService) generateMotivationRecommendations(effectiveness map[string]float64, patterns map[string]interface{}) []string {
	var recommendations []string

	bestStrategy := ""
	bestScore := 0.0
	for strategy, score := range effectiveness {
		if score > bestScore {
			bestScore = score
			bestStrategy = strategy
		}
	}

	if bestStrategy != "" {
		recommendations = append(recommendations,
			fmt.Sprintf("Используй больше стратегий типа '%s' - они наиболее эффективны для тебя", bestStrategy))
	}

	if timePatterns, ok := patterns["time_patterns"].(map[string]int); ok {
		bestTime := ""
		maxCount := 0
		for timeSlot, count := range timePatterns {
			if count > maxCount {
				maxCount = count
				bestTime = timeSlot
			}
		}

		if bestTime != "" {
			recommendations = append(recommendations,
				fmt.Sprintf("Ты наиболее отзывчив на мотивацию в %s", bestTime))
		}
	}

	return recommendations
}

func (s *MotivationService) analyzeOptimalTiming(strategies []MotivationStrategy) map[string]interface{} {
	timing := make(map[string]interface{})

	hourlyEffectiveness := make(map[int]float64)
	for _, strategy := range strategies {
		if strategy.LastUsed != nil {
			hour := strategy.LastUsed.Hour()
			hourlyEffectiveness[hour] = strategy.EffectivenessScore
		}
	}

	timing["hourly_effectiveness"] = hourlyEffectiveness

	return timing
}

func (s *MotivationService) analyzePreferredTones(strategies []MotivationStrategy) []string {

	toneScores := make(map[string]float64)

	for _, strategy := range strategies {
		if tone, ok := strategy.StrategyData["tone"].(string); ok {
			toneScores[tone] = strategy.EffectivenessScore
		}
	}

	var preferredTones []string
	for tone, score := range toneScores {
		if score > 0.7 {
			preferredTones = append(preferredTones, tone)
		}
	}

	return preferredTones
}

func (s *MotivationService) identifySuccessFactors(strategies []MotivationStrategy) []string {
	factors := []string{}

	highEffectiveStrategies := []MotivationStrategy{}
	for _, strategy := range strategies {
		if strategy.EffectivenessScore > 0.8 {
			highEffectiveStrategies = append(highEffectiveStrategies, strategy)
		}
	}

	if len(highEffectiveStrategies) > 0 {
		factors = append(factors, "Персонализированный подход")
		factors = append(factors, "Правильный выбор времени")
		factors = append(factors, "Соответствие настроению")
	}

	return factors
}

func (s *MotivationService) getTimeSlot(hour int) string {
	switch {
	case hour >= 6 && hour < 12:
		return "утро"
	case hour >= 12 && hour < 18:
		return "день"
	case hour >= 18 && hour < 22:
		return "вечер"
	default:
		return "ночь"
	}
}

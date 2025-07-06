package chatgpt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"telegrambot/internal/ai_coach"
	"telegrambot/internal/messagestore/models"
	"telegrambot/pkg/config"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

type ChatGPTService struct {
	client	*openai.Client
	aiCoach	*ai_coach.AICoachService
	db	*sqlx.DB
}

type ChatGPTFunctionCall struct {
	Name		string			`json:"name"`
	Arguments	map[string]interface{}	`json:"arguments"`
}

type ChatGPTFunction struct {
	Name		string				`json:"name"`
	Description	string				`json:"description"`
	Parameters	ChatGPTFunctionParameters	`json:"parameters"`
}

type ChatGPTFunctionParameters struct {
	Type		string				`json:"type"`
	Properties	map[string]ChatGPTProperty	`json:"properties"`
	Required	[]string			`json:"required"`
}

type ChatGPTProperty struct {
	Type		string				`json:"type"`
	Description	string				`json:"description"`
	Enum		[]string			`json:"enum,omitempty"`
	Items		*ChatGPTProperty		`json:"items,omitempty"`
	Properties	map[string]ChatGPTProperty	`json:"properties,omitempty"`
	Minimum		interface{}			`json:"minimum,omitempty"`
	Maximum		interface{}			`json:"maximum,omitempty"`
}

func NewChatGPTService(cfg *config.Config, db *sqlx.DB) *ChatGPTService {
	client := openai.NewClient(cfg.OpenAIKey)
	aiCoach := ai_coach.NewAICoachService(db)

	return &ChatGPTService{
		client:		client,
		aiCoach:	aiCoach,
		db:		db,
	}
}

func (c *ChatGPTService) ProcessMessage(ctx context.Context, userID int64, message string, history []models.MessageHistoryItem) (string, error) {
	logrus.Infof("Обработка сообщения от пользователя %d через Jarvis", userID)

	userContext, err := c.aiCoach.GetCurrentContext(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить контекст пользователя: %v", err)
		userContext = map[string]interface{}{}
	}

	personality, err := c.aiCoach.GetUserPersonality(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить персональность пользователя: %v", err)
	}

	systemPrompt := c.buildJarvisSystemPrompt(userContext, personality)

	jarvisFunctions := GetAllJarvisFunctions()
	functions := c.convertToOpenAIFunctions(jarvisFunctions)

	logrus.Infof("Передаем %d функций в OpenAI для пользователя %d", len(functions), userID)
	for _, f := range functions {
		logrus.Debugf("Функция: %s - %s", f.Name, f.Description)
	}

	messages := c.buildMessages(systemPrompt, message, history)

	logrus.Infof("Отправляем запрос в OpenAI с %d сообщениями и %d функциями", len(messages), len(functions))

	response, functionCall, err := c.sendChatCompletionRequest(ctx, messages, functions)
	if err != nil {
		return "", err
	}

	if functionCall != nil {
		logrus.Infof("ChatGPT вызвал функцию: %s с аргументами: %+v", functionCall.Name, functionCall.Arguments)

		result, _, err := c.handleFunctionCall(functionCall, userID)
		if err != nil {
			logrus.Errorf("Ошибка выполнения функции %s: %v", functionCall.Name, err)
			return fmt.Sprintf("Произошла ошибка при выполнении функции: %v", err), nil
		}

		logrus.Infof("Функция %s выполнена успешно для пользователя %d", functionCall.Name, userID)

		c.updateConversationContext(ctx, userID, message, functionCall.Name)

		return result, nil
	}

	logrus.Infof("ChatGPT НЕ вызвал никаких функций для сообщения: %s", message)

	c.updateConversationContext(ctx, userID, message, "chat")

	c.learnFromInteraction(ctx, userID, message, response)

	return response, nil
}

func (c *ChatGPTService) ProcessAudioMessage(ctx context.Context, userID int64, audioData []byte, history []models.MessageHistoryItem) (string, error) {

	transcription, err := c.transcribeAudio(ctx, audioData)
	if err != nil {
		return "", fmt.Errorf("ошибка транскрибации аудио: %w", err)
	}

	logrus.Infof("Транскрибированное сообщение от пользователя %d: %s", userID, transcription)

	return c.ProcessMessage(ctx, userID, transcription, history)
}

func (c *ChatGPTService) GenerateProactiveMessage(ctx context.Context, userID int64) (string, error) {

	insights, err := c.aiCoach.GenerateInsights(ctx, userID)
	if err != nil {
		return "", err
	}

	if len(insights) == 0 {
		return "", nil
	}

	var topInsight *ai_coach.AIInsight
	for i := range insights {
		if topInsight == nil || insights[i].Priority > topInsight.Priority {
			topInsight = &insights[i]
		}
	}

	if topInsight == nil {
		return "", nil
	}

	personality, err := c.aiCoach.GetUserPersonality(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить персональность: %v", err)
	}

	if personality != nil {
		message := c.aiCoach.GeneratePersonalizedMessage(personality, topInsight.InsightType, map[string]interface{}{
			"insight": topInsight,
		})

		if message == "" {
			message = topInsight.Content
		} else {
			message = message + "\n\n" + topInsight.Content
		}

		return message, nil
	}

	return topInsight.Content, nil
}

func (c *ChatGPTService) buildJarvisSystemPrompt(userContext map[string]interface{}, personality *ai_coach.PersonalityProfile) string {
	prompt := `Ты Jarvis - умный персональный ассистент по достижению целей в системе OKR. 

КРИТИЧЕСКИ ВАЖНО: Когда пользователь упоминает цели, планы, достижения - ОБЯЗАТЕЛЬНО используй функции!

ОБЯЗАТЕЛЬНЫЕ ПРАВИЛА:
1. Когда пользователь говорит о новых целях, планах, мечтах - НЕМЕДЛЕННО используй create_objective
2. Когда пользователь спрашивает про свои цели - ВСЕГДА используй get_objectives
3. Когда говорит о конкретных результатах (подписчики, видео, деньги) - это Key Results для OKR
4. ВСЕГДА создавай структурированные OKR с конкретными измеримыми результатами
5. НЕ спрашивай разрешения - ДЕЙСТВУЙ НЕМЕДЛЕННО!

КОГДА ИСПОЛЬЗОВАТЬ ФУНКЦИИ:
❗ create_objective: "хочу стать...", "планирую...", "моя цель...", "достичь...", упоминания планов/мечт
❗ get_objectives: "мои цели", "что у меня", "покажи цели", "какие цели"
❗ add_key_result_progress: "сделал", "выполнил", упоминания прогресса

СТРУКТУРА OKR:
- Objective: амбициозная качественная цель
- Key Results: 2-5 конкретных измеримых результатов
- Tasks: конкретные действия для достижения

ПРИМЕРЫ ЦЕЛЕЙ И KEY RESULTS:
• "Стать популярным блогером" → KR: "1,000,000 подписчиков", "1000 видео"
• "Похудеть" → KR: "Сбросить 10 кг", "Бегать 5 км за 25 мин"
• "Изучить программирование" → KR: "Создать 5 проектов", "Получить сертификат"

ДОСТУПНЫЕ ФУНКЦИИ:
- create_objective: создание новых целей OKR
- get_objectives: получение списка целей  
- create_key_result: добавление ключевых результатов
- add_key_result_progress: обновление прогресса
- analyze_productivity: анализ продуктивности
- generate_motivation: создание мотивации`

	if userContext != nil {
		if moodCtx, ok := userContext["mood"]; ok {
			prompt += "\n\nТЕКУЩЕЕ НАСТРОЕНИЕ ПОЛЬЗОВАТЕЛЯ:\n" + fmt.Sprintf("%v", moodCtx)
		}

		if activityCtx, ok := userContext["activity"]; ok {
			prompt += "\n\nТЕКУЩАЯ АКТИВНОСТЬ:\n" + fmt.Sprintf("%v", activityCtx)
		}

		if timeCtx, ok := userContext["time"]; ok {
			prompt += "\n\nВРЕМЕННОЙ КОНТЕКСТ:\n" + fmt.Sprintf("%v", timeCtx)
		}
	}

	if personality != nil {
		prompt += fmt.Sprintf("\n\nПЕРСОНАЛЬНОСТЬ ПОЛЬЗОВАТЕЛЯ:\n- Тип: %s\n- Стиль мотивации: %s\n- Стиль общения: %s\n- Уровень активности: %s",
			personality.PersonalityType, personality.MotivationStyle, personality.CommunicationStyle, personality.ActivityLevel)
	}

	prompt += "\n\nАДАПТИРУЙ СВОИ ОТВЕТЫ ПОД ЭТОТ КОНТЕКСТ И ПЕРСОНАЛЬНОСТЬ!"

	return prompt
}

func (c *ChatGPTService) buildMessages(systemPrompt, message string, history []models.MessageHistoryItem) []openai.ChatCompletionMessage {
	var messages []openai.ChatCompletionMessage

	messages = append(messages, openai.ChatCompletionMessage{
		Role:		openai.ChatMessageRoleSystem,
		Content:	systemPrompt,
	})

	historyLimit := 10
	startIndex := 0
	if len(history) > historyLimit {
		startIndex = len(history) - historyLimit
	}

	for i := startIndex; i < len(history); i++ {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:		history[i].Role,
			Content:	history[i].Content,
		})
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:		openai.ChatMessageRoleUser,
		Content:	message,
	})

	return messages
}

func (c *ChatGPTService) sendChatCompletionRequest(ctx context.Context, messages []openai.ChatCompletionMessage, functions []openai.FunctionDefinition) (string, *ChatGPTFunctionCall, error) {
	req := openai.ChatCompletionRequest{
		Model:		openai.GPT4Dot1,
		Messages:	messages,
		Functions:	functions,
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", nil, fmt.Errorf("ошибка запроса к OpenAI: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", nil, fmt.Errorf("нет ответа от OpenAI")
	}

	choice := resp.Choices[0]

	if choice.Message.FunctionCall != nil {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(choice.Message.FunctionCall.Arguments), &args); err != nil {
			return "", nil, fmt.Errorf("ошибка парсинга аргументов функции: %w", err)
		}

		return "", &ChatGPTFunctionCall{
			Name:		choice.Message.FunctionCall.Name,
			Arguments:	args,
		}, nil
	}

	return choice.Message.Content, nil, nil
}

func (c *ChatGPTService) handleFunctionCall(functionCall *ChatGPTFunctionCall, userID int64) (string, *ChatGPTFunction, error) {

	result, function, err := c.handleNewJarvisFunctions(functionCall, userID)
	if err == nil {
		return result, function, nil
	}

	return "", nil, fmt.Errorf("неизвестная функция: %s", functionCall.Name)
}

func (c *ChatGPTService) convertToOpenAIFunctions(jarvisFunctions []ChatGPTFunction) []openai.FunctionDefinition {
	var openAIFunctions []openai.FunctionDefinition

	for _, jf := range jarvisFunctions {
		params := make(map[string]interface{})
		params["type"] = jf.Parameters.Type
		params["properties"] = c.convertProperties(jf.Parameters.Properties)
		if len(jf.Parameters.Required) > 0 {
			params["required"] = jf.Parameters.Required
		}

		openAIFunctions = append(openAIFunctions, openai.FunctionDefinition{
			Name:		jf.Name,
			Description:	jf.Description,
			Parameters:	params,
		})
	}

	return openAIFunctions
}

func (c *ChatGPTService) convertProperties(properties map[string]ChatGPTProperty) map[string]interface{} {
	result := make(map[string]interface{})

	for key, prop := range properties {
		propMap := map[string]interface{}{
			"type":		prop.Type,
			"description":	prop.Description,
		}

		if len(prop.Enum) > 0 {
			propMap["enum"] = prop.Enum
		}

		if prop.Items != nil {
			propMap["items"] = map[string]interface{}{
				"type":		prop.Items.Type,
				"description":	prop.Items.Description,
			}
			if len(prop.Items.Enum) > 0 {
				propMap["items"].(map[string]interface{})["enum"] = prop.Items.Enum
			}
		}

		if prop.Minimum != nil {
			propMap["minimum"] = prop.Minimum
		}

		if prop.Maximum != nil {
			propMap["maximum"] = prop.Maximum
		}

		result[key] = propMap
	}

	return result
}

func (c *ChatGPTService) updateConversationContext(ctx context.Context, userID int64, message, intent string) {
	err := c.aiCoach.UpdateConversationContext(ctx, userID, message, intent)
	if err != nil {
		logrus.Warnf("Не удалось обновить контекст разговора: %v", err)
	}
}

func (c *ChatGPTService) learnFromInteraction(ctx context.Context, userID int64, userMessage, assistantResponse string) {

	behaviorData := map[string]interface{}{
		"message_length":	len(userMessage),
		"response_length":	len(assistantResponse),
		"interaction_time":	time.Now(),
		"message_type":		"text",
	}

	err := c.aiCoach.LearnFromBehavior(ctx, userID, behaviorData)
	if err != nil {
		logrus.Warnf("Не удалось обучиться на взаимодействии: %v", err)
	}
}

func (c *ChatGPTService) transcribeAudio(ctx context.Context, audioData []byte) (string, error) {

	tempFile, err := os.CreateTemp("", "audio-*.ogg")
	if err != nil {
		return "", fmt.Errorf("ошибка создания временного файла: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err = tempFile.Write(audioData); err != nil {
		return "", fmt.Errorf("ошибка записи аудиоданных: %w", err)
	}

	resp, err := c.client.CreateTranscription(
		ctx,
		openai.AudioRequest{
			Model:		openai.Whisper1,
			FilePath:	tempFile.Name(),
			Language:	"ru",
		},
	)
	if err != nil {
		return "", fmt.Errorf("ошибка при транскрибации аудио: %w", err)
	}

	return resp.Text, nil
}

func (c *ChatGPTService) AnalyzeUserProductivity(ctx context.Context, userID int64) (*ai_coach.ProductivityMetrics, error) {
	return c.aiCoach.AnalyzeProductivity(ctx, userID)
}

func (c *ChatGPTService) GenerateUserInsights(ctx context.Context, userID int64) ([]ai_coach.AIInsight, error) {
	return c.aiCoach.GenerateInsights(ctx, userID)
}

func (c *ChatGPTService) PredictGoalSuccess(ctx context.Context, userID int64, objectiveID string) (*ai_coach.CompletionPrediction, error) {
	return c.aiCoach.PredictCompletionProbability(ctx, userID, objectiveID)
}

func (c *ChatGPTService) GenerateMotivation(ctx context.Context, userID int64) (string, error) {
	return c.aiCoach.GenerateMotivation(ctx, userID)
}

func (c *ChatGPTService) CreateWeeklyPlan(ctx context.Context, userID int64) (map[string]interface{}, error) {
	return c.aiCoach.GenerateWeeklyPlan(ctx, userID)
}

func (c *ChatGPTService) CheckUserWellbeing(ctx context.Context, userID int64, stressLevel, sleepQuality, workLifeBalance int) (string, error) {

	err := c.aiCoach.UpdateMoodContext(ctx, userID, (stressLevel+sleepQuality+workLifeBalance)/3, sleepQuality)
	if err != nil {
		logrus.Warnf("Не удалось обновить контекст настроения: %v", err)
	}

	recommendations := c.generateWellbeingRecommendations(stressLevel, sleepQuality, workLifeBalance)

	return recommendations, nil
}

func (c *ChatGPTService) generateWellbeingRecommendations(stress, sleep, balance int) string {
	var recommendations []string

	if stress > 3 {
		recommendations = append(recommendations, "🧘 Рекомендую сделать перерыв и попрактиковать медитацию или дыхательные упражнения")
		recommendations = append(recommendations, "🚶 Короткая прогулка на свежем воздухе поможет снизить стресс")
	}

	if sleep < 3 {
		recommendations = append(recommendations, "😴 Стоит улучшить качество сна: соблюдай режим, избегай экранов перед сном")
		recommendations = append(recommendations, "🌙 Попробуй расслабляющие техники перед сном")
	}

	if balance < 3 {
		recommendations = append(recommendations, "⚖️ Важно найти баланс между работой и отдыхом")
		recommendations = append(recommendations, "🎯 Пересмотри приоритеты и делегируй некоторые задачи")
	}

	if len(recommendations) == 0 {
		return "🌟 Отлично! Твое самочувствие в норме. Продолжай в том же духе!"
	}

	result := "💡 **Рекомендации по самочувствию:**\n\n"
	for _, rec := range recommendations {
		result += rec + "\n\n"
	}

	return result
}

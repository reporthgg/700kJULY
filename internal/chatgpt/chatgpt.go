package chatgpt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"telegrambot/internal/messagestore/models"
	"telegrambot/pkg/config"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

type Service struct {
	client *openai.Client
}

type FunctionCall struct {
	Name		string			`json:"name"`
	Arguments	map[string]interface{}	`json:"arguments"`
}

func NewService(cfg *config.Config) *Service {
	client := openai.NewClient(cfg.OpenAIKey)
	return &Service{
		client: client,
	}
}

func (s *Service) ProcessTextMessage(ctx context.Context, message string, functions []openai.FunctionDefinition) (string, *FunctionCall, error) {
	var messages []openai.ChatCompletionMessage
	messages = append(messages, openai.ChatCompletionMessage{
		Role:		openai.ChatMessageRoleSystem,
		Content:	"Ты полезный ассистент, который может помочь пользователю с различными задачами. Сегодня " + time.Now().Format("2006-01-02") + ". ВАЖНО: При создании новых целей OKR (функция create_objective) используй ТОЛЬКО ключевые результаты, которые пользователь явно указал в текущем сообщении. НЕ используй информацию из предыдущих целей или контекста истории. Каждая цель должна содержать только те ключевые результаты, которые относятся именно к ней и были явно указаны пользователем в текущем запросе.\n\nПри работе с повторяющимися задачами следуй этим правилам:\n1. Если пользователь упоминает новую цель, которой еще нет в системе (например, 'хочу каждый день делать отжимания'), используй функцию create_objective_with_recurring_tasks.\n2. Если пользователь хочет добавить задачи к уже существующему ключевому результату, используй функцию create_recurring_tasks.",
	})
	messages = append(messages, openai.ChatCompletionMessage{
		Role:		openai.ChatMessageRoleUser,
		Content:	message,
	})

	chatReq := openai.ChatCompletionRequest{
		Model:		openai.GPT4Dot1,
		Messages:	messages,
		Functions:	functions,
	}

	resp, err := s.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		logrus.Errorf("Ошибка при запросе к OpenAI: %v", err)
		return "", nil, err
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Message.FunctionCall != nil {
		fc := resp.Choices[0].Message.FunctionCall
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(fc.Arguments), &args); err != nil {
			return "", nil, err
		}
		return "", &FunctionCall{
			Name:		fc.Name,
			Arguments:	args,
		}, nil
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil, nil
	}

	return "", nil, errors.New("нет ответа от OpenAI")
}

func (s *Service) ProcessTextMessageWithHistory(ctx context.Context, message string, history []models.MessageHistoryItem, functions []openai.FunctionDefinition) (string, *FunctionCall, error, *int, *int) {
	var messages []openai.ChatCompletionMessage

	messages = append(messages, openai.ChatCompletionMessage{
		Role:		openai.ChatMessageRoleSystem,
		Content:	"Ты полезный ассистент, который может помочь пользователю с различными задачами. Сегодня " + time.Now().Format("2006-01-02") + ". ВАЖНО: При создании новых целей OKR (функция create_objective) используй ТОЛЬКО ключевые результаты, которые пользователь явно указал в текущем сообщении или в предыдущих 3-х сообщениях. НЕ используй информацию из предыдущих целей или контекста истории. Каждая цель должна содержать только те ключевые результаты, которые относятся именно к ней и были явно указаны пользователем в текущем запросе или в предыдущих 3-х сообщениях.",
	})

	for _, item := range history {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:		item.Role,
			Content:	item.Content,
		})
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:		openai.ChatMessageRoleUser,
		Content:	message,
	})

	chatReq := openai.ChatCompletionRequest{
		Model:		openai.GPT4Dot1,
		Messages:	messages,
		Functions:	functions,
	}

	resp, err := s.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		logrus.Errorf("Ошибка при запросе к OpenAI с историей: %v", err)
		return "", nil, err, nil, nil
	}

	var promptTokens, completionTokens *int
	if resp.Usage.PromptTokens > 0 {
		pt := resp.Usage.PromptTokens
		promptTokens = &pt
	}
	if resp.Usage.CompletionTokens > 0 {
		ct := resp.Usage.CompletionTokens
		completionTokens = &ct
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Message.FunctionCall != nil {
		fc := resp.Choices[0].Message.FunctionCall
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(fc.Arguments), &args); err != nil {
			return "", nil, err, promptTokens, completionTokens
		}
		return "", &FunctionCall{
			Name:		fc.Name,
			Arguments:	args,
		}, nil, promptTokens, completionTokens
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil, nil, promptTokens, completionTokens
	}

	return "", nil, errors.New("нет ответа от OpenAI"), promptTokens, completionTokens
}

func (s *Service) ProcessAudioMessage(ctx context.Context, audioData []byte, functions []openai.FunctionDefinition) (string, *FunctionCall, error) {

	transcription, err := s.transcribeAudio(ctx, audioData)
	if err != nil {
		return "", nil, err
	}

	return s.ProcessTextMessage(ctx, transcription, functions)
}

func (s *Service) ProcessAudioMessageWithHistory(ctx context.Context, audioData []byte, history []models.MessageHistoryItem, functions []openai.FunctionDefinition) (string, *FunctionCall, error, *int, *int) {

	transcription, err := s.transcribeAudio(ctx, audioData)
	if err != nil {
		return "", nil, err, nil, nil
	}

	return s.ProcessTextMessageWithHistory(ctx, transcription, history, functions)
}

func (s *Service) transcribeAudio(ctx context.Context, audioData []byte) (string, error) {

	tempFile, err := os.CreateTemp("", "audio-*.ogg")
	if err != nil {
		return "", fmt.Errorf("ошибка создания временного файла: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err = tempFile.Write(audioData); err != nil {
		return "", fmt.Errorf("ошибка записи аудиоданных: %w", err)
	}

	resp, err := s.client.CreateTranscription(
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

func (s *Service) TranscribeAudio(ctx context.Context, audioData []byte) (string, error) {
	return s.transcribeAudio(ctx, audioData)
}

func DefineFunctions() []openai.FunctionDefinition {
	return []openai.FunctionDefinition{
		{
			Name:		"create_calendar_event",
			Description:	"Создать событие в календаре",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":		"string",
						"description":	"Название события",
					},
					"description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание события",
					},
					"start_time": map[string]interface{}{
						"type":		"string",
						"description":	"Время начала события в формате ISO 8601 (YYYY-MM-DDTHH:MM:SS)",
					},
					"end_time": map[string]interface{}{
						"type":		"string",
						"description":	"Время окончания события в формате ISO 8601 (YYYY-MM-DDTHH:MM:SS)",
					},
				},
				"required":	[]string{"title", "start_time", "end_time"},
			},
		},
		{
			Name:		"create_meeting",
			Description:	"Создать встречу с другим пользователем",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":		"string",
						"description":	"Название встречи",
					},
					"participant_username": map[string]interface{}{
						"type":		"string",
						"description":	"Имя пользователя, с которым встреча",
					},
					"description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание встречи",
					},
					"start_time": map[string]interface{}{
						"type":		"string",
						"description":	"Время начала встречи в формате ISO 8601 (YYYY-MM-DDTHH:MM:SS)",
					},
					"end_time": map[string]interface{}{
						"type":		"string",
						"description":	"Время окончания встречи в формате ISO 8601 (YYYY-MM-DDTHH:MM:SS)",
					},
				},
				"required":	[]string{"title", "participant_username", "start_time", "end_time"},
			},
		},
		{
			Name:		"add_transaction",
			Description:	"Добавить финансовую транзакцию (доход или расход)",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"amount": map[string]interface{}{
						"type":		"number",
						"description":	"Сумма транзакции (положительная для дохода, отрицательная для расхода)",
					},
					"details": map[string]interface{}{
						"type":		"string",
						"description":	"Детали транзакции, например 'продукты', 'зарплата' и т.д.",
					},
					"category": map[string]interface{}{
						"type":		"string",
						"description":	"Категория транзакции (например, 'продукты', 'транспорт', 'развлечения')",
					},
				},
				"required":	[]string{"amount", "details"},
			},
		},
		{
			Name:		"get_financial_summary",
			Description:	"Получить сводку финансов за определенный период",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"period": map[string]interface{}{
						"type":		"string",
						"description":	"Период (day, week, month, year)",
						"enum":		[]string{"day", "week", "month", "year"},
					},
				},
				"required":	[]string{"period"},
			},
		},
		{
			Name:		"create_objective",
			Description:	"Создать цель OKR",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":		"string",
						"description":	"Название цели",
					},
					"sphere": map[string]interface{}{
						"type":		"string",
						"description":	"Сфера цели (бизнес, финансы, здоровье и т.д.)",
					},
					"period": map[string]interface{}{
						"type":		"string",
						"description":	"Период (week, month, quarter, year)",
						"enum":		[]string{"week", "month", "quarter", "year"},
					},
					"deadline": map[string]interface{}{
						"type":		"string",
						"description":	"Дедлайн для цели в формате YYYY-MM-DD",
					},
					"key_results": map[string]interface{}{
						"type":		"array",
						"description":	"Ключевые результаты",
						"items": map[string]interface{}{
							"type":	"object",
							"properties": map[string]interface{}{
								"title": map[string]interface{}{
									"type":		"string",
									"description":	"Название ключевого результата",
								},
								"target": map[string]interface{}{
									"type":		"number",
									"description":	"Целевое значение",
								},
								"unit": map[string]interface{}{
									"type":		"string",
									"description":	"Единица измерения (штуки, проценты, деньги и т.д.)",
								},
								"deadline": map[string]interface{}{
									"type":		"string",
									"description":	"Дедлайн для ключевого результата в формате YYYY-MM-DD",
								},
							},
							"required":	[]string{"title", "target", "unit", "deadline"},
						},
					},
				},
				"required":	[]string{"title", "sphere", "period", "deadline", "key_results"},
			},
		},
		{
			Name:		"create_key_result",
			Description:	"Добавить ключевой результат к существующей цели",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"objective_id": map[string]interface{}{
						"type":		"string",
						"description":	"ID цели, к которой добавляется ключевой результат",
					},
					"objective_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название цели, к которой добавляется ключевой результат (используется, если ID не указан)",
					},
					"title": map[string]interface{}{
						"type":		"string",
						"description":	"Название ключевого результата",
					},
					"target": map[string]interface{}{
						"type":		"number",
						"description":	"Целевое значение",
					},
					"unit": map[string]interface{}{
						"type":		"string",
						"description":	"Единица измерения (штуки, проценты, деньги и т.д.)",
					},
					"deadline": map[string]interface{}{
						"type":		"string",
						"description":	"Дедлайн для ключевого результата в формате YYYY-MM-DD",
					},
				},
				"required":	[]string{"title", "target", "unit", "deadline"},
			},
		},
		{
			Name:		"create_task",
			Description:	"Добавить мини-задачу к ключевому результату",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"key_result_id": map[string]interface{}{
						"type":		"integer",
						"description":	"ID ключевого результата, к которому добавляется задача",
					},
					"key_result_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название ключевого результата (используется, если ID не указан)",
					},
					"objective_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название цели, к которой относится ключевой результат (используется вместе с key_result_description)",
					},
					"title": map[string]interface{}{
						"type":		"string",
						"description":	"Название задачи",
					},
					"target": map[string]interface{}{
						"type":		"number",
						"description":	"Целевое значение",
					},
					"unit": map[string]interface{}{
						"type":		"string",
						"description":	"Единица измерения (штуки, проценты, деньги и т.д.)",
					},
					"deadline": map[string]interface{}{
						"type":		"string",
						"description":	"Дедлайн для задачи в формате YYYY-MM-DD",
					},
				},
				"required":	[]string{"title", "target", "unit", "deadline"},
			},
		},
		{
			Name:		"add_key_result_progress",
			Description:	"Добавить прогресс по ключевому результату",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"key_result_id": map[string]interface{}{
						"type":		"integer",
						"description":	"ID ключевого результата",
					},
					"key_result_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название ключевого результата (используется, если ID не указан)",
					},
					"objective_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название цели, к которой относится ключевой результат (используется вместе с key_result_description)",
					},
					"progress": map[string]interface{}{
						"type":		"number",
						"description":	"Прогресс, который нужно добавить (может быть отрицательным для уменьшения)",
					},
				},
				"required":	[]string{"progress"},
			},
		},
		{
			Name:		"add_task_progress",
			Description:	"Добавить прогресс по мини-задаче",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":		"integer",
						"description":	"ID мини-задачи",
					},
					"task_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название задачи (используется, если ID не указан)",
					},
					"key_result_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название ключевого результата, к которому относится задача (используется вместе с task_description)",
					},
					"progress": map[string]interface{}{
						"type":		"number",
						"description":	"Прогресс, который нужно добавить (может быть отрицательным для уменьшения)",
					},
				},
				"required":	[]string{"progress"},
			},
		},
		{
			Name:		"get_objective_details",
			Description:	"Получить подробную информацию о цели, включая ключевые результаты и задачи",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"objective_id": map[string]interface{}{
						"type":		"string",
						"description":	"ID цели",
					},
				},
				"required":	[]string{"objective_id"},
			},
		},
		{
			Name:		"delete_objective",
			Description:	"Удалить цель и все связанные ключевые результаты и задачи",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"objective_id": map[string]interface{}{
						"type":		"string",
						"description":	"ID цели",
					},
					"objective_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название цели (используется, если ID не указан)",
					},
				},
				"required":	[]string{},
			},
		},
		{
			Name:		"delete_key_result",
			Description:	"Удалить ключевой результат и все связанные задачи",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"key_result_id": map[string]interface{}{
						"type":		"integer",
						"description":	"ID ключевого результата",
					},
					"key_result_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название ключевого результата (используется, если ID не указан)",
					},
					"objective_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название цели, к которой относится ключевой результат (используется вместе с key_result_description)",
					},
				},
				"required":	[]string{},
			},
		},
		{
			Name:		"delete_task",
			Description:	"Удалить мини-задачу",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":		"integer",
						"description":	"ID мини-задачи",
					},
					"task_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название задачи (используется, если ID не указан)",
					},
					"key_result_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание или название ключевого результата, к которому относится задача (используется вместе с task_description)",
					},
				},
				"required":	[]string{},
			},
		},
		{
			Name:		"get_calendar_events",
			Description:	"Получить события из календаря на указанную дату или период",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"date": map[string]interface{}{
						"type":		"string",
						"description":	"Дата, на которую нужно получить события (формат YYYY-MM-DD). Если указана, то period игнорируется.",
					},
					"start_date": map[string]interface{}{
						"type":		"string",
						"description":	"Начальная дата периода (формат YYYY-MM-DD). Используется, если date не указана.",
					},
					"end_date": map[string]interface{}{
						"type":		"string",
						"description":	"Конечная дата периода (формат YYYY-MM-DD). Используется, если date не указана.",
					},
				},
				"required":	[]string{},
			},
		},
		{
			Name:		"update_calendar_event",
			Description:	"Обновить существующее событие в календаре",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"event_id": map[string]interface{}{
						"type":		"string",
						"description":	"ID события, которое нужно обновить",
					},
					"title": map[string]interface{}{
						"type":		"string",
						"description":	"Новое название события",
					},
					"description": map[string]interface{}{
						"type":		"string",
						"description":	"Новое описание события",
					},
					"start_time": map[string]interface{}{
						"type":		"string",
						"description":	"Новое время начала события в формате ISO 8601 (YYYY-MM-DDTHH:MM:SS)",
					},
					"end_time": map[string]interface{}{
						"type":		"string",
						"description":	"Новое время окончания события в формате ISO 8601 (YYYY-MM-DDTHH:MM:SS)",
					},
				},
				"required":	[]string{"event_id"},
			},
		},
		{
			Name:		"delete_calendar_event",
			Description:	"Удалить событие из календаря",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"event_id": map[string]interface{}{
						"type":		"string",
						"description":	"ID события, которое нужно удалить",
					},
				},
				"required":	[]string{"event_id"},
			},
		},
		{
			Name:		"delete_calendar_events_by_date",
			Description:	"Удалить все события из календаря на указанную дату",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"date": map[string]interface{}{
						"type":		"string",
						"description":	"Дата, на которую нужно удалить события (формат YYYY-MM-DD)",
					},
				},
				"required":	[]string{"date"},
			},
		},
		{
			Name:		"find_and_update_event",
			Description:	"Найти и обновить событие по его описанию",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"event_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание события, которое нужно найти (название, дата и/или время)",
					},
					"new_title": map[string]interface{}{
						"type":		"string",
						"description":	"Новое название события (если нужно изменить)",
					},
					"new_description": map[string]interface{}{
						"type":		"string",
						"description":	"Новое описание события (если нужно изменить)",
					},
					"new_date": map[string]interface{}{
						"type":		"string",
						"description":	"Новая дата события в формате YYYY-MM-DD (если нужно изменить)",
					},
					"new_time": map[string]interface{}{
						"type":		"string",
						"description":	"Новое время события в формате HH:MM (если нужно изменить)",
					},
					"time_shift": map[string]interface{}{
						"type":		"integer",
						"description":	"Сдвиг времени в минутах (положительное или отрицательное значение)",
					},
				},
				"required":	[]string{"event_description"},
			},
		},
		{
			Name:		"find_and_delete_event",
			Description:	"Найти и удалить событие по его описанию",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"event_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание события, которое нужно найти и удалить (название, дата и/или время)",
					},
					"date": map[string]interface{}{
						"type":		"string",
						"description":	"Дата события в формате YYYY-MM-DD (если известна)",
					},
				},
				"required":	[]string{"event_description"},
			},
		},
		{
			Name:		"get_objectives",
			Description:	"Получить список всех целей пользователя",
			Parameters: map[string]interface{}{
				"type":		"object",
				"properties":	map[string]interface{}{},
				"required":	[]string{},
			},
		},
		{
			Name:		"get_key_results_by_objective_description",
			Description:	"Получить ключевые результаты по названию или описанию цели",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"objective_description": map[string]interface{}{
						"type":		"string",
						"description":	"Название или описание цели для поиска",
					},
				},
				"required":	[]string{"objective_description"},
			},
		},
		{
			Name:		"create_recurring_tasks",
			Description:	"Добавить набор повторяющихся задач к СУЩЕСТВУЮЩЕМУ ключевому результату. Используй только если ключевой результат уже существует в системе.",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"key_result_id": map[string]interface{}{
						"type":		"integer",
						"description":	"ID ключевого результата",
					},
					"key_result_description": map[string]interface{}{
						"type":		"string",
						"description":	"Описание ключевого результата (если ID не указан)",
					},
					"task_title": map[string]interface{}{
						"type":		"string",
						"description":	"Базовое название задачи (к нему будет добавлена дата)",
					},
					"daily_target": map[string]interface{}{
						"type":		"number",
						"description":	"Ежедневная цель (например, 100 отжиманий)",
					},
					"unit": map[string]interface{}{
						"type":		"string",
						"description":	"Единица измерения (штуки, раз и т.д.)",
					},
					"start_date": map[string]interface{}{
						"type":		"string",
						"description":	"Дата начала в формате YYYY-MM-DD",
					},
					"end_date": map[string]interface{}{
						"type":		"string",
						"description":	"Дата окончания в формате YYYY-MM-DD",
					},
				},
				"required":	[]string{"task_title", "daily_target", "unit", "start_date", "end_date"},
			},
		},
		{
			Name:		"create_objective_with_recurring_tasks",
			Description:	"Создать НОВУЮ цель с ключевым результатом и набором ежедневных задач. Используй, когда пользователь формулирует новую цель, которой еще нет в системе.",
			Parameters: map[string]interface{}{
				"type":	"object",
				"properties": map[string]interface{}{
					"objective_title": map[string]interface{}{
						"type":		"string",
						"description":	"Название цели",
					},
					"sphere": map[string]interface{}{
						"type":		"string",
						"description":	"Сфера цели (здоровье, бизнес, личное развитие и т.д.)",
					},
					"period": map[string]interface{}{
						"type":		"string",
						"description":	"Период цели (day, week, month, quarter, year)",
						"enum":		[]string{"day", "week", "month", "quarter", "year"},
					},
					"deadline": map[string]interface{}{
						"type":		"string",
						"description":	"Дедлайн цели в формате YYYY-MM-DD",
					},
					"key_result_title": map[string]interface{}{
						"type":		"string",
						"description":	"Название ключевого результата",
					},
					"key_result_target": map[string]interface{}{
						"type":		"number",
						"description":	"Целевое значение ключевого результата",
					},
					"key_result_unit": map[string]interface{}{
						"type":		"string",
						"description":	"Единица измерения ключевого результата",
					},
					"task_title": map[string]interface{}{
						"type":		"string",
						"description":	"Базовое название задачи (к нему будет добавлена дата)",
					},
					"daily_target": map[string]interface{}{
						"type":		"number",
						"description":	"Ежедневная цель для задачи",
					},
					"task_unit": map[string]interface{}{
						"type":		"string",
						"description":	"Единица измерения для задачи",
					},
					"start_date": map[string]interface{}{
						"type":		"string",
						"description":	"Дата начала повторяющихся задач в формате YYYY-MM-DD",
					},
					"end_date": map[string]interface{}{
						"type":		"string",
						"description":	"Дата окончания повторяющихся задач в формате YYYY-MM-DD",
					},
				},
				"required": []string{"objective_title", "sphere", "period", "key_result_title",
					"key_result_target", "key_result_unit", "task_title",
					"daily_target", "task_unit", "start_date", "end_date"},
			},
		},
	}
}

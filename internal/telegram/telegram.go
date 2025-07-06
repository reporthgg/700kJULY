package telegram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"telegrambot/internal/calendar"
	"telegrambot/internal/chatgpt"
	"telegrambot/internal/finance"
	"telegrambot/internal/linking"
	"telegrambot/internal/meetings"
	"telegrambot/internal/messagestore"
	"telegrambot/internal/messagestore/models"
	"telegrambot/internal/okr"
	"telegrambot/internal/users"
	"telegrambot/pkg/config"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	bot			*tgbotapi.BotAPI
	chatgptService		*chatgpt.ChatGPTService
	calendarService		*calendar.Service
	meetingsService		*meetings.Service
	financeService		*finance.Service
	okrService		*okr.Service
	messageStoreService	*messagestore.Service
	userService		*users.Service
	linkingService		*linking.Service
	cfg			*config.Config
	db			*sqlx.DB
}

func NewHandler(
	cfg *config.Config,
	chatgptService *chatgpt.ChatGPTService,
	calendarService *calendar.Service,
	meetingsService *meetings.Service,
	financeService *finance.Service,
	okrService *okr.Service,
	messageStoreService *messagestore.Service,
	usrService *users.Service,
	lnkService *linking.Service,
	db *sqlx.DB,
) (*Handler, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("ошибка при инициализации Telegram бота: %v", err)
	}

	logrus.Infof("Telegram бот запущен: %s", bot.Self.UserName)

	return &Handler{
		bot:			bot,
		chatgptService:		chatgptService,
		calendarService:	calendarService,
		meetingsService:	meetingsService,
		financeService:		financeService,
		okrService:		okrService,
		messageStoreService:	messageStoreService,
		userService:		usrService,
		linkingService:		lnkService,
		cfg:			cfg,
		db:			db,
	}, nil
}

func (h *Handler) SetupWebhook() error {
	webhookURL := fmt.Sprintf("https://%s:%s/webhook", h.cfg.ServerHost, h.cfg.ServerPort)

	webhookConfig, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		return fmt.Errorf("ошибка при создании конфига вебхука: %w", err)
	}

	if _, err := h.bot.Request(webhookConfig); err != nil {
		return fmt.Errorf("ошибка при установке вебхука: %v", err)
	}

	return nil
}

func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	update, err := h.bot.HandleUpdate(r)
	if err != nil {
		logrus.Errorf("Ошибка при обработке обновления: %v", err)
		return
	}

	h.handleUpdate(*update)
}

func (h *Handler) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := h.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("ошибка при отправке сообщения: %v", err)
	}
	return nil
}

func (h *Handler) handleUpdate(update tgbotapi.Update) {
	ctx := context.Background()

	if update.Message == nil {
		return
	}

	err := h.meetingsService.StoreUser(ctx, update.Message.From.ID, update.Message.From.UserName, update.Message.From.FirstName)
	if err != nil {
		logrus.Errorf("Ошибка при сохранении пользователя: %v", err)
	}

	if strings.HasPrefix(update.Message.Text, "/start ") {
		parts := strings.Fields(update.Message.Text)
		if len(parts) == 2 {
			linkToken := parts[1]
			h.handleLinkTokenStart(ctx, update.Message.Chat.ID, update.Message.From.ID, linkToken)
			return
		}
	}

	query := `SELECT role FROM users WHERE id = $1`
	var role string
	err = h.db.GetContext(ctx, &role, query, update.Message.From.ID)
	if err != nil {
		logrus.Errorf("Ошибка при получении роли пользователя: %v", err)
		role = "free"
	}

	if role == "free" {
		h.SendMessage(update.Message.Chat.ID, "У вас нет подписки")
		return
	}

	if update.Message.Voice != nil || update.Message.Audio != nil {
		h.handleAudioMessage(ctx, update)
		return
	}

	if update.Message.Command() == "google_auth" {
		h.handleGoogleAuth(ctx, update)
		return
	}

	if update.Message.Text != "" {
		h.handleTextMessage(ctx, update)
		return
	}
}

func (h *Handler) handleAudioMessage(ctx context.Context, update tgbotapi.Update) {
	var fileID string
	if update.Message.Voice != nil {
		fileID = update.Message.Voice.FileID
	} else if update.Message.Audio != nil {
		fileID = update.Message.Audio.FileID
	}

	fileURL, err := h.bot.GetFileDirectURL(fileID)
	if err != nil {
		logrus.Errorf("Ошибка при получении URL файла: %v", err)
		h.SendMessage(update.Message.Chat.ID, "Не удалось получить аудио файл")
		return
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		logrus.Errorf("Ошибка при загрузке файла: %v", err)
		h.SendMessage(update.Message.Chat.ID, "Не удалось загрузить аудио файл")
		return
	}
	defer resp.Body.Close()

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Ошибка при чтении аудио данных: %v", err)
		h.SendMessage(update.Message.Chat.ID, "Не удалось прочитать аудио файл")
		return
	}

	h.SendMessage(update.Message.Chat.ID, "🎧 Обрабатываю ваше аудио сообщение через Jarvis...")

	userID := fmt.Sprintf("%d", update.Message.From.ID)
	history, err := h.messageStoreService.GetMessageHistory(ctx, userID)
	if err != nil {
		logrus.Errorf("Ошибка при получении истории сообщений: %v", err)
		history = []models.MessageHistoryItem{}
	}

	userIDInt64 := update.Message.From.ID
	response, err := h.chatgptService.ProcessAudioMessage(ctx, userIDInt64, audioData, history)
	if err != nil {
		logrus.Errorf("Ошибка при обработке аудио через Jarvis: %v", err)
		h.SendMessage(update.Message.Chat.ID, "Произошла ошибка при обработке аудио")
		return
	}

	messageID, err := h.messageStoreService.StoreUserMessage(ctx, userID, "[Аудио сообщение]", "telegram")
	if err != nil {
		logrus.Errorf("Ошибка при сохранении сообщения пользователя: %v", err)
	}

	var promptTokens, completionTokens *int
	err = h.messageStoreService.StoreAiResponse(ctx, messageID, response, promptTokens, completionTokens)
	if err != nil {
		logrus.Errorf("Ошибка при сохранении ответа ИИ: %v", err)
	}

	h.SendMessage(update.Message.Chat.ID, response)
}

func (h *Handler) handleTextMessage(ctx context.Context, update tgbotapi.Update) {

	userID := fmt.Sprintf("%d", update.Message.From.ID)
	messageID, err := h.messageStoreService.StoreUserMessage(ctx, userID, update.Message.Text, "telegram")
	if err != nil {
		logrus.Errorf("Ошибка при сохранении сообщения пользователя: %v", err)
	}

	history, err := h.messageStoreService.GetMessageHistory(ctx, userID)
	if err != nil {
		logrus.Errorf("Ошибка при получении истории сообщений: %v", err)
		history = []models.MessageHistoryItem{}
	}

	userIDInt64 := update.Message.From.ID
	response, err := h.chatgptService.ProcessMessage(ctx, userIDInt64, update.Message.Text, history)
	if err != nil {
		logrus.Errorf("Ошибка при обработке текста через Jarvis: %v", err)
		h.SendMessage(update.Message.Chat.ID, "Произошла ошибка при обработке сообщения")
		return
	}

	var promptTokens, completionTokens *int
	err = h.messageStoreService.StoreAiResponse(ctx, messageID, response, promptTokens, completionTokens)
	if err != nil {
		logrus.Errorf("Ошибка при сохранении ответа ИИ: %v", err)
	}

	h.SendMessage(update.Message.Chat.ID, response)
}

func (h *Handler) handleFunctionCall(ctx context.Context, chatID int64, userID int64, functionCall *chatgpt.FunctionCall) string {

	var response string

	switch functionCall.Name {
	case "create_calendar_event":
		title, _ := functionCall.Arguments["title"].(string)
		description, _ := functionCall.Arguments["description"].(string)
		startTimeStr, _ := functionCall.Arguments["start_time"].(string)
		endTimeStr, _ := functionCall.Arguments["end_time"].(string)

		if startTimeStr != "" {

			t, err := time.Parse("2006-01-02T15:04:05", startTimeStr)
			if err == nil {

				startTimeStr = time.Date(
					t.Year(), t.Month(), t.Day(),
					t.Hour(), t.Minute(), t.Second(), 0,
					time.Local,
				).Format(time.RFC3339)
			}
		}

		if endTimeStr != "" {

			t, err := time.Parse("2006-01-02T15:04:05", endTimeStr)
			if err == nil {

				endTimeStr = time.Date(
					t.Year(), t.Month(), t.Day(),
					t.Hour(), t.Minute(), t.Second(), 0,
					time.Local,
				).Format(time.RFC3339)
			}
		}

		eventID, err := h.calendarService.CreateEvent(ctx, userID, title, description, startTimeStr, endTimeStr)
		if err != nil {
			logrus.Errorf("Ошибка при создании события: %v", err)
			response = "Не удалось создать событие в календаре"
		} else {
			response = fmt.Sprintf("Событие '%s' успешно создано (ID: %s)", title, eventID)
		}

	case "create_meeting":
		title, _ := functionCall.Arguments["title"].(string)
		participantUsername, _ := functionCall.Arguments["participant_username"].(string)
		description, _ := functionCall.Arguments["description"].(string)
		startTime, _ := functionCall.Arguments["start_time"].(string)
		endTime, _ := functionCall.Arguments["end_time"].(string)

		meetingID, err := h.meetingsService.CreateMeeting(ctx, userID, participantUsername, title, description, startTime, endTime)
		if err != nil {
			logrus.Errorf("Ошибка при создании встречи: %v", err)
			response = "Не удалось создать встречу"
		} else {
			response = fmt.Sprintf("Запрос на встречу '%s' с пользователем @%s успешно отправлен (ID: %s)", title, participantUsername, meetingID)
		}

	case "add_transaction":
		amount, _ := functionCall.Arguments["amount"].(float64)
		details, _ := functionCall.Arguments["details"].(string)
		category, _ := functionCall.Arguments["category"].(string)

		transactionID, err := h.financeService.AddTransaction(ctx, userID, amount, details, category)
		if err != nil {
			logrus.Errorf("Ошибка при добавлении транзакции: %v", err)
			response = "Не удалось добавить финансовую транзакцию"
		} else {
			transactionType := "доход"
			if amount < 0 {
				transactionType = "расход"
				amount = -amount
			}
			response = fmt.Sprintf("Добавлен %s на сумму %.2f (ID: %s)", transactionType, amount, transactionID)
		}

	case "get_financial_summary":
		period, _ := functionCall.Arguments["period"].(string)

		summary, err := h.financeService.GetSummary(ctx, userID, period)
		if err != nil {
			logrus.Errorf("Ошибка при получении финансовой сводки: %v", err)
			response = "Не удалось получить финансовую сводку"
		} else {
			response = fmt.Sprintf("Финансовая сводка за %s:\n\nДоходы: %.2f\nРасходы: %.2f\nБаланс: %.2f",
				translatePeriod(period), summary.Income, summary.Expenses, summary.Balance)

			if len(summary.Categories) > 0 {
				response += "\n\nПо категориям:"
				for category, amount := range summary.Categories {
					response += fmt.Sprintf("\n%s: %.2f", category, amount)
				}
			}
		}

	case "create_objective":
		title, _ := functionCall.Arguments["title"].(string)
		sphere, _ := functionCall.Arguments["sphere"].(string)
		period, _ := functionCall.Arguments["period"].(string)
		deadlineStr, _ := functionCall.Arguments["deadline"].(string)
		keyResultsRaw, _ := functionCall.Arguments["key_results"].([]interface{})

		parsedDeadline, err := time.Parse("2006-01-02", deadlineStr)
		if err != nil {
			logrus.Errorf("Ошибка при разборе даты дедлайна: %v", err)
			response = "Некорректный формат даты дедлайна. Используйте формат YYYY-MM-DD."
			break
		}
		deadline := &parsedDeadline

		var keyResults []okr.KeyResult
		for _, krRaw := range keyResultsRaw {
			kr, ok := krRaw.(map[string]interface{})
			if !ok {
				continue
			}

			title, _ := kr["title"].(string)
			target, _ := kr["target"].(float64)
			unit, _ := kr["unit"].(string)
			krDeadlineStr, _ := kr["deadline"].(string)

			parsedKrDeadline, err := time.Parse("2006-01-02", krDeadlineStr)
			if err != nil {
				logrus.Errorf("Ошибка при разборе даты дедлайна ключевого результата: %v", err)
				response = fmt.Sprintf("Некорректный формат даты дедлайна для ключевого результата '%s'", title)
				break
			}
			krDeadline := &parsedKrDeadline

			keyResults = append(keyResults, okr.KeyResult{
				Title:		title,
				Target:		target,
				Unit:		unit,
				Deadline:	krDeadline,
			})
		}

		objectiveID, err := h.okrService.CreateObjective(ctx, userID, title, sphere, period, deadline, keyResults)
		if err != nil {
			logrus.Errorf("Ошибка при создании цели OKR: %v", err)
			response = "Не удалось создать цель OKR"
		} else {
			response = fmt.Sprintf("Цель '%s' успешно создана (ID: %s) с %d ключевыми результатами. Дедлайн: %s",
				title, objectiveID, len(keyResults), deadline.Format("02.01.2006"))
		}

	case "create_key_result":
		objectiveID, _ := functionCall.Arguments["objective_id"].(string)
		objectiveDescription, _ := functionCall.Arguments["objective_description"].(string)
		title, _ := functionCall.Arguments["title"].(string)
		target, _ := functionCall.Arguments["target"].(float64)
		unit, _ := functionCall.Arguments["unit"].(string)
		deadlineStr, _ := functionCall.Arguments["deadline"].(string)

		if objectiveID == "" && objectiveDescription != "" {
			objectives, err := h.okrService.FindObjectiveByDescription(ctx, userID, objectiveDescription)
			if err != nil {
				logrus.Errorf("Ошибка при поиске цели по описанию: %v", err)
				response = "Не удалось найти цель по вашему описанию"
				break
			}

			if len(objectives) == 0 {
				response = fmt.Sprintf("Не удалось найти цель с описанием '%s'. Пожалуйста, проверьте описание или укажите ID цели.", objectiveDescription)
				break
			}

			objectiveID = objectives[0].ID
			logrus.Infof("Найдена цель по описанию: %s (ID: %s)", objectives[0].Title, objectiveID)
		}

		if objectiveID == "" {
			response = "Для добавления ключевого результата необходимо указать ID цели или её описание"
			break
		}

		parsedDeadline, err := time.Parse("2006-01-02", deadlineStr)
		if err != nil {
			logrus.Errorf("Ошибка при разборе даты дедлайна: %v", err)
			response = "Некорректный формат даты дедлайна. Используйте формат YYYY-MM-DD."
			break
		}
		deadline := &parsedDeadline

		keyResultID, err := h.okrService.CreateKeyResult(ctx, userID, objectiveID, title, target, unit, deadline)
		if err != nil {
			logrus.Errorf("Ошибка при создании ключевого результата: %v", err)
			response = "Не удалось создать ключевой результат"
		} else {

			objective, err := h.okrService.GetObjectiveDetails(ctx, userID, objectiveID)
			objectiveTitle := objectiveID
			if err == nil {
				objectiveTitle = objective.Objective.Title
			}

			response = fmt.Sprintf("Ключевой результат '%s' успешно добавлен к цели '%s'. Дедлайн: %s. Также теперь вы можете добавить мини-задачи к этому ключевому результату с помощью сообщения 'Добавь задачу к ключевому результату %s'",
				title, objectiveTitle, parsedDeadline.Format("02.01.2006"), title)
			fmt.Println(keyResultID)
		}

	case "create_task":
		keyResultIDFloat, _ := functionCall.Arguments["key_result_id"].(float64)
		keyResultID := int64(keyResultIDFloat)
		keyResultDescription, _ := functionCall.Arguments["key_result_description"].(string)
		objectiveDescription, _ := functionCall.Arguments["objective_description"].(string)
		title, _ := functionCall.Arguments["title"].(string)
		target, _ := functionCall.Arguments["target"].(float64)
		unit, _ := functionCall.Arguments["unit"].(string)
		deadlineStr, _ := functionCall.Arguments["deadline"].(string)

		if keyResultID == 0 && keyResultDescription != "" {
			keyResults, err := h.okrService.FindKeyResultByDescription(ctx, userID, keyResultDescription, objectiveDescription)
			if err != nil {
				logrus.Errorf("Ошибка при поиске ключевого результата по описанию: %v", err)
				response = "Не удалось найти ключевой результат по вашему описанию"
				break
			}

			if len(keyResults) == 0 {
				response = fmt.Sprintf("Не удалось найти ключевой результат с описанием '%s'. Пожалуйста, проверьте описание или укажите ID ключевого результата.", keyResultDescription)
				break
			}

			keyResultID = keyResults[0].ID
			logrus.Infof("Найден ключевой результат по описанию: %s (ID: %d)", keyResults[0].Title, keyResultID)
		}

		if keyResultID == 0 {
			response = "Для добавления задачи необходимо указать ID ключевого результата или его описание"
			break
		}

		parsedDeadline, err := time.Parse("2006-01-02", deadlineStr)
		if err != nil {
			logrus.Errorf("Ошибка при разборе даты дедлайна: %v", err)
			response = "Некорректный формат даты дедлайна. Используйте формат YYYY-MM-DD."
			break
		}
		deadline := &parsedDeadline

		taskID, err := h.okrService.CreateTask(ctx, userID, keyResultID, title, target, unit, deadline)
		if err != nil {
			logrus.Errorf("Ошибка при создании задачи: %v", err)
			response = "Не удалось создать задачу"
		} else {

			keyResultTitle := fmt.Sprintf("ID: %d", keyResultID)

			keyResults, err := h.okrService.FindKeyResultByDescription(ctx, userID, "", "")
			if err == nil {
				for _, kr := range keyResults {
					if kr.ID == keyResultID {
						keyResultTitle = kr.Title
						break
					}
				}
			}

			response = fmt.Sprintf("Задача '%s' успешно добавлена к ключевому результату '%s' (ID: %d). Дедлайн: %s",
				title, keyResultTitle, taskID, parsedDeadline.Format("02.01.2006"))
		}

	case "add_key_result_progress":
		keyResultIDFloat, _ := functionCall.Arguments["key_result_id"].(float64)
		keyResultID := int64(keyResultIDFloat)
		keyResultDescription, _ := functionCall.Arguments["key_result_description"].(string)
		objectiveDescription, _ := functionCall.Arguments["objective_description"].(string)
		progress, _ := functionCall.Arguments["progress"].(float64)

		if keyResultID == 0 && keyResultDescription != "" {
			keyResults, err := h.okrService.FindKeyResultByDescription(ctx, userID, keyResultDescription, objectiveDescription)
			if err != nil {
				logrus.Errorf("Ошибка при поиске ключевого результата по описанию: %v", err)
				response = "Не удалось найти ключевой результат по вашему описанию"
				break
			}

			if len(keyResults) == 0 {
				response = fmt.Sprintf("Не удалось найти ключевой результат с описанием '%s'. Пожалуйста, проверьте описание или укажите ID ключевого результата.", keyResultDescription)
				break
			}

			keyResultID = keyResults[0].ID
			logrus.Infof("Найден ключевой результат по описанию: %s (ID: %d)", keyResults[0].Title, keyResultID)
		}

		if keyResultID == 0 {
			response = "Для обновления прогресса необходимо указать ID ключевого результата или его описание"
			break
		}

		exceeded, err := h.okrService.UpdateKeyResultProgress(ctx, userID, keyResultID, progress)
		if err != nil {
			logrus.Errorf("Ошибка при обновлении прогресса: %v", err)
			response = "Не удалось обновить прогресс ключевого результата"
		} else {

			keyResultTitle := fmt.Sprintf("ID: %d", keyResultID)

			keyResults, err := h.okrService.FindKeyResultByDescription(ctx, userID, "", "")
			if err == nil {
				for _, kr := range keyResults {
					if kr.ID == keyResultID {
						keyResultTitle = kr.Title
						break
					}
				}
			}

			if exceeded {
				response = fmt.Sprintf("🎉 Поздравляем! Прогресс ключевого результата '%s' обновлен, и вы превысили целевое значение!", keyResultTitle)
			} else {
				if progress >= 0 {
					response = fmt.Sprintf("Прогресс ключевого результата '%s' увеличен на %.2f", keyResultTitle, progress)
				} else {
					response = fmt.Sprintf("Прогресс ключевого результата '%s' уменьшен на %.2f", keyResultTitle, -progress)
				}
			}
		}

	case "add_task_progress":
		taskIDFloat, _ := functionCall.Arguments["task_id"].(float64)
		taskID := int64(taskIDFloat)
		taskDescription, _ := functionCall.Arguments["task_description"].(string)
		keyResultDescription, _ := functionCall.Arguments["key_result_description"].(string)
		progress, _ := functionCall.Arguments["progress"].(float64)

		if taskID == 0 && taskDescription != "" {
			tasks, err := h.okrService.FindTaskByDescription(ctx, userID, taskDescription, keyResultDescription)
			if err != nil {
				logrus.Errorf("Ошибка при поиске задачи по описанию: %v", err)
				response = "Не удалось найти задачу по вашему описанию"
				break
			}

			if len(tasks) == 0 {
				response = fmt.Sprintf("Не удалось найти задачу с описанием '%s'. Пожалуйста, проверьте описание или укажите ID задачи.", taskDescription)
				break
			}

			taskID = tasks[0].ID
			logrus.Infof("Найдена задача по описанию: %s (ID: %d)", tasks[0].Title, taskID)
		}

		if taskID == 0 {
			response = "Для обновления прогресса необходимо указать ID задачи или её описание"
			break
		}

		exceeded, err := h.okrService.UpdateTaskProgress(ctx, userID, taskID, progress)
		if err != nil {
			logrus.Errorf("Ошибка при обновлении прогресса задачи: %v", err)
			response = "Не удалось обновить прогресс задачи"
		} else {

			taskTitle := fmt.Sprintf("ID: %d", taskID)

			tasks, err := h.okrService.FindTaskByDescription(ctx, userID, "", "")
			if err == nil {
				for _, task := range tasks {
					if task.ID == taskID {
						taskTitle = task.Title
						break
					}
				}
			}

			if exceeded {
				response = fmt.Sprintf("🎉 Поздравляем! Прогресс задачи '%s' обновлен, и вы превысили целевое значение!", taskTitle)
			} else {
				if progress >= 0 {
					response = fmt.Sprintf("Прогресс задачи '%s' увеличен на %.2f", taskTitle, progress)
				} else {
					response = fmt.Sprintf("Прогресс задачи '%s' уменьшен на %.2f", taskTitle, -progress)
				}
			}
		}

	case "get_objectives":
		objectives, err := h.okrService.GetObjectives(ctx, userID)
		if err != nil {
			logrus.Errorf("Ошибка при получении списка целей: %v", err)
			response = "Не удалось получить список ваших целей"
			break
		}

		if len(objectives) == 0 {
			response = "У вас пока нет созданных целей. Вы можете создать новую цель!"
			break
		}

		response = "🎯 Ваши цели:\n\n"

		for i, obj := range objectives {

			details, err := h.okrService.GetObjectiveDetails(ctx, userID, obj.ID)
			if err != nil {

				progress, _ := h.okrService.GetObjectiveProgress(ctx, obj.ID)

				response += fmt.Sprintf("%d. %s (Сфера: %s, Период: %s)\n",
					i+1, obj.Title, obj.Sphere, translatePeriod(obj.Period))
				response += fmt.Sprintf("   Прогресс: %.1f%%\n", progress)

				if obj.Deadline != nil {
					response += fmt.Sprintf("   Дедлайн: %s\n", obj.Deadline.Format("02.01.2006"))
				} else {
					response += "   Дедлайн: не установлен\n"
				}

				response += fmt.Sprintf("   ID: %s\n\n", obj.ID)
				continue
			}

			response += fmt.Sprintf("%d. Objective: %s\n", i+1, details.Objective.Title)
			response += fmt.Sprintf("   Сфера: %s, Период: %s\n", details.Objective.Sphere, translatePeriod(details.Objective.Period))

			if details.Objective.Deadline != nil {
				response += fmt.Sprintf("   Дедлайн: %s\n", details.Objective.Deadline.Format("02.01.2006"))
			} else {
				response += "   Дедлайн: не установлен\n"
			}

			response += fmt.Sprintf("   Общий прогресс: %.1f%%\n\n", details.Progress)

			for j, kr := range details.KeyResults {
				response += fmt.Sprintf("   • Key Result %d: %s\n", j+1, kr.KeyResult.Title)
				response += fmt.Sprintf("     Прогресс: %.1f / %.1f %s (%.1f%%)\n",
					kr.KeyResult.Progress, kr.KeyResult.Target, kr.KeyResult.Unit, kr.Progress)

				if len(kr.Tasks) > 0 {
					response += "     Задачи:\n"
					for k, task := range kr.Tasks {
						response += fmt.Sprintf("     %d. %s (%.1f / %.1f %s)\n",
							k+1, task.Title, task.Progress, task.Target, task.Unit)
					}
				}

				response += "\n"
			}

			response += "\n"
		}

		response += "Чтобы увидеть подробную информацию о конкретной цели, запросите детали по ID цели."

	case "get_objective_details":
		objectiveID, _ := functionCall.Arguments["objective_id"].(string)

		details, err := h.okrService.GetObjectiveDetails(ctx, userID, objectiveID)
		if err != nil {
			logrus.Errorf("Ошибка при получении информации о цели: %v", err)
			response = "Не удалось получить информацию о цели"
			break
		}

		response = fmt.Sprintf("🎯 Objective: %s\n", details.Objective.Title)
		response += fmt.Sprintf("Сфера: %s, Период: %s\n", details.Objective.Sphere, translatePeriod(details.Objective.Period))

		if details.Objective.Deadline != nil {
			response += fmt.Sprintf("Дедлайн: %s\n", details.Objective.Deadline.Format("02.01.2006"))
		} else {
			response += "Дедлайн: не установлен\n"
		}

		response += fmt.Sprintf("Общий прогресс: %.1f%%\n\n", details.Progress)

		if len(details.KeyResults) == 0 {
			response += "У этой цели пока нет ключевых результатов"
		} else {
			response += "📊 Ключевые результаты:\n\n"

			for i, kr := range details.KeyResults {
				response += fmt.Sprintf("%d. Key Result: %s\n", i+1, kr.KeyResult.Title)
				response += fmt.Sprintf("   Прогресс: %.1f / %.1f %s (%.1f%%)\n",
					kr.KeyResult.Progress, kr.KeyResult.Target, kr.KeyResult.Unit, kr.Progress)

				if kr.KeyResult.Deadline != nil {
					response += fmt.Sprintf("   Дедлайн: %s\n", kr.KeyResult.Deadline.Format("02.01.2006"))
				} else {
					response += "   Дедлайн: не установлен\n"
				}

				response += fmt.Sprintf("   ID: %d\n", kr.KeyResult.ID)

				if len(kr.Tasks) > 0 {
					response += "\n   Задачи:\n"
					for j, task := range kr.Tasks {
						response += fmt.Sprintf("   %d. %s\n", j+1, task.Title)
						response += fmt.Sprintf("      Прогресс: %.1f / %.1f %s\n",
							task.Progress, task.Target, task.Unit)

						if task.Deadline != nil {
							response += fmt.Sprintf("      Дедлайн: %s\n", task.Deadline.Format("02.01.2006"))
						} else {
							response += "      Дедлайн: не установлен\n"
						}

						response += fmt.Sprintf("      ID: %d\n", task.ID)
					}
				}

				response += "\n"
			}
		}

		response += "\nID цели: " + details.Objective.ID

	case "delete_objective":
		objectiveID, _ := functionCall.Arguments["objective_id"].(string)
		objectiveDescription, _ := functionCall.Arguments["objective_description"].(string)

		var foundObjective *okr.Objective

		if objectiveID == "" && objectiveDescription != "" {
			objectives, err := h.okrService.FindObjectiveByDescription(ctx, userID, objectiveDescription)
			if err != nil {
				logrus.Errorf("Ошибка при поиске цели по описанию: %v", err)
				response = "Не удалось найти цель по вашему описанию"
				break
			}

			if len(objectives) == 0 {
				response = fmt.Sprintf("Не удалось найти цель с описанием '%s'. Пожалуйста, проверьте описание или укажите ID цели.", objectiveDescription)
				break
			}

			objectiveID = objectives[0].ID
			foundObjective = &objectives[0]
			logrus.Infof("Найдена цель по описанию: %s (ID: %s)", objectives[0].Title, objectiveID)
		}

		if objectiveID == "" {
			response = "Для удаления цели необходимо указать ID или её описание"
			break
		}

		err := h.okrService.DeleteObjective(ctx, userID, objectiveID)
		if err != nil {
			logrus.Errorf("Ошибка при удалении цели: %v", err)
			response = "Не удалось удалить цель"
		} else {

			if foundObjective != nil {
				response = fmt.Sprintf("Цель '%s' и все связанные с ней ключевые результаты и задачи успешно удалены", foundObjective.Title)
			} else {

				response = fmt.Sprintf("Цель с ID %s и все связанные с ней ключевые результаты и задачи успешно удалены", objectiveID)
			}
		}

	case "delete_key_result":
		keyResultIDFloat, _ := functionCall.Arguments["key_result_id"].(float64)
		keyResultID := int64(keyResultIDFloat)
		keyResultDescription, _ := functionCall.Arguments["key_result_description"].(string)
		objectiveDescription, _ := functionCall.Arguments["objective_description"].(string)

		var foundKeyResult *okr.KeyResult

		if keyResultID == 0 && keyResultDescription != "" {
			keyResults, err := h.okrService.FindKeyResultByDescription(ctx, userID, keyResultDescription, objectiveDescription)
			if err != nil {
				logrus.Errorf("Ошибка при поиске ключевого результата по описанию: %v", err)
				response = "Не удалось найти ключевой результат по вашему описанию"
				break
			}

			if len(keyResults) == 0 {
				response = fmt.Sprintf("Не удалось найти ключевой результат с описанием '%s'. Пожалуйста, проверьте описание или укажите ID ключевого результата.", keyResultDescription)
				break
			}

			keyResultID = keyResults[0].ID
			foundKeyResult = &keyResults[0]
			logrus.Infof("Найден ключевой результат по описанию: %s (ID: %d)", keyResults[0].Title, keyResultID)
		}

		if keyResultID == 0 {
			response = "Для удаления ключевого результата необходимо указать ID или его описание"
			break
		}

		err := h.okrService.DeleteKeyResult(ctx, userID, keyResultID)
		if err != nil {
			logrus.Errorf("Ошибка при удалении ключевого результата: %v", err)
			response = "Не удалось удалить ключевой результат"
		} else {

			if foundKeyResult != nil {
				response = fmt.Sprintf("Ключевой результат '%s' и все связанные с ним задачи успешно удалены", foundKeyResult.Title)
			} else {

				response = fmt.Sprintf("Ключевой результат с ID %d и все связанные с ним задачи успешно удалены", keyResultID)
			}
		}

	case "delete_task":
		taskIDFloat, _ := functionCall.Arguments["task_id"].(float64)
		taskID := int64(taskIDFloat)
		taskDescription, _ := functionCall.Arguments["task_description"].(string)
		keyResultDescription, _ := functionCall.Arguments["key_result_description"].(string)

		var foundTask *okr.Task

		if taskID == 0 && taskDescription != "" {
			tasks, err := h.okrService.FindTaskByDescription(ctx, userID, taskDescription, keyResultDescription)
			if err != nil {
				logrus.Errorf("Ошибка при поиске задачи по описанию: %v", err)
				response = "Не удалось найти задачу по вашему описанию"
				break
			}

			if len(tasks) == 0 {
				response = fmt.Sprintf("Не удалось найти задачу с описанием '%s'. Пожалуйста, проверьте описание или укажите ID задачи.", taskDescription)
				break
			}

			taskID = tasks[0].ID
			foundTask = &tasks[0]
			logrus.Infof("Найдена задача по описанию: %s (ID: %d)", tasks[0].Title, taskID)
		}

		if taskID == 0 {
			response = "Для удаления задачи необходимо указать ID задачи или её описание"
			break
		}

		err := h.okrService.DeleteTask(ctx, userID, taskID)
		if err != nil {
			logrus.Errorf("Ошибка при удалении задачи: %v", err)
			response = "Не удалось удалить задачу"
		} else {

			if foundTask != nil {
				response = fmt.Sprintf("Задача '%s' успешно удалена", foundTask.Title)
			} else {

				taskTitle := fmt.Sprintf("с ID %d", taskID)
				response = fmt.Sprintf("Задача %s успешно удалена", taskTitle)
			}
		}

	case "get_calendar_events":
		date, _ := functionCall.Arguments["date"].(string)
		startDate, _ := functionCall.Arguments["start_date"].(string)
		endDate, _ := functionCall.Arguments["end_date"].(string)

		var events []calendar.Event
		var err error

		if date != "" {

			parsedDate, parseErr := time.Parse("2006-01-02", date)
			if parseErr != nil {
				logrus.Errorf("Ошибка при разборе даты: %v", parseErr)
				response = "Некорректный формат даты. Используйте формат YYYY-MM-DD."
				break
			}
			events, err = h.calendarService.GetEventsByDate(ctx, userID, parsedDate)
		} else if startDate != "" && endDate != "" {

			parsedStartDate, parseErr := time.Parse("2006-01-02", startDate)
			if parseErr != nil {
				logrus.Errorf("Ошибка при разборе начальной даты: %v", parseErr)
				response = "Некорректный формат начальной даты. Используйте формат YYYY-MM-DD."
				break
			}
			parsedEndDate, parseErr := time.Parse("2006-01-02", endDate)
			if parseErr != nil {
				logrus.Errorf("Ошибка при разборе конечной даты: %v", parseErr)
				response = "Некорректный формат конечной даты. Используйте формат YYYY-MM-DD."
				break
			}

			parsedEndDate = parsedEndDate.Add(24 * time.Hour)
			events, err = h.calendarService.GetEventsByDateRange(ctx, userID, parsedStartDate, parsedEndDate)
		} else {

			today := time.Now()
			events, err = h.calendarService.GetEventsByDate(ctx, userID, today)
		}

		if err != nil {
			logrus.Errorf("Ошибка при получении событий: %v", err)
			response = "Не удалось получить события из календаря"
			break
		}

		if len(events) == 0 {
			if date != "" {
				response = fmt.Sprintf("У вас нет событий на %s", date)
			} else if startDate != "" && endDate != "" {
				response = fmt.Sprintf("У вас нет событий в период с %s по %s", startDate, endDate)
			} else {
				response = "У вас нет событий на сегодня"
			}
		} else {
			if date != "" {
				response = fmt.Sprintf("События на %s:\n\n", date)
			} else if startDate != "" && endDate != "" {
				response = fmt.Sprintf("События в период с %s по %s:\n\n", startDate, endDate)
			} else {
				response = "События на сегодня:\n\n"
			}

			for _, event := range events {
				response += fmt.Sprintf("🕒 %s - %s\n",
					event.StartTime.Format("15:04"),
					event.Title)

				if event.Description != "" {
					response += fmt.Sprintf("   %s\n", event.Description)
				}

				response += fmt.Sprintf("   (ID: %s)\n\n", event.ID)
			}
		}

	case "update_calendar_event":
		eventID, _ := functionCall.Arguments["event_id"].(string)
		title, _ := functionCall.Arguments["title"].(string)
		description, _ := functionCall.Arguments["description"].(string)
		startTime, _ := functionCall.Arguments["start_time"].(string)
		endTime, _ := functionCall.Arguments["end_time"].(string)

		event, err := h.calendarService.GetEventByID(ctx, userID, eventID)
		if err != nil {
			logrus.Errorf("Ошибка при получении события: %v", err)
			response = "Событие не найдено или не принадлежит вам"
			break
		}

		if title == "" {
			title = event.Title
		}
		if description == "" {
			description = event.Description
		}
		if startTime == "" {
			startTime = event.StartTime.Format(time.RFC3339)
		}
		if endTime == "" {
			endTime = event.EndTime.Format(time.RFC3339)
		}

		err = h.calendarService.UpdateEvent(ctx, userID, eventID, title, description, startTime, endTime)
		if err != nil {
			logrus.Errorf("Ошибка при обновлении события: %v", err)
			response = "Не удалось обновить событие"
			break
		}

		response = fmt.Sprintf("Событие '%s' успешно обновлено", title)

	case "delete_calendar_event":
		eventID, _ := functionCall.Arguments["event_id"].(string)

		err := h.calendarService.DeleteEvent(ctx, userID, eventID)
		if err != nil {
			logrus.Errorf("Ошибка при удалении события: %v", err)
			response = "Не удалось удалить событие"
			break
		}

		response = "Событие успешно удалено"

	case "delete_calendar_events_by_date":
		dateStr, _ := functionCall.Arguments["date"].(string)

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			logrus.Errorf("Ошибка при разборе даты: %v", err)
			response = "Некорректный формат даты. Используйте формат YYYY-MM-DD."
			break
		}

		endDate := date.Add(24 * time.Hour)
		count, err := h.calendarService.DeleteEventsByDateRange(ctx, userID, date, endDate)
		if err != nil {
			logrus.Errorf("Ошибка при удалении событий: %v", err)
			response = "Не удалось удалить события"
			break
		}

		if count == 0 {
			response = fmt.Sprintf("На %s не было запланированных событий", dateStr)
		} else {
			response = fmt.Sprintf("Успешно удалено %d событий на %s", count, dateStr)
		}

	case "find_and_update_event":
		eventDescription, _ := functionCall.Arguments["event_description"].(string)
		newTitle, _ := functionCall.Arguments["new_title"].(string)
		newDescription, _ := functionCall.Arguments["new_description"].(string)
		newDate, _ := functionCall.Arguments["new_date"].(string)
		newTime, _ := functionCall.Arguments["new_time"].(string)
		timeShift, ok := functionCall.Arguments["time_shift"].(float64)

		var searchStartDate, searchEndDate time.Time

		dateFound := false

		russianMonths := map[string]int{
			"января":	1, "февраля": 2, "марта": 3, "апреля": 4,
			"мая":	5, "июня": 6, "июля": 7, "августа": 8,
			"сентября":	9, "октября": 10, "ноября": 11, "декабря": 12,
		}

		monthPattern := regexp.MustCompile(`(\d{1,2})\s+(января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря)(?:\s+(\d{4}))?`)
		if matches := monthPattern.FindStringSubmatch(eventDescription); len(matches) >= 3 {
			day, _ := strconv.Atoi(matches[1])
			month := russianMonths[matches[2]]
			year := time.Now().Year()
			if len(matches) > 3 && matches[3] != "" {
				year, _ = strconv.Atoi(matches[3])
			}

			searchStartDate = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
			searchEndDate = searchStartDate.Add(24 * time.Hour)
			dateFound = true

			logrus.Infof("Найдена дата в тексте: %s", searchStartDate.Format("2006-01-02"))
		}

		datePattern := regexp.MustCompile(`(\d{1,2})[\.\-/](\d{1,2})(?:[\.\-/](\d{2,4}))?`)
		matches := datePattern.FindStringSubmatch(eventDescription)
		if !dateFound && len(matches) >= 3 {
			day, _ := strconv.Atoi(matches[1])
			month, _ := strconv.Atoi(matches[2])
			year := time.Now().Year()
			if len(matches) > 3 && matches[3] != "" {
				year, _ = strconv.Atoi(matches[3])

				if year < 100 {
					year += 2000
				}
			}

			searchStartDate = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
			searchEndDate = searchStartDate.Add(24 * time.Hour)
			dateFound = true

			logrus.Infof("Найдена дата в тексте: %s", searchStartDate.Format("2006-01-02"))
		}

		if !dateFound {

			now := time.Now()
			searchStartDate = now.AddDate(0, -1, 0)
			searchEndDate = now.AddDate(0, 1, 0)

			logrus.Info("Дата не найдена, ищем события за период двух месяцев")
		}

		events, err := h.calendarService.GetEventsByDateRange(ctx, userID, searchStartDate, searchEndDate)
		if err != nil {
			logrus.Errorf("Ошибка при получении событий: %v", err)
			response = "Не удалось найти события для обновления"
			break
		}

		if len(events) == 0 {
			if dateFound {
				response = fmt.Sprintf("На %s у вас нет запланированных событий",
					searchStartDate.Format("02.01.2006"))
			} else {
				response = "У вас нет запланированных событий на ближайшие два месяца"
			}
			break
		}

		matchScore := func(event calendar.Event, description string) int {
			descriptionLower := strings.ToLower(description)
			titleLower := strings.ToLower(event.Title)
			descriptionEventLower := strings.ToLower(event.Description)

			score := 0

			titleWords := strings.Fields(titleLower)
			for _, word := range titleWords {
				if len(word) > 3 && strings.Contains(descriptionLower, word) {
					score += 2
				}
			}

			if strings.Contains(titleLower, descriptionLower) || strings.Contains(descriptionLower, titleLower) {
				score += 5
			}

			if strings.Contains(descriptionEventLower, descriptionLower) {
				score += 2
			}

			timePattern := regexp.MustCompile(`\b(\d{1,2})[:\.](\d{2})\b`)
			if matches := timePattern.FindAllStringSubmatch(descriptionLower, -1); len(matches) > 0 {
				for _, match := range matches {
					hours, _ := strconv.Atoi(match[1])
					minutes, _ := strconv.Atoi(match[2])

					if event.StartTime.Hour() == hours && event.StartTime.Minute() == minutes {
						score += 10
						break
					}
				}
			}

			if score > 0 {
				logrus.Infof("Событие: %s (%s), Счет: %d", event.Title, event.StartTime.Format("15:04"), score)
			}

			return score
		}

		var bestMatch *calendar.Event
		bestScore := 0

		for i, event := range events {
			score := matchScore(event, eventDescription)
			if score > bestScore {
				bestScore = score
				bestMatch = &events[i]
			}
		}

		if bestMatch == nil || bestScore < 3 {
			response = "Не удалось найти событие по вашему описанию. Пожалуйста, уточните детали."
			break
		}

		title := bestMatch.Title
		if newTitle != "" {
			title = newTitle
		}

		description := bestMatch.Description
		if newDescription != "" {
			description = newDescription
		}

		startTime := bestMatch.StartTime
		endTime := bestMatch.EndTime
		duration := endTime.Sub(startTime)

		if ok && timeShift != 0 {
			shiftDuration := time.Duration(timeShift) * time.Minute
			startTime = startTime.Add(shiftDuration)
			endTime = startTime.Add(duration)
		}

		if newDate != "" {
			date, err := time.Parse("2006-01-02", newDate)
			if err == nil {

				startTime = time.Date(
					date.Year(), date.Month(), date.Day(),
					startTime.Hour(), startTime.Minute(), startTime.Second(), 0,
					startTime.Location())
				endTime = startTime.Add(duration)
			}
		}

		if newTime != "" {
			timePattern := regexp.MustCompile(`^(\d{1,2}):(\d{2})$`)
			if matches := timePattern.FindStringSubmatch(newTime); len(matches) >= 3 {
				hours, _ := strconv.Atoi(matches[1])
				minutes, _ := strconv.Atoi(matches[2])

				startTime = time.Date(
					startTime.Year(), startTime.Month(), startTime.Day(),
					hours, minutes, 0, 0,
					startTime.Location())
				endTime = startTime.Add(duration)
			}
		}

		err = h.calendarService.UpdateEvent(ctx, userID, bestMatch.ID, title, description,
			startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
		if err != nil {
			logrus.Errorf("Ошибка при обновлении события: %v", err)
			response = "Не удалось обновить событие"
			break
		}

		response = fmt.Sprintf("Событие '%s' успешно обновлено. Новое время: %s",
			title, startTime.Format("02.01.2006 15:04"))

	case "find_and_delete_event":
		eventDescription, _ := functionCall.Arguments["event_description"].(string)
		dateStr, _ := functionCall.Arguments["date"].(string)

		var searchStartDate, searchEndDate time.Time
		dateFound := false

		if dateStr != "" {

			date, err := time.Parse("2006-01-02", dateStr)
			if err == nil {
				searchStartDate = date
				searchEndDate = date.Add(24 * time.Hour)
				dateFound = true
			}
		}

		if !dateFound {

			russianMonths := map[string]int{
				"января":	1, "февраля": 2, "марта": 3, "апреля": 4,
				"мая":	5, "июня": 6, "июля": 7, "августа": 8,
				"сентября":	9, "октября": 10, "ноября": 11, "декабря": 12,
			}

			monthPattern := regexp.MustCompile(`(\d{1,2})\s+(января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря)(?:\s+(\d{4}))?`)
			if matches := monthPattern.FindStringSubmatch(eventDescription); len(matches) >= 3 {
				day, _ := strconv.Atoi(matches[1])
				month := russianMonths[matches[2]]
				year := time.Now().Year()
				if len(matches) > 3 && matches[3] != "" {
					year, _ = strconv.Atoi(matches[3])
				}

				searchStartDate = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
				searchEndDate = searchStartDate.Add(24 * time.Hour)
				dateFound = true

				logrus.Infof("Найдена дата в тексте: %s", searchStartDate.Format("2006-01-02"))
			}

			datePattern := regexp.MustCompile(`(\d{1,2})[\.\-/](\d{1,2})(?:[\.\-/](\d{2,4}))?`)
			matches := datePattern.FindStringSubmatch(eventDescription)
			if !dateFound && len(matches) >= 3 {
				day, _ := strconv.Atoi(matches[1])
				month, _ := strconv.Atoi(matches[2])
				year := time.Now().Year()
				if len(matches) > 3 && matches[3] != "" {
					year, _ = strconv.Atoi(matches[3])

					if year < 100 {
						year += 2000
					}
				}

				searchStartDate = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
				searchEndDate = searchStartDate.Add(24 * time.Hour)
				dateFound = true

				logrus.Infof("Найдена дата в тексте: %s", searchStartDate.Format("2006-01-02"))
			}
		}

		if !dateFound {

			now := time.Now()
			searchStartDate = now.AddDate(0, -1, 0)
			searchEndDate = now.AddDate(0, 1, 0)

			logrus.Info("Дата не найдена, ищем события за период двух месяцев")
		}

		events, err := h.calendarService.GetEventsByDateRange(ctx, userID, searchStartDate, searchEndDate)
		if err != nil {
			logrus.Errorf("Ошибка при получении событий: %v", err)
			response = "Не удалось найти события для удаления"
			break
		}

		if len(events) == 0 {
			if dateFound {
				response = fmt.Sprintf("На %s у вас нет запланированных событий",
					searchStartDate.Format("02.01.2006"))
			} else {
				response = "У вас нет запланированных событий на ближайшие два месяца"
			}
			break
		}

		matchScore := func(event calendar.Event, description string) int {
			descriptionLower := strings.ToLower(description)
			titleLower := strings.ToLower(event.Title)
			descriptionEventLower := strings.ToLower(event.Description)

			score := 0

			titleWords := strings.Fields(titleLower)
			for _, word := range titleWords {
				if len(word) > 3 && strings.Contains(descriptionLower, word) {
					score += 2
				}
			}

			if strings.Contains(titleLower, descriptionLower) || strings.Contains(descriptionLower, titleLower) {
				score += 5
			}

			if strings.Contains(descriptionEventLower, descriptionLower) {
				score += 2
			}

			timePattern := regexp.MustCompile(`\b(\d{1,2})[:\.](\d{2})\b`)
			if matches := timePattern.FindAllStringSubmatch(descriptionLower, -1); len(matches) > 0 {
				for _, match := range matches {
					hours, _ := strconv.Atoi(match[1])
					minutes, _ := strconv.Atoi(match[2])

					if event.StartTime.Hour() == hours && event.StartTime.Minute() == minutes {
						score += 10
						break
					}
				}
			}

			if score > 0 {
				logrus.Infof("Событие: %s (%s), Счет: %d", event.Title, event.StartTime.Format("15:04"), score)
			}

			return score
		}

		var bestMatch *calendar.Event
		bestScore := 0

		for i, event := range events {
			score := matchScore(event, eventDescription)
			if score > bestScore {
				bestScore = score
				bestMatch = &events[i]
			}
		}

		if bestMatch == nil || bestScore < 3 {
			response = "Не удалось найти событие по вашему описанию. Пожалуйста, уточните детали."
			break
		}

		err = h.calendarService.DeleteEvent(ctx, userID, bestMatch.ID)
		if err != nil {
			logrus.Errorf("Ошибка при удалении события: %v", err)
			response = "Не удалось удалить событие"
			break
		}

		response = fmt.Sprintf("Событие '%s' (%s) успешно удалено",
			bestMatch.Title, bestMatch.StartTime.Format("02.01.2006 15:04"))

	case "get_key_results_by_objective_description":
		description, _ := functionCall.Arguments["objective_description"].(string)

		objectives, err := h.okrService.FindObjectiveByDescription(ctx, userID, description)
		if err != nil {
			logrus.Errorf("Ошибка при поиске целей: %v", err)
			response = "Не удалось найти цели по вашему описанию"
			break
		}

		if len(objectives) == 0 {
			response = fmt.Sprintf("Не удалось найти цель с описанием '%s'. Попробуйте другое описание или посмотрите список всех целей.", description)
			break
		}

		objective := objectives[0]

		details, err := h.okrService.GetObjectiveDetails(ctx, userID, objective.ID)
		if err != nil {
			logrus.Errorf("Ошибка при получении деталей цели: %v", err)
			response = "Не удалось получить информацию о цели"
			break
		}

		response = fmt.Sprintf("🎯 Objective: %s\n", details.Objective.Title)
		response += fmt.Sprintf("Сфера: %s, Период: %s\n", details.Objective.Sphere, translatePeriod(details.Objective.Period))

		if details.Objective.Deadline != nil {
			response += fmt.Sprintf("Дедлайн: %s\n", details.Objective.Deadline.Format("02.01.2006"))
		} else {
			response += "Дедлайн: не установлен\n"
		}

		response += fmt.Sprintf("Общий прогресс: %.1f%%\n\n", details.Progress)

		if len(details.KeyResults) == 0 {
			response += "У этой цели пока нет ключевых результатов"
		} else {
			response += "📊 Ключевые результаты:\n\n"

			for i, kr := range details.KeyResults {
				response += fmt.Sprintf("%d. Key Result: %s\n", i+1, kr.KeyResult.Title)
				response += fmt.Sprintf("   Прогресс: %.1f / %.1f %s (%.1f%%)\n",
					kr.KeyResult.Progress, kr.KeyResult.Target, kr.KeyResult.Unit, kr.Progress)

				if kr.KeyResult.Deadline != nil {
					response += fmt.Sprintf("   Дедлайн: %s\n", kr.KeyResult.Deadline.Format("02.01.2006"))
				} else {
					response += "   Дедлайн: не установлен\n"
				}

				response += fmt.Sprintf("   ID: %d\n", kr.KeyResult.ID)

				if len(kr.Tasks) > 0 {
					response += "\n   Задачи:\n"
					for j, task := range kr.Tasks {
						response += fmt.Sprintf("   %d. %s\n", j+1, task.Title)
						response += fmt.Sprintf("      Прогресс: %.1f / %.1f %s\n",
							task.Progress, task.Target, task.Unit)

						if task.Deadline != nil {
							response += fmt.Sprintf("      Дедлайн: %s\n", task.Deadline.Format("02.01.2006"))
						} else {
							response += "      Дедлайн: не установлен\n"
						}

						response += fmt.Sprintf("      ID: %d\n", task.ID)
					}
				}

				response += "\n"
			}
		}

		response += "\nID цели: " + details.Objective.ID

	case "create_recurring_tasks":
		keyResultIDFloat, _ := functionCall.Arguments["key_result_id"].(float64)
		keyResultID := int64(keyResultIDFloat)
		keyResultDescription, _ := functionCall.Arguments["key_result_description"].(string)
		objectiveDescription, _ := functionCall.Arguments["objective_description"].(string)
		title, _ := functionCall.Arguments["title"].(string)
		target, _ := functionCall.Arguments["target"].(float64)
		unit, _ := functionCall.Arguments["unit"].(string)
		deadlineStr, _ := functionCall.Arguments["deadline"].(string)

		if keyResultID == 0 && keyResultDescription != "" {
			keyResults, err := h.okrService.FindKeyResultByDescription(ctx, userID, keyResultDescription, objectiveDescription)
			if err != nil {
				logrus.Errorf("Ошибка при поиске ключевого результата по описанию: %v", err)
				response = "Не удалось найти ключевой результат по вашему описанию"
				break
			}

			if len(keyResults) == 0 {
				response = fmt.Sprintf("Не удалось найти ключевой результат с описанием '%s'. Пожалуйста, проверьте описание или укажите ID ключевого результата.", keyResultDescription)
				break
			}

			keyResultID = keyResults[0].ID
			logrus.Infof("Найден ключевой результат по описанию: %s (ID: %d)", keyResults[0].Title, keyResultID)
		}

		if keyResultID == 0 {
			response = "Для создания повторяющихся задач необходимо указать ID ключевого результата или его описание"
			break
		}

		startDate, err := time.Parse("2006-01-02", deadlineStr)
		if err != nil {
			logrus.Errorf("Ошибка при разборе даты начала: %v", err)
			response = "Некорректный формат даты начала. Используйте формат YYYY-MM-DD."
			break
		}

		endDate, err := time.Parse("2006-01-02", deadlineStr)
		if err != nil {
			logrus.Errorf("Ошибка при разборе даты окончания: %v", err)
			response = "Некорректный формат даты окончания. Используйте формат YYYY-MM-DD."
			break
		}

		taskIDs, err := h.okrService.CreateRecurringTasks(ctx, userID, keyResultID, title, target, unit, startDate, endDate)
		if err != nil {
			logrus.Errorf("Ошибка при создании повторяющихся задач: %v", err)
			response = "Не удалось создать повторяющиеся задачи"
			break
		}

		keyResultTitle := fmt.Sprintf("ID: %d", keyResultID)
		keyResults, err := h.okrService.FindKeyResultByDescription(ctx, userID, "", "")
		if err == nil {
			for _, kr := range keyResults {
				if kr.ID == keyResultID {
					keyResultTitle = kr.Title
					break
				}
			}
		}

		response = fmt.Sprintf("Успешно создано %d ежедневных задач '%s' с целью %.1f %s в день для ключевого результата '%s'. Период: с %s по %s",
			len(taskIDs), title, target, unit, keyResultTitle,
			startDate.Format("02.01.2006"), endDate.Format("02.01.2006"))

	case "create_objective_with_recurring_tasks":
		objectiveTitle, _ := functionCall.Arguments["objective_title"].(string)
		sphere, _ := functionCall.Arguments["sphere"].(string)
		period, _ := functionCall.Arguments["period"].(string)
		deadlineStr, _ := functionCall.Arguments["deadline"].(string)
		keyResultTitle, _ := functionCall.Arguments["key_result_title"].(string)
		keyResultTarget, _ := functionCall.Arguments["key_result_target"].(float64)
		keyResultUnit, _ := functionCall.Arguments["key_result_unit"].(string)
		taskTitle, _ := functionCall.Arguments["task_title"].(string)
		dailyTarget, _ := functionCall.Arguments["daily_target"].(float64)
		taskUnit, _ := functionCall.Arguments["task_unit"].(string)
		startDateStr, _ := functionCall.Arguments["start_date"].(string)
		endDateStr, _ := functionCall.Arguments["end_date"].(string)

		var objectiveDeadline *time.Time
		if deadlineStr != "" {
			deadline, err := time.Parse("2006-01-02", deadlineStr)
			if err == nil {
				objectiveDeadline = &deadline
			}
		}

		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			logrus.Errorf("Ошибка при разборе даты начала: %v", err)
			response = "Некорректный формат даты начала. Используйте формат YYYY-MM-DD."
			break
		}

		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			logrus.Errorf("Ошибка при разборе даты окончания: %v", err)
			response = "Некорректный формат даты окончания. Используйте формат YYYY-MM-DD."
			break
		}

		var keyResultDeadline *time.Time
		keyResultDeadline = &endDate

		objectiveID, keyResultID, taskIDs, err := h.okrService.CreateObjectiveWithRecurringTasks(
			ctx, userID,
			objectiveTitle, sphere, period, objectiveDeadline,
			keyResultTitle, keyResultTarget, keyResultUnit, keyResultDeadline,
			taskTitle, dailyTarget, taskUnit,
			startDate, endDate,
		)

		if err != nil {
			logrus.Errorf("Ошибка при создании цели с задачами: %v", err)
			response = "Не удалось создать цель с повторяющимися задачами"
			break
		}

		response = fmt.Sprintf("Успешно создана цель '%s' с ключевым результатом '%s' и %d ежедневными задачами '%s'.\n\nПериод: с %s по %s\nЕжедневная цель: %.1f %s\n\nID цели: %s\nID ключевого результата: %d",
			objectiveTitle, keyResultTitle, len(taskIDs), taskTitle,
			startDate.Format("02.01.2006"), endDate.Format("02.01.2006"),
			dailyTarget, taskUnit, objectiveID, keyResultID)

	default:
		response = "Неизвестная функция"
	}

	return response
}

func translatePeriod(period string) string {
	switch period {
	case "day":
		return "день"
	case "week":
		return "неделю"
	case "month":
		return "месяц"
	case "quarter":
		return "квартал"
	case "year":
		return "год"
	default:
		return period
	}
}

func (h *Handler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	userID, err := strconv.ParseInt(state, 10, 64)
	if err != nil {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	err = h.calendarService.HandleGoogleCallback(r.Context(), code, userID)
	if err != nil {
		logrus.Errorf("Ошибка при обработке авторизации Google: %v", err)
		http.Error(w, "Ошибка авторизации", http.StatusInternalServerError)
		return
	}

	h.SendMessage(userID, "Google Calendar успешно подключен! Теперь все события будут автоматически синхронизироваться.")

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
		<html>
			<head><title>Авторизация успешна</title></head>
			<body>
				<h1>Google Calendar успешно подключен!</h1>
				<p>Вы можете закрыть это окно и вернуться к Telegram боту.</p>
			</body>
		</html>
	`))
}

func (h *Handler) handleGoogleAuth(ctx context.Context, update tgbotapi.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	authURL, err := h.calendarService.GetGoogleAuthURL(userID, "telegram")
	if err != nil {
		h.SendMessage(chatID, "Не удалось получить ссылку для авторизации Google Calendar")
		return
	}

	msg := fmt.Sprintf("Для подключения Google Calendar перейдите по ссылке:\n%s", authURL)
	h.SendMessage(chatID, msg)
}

func (h *Handler) handleLinkTokenStart(ctx context.Context, chatID int64, telegramUserID int64, token string) {
	fmt.Println("handleLinkTokenStart", token)
	webUserID, err := h.linkingService.ValidateAndUseLinkToken(token)
	if err != nil {
		logrus.Warnf("Ошибка валидации токена привязки '%s' для telegram_user_id %d: %v", token, telegramUserID, err)
		var errMsg string
		switch {
		case errors.Is(err, linking.ErrTokenNotFound):
			errMsg = "Ссылка для привязки недействительна или устарела. Пожалуйста, сгенерируйте новую ссылку на сайте."
		case errors.Is(err, linking.ErrTokenAlreadyUsed):
			errMsg = "Эта ссылка для привязки уже была использована."
		default:
			errMsg = "Не удалось обработать ссылку для привязки. Попробуйте позже."
		}
		h.SendMessage(chatID, errMsg)
		return
	}

	err = h.userService.LinkTelegramAccount(ctx, webUserID, telegramUserID)
	if err != nil {
		logrus.Errorf("Ошибка привязки telegram_id %d к web_user_id %d: %v", telegramUserID, webUserID, err)
		var errMsg string
		switch {
		case errors.Is(err, users.ErrTelegramIDAlreadyLinkedToOtherUser):
			errMsg = "Этот Telegram-аккаунт уже привязан к другому профилю на сайте."
		case errors.Is(err, users.ErrTelegramIDAlreadyLinkedToThisUser):
			errMsg = "Этот Telegram-аккаунт уже был привязан к вашему профилю на сайте."
		case errors.Is(err, users.ErrUserNotFound):
			errMsg = "Профиль на сайте, к которому вы пытаетесь привязаться, не найден. Возможно, ссылка устарела."
		default:
			errMsg = "Произошла ошибка при привязке вашего Telegram-аккаунта. Попробуйте позже."
		}
		h.SendMessage(chatID, errMsg)
		return
	}

	webUser, err := h.userService.GetWebUserByID(ctx, webUserID)
	var successMsg string
	if err == nil && webUser != nil {
		successMsg = fmt.Sprintf("Ваш Telegram-аккаунт успешно привязан к профилю '%s' на сайте!", webUser.Login)
	} else {
		successMsg = "Ваш Telegram-аккаунт успешно привязан к профилю на сайте!"
		if err != nil {
			logrus.Warnf("Не удалось получить детали web_user %d после привязки: %v", webUserID, err)
		}
	}
	h.SendMessage(chatID, successMsg)
	logrus.Infof("Telegram аккаунт %d успешно привязан к web_user %d (токен: %s)", telegramUserID, webUserID, token)
}

func (h *Handler) GetBotInfo() *tgbotapi.User {
	if h.bot != nil {
		return &h.bot.Self
	}
	return nil
}

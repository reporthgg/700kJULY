package chatgpt

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

var AnalyzeProductivityFunction = ChatGPTFunction{
	Name:		"analyze_productivity",
	Description:	"Анализирует продуктивность пользователя и дает персональные рекомендации",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"time_period": {
				Type:		"string",
				Description:	"Период для анализа (week, month, quarter)",
				Enum:		[]string{"week", "month", "quarter"},
			},
			"include_predictions": {
				Type:		"boolean",
				Description:	"Включить предсказания будущей продуктивности",
			},
		},
		Required:	[]string{"time_period"},
	},
}

var GeneratePersonalInsightsFunction = ChatGPTFunction{
	Name:		"generate_personal_insights",
	Description:	"Генерирует персональные инсайты на основе поведения пользователя",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"insight_types": {
				Type:		"array",
				Description:	"Типы инсайтов для генерации",
				Items: &ChatGPTProperty{
					Type:	"string",
					Enum:	[]string{"productivity", "motivation", "patterns", "risks", "opportunities"},
				},
			},
			"priority_level": {
				Type:		"string",
				Description:	"Уровень приоритета инсайтов",
				Enum:		[]string{"high", "medium", "low", "all"},
			},
		},
		Required:	[]string{},
	},
}

var PredictGoalSuccessFunction = ChatGPTFunction{
	Name:		"predict_goal_success",
	Description:	"Предсказывает вероятность успешного завершения цели",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"goal_id": {
				Type:		"string",
				Description:	"ID цели для анализа",
			},
			"include_recommendations": {
				Type:		"boolean",
				Description:	"Включить рекомендации по улучшению",
			},
		},
		Required:	[]string{"goal_id"},
	},
}

var GenerateMotivationFunction = ChatGPTFunction{
	Name:		"generate_motivation",
	Description:	"Генерирует персональную мотивацию на основе текущего состояния пользователя",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"motivation_type": {
				Type:		"string",
				Description:	"Тип мотивации",
				Enum:		[]string{"achievement", "challenge", "support", "celebration", "recovery"},
			},
			"current_mood": {
				Type:		"integer",
				Description:	"Текущее настроение (1-5)",
				Minimum:	1,
				Maximum:	5,
			},
			"energy_level": {
				Type:		"integer",
				Description:	"Уровень энергии (1-5)",
				Minimum:	1,
				Maximum:	5,
			},
		},
		Required:	[]string{},
	},
}

var CreateMotivationPlanFunction = ChatGPTFunction{
	Name:		"create_motivation_plan",
	Description:	"Создает персональный план мотивации на неделю",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"focus_areas": {
				Type:		"array",
				Description:	"Области фокуса для мотивации",
				Items: &ChatGPTProperty{
					Type: "string",
				},
			},
			"intensity": {
				Type:		"string",
				Description:	"Интенсивность мотивационного плана",
				Enum:		[]string{"light", "moderate", "intense"},
			},
		},
		Required:	[]string{},
	},
}

var GenerateWeeklyPlanFunction = ChatGPTFunction{
	Name:		"generate_weekly_plan",
	Description:	"Генерирует оптимальный недельный план на основе целей и предпочтений",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"available_hours_per_day": {
				Type:		"number",
				Description:	"Доступно часов в день для работы над целями",
			},
			"priority_goals": {
				Type:		"array",
				Description:	"Приоритетные цели для включения в план",
				Items: &ChatGPTProperty{
					Type: "string",
				},
			},
			"include_breaks": {
				Type:		"boolean",
				Description:	"Включить перерывы и отдых в план",
			},
		},
		Required:	[]string{},
	},
}

var OptimizeScheduleFunction = ChatGPTFunction{
	Name:		"optimize_schedule",
	Description:	"Оптимизирует расписание на основе пиковых часов продуктивности",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"current_schedule": {
				Type:		"string",
				Description:	"Текущее расписание в JSON формате",
			},
			"constraints": {
				Type:		"array",
				Description:	"Ограничения расписания",
				Items: &ChatGPTProperty{
					Type: "string",
				},
			},
		},
		Required:	[]string{},
	},
}

var ShareGoalFunction = ChatGPTFunction{
	Name:		"share_goal",
	Description:	"Помогает поделиться целью с друзьями или командой",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"goal_id": {
				Type:		"string",
				Description:	"ID цели для шаринга",
			},
			"sharing_type": {
				Type:		"string",
				Description:	"Тип шаринга",
				Enum:		[]string{"progress_update", "achievement", "help_request", "motivation"},
			},
			"audience": {
				Type:		"string",
				Description:	"Аудитория для шаринга",
				Enum:		[]string{"friends", "team", "public", "family"},
			},
		},
		Required:	[]string{"goal_id", "sharing_type"},
	},
}

var FindAccountabilityPartnerFunction = ChatGPTFunction{
	Name:		"find_accountability_partner",
	Description:	"Помогает найти партнера по ответственности или команду поддержки",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"goal_category": {
				Type:		"string",
				Description:	"Категория цели для поиска партнера",
			},
			"interaction_frequency": {
				Type:		"string",
				Description:	"Желаемая частота взаимодействия",
				Enum:		[]string{"daily", "weekly", "monthly"},
			},
		},
		Required:	[]string{"goal_category"},
	},
}

var UpdatePreferencesFunction = ChatGPTFunction{
	Name:		"update_preferences",
	Description:	"Обновляет предпочтения пользователя на основе обратной связи",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"preference_type": {
				Type:		"string",
				Description:	"Тип предпочтения",
				Enum:		[]string{"communication_style", "motivation_type", "reminder_frequency", "difficulty_level"},
			},
			"new_value": {
				Type:		"string",
				Description:	"Новое значение предпочтения",
			},
			"feedback_reason": {
				Type:		"string",
				Description:	"Причина изменения предпочтения",
			},
		},
		Required:	[]string{"preference_type", "new_value"},
	},
}

var LearnFromFeedbackFunction = ChatGPTFunction{
	Name:		"learn_from_feedback",
	Description:	"Обучается на основе обратной связи пользователя",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"feedback_type": {
				Type:		"string",
				Description:	"Тип обратной связи",
				Enum:		[]string{"positive", "negative", "suggestion", "complaint"},
			},
			"context": {
				Type:		"string",
				Description:	"Контекст обратной связи",
			},
			"specific_feature": {
				Type:		"string",
				Description:	"Конкретная функция, к которой относится обратная связь",
			},
		},
		Required:	[]string{"feedback_type", "context"},
	},
}

var CheckWellbeingFunction = ChatGPTFunction{
	Name:		"check_wellbeing",
	Description:	"Проверяет самочувствие и предлагает рекомендации по благополучию",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"current_stress_level": {
				Type:		"integer",
				Description:	"Текущий уровень стресса (1-5)",
				Minimum:	1,
				Maximum:	5,
			},
			"sleep_quality": {
				Type:		"integer",
				Description:	"Качество сна (1-5)",
				Minimum:	1,
				Maximum:	5,
			},
			"work_life_balance": {
				Type:		"integer",
				Description:	"Баланс работы и жизни (1-5)",
				Minimum:	1,
				Maximum:	5,
			},
		},
		Required:	[]string{},
	},
}

var SuggestBreakFunction = ChatGPTFunction{
	Name:		"suggest_break",
	Description:	"Предлагает персональные рекомендации для перерыва и восстановления",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"work_duration": {
				Type:		"integer",
				Description:	"Сколько минут уже работает",
			},
			"energy_level": {
				Type:		"integer",
				Description:	"Текущий уровень энергии (1-5)",
				Minimum:	1,
				Maximum:	5,
			},
			"break_type": {
				Type:		"string",
				Description:	"Предпочитаемый тип перерыва",
				Enum:		[]string{"active", "passive", "creative", "social", "solo"},
			},
		},
		Required:	[]string{"work_duration"},
	},
}

var CheckAchievementsFunction = ChatGPTFunction{
	Name:		"check_achievements",
	Description:	"Проверяет новые достижения и прогресс в системе геймификации",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"show_progress": {
				Type:		"boolean",
				Description:	"Показать прогресс к следующим достижениям",
			},
			"achievement_category": {
				Type:		"string",
				Description:	"Категория достижений",
				Enum:		[]string{"goals", "completion", "streak", "social", "learning", "all"},
			},
		},
		Required:	[]string{},
	},
}

var CreateChallengeFunction = ChatGPTFunction{
	Name:		"create_challenge",
	Description:	"Создает новый вызов или соревнование для мотивации",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"challenge_type": {
				Type:		"string",
				Description:	"Тип вызова",
				Enum:		[]string{"daily", "weekly", "monthly", "custom"},
			},
			"title": {
				Type:		"string",
				Description:	"Название вызова",
			},
			"description": {
				Type:		"string",
				Description:	"Описание вызова",
			},
			"duration_days": {
				Type:		"integer",
				Description:	"Продолжительность в днях",
				Minimum:	1,
				Maximum:	365,
			},
		},
		Required:	[]string{"challenge_type", "title"},
	},
}

var CreateObjectiveFunction = ChatGPTFunction{
	Name:		"create_objective",
	Description:	"Создать новую цель OKR",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"title": {
				Type:		"string",
				Description:	"Название цели",
			},
			"sphere": {
				Type:		"string",
				Description:	"Сфера цели (бизнес, финансы, здоровье, творчество, образование и т.д.)",
			},
			"period": {
				Type:		"string",
				Description:	"Период (week, month, quarter, year)",
				Enum:		[]string{"week", "month", "quarter", "year"},
			},
			"deadline": {
				Type:		"string",
				Description:	"Дедлайн для цели в формате YYYY-MM-DD",
			},
			"key_results": {
				Type:		"array",
				Description:	"Ключевые результаты (2-5 измеримых целей)",
				Items: &ChatGPTProperty{
					Type:		"object",
					Description:	"Ключевой результат с конкретными параметрами",
					Properties: map[string]ChatGPTProperty{
						"title": {
							Type:		"string",
							Description:	"Название ключевого результата",
						},
						"target": {
							Type:		"number",
							Description:	"Целевое значение (число)",
						},
						"unit": {
							Type:		"string",
							Description:	"Единица измерения (подписчики, видео, кг, рубли, проекты и т.д.)",
						},
						"deadline": {
							Type:		"string",
							Description:	"Дедлайн в формате YYYY-MM-DD",
						},
					},
				},
			},
		},
		Required:	[]string{"title", "sphere", "period", "deadline", "key_results"},
	},
}

var GetObjectivesFunction = ChatGPTFunction{
	Name:		"get_objectives",
	Description:	"Получить список целей пользователя",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"period": {
				Type:		"string",
				Description:	"Период для фильтрации (week, month, quarter, year, all)",
				Enum:		[]string{"week", "month", "quarter", "year", "all"},
			},
			"status": {
				Type:		"string",
				Description:	"Статус для фильтрации (active, completed, paused, all)",
				Enum:		[]string{"active", "completed", "paused", "all"},
			},
		},
		Required:	[]string{},
	},
}

var CreateKeyResultFunction = ChatGPTFunction{
	Name:		"create_key_result",
	Description:	"Добавить ключевой результат к существующей цели",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"objective_id": {
				Type:		"string",
				Description:	"ID цели, к которой добавляется ключевой результат",
			},
			"objective_description": {
				Type:		"string",
				Description:	"Описание или название цели (используется, если ID не указан)",
			},
			"title": {
				Type:		"string",
				Description:	"Название ключевого результата",
			},
			"target": {
				Type:		"number",
				Description:	"Целевое значение",
			},
			"unit": {
				Type:		"string",
				Description:	"Единица измерения (штуки, проценты, деньги, видео, подписчики и т.д.)",
			},
			"deadline": {
				Type:		"string",
				Description:	"Дедлайн для ключевого результата в формате YYYY-MM-DD",
			},
		},
		Required:	[]string{"title", "target", "unit", "deadline"},
	},
}

var AddKeyResultProgressFunction = ChatGPTFunction{
	Name:		"add_key_result_progress",
	Description:	"Добавить прогресс по ключевому результату",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"key_result_id": {
				Type:		"integer",
				Description:	"ID ключевого результата",
			},
			"key_result_description": {
				Type:		"string",
				Description:	"Описание ключевого результата (если ID не указан)",
			},
			"objective_description": {
				Type:		"string",
				Description:	"Описание цели, к которой относится ключевой результат",
			},
			"progress": {
				Type:		"number",
				Description:	"Прогресс, который нужно добавить",
			},
		},
		Required:	[]string{"progress"},
	},
}

var CreateTaskFunction = ChatGPTFunction{
	Name:		"create_task",
	Description:	"Создать мини-задачу для ключевого результата",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"key_result_id": {
				Type:		"integer",
				Description:	"ID ключевого результата, к которому добавляется задача",
			},
			"key_result_description": {
				Type:		"string",
				Description:	"Описание ключевого результата (если ID не указан)",
			},
			"objective_description": {
				Type:		"string",
				Description:	"Описание цели (если ID ключевого результата не указан)",
			},
			"title": {
				Type:		"string",
				Description:	"Название задачи",
			},
			"target": {
				Type:		"number",
				Description:	"Целевое значение для задачи",
			},
			"unit": {
				Type:		"string",
				Description:	"Единица измерения (штуки, проценты, минуты, видео и т.д.)",
			},
			"deadline": {
				Type:		"string",
				Description:	"Дедлайн для задачи в формате YYYY-MM-DD",
			},
		},
		Required:	[]string{"title", "target", "unit", "deadline"},
	},
}

var AddTaskProgressFunction = ChatGPTFunction{
	Name:		"add_task_progress",
	Description:	"Добавить прогресс по задаче",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"task_id": {
				Type:		"integer",
				Description:	"ID задачи",
			},
			"task_description": {
				Type:		"string",
				Description:	"Описание задачи (если ID не указан)",
			},
			"key_result_description": {
				Type:		"string",
				Description:	"Описание ключевого результата",
			},
			"progress": {
				Type:		"number",
				Description:	"Прогресс, который нужно добавить",
			},
		},
		Required:	[]string{"progress"},
	},
}

var GetTasksFunction = ChatGPTFunction{
	Name:		"get_tasks",
	Description:	"Получить задачи по ключевому результату",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"key_result_id": {
				Type:		"integer",
				Description:	"ID ключевого результата",
			},
			"objective_id": {
				Type:		"string",
				Description:	"ID цели для получения всех задач",
			},
		},
		Required:	[]string{},
	},
}

var DeleteObjectiveFunction = ChatGPTFunction{
	Name:		"delete_objective",
	Description:	"Удалить цель полностью (со всеми ключевыми результатами и задачами)",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"objective_id": {
				Type:		"string",
				Description:	"ID цели для удаления",
			},
			"objective_description": {
				Type:		"string",
				Description:	"Описание или название цели (если ID не указан)",
			},
			"confirm": {
				Type:		"boolean",
				Description:	"Подтверждение удаления (обязательно true для удаления)",
			},
		},
		Required:	[]string{"confirm"},
	},
}

var DeleteKeyResultFunction = ChatGPTFunction{
	Name:		"delete_key_result",
	Description:	"Удалить ключевой результат (со всеми связанными задачами)",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"key_result_id": {
				Type:		"integer",
				Description:	"ID ключевого результата для удаления",
			},
			"key_result_description": {
				Type:		"string",
				Description:	"Описание ключевого результата (если ID не указан)",
			},
			"objective_description": {
				Type:		"string",
				Description:	"Описание цели (для уточнения поиска)",
			},
			"confirm": {
				Type:		"boolean",
				Description:	"Подтверждение удаления (обязательно true для удаления)",
			},
		},
		Required:	[]string{"confirm"},
	},
}

var DeleteTaskFunction = ChatGPTFunction{
	Name:		"delete_task",
	Description:	"Удалить задачу",
	Parameters: ChatGPTFunctionParameters{
		Type:	"object",
		Properties: map[string]ChatGPTProperty{
			"task_id": {
				Type:		"integer",
				Description:	"ID задачи для удаления",
			},
			"task_description": {
				Type:		"string",
				Description:	"Описание задачи (если ID не указан)",
			},
			"key_result_description": {
				Type:		"string",
				Description:	"Описание ключевого результата (для уточнения поиска)",
			},
			"confirm": {
				Type:		"boolean",
				Description:	"Подтверждение удаления (обязательно true для удаления)",
			},
		},
		Required:	[]string{"confirm"},
	},
}

func (c *ChatGPTService) handleAnalyzeProductivity(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	timePeriod := "week"
	if tp, ok := args["time_period"].(string); ok {
		timePeriod = tp
	}

	includePredictions := false
	if ip, ok := args["include_predictions"].(bool); ok {
		includePredictions = ip
	}

	ctx := context.Background()
	metrics, err := c.aiCoach.AnalyzeProductivity(ctx, userID)
	if err != nil {
		return "Не удалось проанализировать продуктивность: " + err.Error(), &AnalyzeProductivityFunction, err
	}

	response := fmt.Sprintf("📊 **Анализ продуктивности за %s:**\n\n", getPeriodName(timePeriod))
	response += fmt.Sprintf("• Уровень завершения: %.1f%%\n", metrics.CompletionRate*100)
	response += fmt.Sprintf("• Среднее время задачи: %.1f мин\n", metrics.AverageTaskTime)
	response += fmt.Sprintf("• Серия: %d дней\n", metrics.StreakDays)
	response += fmt.Sprintf("• Уровень: %d (%d очков)\n\n", metrics.Level, metrics.TotalPointsEarned)

	if len(metrics.PeakProductivityHours) > 0 {
		response += fmt.Sprintf("⏰ **Пиковые часы:** %v\n\n", metrics.PeakProductivityHours)
	}

	if len(metrics.ImprovementSuggestions) > 0 {
		response += "💡 **Рекомендации:**\n"
		for _, suggestion := range metrics.ImprovementSuggestions {
			response += fmt.Sprintf("• %s\n", suggestion)
		}
		response += "\n"
	}

	if includePredictions && len(metrics.PredictedOutcomes) > 0 {
		response += "🔮 **Прогнозы:**\n"
		for _, prediction := range metrics.PredictedOutcomes {
			response += fmt.Sprintf("• %s (уверенность: %.1f%%)\n", prediction.Description, prediction.Confidence*100)
		}
	}

	return response, &AnalyzeProductivityFunction, nil
}

func (c *ChatGPTService) handleGeneratePersonalInsights(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	ctx := context.Background()
	insights, err := c.aiCoach.GenerateInsights(ctx, userID)
	if err != nil {
		return "Не удалось сгенерировать инсайты: " + err.Error(), &GeneratePersonalInsightsFunction, err
	}

	if len(insights) == 0 {
		return "🤖 На данный момент новых инсайтов нет. Продолжай работать над своими целями, и я найду новые паттерны для анализа!", &GeneratePersonalInsightsFunction, nil
	}

	response := "💡 **Персональные инсайты:**\n\n"

	for i, insight := range insights {
		if i >= 5 {
			break
		}

		response += fmt.Sprintf("**%s** (%s)\n", insight.Title, getCategoryEmoji(insight.Category))
		response += fmt.Sprintf("%s\n", insight.Content)

		if insight.ActionButtonText != "" {
			response += fmt.Sprintf("👆 %s\n", insight.ActionButtonText)
		}

		response += "\n"
	}

	return response, &GeneratePersonalInsightsFunction, nil
}

func (c *ChatGPTService) handlePredictGoalSuccess(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	goalID, ok := args["goal_id"].(string)
	if !ok {
		return "Не указан ID цели для анализа", &PredictGoalSuccessFunction, fmt.Errorf("goal_id is required")
	}

	includeRecommendations := true
	if ir, ok := args["include_recommendations"].(bool); ok {
		includeRecommendations = ir
	}

	ctx := context.Background()
	prediction, err := c.aiCoach.PredictCompletionProbability(ctx, userID, goalID)
	if err != nil {
		return "Не удалось создать предсказание: " + err.Error(), &PredictGoalSuccessFunction, err
	}

	response := "🎯 **Прогноз успеха цели:**\n\n"
	response += fmt.Sprintf("📊 Вероятность успеха: %.1f%%\n", prediction.Probability*100)
	response += fmt.Sprintf("📅 Ожидаемое завершение: %s\n", prediction.EstimatedCompletionDate.Format("02.01.2006"))
	response += fmt.Sprintf("🎯 Уверенность прогноза: %.1f%%\n\n", prediction.ConfidenceLevel*100)

	if len(prediction.SuccessFactors) > 0 {
		response += "✅ **Факторы успеха:**\n"
		for _, factor := range prediction.SuccessFactors {
			response += fmt.Sprintf("• %s\n", factor)
		}
		response += "\n"
	}

	if len(prediction.RiskFactors) > 0 {
		response += "⚠️ **Факторы риска:**\n"
		for _, risk := range prediction.RiskFactors {
			response += fmt.Sprintf("• %s\n", risk)
		}
		response += "\n"
	}

	if includeRecommendations && len(prediction.Recommendations) > 0 {
		response += "💡 **Рекомендации:**\n"
		for _, rec := range prediction.Recommendations {
			response += fmt.Sprintf("• %s\n", rec)
		}
	}

	return response, &PredictGoalSuccessFunction, nil
}

func (c *ChatGPTService) handleGenerateMotivation(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	ctx := context.Background()
	motivation, err := c.aiCoach.GenerateMotivation(ctx, userID)
	if err != nil {
		return "Не удалось сгенерировать мотивацию: " + err.Error(), &GenerateMotivationFunction, err
	}

	response := "🚀 **Персональная мотивация:**\n\n"
	response += motivation

	if motivationType, ok := args["motivation_type"].(string); ok {
		switch motivationType {
		case "challenge":
			response += "\n\n🎯 **Вызов дня:** Попробуй превзойти вчерашний результат на 10%!"
		case "celebration":
			response += "\n\n🎉 **Время праздновать:** Ты достоин признания за свои достижения!"
		case "recovery":
			response += "\n\n🌸 **Время восстановления:** Помни, отдых - это тоже часть пути к успеху."
		}
	}

	return response, &GenerateMotivationFunction, nil
}

func (c *ChatGPTService) handleCreateMotivationPlan(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	ctx := context.Background()

	goals, err := c.aiCoach.GetActiveUserGoals(ctx, userID)
	if err != nil {
		logrus.Warnf("Не удалось получить цели: %v", err)
		goals = []interface{}{}
	}

	plan, err := c.aiCoach.GenerateMotivationPlan(ctx, userID, goals)
	if err != nil {
		return "Не удалось создать план мотивации: " + err.Error(), &CreateMotivationPlanFunction, err
	}

	response := "📋 **Твой персональный план мотивации на неделю:**\n\n"

	if dailyMotivations, ok := plan["daily_motivations"].(map[string]interface{}); ok {
		days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
		dayNames := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"}

		for i, day := range days {
			if dayPlan, ok := dailyMotivations[day].(map[string]interface{}); ok {
				response += fmt.Sprintf("**%s:**\n", dayNames[i])

				if morning, ok := dayPlan["morning_motivation"].(string); ok {
					response += fmt.Sprintf("🌅 Утро: %s\n", morning)
				}
				if midday, ok := dayPlan["midday_boost"].(string); ok {
					response += fmt.Sprintf("☀️ День: %s\n", midday)
				}
				if evening, ok := dayPlan["evening_reflection"].(string); ok {
					response += fmt.Sprintf("🌙 Вечер: %s\n", evening)
				}

				response += "\n"
			}
		}
	}

	if challenges, ok := plan["challenge_boosts"].([]map[string]interface{}); ok {
		response += "🎯 **Вызовы недели:**\n"
		for _, challenge := range challenges {
			if title, ok := challenge["title"].(string); ok {
				response += fmt.Sprintf("• %s\n", title)
			}
		}
		response += "\n"
	}

	if rewards, ok := plan["reward_schedule"].(map[string]interface{}); ok {
		if dailyRewards, ok := rewards["daily_rewards"].([]string); ok {
			response += "🎁 **Ежедневные награды:**\n"
			for _, reward := range dailyRewards {
				response += fmt.Sprintf("• %s\n", reward)
			}
		}
	}

	return response, &CreateMotivationPlanFunction, nil
}

func (c *ChatGPTService) handleGenerateWeeklyPlan(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	ctx := context.Background()
	plan, err := c.aiCoach.GenerateWeeklyPlan(ctx, userID)
	if err != nil {
		return "Не удалось создать недельный план: " + err.Error(), &GenerateWeeklyPlanFunction, err
	}

	response := "📅 **Твой оптимальный план на неделю:**\n\n"

	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	dayNames := []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"}

	for i, day := range days {
		if dayPlan, ok := plan[day].(map[string]interface{}); ok {
			response += fmt.Sprintf("**%s:**\n", dayNames[i])

			if focus, ok := dayPlan["focus"].(string); ok {
				response += fmt.Sprintf("🎯 Фокус: %s\n", focus)
			}

			if time, ok := dayPlan["time"].(float64); ok {
				response += fmt.Sprintf("⏱️ Время: %.1f часа\n", time)
			}

			response += "\n"
		}
	}

	response += "💡 **Общие рекомендации:**\n"
	response += "• Начинай день с планирования\n"
	response += "• Делай перерывы каждые 45-90 минут\n"
	response += "• Завершай день рефлексией\n"
	response += "• Адаптируй план под свое самочувствие\n"

	return response, &GenerateWeeklyPlanFunction, nil
}

func getPeriodName(period string) string {
	switch period {
	case "week":
		return "неделю"
	case "month":
		return "месяц"
	case "quarter":
		return "квартал"
	default:
		return "период"
	}
}

func getCategoryEmoji(category string) string {
	switch category {
	case "productivity":
		return "📊"
	case "motivation":
		return "🚀"
	case "achievement":
		return "🏆"
	case "planning":
		return "📋"
	case "health":
		return "💪"
	case "learning":
		return "📚"
	default:
		return "💡"
	}
}

func GetAllJarvisFunctions() []ChatGPTFunction {
	return []ChatGPTFunction{

		AnalyzeProductivityFunction,
		GeneratePersonalInsightsFunction,
		PredictGoalSuccessFunction,
		GenerateMotivationFunction,
		CreateMotivationPlanFunction,
		GenerateWeeklyPlanFunction,
		OptimizeScheduleFunction,
		ShareGoalFunction,
		FindAccountabilityPartnerFunction,
		UpdatePreferencesFunction,
		LearnFromFeedbackFunction,
		CheckWellbeingFunction,
		SuggestBreakFunction,
		CheckAchievementsFunction,
		CreateChallengeFunction,
		CreateObjectiveFunction,
		GetObjectivesFunction,
		CreateKeyResultFunction,
		AddKeyResultProgressFunction,
		CreateTaskFunction,
		AddTaskProgressFunction,
		GetTasksFunction,
		DeleteObjectiveFunction,
		DeleteKeyResultFunction,
		DeleteTaskFunction,
	}
}

func (c *ChatGPTService) handleNewJarvisFunctions(functionCall *ChatGPTFunctionCall, userID int64) (string, *ChatGPTFunction, error) {
	args := functionCall.Arguments

	switch functionCall.Name {
	case "analyze_productivity":
		return c.handleAnalyzeProductivity(args, userID)
	case "generate_personal_insights":
		return c.handleGeneratePersonalInsights(args, userID)
	case "predict_goal_success":
		return c.handlePredictGoalSuccess(args, userID)
	case "generate_motivation":
		return c.handleGenerateMotivation(args, userID)
	case "create_motivation_plan":
		return c.handleCreateMotivationPlan(args, userID)
	case "generate_weekly_plan":
		return c.handleGenerateWeeklyPlan(args, userID)

	case "create_objective":
		return c.handleCreateObjective(args, userID)
	case "get_objectives":
		return c.handleGetObjectives(args, userID)
	case "create_key_result":
		return c.handleCreateKeyResult(args, userID)
	case "add_key_result_progress":
		return c.handleAddKeyResultProgress(args, userID)

	case "create_task":
		return c.handleCreateTask(args, userID)
	case "add_task_progress":
		return c.handleAddTaskProgress(args, userID)
	case "get_tasks":
		return c.handleGetTasks(args, userID)
	case "delete_objective":
		return c.handleDeleteObjective(args, userID)
	case "delete_key_result":
		return c.handleDeleteKeyResult(args, userID)
	case "delete_task":
		return c.handleDeleteTask(args, userID)

	default:
		return "", nil, fmt.Errorf("неизвестная функция: %s", functionCall.Name)
	}
}

func (c *ChatGPTService) handleCreateObjective(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	logrus.Infof("Создание цели для пользователя %d с аргументами: %+v", userID, args)

	title, _ := args["title"].(string)
	sphere, _ := args["sphere"].(string)
	period, _ := args["period"].(string)
	deadline, _ := args["deadline"].(string)
	keyResultsInterface, _ := args["key_results"].([]interface{})

	logrus.Infof("Параметры цели: title=%s, sphere=%s, period=%s, deadline=%s, keyResults=%d",
		title, sphere, period, deadline, len(keyResultsInterface))

	if title == "" || sphere == "" || period == "" || deadline == "" {
		logrus.Errorf("Отсутствуют обязательные параметры: title=%s, sphere=%s, period=%s, deadline=%s",
			title, sphere, period, deadline)
		return "❌ Не указаны обязательные параметры для создания цели", &CreateObjectiveFunction, nil
	}

	query := `
		INSERT INTO objectives (id, user_id, title, sphere, period, deadline, status, created_at, updated_at) 
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, 'active', NOW(), NOW()) 
		RETURNING id
	`

	var objectiveID string
	logrus.Infof("Выполняем SQL запрос создания цели: %s", query)
	err := c.db.QueryRow(query, userID, title, sphere, period, deadline).Scan(&objectiveID)
	if err != nil {
		logrus.Errorf("Ошибка создания цели: %v", err)
		return "❌ Не удалось создать цель в базе данных", &CreateObjectiveFunction, fmt.Errorf("database error: %w", err)
	}

	logrus.Infof("Цель создана успешно с ID: %s", objectiveID)

	keyResultsCreated := 0
	logrus.Infof("Обрабатываем %d ключевых результатов", len(keyResultsInterface))

	for i, krInterface := range keyResultsInterface {
		logrus.Infof("Обработка KR #%d: %+v", i+1, krInterface)

		if krMap, ok := krInterface.(map[string]interface{}); ok {
			krTitle, _ := krMap["title"].(string)
			target, _ := krMap["target"].(float64)
			unit, _ := krMap["unit"].(string)
			krDeadline, _ := krMap["deadline"].(string)

			logrus.Infof("KR параметры: title=%s, target=%.1f, unit=%s, deadline=%s",
				krTitle, target, unit, krDeadline)

			if krTitle != "" && target > 0 && unit != "" && krDeadline != "" {
				krQuery := `
					INSERT INTO key_results (objective_id, title, target, unit, deadline, status, progress, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, 'active', 0, NOW(), NOW())
				`

				logrus.Infof("Создаем KR: %s", krTitle)
				_, err := c.db.Exec(krQuery, objectiveID, krTitle, target, unit, krDeadline)
				if err != nil {
					logrus.Errorf("Ошибка создания ключевого результата: %v", err)
				} else {
					keyResultsCreated++
					logrus.Infof("KR создан успешно: %s", krTitle)
				}
			} else {
				logrus.Warnf("KR пропущен из-за неполных данных: title=%s, target=%.1f, unit=%s, deadline=%s",
					krTitle, target, unit, krDeadline)
			}
		} else {
			logrus.Warnf("KR #%d не является объектом: %T", i+1, krInterface)
		}
	}

	response := fmt.Sprintf("🎯 **Цель успешно создана!**\n\n")
	response += fmt.Sprintf("📋 **Название:** %s\n", title)
	response += fmt.Sprintf("🎌 **Сфера:** %s\n", sphere)
	response += fmt.Sprintf("⏰ **Период:** %s\n", getPeriodName(period))
	response += fmt.Sprintf("📅 **Дедлайн:** %s\n", deadline)
	response += fmt.Sprintf("🔑 **Ключевые результаты:** %d создано\n\n", keyResultsCreated)

	response += "✨ Jarvis будет отслеживать твой прогресс и поможет достичь этой цели!"

	return response, &CreateObjectiveFunction, nil
}

func (c *ChatGPTService) handleGetObjectives(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	logrus.Infof("Получение целей для пользователя %d с аргументами: %+v", userID, args)

	period, _ := args["period"].(string)
	status, _ := args["status"].(string)

	if period == "" {
		period = "all"
	}
	if status == "" {
		status = "all"
	}

	logrus.Infof("Фильтры: period=%s, status=%s", period, status)

	query := `
		SELECT o.id, o.title, o.sphere, o.period, o.deadline, o.status, o.created_at,
		       COUNT(kr.id) as key_results_count,
		       COALESCE(AVG(CASE WHEN kr.target > 0 THEN (kr.progress::float / kr.target::float) * 100 END), 0) as avg_progress
		FROM objectives o
		LEFT JOIN key_results kr ON o.id = kr.objective_id
		WHERE o.user_id = $1
	`

	args_list := []interface{}{userID}
	argCount := 1

	if period != "all" {
		argCount++
		query += fmt.Sprintf(" AND o.period = $%d", argCount)
		args_list = append(args_list, period)
	}

	if status != "all" {
		argCount++
		query += fmt.Sprintf(" AND o.status = $%d", argCount)
		args_list = append(args_list, status)
	}

	query += " GROUP BY o.id, o.title, o.sphere, o.period, o.deadline, o.status, o.created_at ORDER BY o.created_at DESC"

	logrus.Infof("Выполняем SQL запрос получения целей: %s с параметрами: %+v", query, args_list)
	rows, err := c.db.Query(query, args_list...)
	if err != nil {
		logrus.Errorf("Ошибка получения целей: %v", err)
		return "❌ Не удалось получить цели из базы данных", &GetObjectivesFunction, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	response := "🎯 **Твои цели:**\n\n"
	objectiveCount := 0

	for rows.Next() {
		var id, title, sphere, period, deadline, status, createdAt string
		var keyResultsCount int
		var avgProgress float64

		err := rows.Scan(&id, &title, &sphere, &period, &deadline, &status, &createdAt, &keyResultsCount, &avgProgress)
		if err != nil {
			continue
		}

		objectiveCount++

		statusEmoji := "🔄"
		switch status {
		case "completed":
			statusEmoji = "✅"
		case "paused":
			statusEmoji = "⏸️"
		case "active":
			statusEmoji = "🎯"
		}

		response += fmt.Sprintf("%s **%s** (%s)\n", statusEmoji, title, sphere)
		response += fmt.Sprintf("📊 Прогресс: %.1f%% | 🔑 KR: %d | 📅 %s\n\n", avgProgress, keyResultsCount, deadline)
	}

	logrus.Infof("Найдено целей для пользователя %d: %d", userID, objectiveCount)

	if objectiveCount == 0 {
		response = "🎯 **У тебя пока нет целей**\n\n"
		response += "💡 Скажи мне о своих планах, и я помогу их структурировать в цели OKR!"
	} else {
		response += fmt.Sprintf("📈 **Всего целей:** %d", objectiveCount)
	}

	logrus.Infof("Возвращаем ответ get_objectives для пользователя %d: %s", userID, response)
	return response, &GetObjectivesFunction, nil
}

func (c *ChatGPTService) handleCreateKeyResult(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	logrus.Infof("Создание ключевого результата для пользователя %d с аргументами: %+v", userID, args)

	title, _ := args["title"].(string)
	target, _ := args["target"].(float64)
	unit, _ := args["unit"].(string)
	deadline, _ := args["deadline"].(string)
	objectiveID, _ := args["objective_id"].(string)
	objectiveDescription, _ := args["objective_description"].(string)

	if title == "" || target <= 0 || unit == "" || deadline == "" {
		return "❌ Не указаны обязательные параметры для создания ключевого результата", &CreateKeyResultFunction, nil
	}

	if objectiveID == "" && objectiveDescription != "" {
		query := `SELECT id FROM objectives WHERE user_id = $1 AND LOWER(title) LIKE LOWER($2) ORDER BY created_at DESC LIMIT 1`
		err := c.db.QueryRow(query, userID, "%"+objectiveDescription+"%").Scan(&objectiveID)
		if err != nil {
			return "❌ Не найдена цель по описанию: " + objectiveDescription, &CreateKeyResultFunction, nil
		}
	}

	if objectiveID == "" {
		return "❌ Не указана цель для ключевого результата", &CreateKeyResultFunction, nil
	}

	var ownerID int64
	checkQuery := `SELECT user_id FROM objectives WHERE id = $1`
	err := c.db.QueryRow(checkQuery, objectiveID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		return "❌ Цель не найдена или не принадлежит пользователю", &CreateKeyResultFunction, nil
	}

	insertQuery := `
		INSERT INTO key_results (objective_id, title, target, unit, deadline, status, progress, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'active', 0, NOW(), NOW())
		RETURNING id
	`

	var keyResultID int64
	err = c.db.QueryRow(insertQuery, objectiveID, title, target, unit, deadline).Scan(&keyResultID)
	if err != nil {
		logrus.Errorf("Ошибка создания ключевого результата: %v", err)
		return "❌ Не удалось создать ключевой результат", &CreateKeyResultFunction, nil
	}

	var objectiveTitle string
	titleQuery := `SELECT title FROM objectives WHERE id = $1`
	c.db.QueryRow(titleQuery, objectiveID).Scan(&objectiveTitle)

	response := fmt.Sprintf("🔑 **Ключевой результат создан!**\n\n")
	response += fmt.Sprintf("📋 **Название:** %s\n", title)
	response += fmt.Sprintf("🎯 **Цель:** %s\n", objectiveTitle)
	response += fmt.Sprintf("📊 **Целевое значение:** %.1f %s\n", target, unit)
	response += fmt.Sprintf("📅 **Дедлайн:** %s\n", deadline)
	response += fmt.Sprintf("🆔 **ID:** %d\n\n", keyResultID)
	response += "✨ Jarvis отслеживает твой прогресс! Используй команду добавления прогресса когда будешь готов обновить результат."

	return response, &CreateKeyResultFunction, nil
}

func (c *ChatGPTService) handleAddKeyResultProgress(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	logrus.Infof("Добавление прогресса ключевого результата для пользователя %d с аргументами: %+v", userID, args)

	keyResultID, hasID := args["key_result_id"].(float64)
	keyResultDescription, _ := args["key_result_description"].(string)
	objectiveDescription, _ := args["objective_description"].(string)
	progress, _ := args["progress"].(float64)

	if progress <= 0 {
		return "❌ Прогресс должен быть больше нуля", &AddKeyResultProgressFunction, nil
	}

	var finalKeyResultID int64

	if !hasID || keyResultID <= 0 {
		if keyResultDescription == "" {
			return "❌ Не указан ID или описание ключевого результата", &AddKeyResultProgressFunction, nil
		}

		var query string
		var params []interface{}

		if objectiveDescription != "" {

			query = `
				SELECT kr.id 
				FROM key_results kr
				JOIN objectives o ON kr.objective_id = o.id
				WHERE o.user_id = $1 
				AND LOWER(kr.title) LIKE LOWER($2)
				AND LOWER(o.title) LIKE LOWER($3)
				ORDER BY kr.created_at DESC LIMIT 1
			`
			params = []interface{}{userID, "%" + keyResultDescription + "%", "%" + objectiveDescription + "%"}
		} else {

			query = `
				SELECT kr.id 
				FROM key_results kr
				JOIN objectives o ON kr.objective_id = o.id
				WHERE o.user_id = $1 
				AND LOWER(kr.title) LIKE LOWER($2)
				ORDER BY kr.created_at DESC LIMIT 1
			`
			params = []interface{}{userID, "%" + keyResultDescription + "%"}
		}

		err := c.db.QueryRow(query, params...).Scan(&finalKeyResultID)
		if err != nil {
			return "❌ Не найден ключевой результат по описанию: " + keyResultDescription, &AddKeyResultProgressFunction, nil
		}
	} else {
		finalKeyResultID = int64(keyResultID)

		checkQuery := `
			SELECT kr.id 
			FROM key_results kr
			JOIN objectives o ON kr.objective_id = o.id
			WHERE kr.id = $1 AND o.user_id = $2
		`
		var checkID int64
		err := c.db.QueryRow(checkQuery, finalKeyResultID, userID).Scan(&checkID)
		if err != nil {
			return "❌ Ключевой результат не найден или не принадлежит пользователю", &AddKeyResultProgressFunction, nil
		}
	}

	type KeyResultData struct {
		Title		string	`db:"title"`
		Target		float64	`db:"target"`
		Unit		string	`db:"unit"`
		Progress	float64	`db:"progress"`
		ObjectiveTitle	string	`db:"objective_title"`
	}

	var krData KeyResultData
	dataQuery := `
		SELECT kr.title, kr.target, kr.unit, kr.progress, o.title as objective_title
		FROM key_results kr
		JOIN objectives o ON kr.objective_id = o.id
		WHERE kr.id = $1
	`
	err := c.db.QueryRow(dataQuery, finalKeyResultID).Scan(
		&krData.Title, &krData.Target, &krData.Unit, &krData.Progress, &krData.ObjectiveTitle,
	)
	if err != nil {
		return "❌ Не удалось получить данные ключевого результата", &AddKeyResultProgressFunction, nil
	}

	newProgress := krData.Progress + progress
	if newProgress > krData.Target {
		newProgress = krData.Target
	}

	updateQuery := `
		UPDATE key_results 
		SET progress = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err = c.db.Exec(updateQuery, newProgress, finalKeyResultID)
	if err != nil {
		logrus.Errorf("Ошибка обновления прогресса: %v", err)
		return "❌ Не удалось обновить прогресс", &AddKeyResultProgressFunction, nil
	}

	completionPercent := (newProgress / krData.Target) * 100
	if completionPercent > 100 {
		completionPercent = 100
	}

	response := fmt.Sprintf("📈 **Прогресс обновлен!**\n\n")
	response += fmt.Sprintf("🔑 **Ключевой результат:** %s\n", krData.Title)
	response += fmt.Sprintf("🎯 **Цель:** %s\n", krData.ObjectiveTitle)
	response += fmt.Sprintf("➕ **Добавлено:** +%.1f %s\n", progress, krData.Unit)
	response += fmt.Sprintf("📊 **Текущий прогресс:** %.1f / %.1f %s (%.1f%%)\n\n",
		newProgress, krData.Target, krData.Unit, completionPercent)

	if completionPercent >= 100 {
		response += "🎉 **Поздравляю! Ключевой результат выполнен на 100%!**\n"
		response += "🏆 Отличная работа! Продолжай в том же духе!"
	} else if completionPercent >= 75 {
		response += "🔥 **Отлично! Ты почти у цели!**\n"
		response += "💪 Осталось совсем немного!"
	} else if completionPercent >= 50 {
		response += "💪 **Хороший прогресс!**\n"
		response += "⚡ Продолжай двигаться к цели!"
	} else {
		response += "🚀 **Каждый шаг приближает к цели!**\n"
		response += "💯 Продолжай работать, результат не заставит себя ждать!"
	}

	return response, &AddKeyResultProgressFunction, nil
}

func (c *ChatGPTService) handleCreateTask(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	logrus.Infof("Создание задачи для пользователя %d с аргументами: %+v", userID, args)

	title, _ := args["title"].(string)
	target, _ := args["target"].(float64)
	unit, _ := args["unit"].(string)
	deadline, _ := args["deadline"].(string)
	keyResultID, hasID := args["key_result_id"].(float64)
	keyResultDescription, _ := args["key_result_description"].(string)
	objectiveDescription, _ := args["objective_description"].(string)

	if title == "" || target <= 0 || unit == "" || deadline == "" {
		return "❌ Не указаны обязательные параметры для создания задачи", &CreateTaskFunction, nil
	}

	var finalKeyResultID int64

	if !hasID || keyResultID <= 0 {
		if keyResultDescription == "" {
			return "❌ Не указан ID или описание ключевого результата", &CreateTaskFunction, nil
		}

		var query string
		var params []interface{}

		if objectiveDescription != "" {

			query = `
				SELECT kr.id 
				FROM key_results kr
				JOIN objectives o ON kr.objective_id = o.id
				WHERE o.user_id = $1 
				AND LOWER(kr.title) LIKE LOWER($2)
				AND LOWER(o.title) LIKE LOWER($3)
				ORDER BY kr.created_at DESC LIMIT 1
			`
			params = []interface{}{userID, "%" + keyResultDescription + "%", "%" + objectiveDescription + "%"}
		} else {

			query = `
				SELECT kr.id 
				FROM key_results kr
				JOIN objectives o ON kr.objective_id = o.id
				WHERE o.user_id = $1 
				AND LOWER(kr.title) LIKE LOWER($2)
				ORDER BY kr.created_at DESC LIMIT 1
			`
			params = []interface{}{userID, "%" + keyResultDescription + "%"}
		}

		err := c.db.QueryRow(query, params...).Scan(&finalKeyResultID)
		if err != nil {
			return "❌ Не найден ключевой результат по описанию: " + keyResultDescription, &CreateTaskFunction, nil
		}
	} else {
		finalKeyResultID = int64(keyResultID)

		checkQuery := `
			SELECT kr.id 
			FROM key_results kr
			JOIN objectives o ON kr.objective_id = o.id
			WHERE kr.id = $1 AND o.user_id = $2
		`
		var checkID int64
		err := c.db.QueryRow(checkQuery, finalKeyResultID, userID).Scan(&checkID)
		if err != nil {
			return "❌ Ключевой результат не найден или не принадлежит пользователю", &CreateTaskFunction, nil
		}
	}

	insertQuery := `
		INSERT INTO tasks (key_result_id, title, target, unit, deadline, status, progress, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'active', 0, NOW(), NOW())
		RETURNING id
	`

	var taskID int64
	err := c.db.QueryRow(insertQuery, finalKeyResultID, title, target, unit, deadline).Scan(&taskID)
	if err != nil {
		logrus.Errorf("Ошибка создания задачи: %v", err)
		return "❌ Не удалось создать задачу", &CreateTaskFunction, nil
	}

	type TaskContextData struct {
		KeyResultTitle	string	`db:"kr_title"`
		ObjectiveTitle	string	`db:"obj_title"`
	}

	var contextData TaskContextData
	contextQuery := `
		SELECT kr.title as kr_title, o.title as obj_title
		FROM key_results kr
		JOIN objectives o ON kr.objective_id = o.id
		WHERE kr.id = $1
	`
	c.db.QueryRow(contextQuery, finalKeyResultID).Scan(&contextData.KeyResultTitle, &contextData.ObjectiveTitle)

	response := fmt.Sprintf("📋 **Задача создана!**\n\n")
	response += fmt.Sprintf("📝 **Название:** %s\n", title)
	response += fmt.Sprintf("🔑 **Ключевой результат:** %s\n", contextData.KeyResultTitle)
	response += fmt.Sprintf("🎯 **Цель:** %s\n", contextData.ObjectiveTitle)
	response += fmt.Sprintf("📊 **Целевое значение:** %.1f %s\n", target, unit)
	response += fmt.Sprintf("📅 **Дедлайн:** %s\n", deadline)
	response += fmt.Sprintf("🆔 **ID:** %d\n\n", taskID)
	response += "🚀 Отличная детализация! Jarvis поможет отслеживать выполнение этой задачи и автоматически обновит прогресс по ключевому результату."

	return response, &CreateTaskFunction, nil
}

func (c *ChatGPTService) handleAddTaskProgress(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	logrus.Infof("Добавление прогресса задачи для пользователя %d с аргументами: %+v", userID, args)

	taskID, hasID := args["task_id"].(float64)
	taskDescription, _ := args["task_description"].(string)
	keyResultDescription, _ := args["key_result_description"].(string)
	progress, _ := args["progress"].(float64)

	if progress <= 0 {
		return "❌ Прогресс должен быть больше нуля", &AddTaskProgressFunction, nil
	}

	var finalTaskID int64

	if !hasID || taskID <= 0 {
		if taskDescription == "" {
			return "❌ Не указан ID или описание задачи", &AddTaskProgressFunction, nil
		}

		var query string
		var params []interface{}

		if keyResultDescription != "" {

			query = `
				SELECT t.id 
				FROM tasks t
				JOIN key_results kr ON t.key_result_id = kr.id
				JOIN objectives o ON kr.objective_id = o.id
				WHERE o.user_id = $1 
				AND LOWER(t.title) LIKE LOWER($2)
				AND LOWER(kr.title) LIKE LOWER($3)
				ORDER BY t.created_at DESC LIMIT 1
			`
			params = []interface{}{userID, "%" + taskDescription + "%", "%" + keyResultDescription + "%"}
		} else {

			query = `
				SELECT t.id 
				FROM tasks t
				JOIN key_results kr ON t.key_result_id = kr.id
				JOIN objectives o ON kr.objective_id = o.id
				WHERE o.user_id = $1 
				AND LOWER(t.title) LIKE LOWER($2)
				ORDER BY t.created_at DESC LIMIT 1
			`
			params = []interface{}{userID, "%" + taskDescription + "%"}
		}

		err := c.db.QueryRow(query, params...).Scan(&finalTaskID)
		if err != nil {
			return "❌ Не найдена задача по описанию: " + taskDescription, &AddTaskProgressFunction, nil
		}
	} else {
		finalTaskID = int64(taskID)

		checkQuery := `
			SELECT t.id 
			FROM tasks t
			JOIN key_results kr ON t.key_result_id = kr.id
			JOIN objectives o ON kr.objective_id = o.id
			WHERE t.id = $1 AND o.user_id = $2
		`
		var checkID int64
		err := c.db.QueryRow(checkQuery, finalTaskID, userID).Scan(&checkID)
		if err != nil {
			return "❌ Задача не найдена или не принадлежит пользователю", &AddTaskProgressFunction, nil
		}
	}

	type TaskData struct {
		Title		string	`db:"title"`
		Target		float64	`db:"target"`
		Unit		string	`db:"unit"`
		Progress	float64	`db:"progress"`
		KeyResultID	int64	`db:"key_result_id"`
		KeyResultTitle	string	`db:"kr_title"`
		ObjectiveTitle	string	`db:"obj_title"`
	}

	var taskData TaskData
	dataQuery := `
		SELECT t.title, t.target, t.unit, t.progress, t.key_result_id,
		       kr.title as kr_title, o.title as obj_title
		FROM tasks t
		JOIN key_results kr ON t.key_result_id = kr.id
		JOIN objectives o ON kr.objective_id = o.id
		WHERE t.id = $1
	`
	err := c.db.QueryRow(dataQuery, finalTaskID).Scan(
		&taskData.Title, &taskData.Target, &taskData.Unit, &taskData.Progress,
		&taskData.KeyResultID, &taskData.KeyResultTitle, &taskData.ObjectiveTitle,
	)
	if err != nil {
		return "❌ Не удалось получить данные задачи", &AddTaskProgressFunction, nil
	}

	newTaskProgress := taskData.Progress + progress
	if newTaskProgress > taskData.Target {
		newTaskProgress = taskData.Target
	}

	updateTaskQuery := `
		UPDATE tasks 
		SET progress = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err = c.db.Exec(updateTaskQuery, newTaskProgress, finalTaskID)
	if err != nil {
		logrus.Errorf("Ошибка обновления прогресса задачи: %v", err)
		return "❌ Не удалось обновить прогресс задачи", &AddTaskProgressFunction, nil
	}

	var krUpdateInfo string
	taskCompletionPercent := (newTaskProgress / taskData.Target) * 100
	if taskCompletionPercent >= 100 && newTaskProgress == taskData.Target {

		addKRProgressQuery := `
			UPDATE key_results 
			SET progress = progress + $1, updated_at = NOW()
			WHERE id = $2
		`
		_, err = c.db.Exec(addKRProgressQuery, taskData.Target, taskData.KeyResultID)
		if err == nil {
			krUpdateInfo = "\n🎯 **Автоматически обновлен ключевой результат:** +" + fmt.Sprintf("%.1f %s", taskData.Target, taskData.Unit)
		}
	}

	response := fmt.Sprintf("📋 **Прогресс задачи обновлен!**\n\n")
	response += fmt.Sprintf("📝 **Задача:** %s\n", taskData.Title)
	response += fmt.Sprintf("🔑 **Ключевой результат:** %s\n", taskData.KeyResultTitle)
	response += fmt.Sprintf("🎯 **Цель:** %s\n", taskData.ObjectiveTitle)
	response += fmt.Sprintf("➕ **Добавлено:** +%.1f %s\n", progress, taskData.Unit)
	response += fmt.Sprintf("📊 **Текущий прогресс:** %.1f / %.1f %s (%.1f%%)\n",
		newTaskProgress, taskData.Target, taskData.Unit, taskCompletionPercent)

	if krUpdateInfo != "" {
		response += krUpdateInfo
	}

	response += "\n"

	if taskCompletionPercent >= 100 {
		response += "🎉 **Задача выполнена на 100%!**\n"
		response += "🏆 Превосходно! Двигаемся к ключевому результату!"
	} else if taskCompletionPercent >= 75 {
		response += "🔥 **Почти готово!**\n"
		response += "💪 Финишная прямая!"
	} else if taskCompletionPercent >= 50 {
		response += "💪 **Хороший темп!**\n"
		response += "⚡ Продолжай в том же духе!"
	} else {
		response += "🚀 **Каждый шаг важен!**\n"
		response += "💯 Отличная работа над задачей!"
	}

	return response, &AddTaskProgressFunction, nil
}

func (c *ChatGPTService) handleGetTasks(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	logrus.Infof("Получение задач для пользователя %d с аргументами: %+v", userID, args)

	keyResultID, hasKRID := args["key_result_id"].(float64)
	objectiveID, _ := args["objective_id"].(string)

	var query string
	var params []interface{}

	if hasKRID && keyResultID > 0 {

		query = `
			SELECT t.id, t.title, t.target, t.unit, t.progress, t.deadline, t.status,
			       kr.title as kr_title, o.title as obj_title
			FROM tasks t
			JOIN key_results kr ON t.key_result_id = kr.id
			JOIN objectives o ON kr.objective_id = o.id
			WHERE t.key_result_id = $1 AND o.user_id = $2
			ORDER BY t.created_at DESC
		`
		params = []interface{}{int64(keyResultID), userID}
	} else if objectiveID != "" {

		query = `
			SELECT t.id, t.title, t.target, t.unit, t.progress, t.deadline, t.status,
			       kr.title as kr_title, o.title as obj_title
			FROM tasks t
			JOIN key_results kr ON t.key_result_id = kr.id
			JOIN objectives o ON kr.objective_id = o.id
			WHERE o.id = $1 AND o.user_id = $2
			ORDER BY kr.created_at, t.created_at DESC
		`
		params = []interface{}{objectiveID, userID}
	} else {

		query = `
			SELECT t.id, t.title, t.target, t.unit, t.progress, t.deadline, t.status,
			       kr.title as kr_title, o.title as obj_title
			FROM tasks t
			JOIN key_results kr ON t.key_result_id = kr.id
			JOIN objectives o ON kr.objective_id = o.id
			WHERE o.user_id = $1
			ORDER BY t.created_at DESC
			LIMIT 20
		`
		params = []interface{}{userID}
	}

	rows, err := c.db.Query(query, params...)
	if err != nil {
		logrus.Errorf("Ошибка получения задач: %v", err)
		return "❌ Не удалось получить задачи из базы данных", &GetTasksFunction, nil
	}
	defer rows.Close()

	response := "📋 **Твои задачи:**\n\n"
	taskCount := 0
	currentKR := ""

	for rows.Next() {
		var taskID int64
		var title, unit, deadline, status, krTitle, objTitle string
		var target, progress float64

		err := rows.Scan(&taskID, &title, &target, &unit, &progress, &deadline, &status,
			&krTitle, &objTitle)
		if err != nil {
			continue
		}

		taskCount++

		if objectiveID != "" && krTitle != currentKR {
			if currentKR != "" {
				response += "\n"
			}
			response += fmt.Sprintf("🔑 **%s**\n", krTitle)
			currentKR = krTitle
		}

		statusEmoji := "📋"
		switch status {
		case "completed":
			statusEmoji = "✅"
		case "paused":
			statusEmoji = "⏸️"
		case "active":
			statusEmoji = "🔄"
		}

		completionPercent := (progress / target) * 100
		if completionPercent > 100 {
			completionPercent = 100
		}

		response += fmt.Sprintf("%s **%s**\n", statusEmoji, title)
		response += fmt.Sprintf("   📊 %.1f / %.1f %s (%.1f%%) | 📅 %s\n",
			progress, target, unit, completionPercent, deadline)

		if objectiveID == "" && hasKRID == false {

			response += fmt.Sprintf("   🎯 %s → 🔑 %s\n", objTitle, krTitle)
		}

		response += "\n"
	}

	if taskCount == 0 {
		response = "📋 **Задач пока нет**\n\n"
		if hasKRID && keyResultID > 0 {
			response += "💡 Создай задачи для детализации ключевого результата!"
		} else if objectiveID != "" {
			response += "💡 Создай задачи для ключевых результатов этой цели!"
		} else {
			response += "💡 Создай цели и разбей их на ключевые результаты и задачи!"
		}
	} else {
		response += fmt.Sprintf("📈 **Всего задач:** %d", taskCount)

		if taskCount >= 10 {
			response += "\n🔥 Wow! Ты отлично детализируешь свои цели!"
		} else if taskCount >= 5 {
			response += "\n💪 Хорошая детализация целей!"
		} else {
			response += "\n🚀 Отличное начало!"
		}
	}

	return response, &GetTasksFunction, nil
}

func (c *ChatGPTService) handleDeleteObjective(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	logrus.Infof("Удаление цели для пользователя %d с аргументами: %+v", userID, args)

	objectiveID, _ := args["objective_id"].(string)
	objectiveDescription, _ := args["objective_description"].(string)
	confirm, _ := args["confirm"].(bool)

	if !confirm {
		return "❌ Для удаления цели необходимо подтверждение. Скажи что-то вроде 'да, удали цель'", &DeleteObjectiveFunction, nil
	}

	if objectiveID == "" && objectiveDescription != "" {
		query := `SELECT id FROM objectives WHERE user_id = $1 AND LOWER(title) LIKE LOWER($2) ORDER BY created_at DESC LIMIT 1`
		err := c.db.QueryRow(query, userID, "%"+objectiveDescription+"%").Scan(&objectiveID)
		if err != nil {
			return "❌ Не найдена цель по описанию: " + objectiveDescription, &DeleteObjectiveFunction, nil
		}
	}

	if objectiveID == "" {
		return "❌ Не указана цель для удаления", &DeleteObjectiveFunction, nil
	}

	var objectiveTitle string
	titleQuery := `SELECT title FROM objectives WHERE id = $1 AND user_id = $2`
	err := c.db.QueryRow(titleQuery, objectiveID, userID).Scan(&objectiveTitle)
	if err != nil {
		return "❌ Цель не найдена или не принадлежит пользователю", &DeleteObjectiveFunction, nil
	}

	deleteQuery := `DELETE FROM objectives WHERE id = $1 AND user_id = $2`
	result, err := c.db.Exec(deleteQuery, objectiveID, userID)
	if err != nil {
		logrus.Errorf("Ошибка удаления цели: %v", err)
		return "❌ Не удалось удалить цель из базы данных", &DeleteObjectiveFunction, nil
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return "❌ Цель не найдена", &DeleteObjectiveFunction, nil
	}

	response := fmt.Sprintf("🗑️ **Цель удалена!**\n\n")
	response += fmt.Sprintf("📋 **Удаленная цель:** %s\n\n", objectiveTitle)
	response += "⚠️ Все связанные ключевые результаты и задачи также удалены"

	return response, &DeleteObjectiveFunction, nil
}

func (c *ChatGPTService) handleDeleteKeyResult(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	logrus.Infof("Удаление ключевого результата для пользователя %d с аргументами: %+v", userID, args)

	keyResultID, hasID := args["key_result_id"].(float64)
	keyResultDescription, _ := args["key_result_description"].(string)
	objectiveDescription, _ := args["objective_description"].(string)
	confirm, _ := args["confirm"].(bool)

	if !confirm {
		return "❌ Для удаления ключевого результата необходимо подтверждение. Скажи что-то вроде 'да, удали ключевой результат'", &DeleteKeyResultFunction, nil
	}

	var finalKeyResultID int64

	if !hasID || keyResultID <= 0 {
		if keyResultDescription == "" {
			return "❌ Не указан ID или описание ключевого результата", &DeleteKeyResultFunction, nil
		}

		var query string
		var params []interface{}

		if objectiveDescription != "" {

			query = `
				SELECT kr.id 
				FROM key_results kr
				JOIN objectives o ON kr.objective_id = o.id
				WHERE o.user_id = $1 
				AND LOWER(kr.title) LIKE LOWER($2)
				AND LOWER(o.title) LIKE LOWER($3)
				ORDER BY kr.created_at DESC LIMIT 1
			`
			params = []interface{}{userID, "%" + keyResultDescription + "%", "%" + objectiveDescription + "%"}
		} else {

			query = `
				SELECT kr.id 
				FROM key_results kr
				JOIN objectives o ON kr.objective_id = o.id
				WHERE o.user_id = $1 
				AND LOWER(kr.title) LIKE LOWER($2)
				ORDER BY kr.created_at DESC LIMIT 1
			`
			params = []interface{}{userID, "%" + keyResultDescription + "%"}
		}

		err := c.db.QueryRow(query, params...).Scan(&finalKeyResultID)
		if err != nil {
			return "❌ Не найден ключевой результат по описанию: " + keyResultDescription, &DeleteKeyResultFunction, nil
		}
	} else {
		finalKeyResultID = int64(keyResultID)
	}

	var krTitle, objectiveTitle string
	titleQuery := `
		SELECT kr.title, o.title
		FROM key_results kr
		JOIN objectives o ON kr.objective_id = o.id
		WHERE kr.id = $1 AND o.user_id = $2
	`
	err := c.db.QueryRow(titleQuery, finalKeyResultID, userID).Scan(&krTitle, &objectiveTitle)
	if err != nil {
		return "❌ Ключевой результат не найден или не принадлежит пользователю", &DeleteKeyResultFunction, nil
	}

	deleteQuery := `DELETE FROM key_results WHERE id = $1`
	result, err := c.db.Exec(deleteQuery, finalKeyResultID)
	if err != nil {
		logrus.Errorf("Ошибка удаления ключевого результата: %v", err)
		return "❌ Не удалось удалить ключевой результат", &DeleteKeyResultFunction, nil
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return "❌ Ключевой результат не найден", &DeleteKeyResultFunction, nil
	}

	response := fmt.Sprintf("🗑️ **Ключевой результат удален!**\n\n")
	response += fmt.Sprintf("🔑 **Удаленный KR:** %s\n", krTitle)
	response += fmt.Sprintf("🎯 **Цель:** %s\n\n", objectiveTitle)
	response += "⚠️ Все связанные задачи также удалены"

	return response, &DeleteKeyResultFunction, nil
}

func (c *ChatGPTService) handleDeleteTask(args map[string]interface{}, userID int64) (string, *ChatGPTFunction, error) {
	logrus.Infof("Удаление задачи для пользователя %d с аргументами: %+v", userID, args)

	taskID, hasID := args["task_id"].(float64)
	taskDescription, _ := args["task_description"].(string)
	keyResultDescription, _ := args["key_result_description"].(string)
	confirm, _ := args["confirm"].(bool)

	if !confirm {
		return "❌ Для удаления задачи необходимо подтверждение. Скажи что-то вроде 'да, удали задачу'", &DeleteTaskFunction, nil
	}

	var finalTaskID int64

	if !hasID || taskID <= 0 {
		if taskDescription == "" {
			return "❌ Не указан ID или описание задачи", &DeleteTaskFunction, nil
		}

		var query string
		var params []interface{}

		if keyResultDescription != "" {

			query = `
				SELECT t.id 
				FROM tasks t
				JOIN key_results kr ON t.key_result_id = kr.id
				JOIN objectives o ON kr.objective_id = o.id
				WHERE o.user_id = $1 
				AND LOWER(t.title) LIKE LOWER($2)
				AND LOWER(kr.title) LIKE LOWER($3)
				ORDER BY t.created_at DESC LIMIT 1
			`
			params = []interface{}{userID, "%" + taskDescription + "%", "%" + keyResultDescription + "%"}
		} else {

			query = `
				SELECT t.id 
				FROM tasks t
				JOIN key_results kr ON t.key_result_id = kr.id
				JOIN objectives o ON kr.objective_id = o.id
				WHERE o.user_id = $1 
				AND LOWER(t.title) LIKE LOWER($2)
				ORDER BY t.created_at DESC LIMIT 1
			`
			params = []interface{}{userID, "%" + taskDescription + "%"}
		}

		err := c.db.QueryRow(query, params...).Scan(&finalTaskID)
		if err != nil {
			return "❌ Не найдена задача по описанию: " + taskDescription, &DeleteTaskFunction, nil
		}
	} else {
		finalTaskID = int64(taskID)
	}

	var taskTitle, krTitle, objectiveTitle string
	titleQuery := `
		SELECT t.title, kr.title, o.title
		FROM tasks t
		JOIN key_results kr ON t.key_result_id = kr.id
		JOIN objectives o ON kr.objective_id = o.id
		WHERE t.id = $1 AND o.user_id = $2
	`
	err := c.db.QueryRow(titleQuery, finalTaskID, userID).Scan(&taskTitle, &krTitle, &objectiveTitle)
	if err != nil {
		return "❌ Задача не найдена или не принадлежит пользователю", &DeleteTaskFunction, nil
	}

	deleteQuery := `DELETE FROM tasks WHERE id = $1`
	result, err := c.db.Exec(deleteQuery, finalTaskID)
	if err != nil {
		logrus.Errorf("Ошибка удаления задачи: %v", err)
		return "❌ Не удалось удалить задачу", &DeleteTaskFunction, nil
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return "❌ Задача не найдена", &DeleteTaskFunction, nil
	}

	response := fmt.Sprintf("🗑️ **Задача удалена!**\n\n")
	response += fmt.Sprintf("📝 **Удаленная задача:** %s\n", taskTitle)
	response += fmt.Sprintf("🔑 **Ключевой результат:** %s\n", krTitle)
	response += fmt.Sprintf("🎯 **Цель:** %s", objectiveTitle)

	return response, &DeleteTaskFunction, nil
}

CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id           BIGINT PRIMARY KEY,
    username     VARCHAR(255),
    first_name   VARCHAR(255),
    role         VARCHAR(20) NOT NULL DEFAULT 'free',
    personality_type     VARCHAR(50) DEFAULT 'balanced',
    motivation_style     VARCHAR(50) DEFAULT 'achievement',
    communication_style  VARCHAR(50) DEFAULT 'friendly',
    activity_level       VARCHAR(50) DEFAULT 'moderate',
    preferred_reminder_time TIME DEFAULT '09:00:00',
    timezone             VARCHAR(50) DEFAULT 'UTC+3',
    total_points         INT DEFAULT 0,
    level                INT DEFAULT 1,
    streak_days          INT DEFAULT 0,
    last_activity_date   DATE,
    jarvis_settings      JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS events (
    id              VARCHAR(36) PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    reminder_sent   BOOLEAN DEFAULT FALSE,
    google_event_id VARCHAR(255),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS meetings (
    id             VARCHAR(36) PRIMARY KEY,
    initiator_id   BIGINT NOT NULL REFERENCES users(id),
    participant_id BIGINT NOT NULL REFERENCES users(id),
    title          VARCHAR(255) NOT NULL,
    description    TEXT,
    start_time     TIMESTAMPTZ NOT NULL,
    end_time       TIMESTAMPTZ NOT NULL,
    confirmed      BOOLEAN DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transactions (
    id        VARCHAR(36) PRIMARY KEY,
    user_id   BIGINT NOT NULL REFERENCES users(id),
    amount    DECIMAL(12,2) NOT NULL,
    details   TEXT,
    category  VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS objective_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    color VARCHAR(7) DEFAULT '#3498db',
    icon VARCHAR(50) DEFAULT '🎯',
    description TEXT,
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS objective_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category_id INT REFERENCES objective_categories(id),
    description TEXT,
    template_data JSONB,
    difficulty_level INT DEFAULT 3 CHECK (difficulty_level >= 1 AND difficulty_level <= 5),
    estimated_days INT DEFAULT 30,
    usage_count INT DEFAULT 0,
    success_rate FLOAT DEFAULT 0.0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS objectives (
    id        VARCHAR(36) PRIMARY KEY,
    user_id   BIGINT NOT NULL REFERENCES users(id),
    title     VARCHAR(255) NOT NULL,
    sphere    VARCHAR(255),
    period    VARCHAR(50)  NOT NULL,
    deadline  TIMESTAMPTZ,
    category_id INT REFERENCES objective_categories(id),
    priority INT DEFAULT 3 CHECK (priority >= 1 AND priority <= 5),
    status VARCHAR(20) DEFAULT 'active',
    parent_objective_id VARCHAR(36) REFERENCES objectives(id),
    estimated_hours FLOAT DEFAULT 0,
    actual_hours FLOAT DEFAULT 0,
    difficulty_level INT DEFAULT 3 CHECK (difficulty_level >= 1 AND difficulty_level <= 5),
    motivation_text TEXT,
    reward_text TEXT,
    celebration_message TEXT,
    tags TEXT[] DEFAULT '{}',
    template_id INT REFERENCES objective_templates(id),
    auto_created BOOLEAN DEFAULT FALSE,
    completion_date TIMESTAMPTZ,
    success_score FLOAT DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS key_results (
    id           BIGSERIAL PRIMARY KEY,
    objective_id VARCHAR(36) NOT NULL REFERENCES objectives(id) ON DELETE CASCADE,
    title        VARCHAR(255) NOT NULL,
    target       DECIMAL(12,2) NOT NULL,
    unit         VARCHAR(50),
    progress     DECIMAL(12,2) DEFAULT 0,
    priority INT DEFAULT 3 CHECK (priority >= 1 AND priority <= 5),
    status VARCHAR(20) DEFAULT 'active',
    estimated_hours FLOAT DEFAULT 0,
    actual_hours FLOAT DEFAULT 0,
    difficulty_level INT DEFAULT 3,
    is_milestone BOOLEAN DEFAULT FALSE,
    completion_date TIMESTAMPTZ,
    deadline     TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tasks (
    id            BIGSERIAL PRIMARY KEY,
    key_result_id BIGINT NOT NULL REFERENCES key_results(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    target        DOUBLE PRECISION NOT NULL,
    unit          TEXT NOT NULL,
    progress      DOUBLE PRECISION NOT NULL DEFAULT 0,
    priority INT DEFAULT 3 CHECK (priority >= 1 AND priority <= 5),
    status VARCHAR(20) DEFAULT 'active',
    estimated_hours FLOAT DEFAULT 0,
    actual_hours FLOAT DEFAULT 0,
    difficulty_level INT DEFAULT 3,
    is_recurring BOOLEAN DEFAULT FALSE,
    recurrence_pattern VARCHAR(50),
    next_occurrence DATE,
    completion_date TIMESTAMPTZ,
    mood_after_completion INT CHECK (mood_after_completion >= 1 AND mood_after_completion <= 5),
    deadline      TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS google_tokens (
    user_id       BIGINT PRIMARY KEY REFERENCES users(id),
    access_token  TEXT NOT NULL,
    refresh_token TEXT,
    token_type    VARCHAR(50) NOT NULL,
    expiry        TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS google_sync_state (
    user_id        BIGINT PRIMARY KEY,
    last_sync_time TIMESTAMPTZ NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_timestamp_google_sync_state
BEFORE UPDATE ON google_sync_state
FOR EACH ROW EXECUTE PROCEDURE trigger_set_timestamp();

CREATE TABLE IF NOT EXISTS web_users (
    id            BIGSERIAL PRIMARY KEY,
    login         VARCHAR(255) UNIQUE NOT NULL,
    email         VARCHAR(255) UNIQUE,
    phone         VARCHAR(50)  UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    telegram_ids  BIGINT[],
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_timestamp_web_users
BEFORE UPDATE ON web_users
FOR EACH ROW EXECUTE PROCEDURE trigger_set_timestamp();

CREATE TABLE IF NOT EXISTS user_messages (
    id              BIGSERIAL PRIMARY KEY,
    user_identifier VARCHAR(255) NOT NULL,
    message_text    TEXT NOT NULL,
    platform        VARCHAR(50) DEFAULT 'telegram',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_responses (
    id               BIGSERIAL PRIMARY KEY,
    user_message_id  BIGINT NOT NULL REFERENCES user_messages(id) ON DELETE CASCADE,
    response_text    TEXT NOT NULL,
    prompt_tokens    INTEGER,
    completion_tokens INTEGER,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS okr_report_settings (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    report_period    VARCHAR(50) NOT NULL, -- day, week, month
    day_of_week      SMALLINT,            -- 1 (Пн) - 7 (Вс)
    hour             INTEGER NOT NULL,    -- 0-23
    minute           INTEGER NOT NULL,    -- 0-59
    enabled          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_report_sent TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS events_user_id_idx            ON events(user_id);
CREATE INDEX IF NOT EXISTS events_start_time_idx         ON events(start_time);
CREATE INDEX IF NOT EXISTS events_google_event_id_idx    ON events(google_event_id);

CREATE INDEX IF NOT EXISTS meetings_initiator_id_idx     ON meetings(initiator_id);
CREATE INDEX IF NOT EXISTS meetings_participant_id_idx   ON meetings(participant_id);

CREATE INDEX IF NOT EXISTS transactions_user_id_idx      ON transactions(user_id);
CREATE INDEX IF NOT EXISTS transactions_created_at_idx   ON transactions(created_at);

CREATE INDEX IF NOT EXISTS objectives_user_id_idx        ON objectives(user_id);
CREATE INDEX IF NOT EXISTS key_results_objective_id_idx  ON key_results(objective_id);
CREATE INDEX IF NOT EXISTS tasks_key_result_id_idx       ON tasks(key_result_id);

CREATE INDEX IF NOT EXISTS user_messages_user_identifier_idx ON user_messages(user_identifier);
CREATE INDEX IF NOT EXISTS user_messages_created_at_idx      ON user_messages(created_at);
CREATE INDEX IF NOT EXISTS ai_responses_user_message_id_idx  ON ai_responses(user_message_id);

CREATE INDEX IF NOT EXISTS okr_report_settings_user_id_idx  ON okr_report_settings(user_id);

CREATE INDEX IF NOT EXISTS idx_web_users_login             ON web_users(login);
CREATE INDEX IF NOT EXISTS idx_web_users_telegram_ids      ON web_users USING GIN (telegram_ids);

CREATE INDEX IF NOT EXISTS idx_key_results_status ON key_results(status) WHERE status IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status) WHERE status IS NOT NULL;

CREATE TABLE IF NOT EXISTS ai_insights (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    insight_type VARCHAR(50) NOT NULL,
    category VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    action_button_text VARCHAR(100),
    action_data JSONB,
    priority INT DEFAULT 3 CHECK (priority >= 1 AND priority <= 5),
    objective_id VARCHAR(36) REFERENCES objectives(id),
    key_result_id BIGINT REFERENCES key_results(id),
    task_id BIGINT REFERENCES tasks(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    shown_at TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    effectiveness_score FLOAT DEFAULT 0.0
);

CREATE TABLE IF NOT EXISTS habit_tracking (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    objective_id VARCHAR(36) REFERENCES objectives(id),
    key_result_id BIGINT REFERENCES key_results(id),
    task_id BIGINT REFERENCES tasks(id),
    date DATE NOT NULL,
    completed BOOLEAN DEFAULT FALSE,
    completion_percentage FLOAT DEFAULT 0.0,
    time_spent_minutes INT DEFAULT 0,
    mood_before INT CHECK (mood_before >= 1 AND mood_before <= 5),
    mood_after INT CHECK (mood_after >= 1 AND mood_after <= 5),
    energy_level INT CHECK (energy_level >= 1 AND energy_level <= 5),
    notes TEXT,
    weather VARCHAR(50),
    location VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS achievement_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    icon VARCHAR(50) DEFAULT '🏆',
    category VARCHAR(50) DEFAULT 'general',
    points INT DEFAULT 10,
    rarity VARCHAR(20) DEFAULT 'common',
    requirements JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_achievements (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    achievement_id INT REFERENCES achievement_types(id),
    earned_at TIMESTAMPTZ DEFAULT NOW(),
    objective_id VARCHAR(36) REFERENCES objectives(id),
    key_result_id BIGINT REFERENCES key_results(id),
    task_id BIGINT REFERENCES tasks(id),
    progress_when_earned JSONB,
    celebration_shown BOOLEAN DEFAULT FALSE,
    UNIQUE(user_id, achievement_id)
);

CREATE TABLE IF NOT EXISTS user_context (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    context_type VARCHAR(50) NOT NULL,
    context_data JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS user_behavior_patterns (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    pattern_type VARCHAR(50) NOT NULL,
    pattern_data JSONB NOT NULL,
    confidence_score FLOAT DEFAULT 0.0,
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    samples_count INT DEFAULT 1
);

CREATE TABLE IF NOT EXISTS motivation_strategies (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    strategy_type VARCHAR(50) NOT NULL,
    strategy_data JSONB NOT NULL,
    effectiveness_score FLOAT DEFAULT 0.0,
    usage_count INT DEFAULT 0,
    last_used TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS goal_predictions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    objective_id VARCHAR(36) REFERENCES objectives(id),
    key_result_id BIGINT REFERENCES key_results(id),
    task_id BIGINT REFERENCES tasks(id),
    prediction_type VARCHAR(50) NOT NULL,
    predicted_value FLOAT,
    predicted_date DATE,
    confidence_score FLOAT DEFAULT 0.0,
    factors JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    actual_outcome FLOAT,
    actual_date DATE
);

CREATE TABLE IF NOT EXISTS user_teams (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    team_type VARCHAR(50) DEFAULT 'private',
    max_members INT DEFAULT 10,
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS team_members (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT REFERENCES user_teams(id),
    user_id BIGINT REFERENCES users(id),
    role VARCHAR(20) DEFAULT 'member',
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    points_contributed INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    UNIQUE(team_id, user_id)
);

CREATE TABLE IF NOT EXISTS shared_objectives (
    id BIGSERIAL PRIMARY KEY,
    objective_id VARCHAR(36) REFERENCES objectives(id),
    team_id BIGINT REFERENCES user_teams(id),
    shared_by BIGINT REFERENCES users(id),
    can_edit BOOLEAN DEFAULT FALSE,
    can_view_progress BOOLEAN DEFAULT TRUE,
    shared_at TIMESTAMPTZ DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS smart_reminders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    objective_id VARCHAR(36) REFERENCES objectives(id),
    key_result_id BIGINT REFERENCES key_results(id),
    task_id BIGINT REFERENCES tasks(id),
    reminder_type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    is_adaptive BOOLEAN DEFAULT TRUE,
    adaptation_data JSONB,
    priority INT DEFAULT 3 CHECK (priority >= 1 AND priority <= 5),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO objective_categories (name, color, icon, description, sort_order) VALUES
('Карьера и работа', '#3498db', '💼', 'Профессиональное развитие, карьерные цели', 1),
('Здоровье и спорт', '#e74c3c', '💪', 'Физическое здоровье, фитнес, спорт', 2),
('Финансы', '#f39c12', '💰', 'Финансовые цели, инвестиции, бюджет', 3),
('Личностное развитие', '#9b59b6', '🎯', 'Саморазвитие, навыки, образование', 4),
('Семья и отношения', '#e67e22', '👨‍👩‍👧‍👦', 'Семейные цели, отношения', 5),
('Хобби и творчество', '#1abc9c', '🎨', 'Творческие проекты, хобби', 6),
('Путешествия', '#34495e', '✈️', 'Путешествия и приключения', 7),
('Дом и быт', '#95a5a6', '🏠', 'Домашние дела, быт, организация', 8)
ON CONFLICT (name) DO NOTHING;

INSERT INTO achievement_types (name, description, icon, category, points, rarity, requirements) VALUES
('Первый шаг', 'Создал свою первую цель', '🎯', 'goals', 10, 'common', '{"goals_created": 1}'),
('Целеустремленный', 'Создал 5 целей', '🎯', 'goals', 25, 'common', '{"goals_created": 5}'),
('Мастер планирования', 'Создал 25 целей', '📋', 'goals', 100, 'rare', '{"goals_created": 25}'),
('Завершитель', 'Завершил первую цель', '✅', 'completion', 50, 'common', '{"goals_completed": 1}'),
('Надежный исполнитель', 'Завершил 10 целей', '🏆', 'completion', 200, 'rare', '{"goals_completed": 10}'),
('Легенда продуктивности', 'Завершил 50 целей', '👑', 'completion', 1000, 'legendary', '{"goals_completed": 50}'),
('Марафонец', 'Поддерживал серию выполнения 7 дней подряд', '🔥', 'streak', 75, 'common', '{"streak_days": 7}'),
('Несгибаемый', 'Поддерживал серию выполнения 30 дней подряд', '💪', 'streak', 300, 'epic', '{"streak_days": 30}'),
('Перфекционист', 'Выполнил цель на 100%', '⭐', 'quality', 100, 'rare', '{"perfect_completion": 1}'),
('Скоростной', 'Завершил цель досрочно', '⚡', 'speed', 75, 'common', '{"early_completion": 1}'),
('Социальный', 'Поделился целью с командой', '👥', 'social', 50, 'common', '{"shared_goals": 1}'),
('Наставник', 'Помог другу достичь цели', '🤝', 'social', 150, 'rare', '{"helped_friends": 1}')
ON CONFLICT (name) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_objectives_category_user ON objectives(category_id, user_id);
CREATE INDEX IF NOT EXISTS idx_objectives_status_user   ON objectives(status, user_id);
CREATE INDEX IF NOT EXISTS idx_objectives_priority_user ON objectives(priority, user_id);

CREATE INDEX IF NOT EXISTS idx_ai_insights_user_active ON ai_insights(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_ai_insights_type_category ON ai_insights(insight_type, category);

CREATE INDEX IF NOT EXISTS idx_habit_tracking_user_date ON habit_tracking(user_id, date);

CREATE INDEX IF NOT EXISTS idx_user_context_user_active ON user_context(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_behavior_patterns_user_type ON user_behavior_patterns(user_id, pattern_type);
CREATE INDEX IF NOT EXISTS idx_predictions_user_type ON goal_predictions(user_id, prediction_type);

CREATE INDEX IF NOT EXISTS idx_smart_reminders_user_scheduled ON smart_reminders(user_id, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_smart_reminders_active ON smart_reminders(is_active, scheduled_at);
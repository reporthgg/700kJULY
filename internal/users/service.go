package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"telegrambot/internal/auth"

	"github.com/sirupsen/logrus"
)

var (
	ErrUserNotFound				= errors.New("пользователь не найден")
	ErrUserAlreadyExists			= errors.New("пользователь с таким логином уже существует")
	ErrInvalidCredentials			= errors.New("неверный логин или пароль")
	ErrTelegramIDAlreadyLinkedToOtherUser	= errors.New("этот Telegram аккаунт уже привязан к другому веб-пользователю")
	ErrTelegramIDAlreadyLinkedToThisUser	= errors.New("этот Telegram аккаунт уже привязан к вашему веб-профилю")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) RegisterWebUser(ctx context.Context, login, password string, email *string, phone *string) (*WebUser, error) {
	existingUser, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil && !errors.Is(err, sql.ErrNoRows) && existingUser != nil {
		logrus.Errorf("Ошибка при проверке существующего пользователя '%s': %v", login, err)
		return nil, fmt.Errorf("внутренняя ошибка сервера при проверке пользователя")
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		logrus.Errorf("Ошибка хеширования пароля для пользователя '%s': %v", login, err)
		return nil, fmt.Errorf("внутренняя ошибка сервера при хешировании пароля")
	}

	user, err := s.repo.CreateUser(ctx, login, hashedPassword, email, phone)
	if err != nil {
		logrus.Errorf("Ошибка создания пользователя '%s' в репозитории: %v", login, err)
		return nil, fmt.Errorf("внутренняя ошибка сервера при создании пользователя")
	}
	return user, nil
}

func (s *Service) AuthenticateWebUser(ctx context.Context, login, password string) (*WebUser, error) {
	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || user == nil {
			return nil, ErrInvalidCredentials
		}
		logrus.Errorf("Ошибка при получении пользователя '%s' для аутентификации: %v", login, err)
		return nil, fmt.Errorf("внутренняя ошибка сервера при аутентификации")
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (s *Service) GetWebUserByID(ctx context.Context, id int64) (*WebUser, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || user == nil {
			return nil, ErrUserNotFound
		}
		logrus.Errorf("Ошибка при получении пользователя по ID %d: %v", id, err)
		return nil, fmt.Errorf("внутренняя ошибка сервера")
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *Service) LinkTelegramAccount(ctx context.Context, webUserID int64, telegramID int64) error {

	existingWebUserWithThisTelegramID, err := s.repo.GetWebUserByTelegramID(ctx, telegramID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logrus.Errorf("Ошибка при проверке telegram_id %d на существующую привязку: %v", telegramID, err)
		return fmt.Errorf("внутренняя ошибка сервера при проверке привязки Telegram")
	}
	if existingWebUserWithThisTelegramID != nil && existingWebUserWithThisTelegramID.ID != webUserID {
		logrus.Warnf("Попытка привязать telegram_id %d к web_user %d, но он уже привязан к web_user %d",
			telegramID, webUserID, existingWebUserWithThisTelegramID.ID)
		return ErrTelegramIDAlreadyLinkedToOtherUser
	}

	webUser, err := s.repo.GetUserByID(ctx, webUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || webUser == nil {
			logrus.Errorf("Web-пользователь с ID %d не найден при попытке привязки telegram_id %d", webUserID, telegramID)
			return ErrUserNotFound
		}
		logrus.Errorf("Ошибка получения web_user %d для привязки telegram_id %d: %v", webUserID, telegramID, err)
		return fmt.Errorf("внутренняя ошибка сервера")
	}
	if webUser == nil {
		logrus.Errorf("Web-пользователь с ID %d вернулся nil при попытке привязки telegram_id %d", webUserID, telegramID)
		return ErrUserNotFound
	}

	for _, existingTgID := range webUser.TelegramIDs {
		if existingTgID == telegramID {
			logrus.Infof("Telegram ID %d уже привязан к web_user %d.", telegramID, webUserID)
			return ErrTelegramIDAlreadyLinkedToThisUser
		}
	}

	_, err = s.repo.AddTelegramIDToWebUser(ctx, webUserID, telegramID)
	if err != nil {
		logrus.Errorf("Ошибка при добавлении telegram_id %d к web_user %d в репозитории: %v", telegramID, webUserID, err)
		return fmt.Errorf("внутренняя ошибка сервера при привязке Telegram")
	}

	logrus.Infof("Telegram ID %d успешно привязан к web_user %d", telegramID, webUserID)
	return nil
}

func (s *Service) FindWebUserByTelegramID(ctx context.Context, telegramID int64) (*WebUser, error) {
	user, err := s.repo.GetWebUserByTelegramID(ctx, telegramID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || user == nil {
			return nil, ErrUserNotFound
		}
		logrus.Errorf("Ошибка при поиске web_user по telegram_id %d: %v", telegramID, err)
		return nil, fmt.Errorf("внутренняя ошибка сервера при поиске по Telegram ID")
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

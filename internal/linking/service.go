package linking

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	ErrTokenNotFound		= errors.New("токен привязки не найден или истек")
	ErrTokenAlreadyUsed		= errors.New("токен привязки уже был использован")
	ErrFailedToGenerateToken	= errors.New("не удалось сгенерировать токен привязки")
)

const (
	linkTokenTTL		= 10 * time.Minute
	linkTokenLengthBytes	= 16
)

type LinkTokenInfo struct {
	WebUserID	int64
	ExpiresAt	time.Time
	Used		bool
}

type Service struct {
	tokens	map[string]LinkTokenInfo
	mu	sync.RWMutex
}

func NewService() *Service {
	s := &Service{
		tokens: make(map[string]LinkTokenInfo),
	}
	go s.cleanupExpiredTokens()
	return s
}

func (s *Service) GenerateLinkToken(webUserID int64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bytes := make([]byte, linkTokenLengthBytes)
	if _, err := rand.Read(bytes); err != nil {
		logrus.Errorf("Ошибка генерации случайных байт для токена привязки: %v", err)
		return "", ErrFailedToGenerateToken
	}
	token := hex.EncodeToString(bytes)

	s.tokens[token] = LinkTokenInfo{
		WebUserID:	webUserID,
		ExpiresAt:	time.Now().Add(linkTokenTTL),
		Used:		false,
	}
	logrus.Debugf("Сгенерирован токен привязки '%s' для web_user_id %d, истекает в %v", token, webUserID, s.tokens[token].ExpiresAt)
	return token, nil
}

func (s *Service) ValidateAndUseLinkToken(token string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, exists := s.tokens[token]
	if !exists {
		logrus.Warnf("Попытка использовать несуществующий токен привязки: %s", token)
		return 0, ErrTokenNotFound
	}

	if time.Now().After(info.ExpiresAt) {
		logrus.Warnf("Попытка использовать истекший токен привязки: %s (истек %v)", token, info.ExpiresAt)

		delete(s.tokens, token)
		return 0, ErrTokenNotFound
	}

	if info.Used {
		logrus.Warnf("Попытка повторно использовать токен привязки: %s", token)
		return 0, ErrTokenAlreadyUsed
	}

	info.Used = true
	s.tokens[token] = info

	logrus.Infof("Токен привязки '%s' успешно валидирован и использован для web_user_id %d", token, info.WebUserID)
	return info.WebUserID, nil
}

func (s *Service) cleanupExpiredTokens() {
	ticker := time.NewTicker(linkTokenTTL / 2)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, info := range s.tokens {

			if now.After(info.ExpiresAt) || info.Used {
				logrus.Debugf("Очистка токена привязки: %s (истек: %v, использован: %t)", token, now.After(info.ExpiresAt), info.Used)
				delete(s.tokens, token)
			}
		}
		s.mu.Unlock()
	}
}

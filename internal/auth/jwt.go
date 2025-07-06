package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID	int64	`json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateJWTToken(userID int64, signingKey string, expirationTime time.Duration) (string, error) {
	expiration := time.Now().Add(expirationTime)
	claims := &Claims{
		UserID:	userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt:	jwt.NewNumericDate(expiration),
			IssuedAt:	jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(signingKey))
	if err != nil {
		return "", fmt.Errorf("ошибка при подписи токена: %w", err)
	}
	return tokenString, nil
}

func ValidateJWTToken(tokenString string, signingKey string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}
		return []byte(signingKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга токена: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("невалидный токен")
	}

	return claims, nil
}

func JWTMiddleware(next http.Handler, signingKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Отсутствует заголовок Authorization", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Некорректный формат заголовка Authorization (ожидается Bearer <token>)", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		claims, err := ValidateJWTToken(tokenString, signingKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Невалидный токен: %v", err), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value("userID").(int64)
	return userID, ok
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

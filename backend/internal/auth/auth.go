package auth

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	jwtSecret []byte
	secretMu  sync.RWMutex
)

func SetSecret(secret string) {
	secretMu.Lock()
	defer secretMu.Unlock()
	jwtSecret = []byte(secret)
}

func getSecret() []byte {
	secretMu.RLock()
	if len(jwtSecret) > 0 {
		defer secretMu.RUnlock()
		return jwtSecret
	}
	secretMu.RUnlock()

	secretMu.Lock()
	defer secretMu.Unlock()
	if len(jwtSecret) > 0 {
		return jwtSecret
	}
	if s := os.Getenv("JWT_SECRET"); s != "" {
		jwtSecret = []byte(s)
		return jwtSecret
	}
	jwtSecret = []byte("weeky-dev-secret-change-in-production")
	return jwtSecret
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

func GenerateToken(userID int64, email string, isAdmin bool) (string, error) {
	claims := Claims{
		UserID:    userID,
		Email:     email,
		IsAdmin:   isAdmin,
		TokenType: AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getSecret())
}

func GenerateRefreshToken(userID int64, email string, isAdmin bool) (string, error) {
	claims := Claims{
		UserID:    userID,
		Email:     email,
		IsAdmin:   isAdmin,
		TokenType: RefreshToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getSecret())
}

func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return getSecret(), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

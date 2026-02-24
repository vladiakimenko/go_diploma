package auth

import (
	"blog-api/pkg/settings"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// types
type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

// errors
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// config
type JWTConfig struct {
	JWTSecret             string
	AccessTokenTTLMinutes int
	RefreshTokenTTLHours  int
}

func (c *JWTConfig) Setup() []settings.EnvLoadable {
	return []settings.EnvLoadable{
		settings.Item[string]{Name: "JWT_SECRET", Default: settings.NoDefault, Field: &c.JWTSecret},
		settings.Item[int]{Name: "JWT_ACCESS_TOKEN_TTL_MINUTES", Default: 5, Field: &c.AccessTokenTTLMinutes},
		settings.Item[int]{Name: "REFRESH_TOKEN_TTL_HOURS", Default: 24, Field: &c.RefreshTokenTTLHours},
	}
}

// manager
type JWTManager struct {
	config          *JWTConfig
	RefreshTokenTTL time.Duration
}

func NewJWTManager(cfg *JWTConfig) *JWTManager {
	return &JWTManager{
		config:          cfg,
		RefreshTokenTTL: time.Duration(cfg.RefreshTokenTTLHours) * time.Hour,
	}
}

func (m *JWTManager) GenerateToken(ctx context.Context, userID int) (string, time.Time, error) {
	if err := ctx.Err(); err != nil {
		return "", time.Time{}, err
	}

	now := time.Now()
	expires := now.Add(time.Duration(m.config.AccessTokenTTLMinutes) * time.Minute)

	claims := Claims{
		UserID: userID,
	}
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(expires)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(m.config.JWTSecret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, expires, nil
}


func (m *JWTManager) KeyFunc(token *jwt.Token) (any, error) {
	if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
		return nil, fmt.Errorf("wrong encryption algorythm: %v", token.Header["alg"])
	}
	return []byte(m.config.JWTSecret), nil
}

func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	claims := Claims{}
	if _, err := jwt.ParseWithClaims(tokenString, &claims, m.KeyFunc); err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return &claims, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	return &claims, nil
}

func (m *JWTManager) RefreshToken(ctx context.Context, tokenString string) (string, time.Time, error) {
    claims, err := m.ValidateToken(tokenString)
    if err != nil && !errors.Is(err, ErrExpiredToken) {
        return "", time.Time{}, err
    }
    return m.GenerateToken(ctx, claims.UserID)
}


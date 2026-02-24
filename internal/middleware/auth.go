package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"blog-api/pkg/auth"
	"blog-api/pkg/exception"
)

const UserIDKey contextKey = "userID"

const AuthorizationHeader string = "Authorization"
const AuthHeaderPrefix string = "Bearer "

type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get(AuthorizationHeader)
			if authHeader == "" {
				exception.WriteApiError(w, exception.TokenNotProvidedError(
					fmt.Sprintf("%s header missing", AuthorizationHeader)),
				)
				return
			}

			if !strings.HasPrefix(authHeader, AuthHeaderPrefix) {
				exception.WriteApiError(w, exception.TokenInvalidError("Invalid authorization scheme"))
				return
			}

			token := strings.TrimPrefix(authHeader, AuthHeaderPrefix)
			if token == "" {
				exception.WriteApiError(w, exception.TokenNotProvidedError(
					fmt.Sprintf("Failed to extract token from %s header", AuthorizationHeader)),
				)
				return
			}

			claims, err := m.jwtManager.ValidateToken(token)
			if err != nil {
				if errors.Is(err, auth.ErrExpiredToken) {
					exception.WriteApiError(w, exception.TokenExpiredError("token expired"))
					return
				}
				exception.WriteApiError(w, exception.TokenInvalidError("token invalid"))
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}

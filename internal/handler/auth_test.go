package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"

	"blog-api/internal/middleware"
	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/internal/service"
	"blog-api/pkg/auth"
	"blog-api/pkg/logging"
)

// data
var (
	validUUID, _   = uuid.NewV4()
	expiredUUID, _ = uuid.NewV4()
)

// setup
func setupAuthTestRouter() http.Handler {
	logging.Init(&logging.LoggerConfig{})

	userRepo := repository.NewInMemoryUserRepo()
	refreshRepo := repository.NewInMemoryRefreshTokenRepo()

	refreshRepo.Store(validUUID.String(), 1, time.Now().Add(time.Hour))
	refreshRepo.Store(expiredUUID.String(), 1, time.Now().Add(-time.Hour))

	jwtCfg := &auth.JWTConfig{RefreshTokenTTLHours: 24}
	jwtManager := auth.NewJWTManager(jwtCfg)

	passCfg := &auth.PasswordConfig{
		MinLength:         6,
		Cost:              4,
		CaseShiftRequired: false,
		DigitsRequired:    false,
		SymbolsRequired:   false,
	}
	passManager := auth.NewPasswordManager(passCfg)

	userService := service.NewUserService(userRepo, refreshRepo, jwtManager, passManager)
	authHandler := NewAuthHandler(userService)

	router := chi.NewRouter()

	router.Post("/api/register", middleware.ModelBodyMiddleware[model.UserCreateRequest](authHandler.Register))
	router.Post("/api/login", middleware.ModelBodyMiddleware[model.UserLoginRequest](authHandler.Login))
	router.Post("/api/refresh", middleware.ModelBodyMiddleware[model.RefreshTokenRequest](authHandler.Refresh))

	protected := chi.NewRouter()
	protected.Use(mockAuthMiddleware())
	protected.Get("/api/users/{userID}", authHandler.GetProfile)
	router.Mount("/", protected)

	return router
}

// validators
func validateHeaders(t *testing.T, res *http.Response) {
	if res.StatusCode == http.StatusNoContent {
		return
	}
	contentType := res.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("incorrect 'Content-Type' header: %q", contentType)
	}
}

func validateStatus(t *testing.T, result *http.Response, wanted int) {
	status := result.StatusCode
	if status != wanted {
		t.Errorf("expected status: %d, recieved: %d", wanted, status)
	}
}

func validateJsonResponse[T any](t *testing.T, result *http.Response) {
	bodyBytes, err := io.ReadAll(result.Body)
	if err != nil {
		t.Errorf("failed to read the response body: %v", err)
		return
	}

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
		t.Errorf("failed to parse the body as json: %v", err)
		t.Logf("Response body: %s", string(bodyBytes))
		return
	}

	errCountBefore := t.Failed()
	validateStructFields(reflect.TypeOf((*T)(nil)).Elem(), rawMap, t)

	if t.Failed() && !errCountBefore {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, bodyBytes, "", "  "); err != nil {
			t.Logf("Response body: %s", string(bodyBytes))
		} else {
			t.Logf("Response body:\n%s", pretty.String())
		}
	}
}

func validateStructFields(typ reflect.Type, rawMap map[string]json.RawMessage, t *testing.T) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// skip ignored JSON fields
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		jsonKey := strings.Split(jsonTag, ",")[0]
		if jsonKey == "" {
			jsonKey = field.Name
		}

		value, exists := rawMap[jsonKey]

		// skip optional fields if omitempty
		if !exists && strings.Contains(jsonTag, "omitempty") {
			continue
		}

		if !exists {
			t.Errorf("field %q is missing in the response body", jsonKey)
			continue
		}

		// skip null pointers
		if string(value) == "null" {
			continue
		}

		// recursively validate nested structs
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct {
			var nestedMap map[string]json.RawMessage
			if err := json.Unmarshal(value, &nestedMap); err == nil {
				validateStructFields(fieldType, nestedMap, t)
			}
		}
	}
}

// helpers
func setActorID(ctx context.Context, id int) context.Context {
	return context.WithValue(ctx, middleware.UserIDKey, id)
}

func mockAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actorID, ok := getActorID(r.Context())
			if !ok || actorID == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// tests
func TestAuthHandler(t *testing.T) {
	router := setupAuthTestRouter()

	tests := []struct {
		name       string
		method     string
		url        string
		body       any
		actorID    int
		wantStatus int
		validate   any
	}{
		// register
		{"Register user", http.MethodPost, "/api/register", model.UserCreateRequest{Username: "tester", Email: "tester@example.com", Password: "password"}, 0, http.StatusCreated, model.TokenResponse{}},

		// login
		{"Login user", http.MethodPost, "/api/login", model.UserLoginRequest{Email: "tester@example.com", Password: "password"}, 0, http.StatusOK, model.TokenResponse{}},

		// refresh
		{"Refresh token valid", http.MethodPost, "/api/refresh", model.RefreshTokenRequest{RefreshToken: validUUID.String()}, 0, http.StatusOK, model.TokenResponse{}},
		{"Refresh token expired", http.MethodPost, "/api/refresh", model.RefreshTokenRequest{RefreshToken: expiredUUID.String()}, 0, http.StatusBadRequest, nil},

		// profile
		{"Get profile valid", http.MethodGet, "/api/users/1", nil, 1, http.StatusOK, model.User{}},
		{"Get profile forbidden", http.MethodGet, "/api/users/2", nil, 1, http.StatusForbidden, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.url, bytes.NewReader(bodyBytes))
			if tt.actorID != 0 {
				req = req.WithContext(setActorID(req.Context(), tt.actorID))
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			validateStatus(t, res, tt.wantStatus)
			validateHeaders(t, res)

			if res.StatusCode < http.StatusBadRequest && tt.validate != nil {
				switch tt.validate.(type) {
				case model.User:
					validateJsonResponse[model.User](t, res)
				case model.TokenResponse:
					validateJsonResponse[model.TokenResponse](t, res)
				}
			}
		})
	}
}

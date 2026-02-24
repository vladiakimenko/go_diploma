package handler

import (
	"net/http"
	"strconv"

	"blog-api/internal/model"
	"blog-api/internal/service"
	"blog-api/pkg/exception"

	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	userService *service.UserService
}

func NewAuthHandler(userService *service.UserService) *AuthHandler {
	return &AuthHandler{
		userService: userService,
	}
}

// POST /api/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	body, ok := getParsedBody[model.UserCreateRequest](r)
	if !ok {
		exception.WriteApiError(w, exception.BadRequestError("Invalid request body"))
		return
	}

	result, err := h.userService.Register(r.Context(), body)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// POST /api/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	body, ok := getParsedBody[model.UserLoginRequest](r)
	if !ok {
		exception.WriteApiError(w, exception.BadRequestError("Invalid request body"))
		return
	}

	result, err := h.userService.Login(r.Context(), body)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// POST /api/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	body, ok := getParsedBody[model.RefreshTokenRequest](r)
	if !ok {
		exception.WriteApiError(w, exception.BadRequestError("Invalid request body"))
		return
	}

	result, err := h.userService.RefreshToken(r.Context(), body)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GET /api/users/{userID}
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {

	actorID, ok := getActorID(r.Context())
	if !ok {
		exception.WriteApiError(w, exception.InternalServerError("Auth misconfigured"))
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		exception.WriteApiError(w, exception.BadRequestError("Invalid user ID"))
		return
	}

	if actorID != userID {
		exception.WriteApiError(w, exception.ForbiddenError("Access deined"))
		return
	}

	result, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		exception.WriteApiError(w, mapServiceError(err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

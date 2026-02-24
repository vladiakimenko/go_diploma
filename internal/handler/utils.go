package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"blog-api/internal/middleware"
	"blog-api/internal/model"
	"blog-api/internal/service"
	"blog-api/pkg/exception"
	"blog-api/pkg/logging"
	"blog-api/pkg/validator"
)

var logger = logging.L()

func writeJSON(
	w http.ResponseWriter,
	status int,
	v any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		exception.WriteApiError(
			w,
			exception.InternalServerError(err.Error()),
		)
	}
}

func getParsedBody[T any](
	r *http.Request,
) (*T, bool) {
	data := r.Context().Value(middleware.ParsedBodyKey).(*T)
	if err := validator.ModelValidate(data); err != nil {
		logger.Error("%s", err.Error())
		return nil, false
	}
	return data, true
}

func getActorID(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(middleware.UserIDKey).(int)
	if !ok {
		logger.Error("%s missing in context", middleware.UserIDKey)
		return 0, false
	}
	return userID, true
}

func getPaginationParams(r *http.Request) (*model.PaginationParams, bool) {
	pagination := &model.PaginationParams{}
	query := r.URL.Query()

	if l := query.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			pagination.Limit = &v
		} else {
			logger.Error("Invalid limit value: %v", err)
			return nil, false
		}
	}

	if o := query.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			pagination.Offset = &v
		} else {
			logger.Error("Invalid offset value: %v", err)
			return nil, false
		}
	}

	if err := validator.ModelValidate(pagination); err != nil {
		logger.Error("%s", err.Error())
		return nil, false
	}

	return pagination, true
}

func writePaginatedJSON[T any](
	w http.ResponseWriter,
	status int,
	data T,
	pagination *model.PaginationParams,
	total int,
) {
	limit, offset := 0, 0
	if pagination != nil {
		if pagination.Limit != nil {
			limit = *pagination.Limit
		}
		if pagination.Offset != nil {
			offset = *pagination.Offset
		}
	}

	resp := model.PaginatedResponse[T]{
		Data:   data,
		Limit:  limit,
		Offset: offset,
		Total:  total,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		exception.WriteApiError(
			w,
			exception.InternalServerError(err.Error()),
		)
	}
}

func mapServiceError(err error) *exception.ApiError {
	switch {

	// user
	case errors.Is(err, service.ErrUserNotFound):
		return exception.NotFoundError(err.Error())

	case errors.Is(err, service.ErrUserAlreadyExists):
		return exception.ConflictError(err.Error())

	case errors.Is(err, service.ErrInvalidCredentials):
		return exception.BadRequestError(err.Error())

	case errors.Is(err, service.ErrInvalidRefreshToken):
		return exception.BadRequestError(err.Error())

	case errors.Is(err, service.ErrRefreshTokenExpired):
		return exception.BadRequestError(err.Error())

	case errors.Is(err, service.ErrWeakPassword):
		return exception.BadRequestError(err.Error())

	case errors.Is(err, service.ErrTokenGeneration):
		return exception.InternalServerError(err.Error())

	case errors.Is(err, service.ErrPasswordHash):
		return exception.InternalServerError(err.Error())

	// post
	case errors.Is(err, service.ErrPostNotFound):
		return exception.NotFoundError(err.Error())

	// comment
	case errors.Is(err, service.ErrCommentNotFound):
		return exception.NotFoundError(err.Error())

	// common
	case errors.Is(err, service.ErrForbidden):
		return exception.ForbiddenError(err.Error())

	case errors.Is(err, service.ErrDatabase):
		return exception.DatabaseError(err.Error())

	default:
		return exception.InternalServerError("unknown error")
	}
}

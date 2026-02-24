package exception

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
)

// Error codes
const (
	// Auth
	AuthTokenMissing = 10
	AuthTokenInvalid = 11
	AuthTokenExpired = 12

	// Safety
	TooManyRequests   = 20
	RequestBodyTooBig = 21

	// Access
	AuthForbidden = 30
	UserBlocked   = 31

	// Validation
	InvalidBodyStructure = 40

	// Resource
	NotExist     = 50
	AlreadyExist = 51

	// Server
	Programming    = 60
	Database       = 61
	ForeignService = 62

	// Unknown
	Unknown = 0
)

type ApiError struct {
	Status  int    `json:"-"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ApiError implements the error interface
func (e *ApiError) Error() string {
	return fmt.Sprintf("Api error: %d(%d): %s", e.Status, e.Code, e.Message)
}

func (e *ApiError) MarshalJSON() ([]byte, error) {
	type jsonStruct struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	return json.Marshal(
		jsonStruct{
			Code:    e.Code,
			Message: e.Message,
		},
	)
}

func NewApiError(status int, code int, msg string) *ApiError {
	return &ApiError{
		Status:  status,
		Code:    code,
		Message: msg,
	}
}

// client
func TokenNotProvidedError(msg string) *ApiError {
	return NewApiError(http.StatusUnauthorized, AuthTokenMissing, msg)
}

func TokenInvalidError(msg string) *ApiError {
	return NewApiError(http.StatusUnauthorized, AuthTokenInvalid, msg)
}

func TokenExpiredError(msg string) *ApiError {
	return NewApiError(http.StatusUnauthorized, AuthTokenExpired, msg)
}

func TooManyRequestsError(msg string) *ApiError {
	return NewApiError(http.StatusTooManyRequests, TooManyRequests, msg)
}

func RequestBodyTooLargeError(msg string) *ApiError {
	return NewApiError(http.StatusRequestEntityTooLarge, RequestBodyTooBig, msg)
}

func BadRequestError(msg string) *ApiError {
	return NewApiError(http.StatusBadRequest, InvalidBodyStructure, msg)
}

func UnauthorizedError(msg string) *ApiError {
	return NewApiError(http.StatusUnauthorized, AuthTokenInvalid, msg)
}

func ForbiddenError(msg string) *ApiError {
	return NewApiError(http.StatusForbidden, AuthForbidden, msg)
}

func NotFoundError(msg string) *ApiError {
	return NewApiError(http.StatusNotFound, NotExist, msg)
}

func ConflictError(msg string) *ApiError {
	return NewApiError(http.StatusConflict, AlreadyExist, msg)
}

// server
func InternalServerError(msg string) *ApiError {
	return NewApiError(http.StatusInternalServerError, Programming, msg)
}

func DatabaseError(msg string) *ApiError {
	return NewApiError(http.StatusInternalServerError, Database, msg)
}

func ForeignServiceError(msg string) *ApiError {
	return NewApiError(http.StatusInternalServerError, ForeignService, msg)
}

func UnknownError(msg string) *ApiError {
	return NewApiError(http.StatusInternalServerError, Unknown, msg)
}

// PanicRecoverMiddleware recovers from panics, logs and returns a json Unknown ApiError: 500(0)
func PanicRecoverMiddleware(next http.Handler, logFunc func(msg string, args ...any)) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if logFunc != nil {
						logFunc("panic recovered: %v\n%s", rec, string(debug.Stack()))
					}
					apiErr := UnknownError("Unexpected server error occurred")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(apiErr.Status)
					json.NewEncoder(w).Encode(apiErr)
				}
			}()
			next.ServeHTTP(w, r)
		},
	)
}

// WriteApiError serializes and writes a known ApiError to the response
func WriteApiError(w http.ResponseWriter, apiErr *ApiError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	json.NewEncoder(w).Encode(apiErr)
}

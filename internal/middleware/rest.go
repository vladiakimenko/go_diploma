package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"blog-api/pkg/exception"
)

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		},
	)
}

func AllowedMethodsMiddleware(method string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.Method != method {
					exception.WriteApiError(w, exception.BadRequestError("Method "+r.Method+" not allowed"))
					return
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

func PanicRecoverMiddleware(next http.Handler) http.Handler {
	return exception.PanicRecoverMiddleware(
		next, func(msg string, args ...any) {
			log.Printf(msg, args...)
		},
	)
}

const MaxBodySize int64 = 10 * 1024 * 1024
const ParsedBodyKey contextKey = "parsedBody"

func ModelBodyMiddleware[T any](next func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, MaxBodySize)
			defer r.Body.Close()

			var data T
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()

			if err := decoder.Decode(&data); err != nil {
				if _, ok := err.(*http.MaxBytesError); ok {
					exception.WriteApiError(w, exception.RequestBodyTooLargeError("Request body too large"))
					return
				}
				exception.WriteApiError(w, exception.RequestBodyTooLargeError("Could not parse body as JSON"))
				return
			}

			if decoder.More() {
				exception.WriteApiError(w, exception.BadRequestError("More than one JSON object in payload"))
				return
			}

			ctx := context.WithValue(r.Context(), ParsedBodyKey, &data)
			r = r.WithContext(ctx)
		}

		next(w, r)
	}
}

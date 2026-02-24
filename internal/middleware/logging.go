package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
)

const XRayHeader string = "X-XRay-ID"

const XRayContextKey contextKey = "xRay"

type ResponseWriterWrapper struct {
	http.ResponseWriter
	status int
}

func (w *ResponseWriterWrapper) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &ResponseWriterWrapper{ResponseWriter: w}
			next.ServeHTTP(wrapped, r)
			status := wrapped.status
			if status == 0 { // if never called http writes 200
				status = http.StatusOK
			}
			xrayID := ""
			if val := r.Context().Value(XRayContextKey); val != nil {
				if s, ok := val.(string); ok {
					xrayID = s
				}
			}
			duration := time.Since(start)
			log.Printf("%s %s - status: %d - duration: %s - x_ray: %s", r.Method, r.RequestURI, status, duration, xrayID)
		},
	)
}

func XRayMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			xrayID := r.Header.Get(XRayHeader)
			if xrayID == "" {
				xrayID = uuid.Must(uuid.NewV4()).String()
			}

			ctx := context.WithValue(r.Context(), XRayContextKey, xrayID)
			w.Header().Set(XRayHeader, xrayID)

			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}

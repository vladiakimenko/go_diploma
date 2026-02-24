package middleware

import (
	"net/http"

	"blog-api/pkg/exception"
	"blog-api/pkg/settings"
	"blog-api/pkg/throttle"
)

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (c *RedisConfig) Setup() []settings.EnvLoadable {
	return []settings.EnvLoadable{
		settings.Item[string]{Name: "REDIS_HOST", Default: "localhost", Field: &c.Host},
		settings.Item[int]{Name: "REDIS_PORT", Default: 6379, Field: &c.Port},
		settings.Item[string]{Name: "REDIS_PASSWORD", Default: "", Field: &c.Password},
		settings.Item[int]{Name: "REDIS_DB", Default: 0, Field: &c.DB},
	}
}

func RateLimiterMiddleware(throttler *throttle.RedisThrottler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ip := throttle.GetClientIP(r)

				allowed, err := throttler.Allow(r.Context(), ip)
				if err != nil {
					exception.WriteApiError(w, exception.ForeignServiceError("Throttling broker connection failed"))
					return
				}

				if !allowed {
					exception.WriteApiError(w, exception.TooManyRequestsError("Too many requests"))
					return
				}

				next.ServeHTTP(w, r)
			},
		)
	}
}

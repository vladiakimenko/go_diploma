package throttle

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var sharedClient *redis.Client

type RedisThrottler struct {
	redisClient *redis.Client
	Limit       int
	Window      time.Duration
	KeyPrefix   string
}

func InitRedis(addr, password string, db int) {
	sharedClient = redis.NewClient(
		&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sharedClient.Ping(ctx).Err(); err != nil {
		panic(fmt.Errorf("failed to connect to Redis at %s: %w", addr, err))
	}
}

func NewThrottler(keyPrefix string, limit int, window time.Duration) *RedisThrottler {
	return &RedisThrottler{
		redisClient: sharedClient,
		Limit:       limit,
		Window:      window,
		KeyPrefix:   keyPrefix,
	}
}

func (r *RedisThrottler) getKey(clientID string) string {
	return fmt.Sprintf("%s:%s", r.KeyPrefix, clientID)
}

func (r *RedisThrottler) Allow(ctx context.Context, clientID string) (bool, error) {
	key := r.getKey(clientID)
	now := time.Now().Unix()

	pipe := r.redisClient.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", now-int64(r.Window.Seconds())))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
	pipe.Expire(ctx, key, r.Window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count, _ := countCmd.Result()
	return count < int64(r.Limit), nil
}

func GetClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	trusted := os.Getenv("TRUSTED_PROXIES")
	if trusted == "" {
		return host
	}

	remoteIP := net.ParseIP(host)
	if remoteIP == nil {
		return host
	}

	trustedList := strings.Split(trusted, ",")
	isTrusted := false
	for _, cidr := range trustedList {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			if cidr == host {
				isTrusted = true
				break
			}
			continue
		}
		if network.Contains(remoteIP) {
			isTrusted = true
			break
		}
	}

	if !isTrusted {
		return host
	}

	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return host
	}
	parts := strings.Split(xff, ",")
	clientIP := strings.TrimSpace(parts[0])
	if net.ParseIP(clientIP) != nil {
		return clientIP
	}

	return host
}

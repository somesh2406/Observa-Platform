package ratelimit

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter reports whether a request for tenantID may proceed.
// Errors mean the quota could not be checked reliably; callers must fail closed.
type Limiter interface {
	Allow(ctx context.Context, tenantID string) (bool, error)
}

// RedisOptions configures the shared Redis-backed tenant quota.
type RedisOptions struct {
	Address     string
	Username    string
	Password    string
	DB          int
	Requests    int
	Window      time.Duration
	DialTimeout time.Duration
	TLS         bool
}

// RedisLimiter uses a shared Redis counter so all ingestion replicas enforce
// the same tenant quota.
type RedisLimiter struct {
	client   *redis.Client
	requests int
	window   time.Duration
}

// incrementScript atomically increments a tenant's fixed-window counter.
// The expiration is set only when the counter is created, so subsequent
// requests cannot extend the window indefinitely.
var incrementScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
    redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

func NewRedisLimiter(o RedisOptions) (*RedisLimiter, error) {
	if strings.TrimSpace(o.Address) == "" {
		return nil, errors.New("Redis limiter address is required")
	}
	if o.DB < 0 {
		return nil, errors.New("Redis limiter DB must not be negative")
	}
	if o.Requests < 1 {
		return nil, errors.New("Redis limiter requests must be positive")
	}
	if o.Window <= 0 {
		return nil, errors.New("Redis limiter window must be positive")
	}
	if o.Window.Milliseconds() < 1 {
		return nil, errors.New("Redis limiter window must be at least 1ms")
	}
	if o.DialTimeout <= 0 {
		o.DialTimeout = 2 * time.Second
	}

	options := &redis.Options{
		Addr:         strings.TrimSpace(o.Address),
		Username:     o.Username,
		Password:     o.Password,
		DB:           o.DB,
		DialTimeout:  o.DialTimeout,
		ReadTimeout:  o.DialTimeout,
		WriteTimeout: o.DialTimeout,
	}

	if o.TLS {
		options.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := redis.NewClient(options)

	// Fail startup if Redis is unavailable or credentials/TLS are incorrect.
	// Do not silently start with an ineffective quota implementation.
	ctx, cancel := context.WithTimeout(context.Background(), o.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect to tenant rate-limit Redis: %w", err)
	}

	return &RedisLimiter{
		client:   client,
		requests: o.Requests,
		window:   o.Window,
	}, nil
}

func (l *RedisLimiter) Allow(ctx context.Context, tenantID string) (bool, error) {
	if l == nil || l.client == nil {
		return false, errors.New("Redis limiter is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if strings.TrimSpace(tenantID) == "" {
		return false, errors.New("tenant id is required for rate limiting")
	}

	// Hash tenant IDs so raw tenant identifiers do not appear in Redis keys.
	digest := sha256.Sum256([]byte(tenantID))
	key := "observa:ingestion:tenant-limit:v1:" +
		hex.EncodeToString(digest[:])

	count, err := incrementScript.Run(
		ctx,
		l.client,
		[]string{key},
		l.window.Milliseconds(),
	).Int64()
	if err != nil {
		return false, fmt.Errorf("execute tenant rate-limit script: %w", err)
	}

	return count <= int64(l.requests), nil
}

// Close releases Redis connections. Call it during application shutdown.
func (l *RedisLimiter) Close() error {
	if l == nil || l.client == nil {
		return nil
	}
	return l.client.Close()
}

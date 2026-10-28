package ratelimit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func newTestRedisLimiter(t *testing.T, requests int, window time.Duration) *RedisLimiter {
	t.Helper()

	address := strings.TrimSpace(os.Getenv("OBSERVA_TEST_REDIS_ADDR"))
	if address == "" {
		t.Skip("set OBSERVA_TEST_REDIS_ADDR to run Redis integration tests")
	}

	limiter, err := NewRedisLimiter(RedisOptions{
		Address:     address,
		Password:    os.Getenv("OBSERVA_TEST_REDIS_PASSWORD"),
		DB:          0,
		Requests:    requests,
		Window:      window,
		DialTimeout: 2 * time.Second,
		TLS:         false,
	})
	if err != nil {
		t.Fatalf("initialize Redis limiter: %v", err)
	}

	t.Cleanup(func() {
		if err := limiter.Close(); err != nil {
			t.Errorf("close Redis limiter: %v", err)
		}
	})

	return limiter
}

func TestRedisLimiterEnforcesQuotaPerTenant(t *testing.T) {
	limiter := newTestRedisLimiter(t, 1, time.Minute)
	ctx := context.Background()

	tenantA := fmt.Sprintf("test-tenant-a-%d", time.Now().UnixNano())
	tenantB := fmt.Sprintf("test-tenant-b-%d", time.Now().UnixNano())

	allowed, err := limiter.Allow(ctx, tenantA)
	if err != nil || !allowed {
		t.Fatalf("first tenant A request: allowed=%v err=%v", allowed, err)
	}

	allowed, err = limiter.Allow(ctx, tenantA)
	if err != nil || allowed {
		t.Fatalf("second tenant A request should be rejected: allowed=%v err=%v", allowed, err)
	}

	allowed, err = limiter.Allow(ctx, tenantB)
	if err != nil || !allowed {
		t.Fatalf("first tenant B request should have its own quota: allowed=%v err=%v", allowed, err)
	}
}

func TestRedisLimiterRejectsBlankTenant(t *testing.T) {
	limiter := newTestRedisLimiter(t, 5, time.Minute)

	if _, err := limiter.Allow(context.Background(), " "); err == nil {
		t.Fatal("expected blank tenant ID to fail")
	}
}

func TestRedisLimiterRejectsCancelledContext(t *testing.T) {
	limiter := newTestRedisLimiter(t, 5, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := limiter.Allow(ctx, "test-cancelled-tenant"); err == nil {
		t.Fatal("expected cancelled context to fail")
	}
}

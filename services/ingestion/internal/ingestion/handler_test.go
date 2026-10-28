package ingestion

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

type fakePublisher struct {
	publishErr error
	calls      int
	readyErr   error
	tenantID   string
	sourceID   string
	sessionID  string
}

func (f *fakePublisher) Publish(_ context.Context, tenantID, sourceID, _ string, _ time.Time, sessionID string, _ []byte) error {
	f.calls++
	f.tenantID, f.sourceID, f.sessionID = tenantID, sourceID, sessionID
	return f.publishErr
}
func (f *fakePublisher) Ready(context.Context) error { return f.readyErr }
func (f *fakePublisher) Close()                      {}

const validEvent = `{"specversion":"1.0","id":"test-event-1","source":"https://bank.example/app","type":"com.observa.user_action","time":"2026-10-16T10:30:12.450Z","datacontenttype":"application/json","dataschema":"https://somesh2406.github.io/observa-event-contracts/schemas/event-data-v1.schema.json","data":{"event_name":"loan.submit"}}`
const validSessionEvent = `{"specversion":"1.0","id":"test-event-2","source":"https://bank.example/app","type":"com.observa.user_action","time":"2026-10-16T10:30:12.450Z","datacontenttype":"application/json","dataschema":"https://somesh2406.github.io/observa-event-contracts/schemas/event-data-v1.schema.json","data":{"event_name":"loan.submit","context":{"session_id":"sess-1"}}}`

func testHandler(p *fakePublisher, max int64) http.Handler {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	router := chi.NewRouter()
	NewHandler(p, Identity{TenantID: "tenant-test", SourceID: "source-test"}, true, nil, max, time.Second, logger).Register(router)
	return router
}

func TestValidEventAcceptedOnlyAfterPublisherSuccess(t *testing.T) {
	p := &fakePublisher{}
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(validEvent))
	req.Header.Set("Content-Type", "application/cloudevents+json")
	rr := httptest.NewRecorder()
	testHandler(p, 1024*1024).ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if p.calls != 1 {
		t.Fatalf("publisher calls=%d, want 1", p.calls)
	}
}

func TestInvalidEventIsNotPublished(t *testing.T) {
	p := &fakePublisher{}
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(`{"specversion":"0.3"}`))
	req.Header.Set("Content-Type", "application/cloudevents+json")
	rr := httptest.NewRecorder()
	testHandler(p, 1024).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
	if p.calls != 0 {
		t.Fatalf("publisher called for invalid event")
	}
}

func TestPublicationFailureNeverReturnsAccepted(t *testing.T) {
	p := &fakePublisher{publishErr: errors.New("broker down")}
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(validEvent))
	req.Header.Set("Content-Type", "application/cloudevents+json")
	rr := httptest.NewRecorder()
	testHandler(p, 1024*1024).ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestOversizedBodyRejected(t *testing.T) {
	p := &fakePublisher{}
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(validEvent))
	req.Header.Set("Content-Type", "application/cloudevents+json")
	rr := httptest.NewRecorder()
	testHandler(p, 8).ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d", rr.Code)
	}
	if p.calls != 0 {
		t.Fatalf("publisher called for oversized event")
	}
}

func TestContentTypeRequired(t *testing.T) {
	p := &fakePublisher{}
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(validEvent))
	rr := httptest.NewRecorder()
	testHandler(p, 1024*1024).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestEmptyBodyRejectedWithoutPublishing(t *testing.T) {
	p := &fakePublisher{}
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader("  \n"))
	req.Header.Set("Content-Type", "application/cloudevents+json")
	rr := httptest.NewRecorder()
	testHandler(p, 1024).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if p.calls != 0 {
		t.Fatal("publisher called for empty body")
	}
}

func TestReadyReturnsUnavailableWhenKafkaPingFails(t *testing.T) {
	p := &fakePublisher{readyErr: errors.New("broker unavailable")}
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()
	testHandler(p, 1024).ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestLivenessDoesNotRequireKafka(t *testing.T) {
	p := &fakePublisher{readyErr: errors.New("broker unavailable")}
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rr := httptest.NewRecorder()
	testHandler(p, 1024).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestMalformedJSONRejectedWithoutPublishing(t *testing.T) {
	p := &fakePublisher{}
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(`{"specversion":`))
	req.Header.Set("Content-Type", "application/cloudevents+json")
	rr := httptest.NewRecorder()
	testHandler(p, 1024).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if p.calls != 0 {
		t.Fatal("publisher called for malformed JSON")
	}
}

func TestGatewayIdentityRequiredOutsideUnauthenticatedDevelopmentMode(t *testing.T) {
	p := &fakePublisher{}
	router := chi.NewRouter()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	NewHandler(p, Identity{}, false, nil, 1024*1024, time.Second, logger).Register(router)
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(validEvent))
	req.Header.Set("Content-Type", "application/cloudevents+json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if p.calls != 0 {
		t.Fatal("publisher called without trusted identity")
	}
}

func TestGatewayIdentityAndSessionIDReachPublisher(t *testing.T) {
	p := &fakePublisher{}
	router := chi.NewRouter()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	NewHandler(p, Identity{}, false, nil, 1024*1024, time.Second, logger).Register(router)
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(validSessionEvent))
	req.Header.Set("Content-Type", "application/cloudevents+json")
	req.Header.Set("X-Observa-Tenant", "tenant-from-jwt")
	req.Header.Set("X-Observa-Source", "source-from-jwt")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if p.tenantID != "tenant-from-jwt" || p.sourceID != "source-from-jwt" || p.sessionID != "sess-1" {
		t.Fatalf("publisher identity/session = %q/%q/%q", p.tenantID, p.sourceID, p.sessionID)
	}
}

type fakeLimiter struct {
	calls   int
	allowed bool
	err     error
}

func (f *fakeLimiter) Allow(
	ctx context.Context,
	tenantID string,
) (bool, error) {
	f.calls++
	return f.allowed, f.err
}

func TestTenantLimiterRunsAfterEventValidation(t *testing.T) {
	p := &fakePublisher{}
	logger := slog.New(
		slog.NewTextHandler(
			os.Stderr,
			&slog.HandlerOptions{Level: slog.LevelError},
		),
	)

	router := chi.NewRouter()
	limiter := &fakeLimiter{allowed: true}

	NewHandler(
		p,
		Identity{TenantID: "tenant-test", SourceID: "source-test"},
		true,
		limiter,
		1024*1024,
		time.Second,
		logger,
	).Register(router)

	invalid := httptest.NewRequest(
		http.MethodPost,
		"/v1/events",
		strings.NewReader(`{"specversion":"0.3"}`),
	)
	invalid.Header.Set("Content-Type", "application/cloudevents+json")

	invalidRR := httptest.NewRecorder()
	router.ServeHTTP(invalidRR, invalid)

	if invalidRR.Code != http.StatusBadRequest {
		t.Fatalf("invalid event status=%d, want 400", invalidRR.Code)
	}

	if limiter.calls != 0 {
		t.Fatalf("limiter called for invalid event: calls=%d, want 0", limiter.calls)
	}

	// First valid request is allowed.
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/events",
		strings.NewReader(validEvent),
	)
	req.Header.Set("Content-Type", "application/cloudevents+json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("first valid request status=%d, want 202; body=%s", rr.Code, rr.Body.String())
	}

	if limiter.calls != 1 {
		t.Fatalf("limiter calls=%d, want 1", limiter.calls)
	}
}

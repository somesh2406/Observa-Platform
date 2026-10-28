package ingestion

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/observa-platform/observa/libs/eventcontract"
	"github.com/observa-platform/observa/services/ingestion/internal/publisher"
	"github.com/observa-platform/observa/services/ingestion/internal/ratelimit"
)

const cloudEventsJSON = "application/cloudevents+json"

type Identity struct {
	TenantID string
	SourceID string
}

type Handler struct {
	publisher            publisher.EventPublisher
	identity             Identity
	allowUnauthenticated bool
	limiter              ratelimit.Limiter
	maxBodyBytes         int64
	publishTimeout       time.Duration
	logger               *slog.Logger
	now                  func() time.Time
	newID                func() (string, error)
}

func NewHandler(p publisher.EventPublisher, identity Identity, allowUnauthenticated bool, limiter ratelimit.Limiter, maxBodyBytes int64, publishTimeout time.Duration, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		publisher:            p,
		identity:             identity,
		allowUnauthenticated: allowUnauthenticated,
		limiter:              limiter,
		maxBodyBytes:         maxBodyBytes,
		publishTimeout:       publishTimeout,
		logger:               logger,
		now:                  time.Now,
		newID:                randomID,
	}
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.publisher.Ready(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "Kafka is not currently reachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != cloudEventsJSON {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/cloudevents+json")
		return
	}

	if r.ContentLength > h.maxBodyBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds the configured limit")
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, h.maxBodyBytes))
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds the configured limit")
		} else {
			writeError(w, http.StatusBadRequest, "invalid_request_body", "Request body could not be read")
		}
		return
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_event", "Request body must contain one event")
		return
	}

	event, err := eventcontract.Decode(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_event", "Event does not conform to the Observa event contract")
		return
	}

	// Tenant identity is resolved only after parsing the CloudEvent. In production,
	// these headers must be overwritten from validated JWT claims by Envoy and the
	// ingestion Service must only be reachable from the gateway network path.
	identity := h.identity
	if !h.allowUnauthenticated {
		identity = Identity{
			TenantID: strings.TrimSpace(r.Header.Get("X-Observa-Tenant")),
			SourceID: strings.TrimSpace(r.Header.Get("X-Observa-Source")),
		}
		if identity.TenantID == "" || identity.SourceID == "" {
			writeError(w, http.StatusUnauthorized, "missing_trusted_identity", "Verified tenant and source identity are required")
			return
		}
	}

	if h.limiter == nil {
		writeError(
			w,
			http.StatusServiceUnavailable,
			"rate_limiter_unavailable",
			"Tenant quota could not be checked",
		)
		return
	}

	allowed, limitErr := h.limiter.Allow(r.Context(), identity.TenantID)
	if limitErr != nil {
		h.logger.Error("tenant rate limiter unavailable", "tenant_id", identity.TenantID, "error", limitErr)
		writeError(w, http.StatusServiceUnavailable, "rate_limiter_unavailable", "Tenant quota could not be checked")
		return
	}
	if !allowed {
		writeError(w, http.StatusTooManyRequests, "tenant_rate_limit_exceeded", "Tenant ingestion quota exceeded")
		return
	}

	requestID, err := h.newID()
	if err != nil {
		h.logger.Error("generate ingestion id failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "The request could not be processed")
		return
	}

	receivedAt := h.now().UTC()
	ctx, cancel := context.WithTimeout(r.Context(), h.publishTimeout)
	defer cancel()

	sessionID := ""
	if event.Data.Context != nil && event.Data.Context.SessionID != nil {
		sessionID = *event.Data.Context.SessionID
	}
	// This handler returns 202 only after broker confirmation. Because delivery is
	// at-least-once, consumers outside this service must deduplicate by event.id.
	if err := h.publisher.Publish(ctx, identity.TenantID, identity.SourceID, requestID, receivedAt, sessionID, body); err != nil {
		h.logger.Warn("event publication unconfirmed", "ingestion_id", requestID, "event_id", event.ID, "error", err)
		writeError(w, http.StatusServiceUnavailable, "publication_unconfirmed", "Event publication was not confirmed; retry with the same event id")
		return
	}

	h.logger.Info("event accepted", "ingestion_id", requestID, "event_id", event.ID, "tenant_id", identity.TenantID, "source_id", identity.SourceID)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":       "accepted",
		"id":           event.ID,
		"ingestion_id": requestID,
		"received_at":  receivedAt.Format(time.RFC3339Nano),
	})
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

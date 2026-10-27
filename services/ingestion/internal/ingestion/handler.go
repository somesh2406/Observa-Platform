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
)

const cloudEventsJSON = "application/cloudevents+json"

type Identity struct {
	TenantID string
	SourceID string
}

type Handler struct {
	publisher      publisher.EventPublisher
	identity       Identity
	maxBodyBytes   int64
	publishTimeout time.Duration
	logger         *slog.Logger
	now            func() time.Time
	newID          func() (string, error)
}

func NewHandler(p publisher.EventPublisher, identity Identity, maxBodyBytes int64, publishTimeout time.Duration, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		publisher:      p,
		identity:       identity,
		maxBodyBytes:   maxBodyBytes,
		publishTimeout: publishTimeout,
		logger:         logger,
		now:            time.Now,
		newID:          randomID,
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

	requestID, err := h.newID()
	if err != nil {
		h.logger.Error("generate ingestion id failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "The request could not be processed")
		return
	}

	receivedAt := h.now().UTC()
	ctx, cancel := context.WithTimeout(r.Context(), h.publishTimeout)
	defer cancel()

	if err := h.publisher.Publish(ctx, h.identity.TenantID, h.identity.SourceID, requestID, receivedAt, event.ID, body); err != nil {
		h.logger.Warn("event publication unconfirmed", "ingestion_id", requestID, "event_id", event.ID, "error", err)
		writeError(w, http.StatusServiceUnavailable, "publication_unconfirmed", "Event publication was not confirmed; retry with the same event id")
		return
	}

	h.logger.Info("event accepted", "ingestion_id", requestID, "event_id", event.ID, "tenant_id", h.identity.TenantID, "source_id", h.identity.SourceID)

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

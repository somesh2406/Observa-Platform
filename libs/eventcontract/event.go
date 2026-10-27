// Package eventcontract validates the Observa CloudEvents 1.0 structured JSON profile.
// It intentionally has no transport, Kafka, storage, or authentication dependencies.
package eventcontract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	CloudEventsSpecVersion = "1.0"
	JSONContentType        = "application/json"
	EventDataSchema        = "https://somesh2406.github.io/observa-event-contracts/schemas/event-data-v1.schema.json"
	MaxEventIDLength       = 128
	MaxSourceLength        = 512
	MaxEventNameLength     = 128
	MaxAttributes          = 64
)

var (
	eventNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._:-][a-z0-9]+)*$`)
	traceIDPattern   = regexp.MustCompile(`^[0-9a-f]{32}$`)
	spanIDPattern    = regexp.MustCompile(`^[0-9a-f]{16}$`)
)

// Event is the producer-facing CloudEvents envelope used by Observa v1.
// Keep this type aligned with schemas/cloudevent-v1.schema.json and
// schemas/event-data-v1.schema.json. Tenant identity is deliberately absent.
type Event struct {
	SpecVersion     string    `json:"specversion"`
	ID              string    `json:"id"`
	Source          string    `json:"source"`
	Type            string    `json:"type"`
	Time            string    `json:"time"`
	Subject         *string   `json:"subject,omitempty"`
	DataContentType string    `json:"datacontenttype"`
	DataSchema      string    `json:"dataschema"`
	Data            EventData `json:"data"`
}

type EventData struct {
	EventName       string         `json:"event_name"`
	Outcome         *string        `json:"outcome,omitempty"`
	Context         *EventContext  `json:"context,omitempty"`
	Attributes      map[string]any `json:"attributes,omitempty"`
	Error           *EventError    `json:"error,omitempty"`
	ProducerVersion *string        `json:"producer_version,omitempty"`
}

type EventContext struct {
	UserID                *string `json:"user_id,omitempty"`
	BusinessApplicationID *string `json:"business_application_id,omitempty"`
	SessionID             *string `json:"session_id,omitempty"`
	RequestID             *string `json:"request_id,omitempty"`
	TraceID               *string `json:"trace_id,omitempty"`
	SpanID                *string `json:"span_id,omitempty"`
}

type EventError struct {
	Code     *string `json:"code,omitempty"`
	Category *string `json:"category,omitempty"`
	Message  *string `json:"message,omitempty"`
}

// Decode parses one JSON event, rejects unknown fields, and validates the profile.
// Body-size limits must be enforced by the HTTP boundary before calling Decode.
func Decode(raw []byte) (Event, error) {
	if err := rejectExplicitNulls(raw); err != nil {
		return Event{}, err
	}
	var event Event
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		return Event{}, fmt.Errorf("decode event: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return Event{}, errors.New("decode event: multiple JSON values are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return Event{}, fmt.Errorf("decode event trailing data: %w", err)
	}
	if err := event.Validate(); err != nil {
		return Event{}, err
	}
	return event, nil
}

// rejectExplicitNulls preserves JSON Schema's distinction between an omitted
// optional property and a property explicitly set to null. Go pointers alone
// cannot distinguish those two inputs after unmarshalling.
func rejectExplicitNulls(raw []byte) error {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil // Decode below returns the canonical JSON parse error.
	}
	if err := rejectNullProperties(envelope, "subject"); err != nil {
		return err
	}
	dataRaw, ok := envelope["data"]
	if !ok || string(bytes.TrimSpace(dataRaw)) == "null" {
		return nil // Required/type validation below reports the invalid data value.
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(dataRaw, &data); err != nil {
		return nil
	}
	if err := rejectNullProperties(data, "outcome", "context", "attributes", "error", "producer_version"); err != nil {
		return err
	}
	if rawContext, ok := data["context"]; ok {
		var context map[string]json.RawMessage
		if err := json.Unmarshal(rawContext, &context); err == nil {
			if err := rejectNullProperties(context, "user_id", "business_application_id", "session_id", "request_id", "trace_id", "span_id"); err != nil {
				return err
			}
		}
	}
	if rawError, ok := data["error"]; ok {
		var eventError map[string]json.RawMessage
		if err := json.Unmarshal(rawError, &eventError); err == nil {
			if err := rejectNullProperties(eventError, "code", "category", "message"); err != nil {
				return err
			}
		}
	}
	return nil
}

func rejectNullProperties(object map[string]json.RawMessage, names ...string) error {
	for _, name := range names {
		if value, exists := object[name]; exists && string(bytes.TrimSpace(value)) == "null" {
			return fmt.Errorf("%s must not be null when provided", name)
		}
	}
	return nil
}

// Validate enforces the Observa v1 profile. Authentication, source authorization,
// payload byte limits, privacy policy, and clock-skew policy belong to later layers.
func (e Event) Validate() error {
	if e.SpecVersion != CloudEventsSpecVersion {
		return fmt.Errorf("specversion must be %q", CloudEventsSpecVersion)
	}
	if strings.TrimSpace(e.ID) == "" || utf8.RuneCountInString(e.ID) > MaxEventIDLength {
		return fmt.Errorf("id must contain 1-%d characters", MaxEventIDLength)
	}
	if err := validateSource(e.Source); err != nil {
		return err
	}
	switch e.Type {
	case "com.observa.user_action", "com.observa.business_state", "com.observa.technical_signal":
	default:
		return fmt.Errorf("unsupported type %q", e.Type)
	}
	if _, err := time.Parse(time.RFC3339Nano, e.Time); err != nil {
		return fmt.Errorf("time must be an RFC3339 timestamp: %w", err)
	}
	if e.Subject != nil && (len(*e.Subject) == 0 || utf8.RuneCountInString(*e.Subject) > 256) {
		return errors.New("subject must contain 1-256 characters when provided")
	}
	if e.DataContentType != JSONContentType {
		return fmt.Errorf("datacontenttype must be %q", JSONContentType)
	}
	if e.DataSchema != EventDataSchema {
		return fmt.Errorf("dataschema must be %q", EventDataSchema)
	}
	return e.Data.validate()
}

func validateSource(source string) error {
	if strings.TrimSpace(source) == "" || utf8.RuneCountInString(source) > MaxSourceLength {
		return fmt.Errorf("source must contain 1-%d characters", MaxSourceLength)
	}
	if strings.ContainsAny(source, "\r\n\t ") {
		return errors.New("source must be a valid URI-reference without whitespace")
	}
	if _, err := url.ParseRequestURI(source); err != nil {
		// ParseRequestURI rejects some valid URI references (e.g. relative paths),
		// so use url.Parse as the URI-reference parser and reject malformed escapes.
		if _, fallbackErr := url.Parse(source); fallbackErr != nil {
			return fmt.Errorf("source must be a valid URI-reference: %w", fallbackErr)
		}
	}
	return nil
}

func (d EventData) validate() error {
	if len(d.EventName) == 0 || utf8.RuneCountInString(d.EventName) > MaxEventNameLength || !eventNamePattern.MatchString(d.EventName) {
		return fmt.Errorf("data.event_name must match %s and contain at most %d characters", eventNamePattern, MaxEventNameLength)
	}
	if d.Outcome != nil {
		switch *d.Outcome {
		case "succeeded", "failed", "denied", "cancelled", "unknown":
		default:
			return fmt.Errorf("unsupported data.outcome %q", *d.Outcome)
		}
	}
	if d.Context != nil {
		if err := d.Context.validate(); err != nil {
			return err
		}
	}
	if len(d.Attributes) > MaxAttributes {
		return fmt.Errorf("data.attributes may contain at most %d properties", MaxAttributes)
	}
	for key := range d.Attributes {
		if strings.TrimSpace(key) == "" || utf8.RuneCountInString(key) > 64 {
			return errors.New("data.attributes keys must contain 1-64 characters")
		}
	}
	if d.Error != nil {
		if d.Error.Code == nil && d.Error.Category == nil && d.Error.Message == nil {
			return errors.New("data.error must contain at least one of code, category, or message")
		}
		if d.Error.Code != nil && (len(*d.Error.Code) == 0 || utf8.RuneCountInString(*d.Error.Code) > 128) {
			return errors.New("data.error.code must contain 1-128 characters")
		}
		if d.Error.Category != nil && (len(*d.Error.Category) == 0 || utf8.RuneCountInString(*d.Error.Category) > 64) {
			return errors.New("data.error.category must contain 1-64 characters")
		}
		if d.Error.Message != nil && utf8.RuneCountInString(*d.Error.Message) > 512 {
			return errors.New("data.error.message must contain at most 512 characters")
		}
	}
	if d.ProducerVersion != nil && (len(*d.ProducerVersion) == 0 || utf8.RuneCountInString(*d.ProducerVersion) > 64) {
		return errors.New("data.producer_version must contain 1-64 characters when provided")
	}
	return nil
}

func (c EventContext) validate() error {
	for name, value := range map[string]*string{
		"user_id":                 c.UserID,
		"business_application_id": c.BusinessApplicationID,
		"session_id":              c.SessionID,
		"request_id":              c.RequestID,
	} {
		if value != nil && (len(*value) == 0 || utf8.RuneCountInString(*value) > 128) {
			return fmt.Errorf("data.context.%s must contain 1-128 characters when provided", name)
		}
	}
	if c.TraceID != nil && (!traceIDPattern.MatchString(*c.TraceID) || strings.Trim(*c.TraceID, "0") == "") {
		return errors.New("data.context.trace_id must be a non-zero 32-character lowercase hexadecimal ID")
	}
	if c.SpanID != nil && (!spanIDPattern.MatchString(*c.SpanID) || strings.Trim(*c.SpanID, "0") == "") {
		return errors.New("data.context.span_id must be a non-zero 16-character lowercase hexadecimal ID")
	}
	return nil
}

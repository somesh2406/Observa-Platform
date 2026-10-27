package eventcontract

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func validEventJSON() string {
	return `{
		"specversion":"1.0",
		"id":"event-123",
		"source":"https://bank.example/apps/mobile",
		"type":"com.observa.user_action",
		"time":"2026-10-16T10:30:12.450Z",
		"datacontenttype":"application/json",
		"dataschema":"https://somesh2406.github.io/observa-event-contracts/schemas/event-data-v1.schema.json",
		"data":{"event_name":"loan.submit","outcome":"succeeded","context":{"user_id":"usr_token_1","business_application_id":"loan_token_1","session_id":"sess-1","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7"},"attributes":{"channel":"mobile"}}
	}`
}

type contractCase struct {
	Name  string          `json:"name"`
	Valid bool            `json:"valid"`
	Event json.RawMessage `json:"event"`
}

// TestSharedContractCorpus is the cross-implementation contract gate: the same
// corpus is consumed by this Go validator and scripts/validate_event_schemas.py.
func TestSharedContractCorpus(t *testing.T) {
	content, err := os.ReadFile("../../tests/fixtures/event-contract-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []contractCase
	if err := json.Unmarshal(content, &cases); err != nil {
		t.Fatalf("decode shared contract corpus: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := Decode(tc.Event)
			if tc.Valid && err != nil {
				t.Fatalf("Go validator rejected event accepted by expected contract: %v", err)
			}
			if !tc.Valid && err == nil {
				t.Fatal("Go validator accepted event expected to be invalid")
			}
		})
	}
}

func TestDecodeAcceptsValidEvent(t *testing.T) {
	got, err := Decode([]byte(validEventJSON()))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.ID != "event-123" || got.Data.EventName != "loan.submit" {
		t.Fatalf("unexpected decoded event: %#v", got)
	}
}

func TestDecodeAcceptsAllSupportedTypes(t *testing.T) {
	for _, eventType := range []string{"com.observa.user_action", "com.observa.business_state", "com.observa.technical_signal"} {
		t.Run(eventType, func(t *testing.T) {
			input := strings.Replace(validEventJSON(), `"com.observa.user_action"`, `"`+eventType+`"`, 1)
			if _, err := Decode([]byte(input)); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
		})
	}
}

func TestDecodeRejectsInvalidEvents(t *testing.T) {
	tests := []struct {
		name string
		edit func(string) string
	}{
		{"unsupported specversion", func(s string) string { return strings.Replace(s, `"specversion":"1.0"`, `"specversion":"2.0"`, 1) }},
		{"empty id", func(s string) string { return strings.Replace(s, `"id":"event-123"`, `"id":" "`, 1) }},
		{"invalid source URI", func(s string) string {
			return strings.Replace(s, `"source":"https://bank.example/apps/mobile"`, `"source":"https://bad host/"`, 1)
		}},
		{"unsupported type", func(s string) string {
			return strings.Replace(s, `"com.observa.user_action"`, `"com.observa.other"`, 1)
		}},
		{"invalid timestamp", func(s string) string { return strings.Replace(s, `"2026-10-16T10:30:12.450Z"`, `"yesterday"`, 1) }},
		{"wrong content type", func(s string) string {
			return strings.Replace(s, `"datacontenttype":"application/json"`, `"datacontenttype":"text/plain"`, 1)
		}},
		{"wrong schema", func(s string) string { return strings.Replace(s, EventDataSchema, "https://example.org/schema/v2", 1) }},
		{"invalid event name", func(s string) string { return strings.Replace(s, `"loan.submit"`, `"Loan Submit"`, 1) }},
		{"zero trace id", func(s string) string {
			return strings.Replace(s, `"4bf92f3577b34da6a3ce929d0e0e4736"`, strings.Repeat("0", 32), 1)
		}},
		{"unknown envelope field", func(s string) string {
			return strings.Replace(s, `"specversion":"1.0",`, `"specversion":"1.0","tenant_id":"attacker",`, 1)
		}},
		{"unknown nested field", func(s string) string {
			return strings.Replace(s, `"event_name":"loan.submit",`, `"event_name":"loan.submit","tenant_id":"attacker",`, 1)
		}},
		{"legacy user_ref field", func(s string) string {
			return strings.Replace(s, `"user_id":"usr_token_1"`, `"user_ref":"usr_token_1"`, 1)
		}},
		{"legacy business_application_ref field", func(s string) string {
			return strings.Replace(s, `"business_application_id":"loan_token_1"`, `"business_application_ref":"loan_token_1"`, 1)
		}},
		{"user name not permitted in context", func(s string) string {
			return strings.Replace(s, `"user_id":"usr_token_1"`, `"user_id":"usr_token_1","user_name":"Jane Doe"`, 1)
		}},
		{"applicant name not permitted in context", func(s string) string {
			return strings.Replace(s, `"business_application_id":"loan_token_1"`, `"business_application_id":"loan_token_1","applicant_name":"Jane Doe"`, 1)
		}},
		{"multiple JSON values", func(s string) string { return s + ` {}` }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Decode([]byte(tt.edit(validEventJSON()))); err == nil {
				t.Fatal("Decode() succeeded; expected validation error")
			}
		})
	}
}

func TestDecodeAcceptsContractExample(t *testing.T) {
	content, err := os.ReadFile("../../examples/user-action.cloudevent.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(content); err != nil {
		t.Fatalf("example does not validate: %v", err)
	}
}

func TestDecodeRejectsTooManyAttributes(t *testing.T) {
	var attributes strings.Builder
	attributes.WriteString(`{"specversion":"1.0","id":"event-123","source":"https://bank.example/apps/mobile","type":"com.observa.user_action","time":"2026-10-16T10:30:12Z","datacontenttype":"application/json","dataschema":"` + EventDataSchema + `","data":{"event_name":"loan.submit","attributes":{`)
	for i := 0; i < MaxAttributes+1; i++ {
		if i > 0 {
			attributes.WriteByte(',')
		}
		attributes.WriteString(`"key` + string(rune('a'+i)) + `":"value"`)
	}
	attributes.WriteString(`}}}`)
	if _, err := Decode([]byte(attributes.String())); err == nil {
		t.Fatal("Decode() succeeded; expected attribute limit error")
	}
}

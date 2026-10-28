package publisher

import (
	"testing"
	"time"
)

func TestNewKafkaRejectsMissingBrokers(t *testing.T) {
	_, err := NewKafka(Options{Topic: "events", DeliveryTimeout: time.Second})
	if err == nil {
		t.Fatal("expected missing brokers to fail")
	}
}

func TestNewKafkaRejectsMissingTopic(t *testing.T) {
	_, err := NewKafka(Options{Brokers: []string{"127.0.0.1:9092"}, DeliveryTimeout: time.Second})
	if err == nil {
		t.Fatal("expected missing topic to fail")
	}
}

func TestNewKafkaRejectsPartialSASLCredentials(t *testing.T) {
	_, err := NewKafka(Options{
		Brokers: []string{"127.0.0.1:9092"}, Topic: "events",
		DeliveryTimeout: time.Second, SASLMechanism: "PLAIN", Username: "user",
	})
	if err == nil {
		t.Fatal("expected partial SASL credentials to fail")
	}
}

func TestNewKafkaRejectsUnknownSASLMechanism(t *testing.T) {
	_, err := NewKafka(Options{
		Brokers: []string{"127.0.0.1:9092"}, Topic: "events",
		DeliveryTimeout: time.Second, SASLMechanism: "UNKNOWN", Username: "user", Password: "pass",
	})
	if err == nil {
		t.Fatal("expected unsupported SASL mechanism to fail")
	}
}

func TestKafkaRecordKeyUsesTenantAndSourceWithoutSession(t *testing.T) {
	got := string(kafkaRecordKey("tenant-a", "source-b", ""))
	if got != "tenant-a:source-b" {
		t.Fatalf("key=%q, want tenant-a:source-b", got)
	}
}

func TestKafkaRecordKeyIncludesSessionAndExcludesEventID(t *testing.T) {
	got := string(kafkaRecordKey("tenant-a", "source-b", "session-c"))
	if got != "tenant-a:source-b:session-c" {
		t.Fatalf("key=%q, want tenant-a:source-b:session-c", got)
	}
}

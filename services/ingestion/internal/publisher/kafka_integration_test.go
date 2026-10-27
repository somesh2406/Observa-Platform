package publisher

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestKafkaPublishIntegration(t *testing.T) {
	brokersRaw := strings.TrimSpace(os.Getenv("OBSERVA_KAFKA_INTEGRATION_BROKERS"))
	topic := strings.TrimSpace(os.Getenv("OBSERVA_KAFKA_INTEGRATION_TOPIC"))
	if brokersRaw == "" || topic == "" {
		t.Skip("set OBSERVA_KAFKA_INTEGRATION_BROKERS and OBSERVA_KAFKA_INTEGRATION_TOPIC to run Kafka integration test")
	}
	brokers := strings.Split(brokersRaw, ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}
	pub, err := NewKafka(Options{
		Brokers: brokers, Topic: topic, DeliveryTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewKafka: %v", err)
	}
	defer pub.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := pub.Ready(ctx); err != nil {
		t.Fatalf("Kafka readiness ping: %v", err)
	}
	eventID := "integration-test-" + time.Now().UTC().Format("20060102T150405.000000000")
	body := []byte(`{"integration_test_id":"` + eventID + `"}`)
	if err := pub.Publish(ctx, "tenant-integration-test", "source-integration-test", eventID, time.Now().UTC(), eventID, body); err != nil {
		t.Fatalf("Kafka publish: %v", err)
	}
}

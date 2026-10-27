package publisher

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

type EventPublisher interface {
	Publish(context.Context, string, string, string, time.Time, string, []byte) error
	Ready(context.Context) error
	Close()
}

type Options struct {
	Brokers         []string
	Topic           string
	DeliveryTimeout time.Duration
	SASLMechanism   string
	Username        string
	Password        string
	TLS             bool
}

type Kafka struct {
	client  *kgo.Client
	topic   string
	timeout time.Duration
}

func NewKafka(o Options) (*Kafka, error) {
	if len(o.Brokers) == 0 || o.Topic == "" {
		return nil, fmt.Errorf("Kafka brokers and topic are required")
	}
	opts := []kgo.Opt{
		kgo.SeedBrokers(o.Brokers...),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordRetries(10),
		kgo.RecordDeliveryTimeout(o.DeliveryTimeout),
		// The HTTP deadline must be enforceable even for an in-flight record.
		// Cancellation can create an ambiguous outcome; the caller returns non-202
		// and downstream consumers must deduplicate repeated logical event identities.
		kgo.AllowIdempotentProduceCancellation(),
		kgo.ProduceRequestTimeout(o.DeliveryTimeout),
		kgo.DialTimeout(3 * time.Second),
	}
	if o.TLS {
		opts = append(opts, kgo.DialTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}))
	}
	if o.SASLMechanism != "" || o.Username != "" || o.Password != "" {
		if o.Username == "" || o.Password == "" || o.SASLMechanism == "" {
			return nil, fmt.Errorf("Kafka SASL mechanism, username, and password must all be configured")
		}
		switch o.SASLMechanism {
		case "PLAIN":
			opts = append(opts, kgo.SASL(plain.Auth{User: o.Username, Pass: o.Password}.AsMechanism()))
		case "SCRAM-SHA-256":
			opts = append(opts, kgo.SASL(scram.Auth{User: o.Username, Pass: o.Password}.AsSha256Mechanism()))
		case "SCRAM-SHA-512":
			opts = append(opts, kgo.SASL(scram.Auth{User: o.Username, Pass: o.Password}.AsSha512Mechanism()))
		default:
			return nil, fmt.Errorf("unsupported Kafka SASL mechanism %q", o.SASLMechanism)
		}
	}
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("create Kafka producer: %w", err)
	}
	return &Kafka{client: client, topic: o.Topic, timeout: o.DeliveryTimeout}, nil
}

func (p *Kafka) Publish(ctx context.Context, tenantID, sourceID, ingestionID string, receivedAt time.Time, eventID string, rawEvent []byte) error {
	key, err := json.Marshal([3]string{tenantID, sourceID, eventID})
	if err != nil {
		return fmt.Errorf("encode Kafka record key: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	record := &kgo.Record{Topic: p.topic, Key: key, Value: rawEvent, Headers: []kgo.RecordHeader{
		{Key: "observa_tenant_id", Value: []byte(tenantID)},
		{Key: "observa_source_id", Value: []byte(sourceID)},
		{Key: "observa_ingestion_id", Value: []byte(ingestionID)},
		{Key: "observa_received_at", Value: []byte(receivedAt.UTC().Format(time.RFC3339Nano))},
		{Key: "observa_contract_version", Value: []byte("1")},
	}}
	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("Kafka delivery not confirmed: %w", err)
	}
	return nil
}

func (p *Kafka) Ready(ctx context.Context) error { return p.client.Ping(ctx) }

func (p *Kafka) Close() { p.client.Close() }

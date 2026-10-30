package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/observa-platform/observa/services/ingestion/internal/kafkatopics"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Kafka topic provisioning failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	environment := strings.TrimSpace(os.Getenv("OBSERVA_ENVIRONMENT"))
	if environment == "" {
		return fmt.Errorf("OBSERVA_ENVIRONMENT is required")
	}

	cataloguePath := strings.TrimSpace(os.Getenv("KAFKA_TOPIC_CATALOG"))
	if cataloguePath == "" {
		return fmt.Errorf("KAFKA_TOPIC_CATALOG is required")
	}

	brokersValue := strings.TrimSpace(os.Getenv("KAFKA_BROKERS"))
	if brokersValue == "" {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}
	brokers := strings.Split(brokersValue, ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
		if brokers[i] == "" {
			return fmt.Errorf("KAFKA_BROKERS contains an empty broker address")
		}
	}

	desired, err := kafkatopics.LoadCatalogue(cataloguePath, environment)
	if err != nil {
		return err
	}

	options := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ClientID("observa-topic-provisioner"),
		kgo.DialTimeout(3 * time.Second),
	}

	if strings.EqualFold(os.Getenv("KAFKA_TLS"), "true") {
		options = append(options, kgo.DialTLSConfig(&tls.Config{
			MinVersion: tls.VersionTLS12,
		}))
	}

	mechanism := strings.ToUpper(strings.TrimSpace(os.Getenv("KAFKA_SASL_MECHANISM")))
	username := os.Getenv("KAFKA_USERNAME")
	password := os.Getenv("KAFKA_PASSWORD")

	if mechanism != "" || username != "" || password != "" {
		if mechanism == "" || username == "" || password == "" {
			return fmt.Errorf(
				"KAFKA_SASL_MECHANISM, KAFKA_USERNAME, and KAFKA_PASSWORD must all be set",
			)
		}

		switch mechanism {
		case "PLAIN":
			options = append(options, kgo.SASL(
				plain.Auth{User: username, Pass: password}.AsMechanism(),
			))
		case "SCRAM-SHA-256":
			options = append(options, kgo.SASL(
				scram.Auth{User: username, Pass: password}.AsSha256Mechanism(),
			))
		case "SCRAM-SHA-512":
			options = append(options, kgo.SASL(
				scram.Auth{User: username, Pass: password}.AsSha512Mechanism(),
			))
		default:
			return fmt.Errorf("unsupported KAFKA_SASL_MECHANISM %q", mechanism)
		}
	}

	client, err := kgo.NewClient(options...)
	if err != nil {
		return fmt.Errorf("create Kafka admin client: %w", err)
	}

	provisioner := kafkatopics.NewProvisioner(client)
	defer provisioner.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Kafka health checks reduce startup races; bounded retries cover the
	// remaining gap between container health and Admin API availability.
	if err := waitForKafka(ctx, client); err != nil {
		return err
	}

	slog.Info(
		"starting topic reconciliation",
		"environment", environment,
		"topic_count", len(desired),
		"brokers", brokers,
	)

	if err := provisioner.Run(ctx, desired); err != nil {
		return err
	}

	slog.Info("Kafka topic provisioning completed", "topic_count", len(desired))
	return nil
}

func waitForKafka(ctx context.Context, client *kgo.Client) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := client.Ping(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("Kafka did not become available before timeout: %w (last error: %v)",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

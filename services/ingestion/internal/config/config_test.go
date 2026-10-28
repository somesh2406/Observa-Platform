package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

const validConfig = `{
  "service":"test",
  "environment":"development",
  "http":{"address":"127.0.0.1:8080","read_header_timeout":"5s","read_timeout":"15s","write_timeout":"15s","idle_timeout":"60s","shutdown_timeout":"15s","publish_timeout":"5s","max_body_bytes":1048576},
  "development":{"allow_unauthenticated":true,"tenant_id":"tenant-test","source_id":"source-test"},
  "kafka":{"brokers":["127.0.0.1:9092"],"topic":"observa.events.v1","delivery_timeout":"5s","tls":false,"sasl_mechanism":""},
  "vault":{"enabled":false,"address":"http://127.0.0.1:8200","kv_v2_path":"observa/ingestion","token_env":"VAULT_TOKEN","request_timeout":"3s","skip_tls_verify":false},
  "logger":{"level":"info","format":"json","add_source":true}
}`

func TestAcceptsExplicitLoopbackDevelopmentMode(t *testing.T) {
	path := writeConfig(t, validConfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Kafka.Topic != "observa.events.v1" {
		t.Fatalf("topic=%q", cfg.Kafka.Topic)
	}
	if cfg.HTTP.ReadTimeout.String() != "15s" {
		t.Fatalf("read timeout=%s", cfg.HTTP.ReadTimeout.String())
	}
}

func TestAllowsGatewayAuthenticatedProductionMode(t *testing.T) {
	content := strings.Replace(validConfig, `"environment":"development"`, `"environment":"production"`, 1)
	content = strings.Replace(content, `"allow_unauthenticated":true`, `"allow_unauthenticated":false`, 1)
	if _, err := Load(writeConfig(t, content)); err != nil {
		t.Fatalf("gateway-authenticated production mode should load: %v", err)
	}
}

func TestRejectsUnauthenticatedModeOnNonLoopbackAddress(t *testing.T) {
	content := strings.Replace(validConfig, `"127.0.0.1:8080"`, `"0.0.0.0:8080"`, 1)
	_, err := Load(writeConfig(t, content))
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("expected loopback guard, got %v", err)
	}
}

func TestRejectsUnknownConfigurationFields(t *testing.T) {
	content := strings.TrimSuffix(validConfig, "}") + `,"unexpected":true}`
	_, err := Load(writeConfig(t, content))
	if err == nil {
		t.Fatal("expected unknown field to be rejected")
	}
}

func TestEnvironmentOverridesConfigurationFile(t *testing.T) {
	t.Setenv("OBSERVA_KAFKA_BROKERS", "127.0.0.1:9092,127.0.0.1:9094")
	cfg, err := Load(writeConfig(t, validConfig))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Kafka.Brokers) != 2 || cfg.Kafka.Brokers[1] != "127.0.0.1:9094" {
		t.Fatalf("brokers=%v", cfg.Kafka.Brokers)
	}
}

func TestVaultPathMustBeRelativeToMount(t *testing.T) {
	content := strings.Replace(validConfig, `"enabled":false`, `"enabled":true`, 1)
	content = strings.Replace(content, `"sasl_mechanism":""`, `"sasl_mechanism":"PLAIN"`, 1)
	content = strings.Replace(content, `"tls":false`, `"tls":true`, 1)
	content = strings.Replace(content, `"kv_v2_path":"observa/ingestion"`, `"kv_v2_path":"secret/data/observa/ingestion"`, 1)
	_, err := Load(writeConfig(t, content))
	if err == nil || !strings.Contains(err.Error(), "relative to the KV v2 mount") {
		t.Fatalf("expected path validation error, got %v", err)
	}
}

func TestVaultRequiresSASLConfiguration(t *testing.T) {
	content := strings.Replace(validConfig, `"enabled":false`, `"enabled":true`, 1)
	_, err := Load(writeConfig(t, content))
	if err == nil || !strings.Contains(err.Error(), "sasl_mechanism") {
		t.Fatalf("expected SASL requirement, got %v", err)
	}
}

func TestEnvironmentVariableCanSupplyKeyAbsentFromConfigFile(t *testing.T) {
	t.Setenv("OBSERVA_HTTP_ADDRESS", "127.0.0.1:8181")
	content := `{
	  "environment":"development",
	  "development":{"allow_unauthenticated":true,"tenant_id":"tenant-test","source_id":"source-test"},
	  "kafka":{"brokers":["127.0.0.1:9092"],"topic":"observa.events.v1"}
	}`
	cfg, err := Load(writeConfig(t, content))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTP.Address != "127.0.0.1:8181" {
		t.Fatalf("HTTP address=%q, want environment-only override", cfg.HTTP.Address)
	}
}

func TestRejectsLocalhostNameForUnauthenticatedMode(t *testing.T) {
	content := strings.Replace(validConfig, `"127.0.0.1:8080"`, `"localhost:8080"`, 1)
	_, err := Load(writeConfig(t, content))
	if err == nil || !strings.Contains(err.Error(), "127.0.0.1 or ::1") {
		t.Fatalf("expected explicit loopback IP error, got %v", err)
	}
}

func TestRejectsSASLPlainWithoutTLS(t *testing.T) {
	content := strings.Replace(validConfig, `"sasl_mechanism":""`, `"sasl_mechanism":"PLAIN"`, 1)
	content = strings.Replace(content, `"enabled":false`, `"enabled":true`, 1)
	_, err := Load(writeConfig(t, content))
	if err == nil || !strings.Contains(err.Error(), "PLAIN requires kafka.tls=true") {
		t.Fatalf("expected PLAIN TLS validation error, got %v", err)
	}
}

func TestRejectsUnknownTenantRateLimitBackend(t *testing.T) {
	content := strings.TrimSuffix(validConfig, "}") + `,"tenant_rate_limit":{"backend":"unknown","requests":10,"window":"1m"}}`
	_, err := Load(writeConfig(t, content))
	if err == nil || !strings.Contains(err.Error(), "backend must be memory or redis") {
		t.Fatalf("expected rate limiter backend validation error, got %v", err)
	}
}

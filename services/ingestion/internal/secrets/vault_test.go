package secrets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchKafkaCredentialsReadsKVV2Secret(t *testing.T) {
	const token = "test-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/data/observa/ingestion" {
			t.Errorf("unexpected Vault path %q", r.URL.Path)
		}
		if got := r.Header.Get("X-Vault-Token"); got != token {
			t.Errorf("token=%q, want test token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"data": map[string]any{
					"kafka_username": "user-from-vault",
					"kafka_password": "password-from-vault",
				},
				"metadata": map[string]any{"version": 1},
			},
		})
	}))
	defer server.Close()

	client, err := NewVault(server.URL, token, false)
	if err != nil {
		t.Fatalf("NewVault: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	user, pass, err := client.FetchKafkaCredentials(ctx, "observa/ingestion", "kafka_username", "kafka_password")
	if err != nil {
		t.Fatalf("FetchKafkaCredentials: %v", err)
	}
	if user != "user-from-vault" || pass != "password-from-vault" {
		t.Fatalf("credentials not read correctly: username=%q password-empty=%v", user, pass == "")
	}
}

func TestFetchKafkaCredentialsRejectsMountPrefixedPath(t *testing.T) {
	client := &VaultClient{}
	_, _, err := client.FetchKafkaCredentials(context.Background(), "secret/data/observa/ingestion", "u", "p")
	if err == nil {
		t.Fatal("expected invalid path to be rejected")
	}
}

func TestFetchKafkaCredentialsRejectsMissingCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"data":{"kafka_username":"only-user"},"metadata":{}}}`))
	}))
	defer server.Close()
	client, err := NewVault(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewVault: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err = client.FetchKafkaCredentials(ctx, "observa/ingestion", "kafka_username", "kafka_password")
	if err == nil {
		t.Fatal("expected missing password to fail")
	}
}

func TestNewVaultRequiresAddressAndToken(t *testing.T) {
	if _, err := NewVault("", "token", false); err == nil {
		t.Fatal("expected empty address to fail")
	}
	if _, err := NewVault("http://127.0.0.1:8200", "", false); err == nil {
		t.Fatal("expected empty token to fail")
	}
}

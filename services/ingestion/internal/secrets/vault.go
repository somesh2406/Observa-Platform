package secrets

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
)

type VaultClient struct {
	client *vaultapi.Client
}

// NewVault creates a Vault API client. skipTLSVerify is intentionally supported
// only for isolated local development and is rejected by config validation elsewhere.
func NewVault(address, token string, skipTLSVerify bool) (*VaultClient, error) {
	if strings.TrimSpace(address) == "" || strings.TrimSpace(token) == "" {
		return nil, errors.New("vault address and token are required")
	}

	cfg := vaultapi.DefaultConfig()
	cfg.Address = address
	if skipTLSVerify {
		baseTransport := http.DefaultTransport
		if cfg.HttpClient.Transport != nil {
			baseTransport = cfg.HttpClient.Transport
		}
		transport, ok := baseTransport.(*http.Transport)
		if !ok {
			return nil, errors.New("Vault HTTP transport does not support TLS customization")
		}
		transport = transport.Clone()
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true} // #nosec G402 -- explicitly gated to local development
		transport.TLSClientConfig = tlsConfig
		cfg.HttpClient.Transport = transport
	}

	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize Vault client: %w", err)
	}
	client.SetToken(token)
	return &VaultClient{client: client}, nil
}

// FetchKafkaCredentials expects a path relative to the "secret" KV v2 mount,
// e.g. "observa/ingestion". Do not pass "secret/data/..." here.
func (v *VaultClient) FetchKafkaCredentials(ctx context.Context, path, userKey, passKey string) (string, string, error) {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" || strings.HasPrefix(path, "data/") || strings.HasPrefix(path, "secret/") {
		return "", "", errors.New("Vault KV v2 path must be relative to the secret mount, e.g. observa/ingestion")
	}
	secret, err := v.client.KVv2("secret").Get(ctx, path)
	if err != nil {
		return "", "", fmt.Errorf("read Vault KV v2 secret at %q: %w", path, err)
	}
	if secret == nil || secret.Data == nil {
		return "", "", errors.New("Vault secret contains no data")
	}
	username, _ := secret.Data[userKey].(string)
	password, _ := secret.Data[passKey].(string)
	if strings.TrimSpace(username) == "" || password == "" {
		return "", "", errors.New("Vault secret does not contain non-empty Kafka credentials")
	}
	return username, password, nil
}

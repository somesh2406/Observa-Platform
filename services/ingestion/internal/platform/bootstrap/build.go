package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/observa-platform/observa/services/ingestion/internal/config"
	"github.com/observa-platform/observa/services/ingestion/internal/ingestion"
	logger "github.com/observa-platform/observa/services/ingestion/internal/platform"
	"github.com/observa-platform/observa/services/ingestion/internal/publisher"
	"github.com/observa-platform/observa/services/ingestion/internal/ratelimit"
	"github.com/observa-platform/observa/services/ingestion/internal/secrets"
)

func Build(configPath string) (*Application, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}

	log := logger.New(cfg.Logger, cfg.Service)

	// Keep secret material outside the general configuration model.
	var kafkaUsername, kafkaPassword string
	if cfg.Vault.Enabled {
		token := os.Getenv(cfg.Vault.TokenEnv)
		vaultClient, err := secrets.NewVault(cfg.Vault.Address, token, cfg.Vault.SkipTLSVerify)
		if err != nil {
			return nil, fmt.Errorf("vault init failed: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Vault.RequestTimeout)
		defer cancel()

		user, pass, err := vaultClient.FetchKafkaCredentials(
			ctx,
			cfg.Vault.KVV2Path,
			cfg.Kafka.UsernameSecretKey,
			cfg.Kafka.PasswordSecretKey,
		)
		if err != nil {
			return nil, fmt.Errorf("secret retrieval failed: %w", err)
		}

		kafkaUsername, kafkaPassword = user, pass
	}

	kafkaOpts := publisher.Options{
		Brokers:         cfg.Kafka.Brokers,
		Topic:           cfg.Kafka.Topic,
		DeliveryTimeout: cfg.Kafka.DeliveryTimeout,
		SASLMechanism:   cfg.Kafka.SASLMechanism,
		Username:        kafkaUsername,
		Password:        kafkaPassword,
		TLS:             cfg.Kafka.TLS,
	}
	kafkaPub, err := publisher.NewKafka(kafkaOpts)
	if err != nil {
		return nil, fmt.Errorf("kafka init failed: %w", err)
	}

	router := chi.NewRouter()

	// 1. Chi Middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer) // Prevents application crashes from handler panics
	router.Use(middleware.Timeout(cfg.HTTP.ReadTimeout))

	// 2. Custom Security Middleware
	router.Use(securityHeaders)

	// 3. Register Domain Handlers. In authenticated mode, request identity is
	// derived from JWT claim headers set by Envoy; local dev mode uses configured identity.
	ident := ingestion.Identity{TenantID: cfg.Development.TenantID, SourceID: cfg.Development.SourceID}

	password := ""
	if cfg.TenantRateLimit.RedisPasswordEnv != "" {
		password = os.Getenv(cfg.TenantRateLimit.RedisPasswordEnv)
	}

	username := ""
	if cfg.TenantRateLimit.RedisUsernameEnv != "" {
		username = os.Getenv(cfg.TenantRateLimit.RedisUsernameEnv)
	}

	tenantLimiter, err := ratelimit.NewRedisLimiter(ratelimit.RedisOptions{
		Address:     cfg.TenantRateLimit.RedisAddress,
		Username:    username,
		Password:    password,
		DB:          cfg.TenantRateLimit.RedisDB,
		Requests:    cfg.TenantRateLimit.Requests,
		Window:      cfg.TenantRateLimit.Window,
		DialTimeout: cfg.TenantRateLimit.DialTimeout,
		TLS:         cfg.TenantRateLimit.RedisTLS,
	})
	if err != nil {
		kafkaPub.Close()
		return nil, fmt.Errorf("tenant rate limiter init failed: %w", err)
	}

	handler := ingestion.NewHandler(kafkaPub, ident, cfg.Development.AllowUnauthenticated, tenantLimiter, cfg.HTTP.MaxBodyBytes, cfg.HTTP.PublishTimeout, log)
	handler.Register(router)

	server := &http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           router, // Chi implements http.Handler
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	return &Application{
		server:          server,
		logger:          log,
		shutdownTimeout: cfg.HTTP.ShutdownTimeout,
		cleanup: func() {
			kafkaPub.Close()
			if err := tenantLimiter.Close(); err != nil {
				log.Error("close tenant rate limiter", "error", err)
			}
		},
	}, nil
}

// Chi middleware uses the exact same signature as standard net/http middleware
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")

		if r.URL.Path == "/v1/events" {
			w.Header().Set("Vary", "Content-Type")
		}

		next.ServeHTTP(w, r)
	})
}

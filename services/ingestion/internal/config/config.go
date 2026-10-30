package config

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Service         string                `mapstructure:"service"`
	Environment     string                `mapstructure:"environment"`
	HTTP            HTTPConfig            `mapstructure:"http"`
	Development     DevelopmentConfig     `mapstructure:"development"`
	Kafka           KafkaConfig           `mapstructure:"kafka"`
	Vault           VaultConfig           `mapstructure:"vault"`
	TenantRateLimit TenantRateLimitConfig `mapstructure:"tenant_rate_limit"`
	Logger          LoggerConfig          `mapstructure:"logger"`
}

type HTTPConfig struct {
	Address           string        `mapstructure:"address"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
	MaxBodyBytes      int64         `mapstructure:"max_body_bytes"`
	PublishTimeout    time.Duration `mapstructure:"publish_timeout"`
}

type DevelopmentConfig struct {
	AllowUnauthenticated bool   `mapstructure:"allow_unauthenticated"`
	TenantID             string `mapstructure:"tenant_id"`
	SourceID             string `mapstructure:"source_id"`
}

type KafkaConfig struct {
	Brokers           []string      `mapstructure:"brokers"`
	Topic             string        `mapstructure:"topic"`
	DeliveryTimeout   time.Duration `mapstructure:"delivery_timeout"`
	SASLMechanism     string        `mapstructure:"sasl_mechanism"`
	TLS               bool          `mapstructure:"tls"`
	UsernameSecretKey string        `mapstructure:"username_secret_key"`
	PasswordSecretKey string        `mapstructure:"password_secret_key"`
}

type VaultConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	Address        string        `mapstructure:"address"`
	KVV2Path       string        `mapstructure:"kv_v2_path"`
	TokenEnv       string        `mapstructure:"token_env"`
	RequestTimeout time.Duration `mapstructure:"request_timeout"`
	SkipTLSVerify  bool          `mapstructure:"skip_tls_verify"`
}

type TenantRateLimitConfig struct {
	Requests         int           `mapstructure:"requests"`
	Window           time.Duration `mapstructure:"window"`
	RedisAddress     string        `mapstructure:"redis_address"`
	RedisUsernameEnv string        `mapstructure:"redis_username_env"`
	RedisPasswordEnv string        `mapstructure:"redis_password_env"`
	RedisDB          int           `mapstructure:"redis_db"`
	RedisTLS         bool          `mapstructure:"redis_tls"`
	DialTimeout      time.Duration `mapstructure:"dial_timeout"`
}

type LoggerConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"`
	AddSource bool   `mapstructure:"add_source"`
}

func Load(configPath string) (Config, error) {
	var cfg Config
	v := viper.New()

	v.SetEnvPrefix("OBSERVA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if strings.TrimSpace(configPath) != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("ingestion.local")
		v.SetConfigType("json")
		v.AddConfigPath("./configs")
	}

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return cfg, fmt.Errorf("read configuration file: %w", err)
		}
	}

	if err := v.UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("decode configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("service", "Observa Ingestion Service")
	v.SetDefault("http.address", "127.0.0.1:8080")
	v.SetDefault("http.read_header_timeout", "5s")
	v.SetDefault("http.read_timeout", "15s")
	v.SetDefault("http.write_timeout", "15s")
	v.SetDefault("http.idle_timeout", "60s")
	v.SetDefault("http.shutdown_timeout", "15s")
	v.SetDefault("http.publish_timeout", "5s")
	v.SetDefault("http.max_body_bytes", int64(1<<20))
	v.SetDefault("kafka.delivery_timeout", "5s")
	v.SetDefault("vault.request_timeout", "3s")
	v.SetDefault("vault.token_env", "VAULT_TOKEN")
	v.SetDefault("kafka.username_secret_key", "kafka_username")
	v.SetDefault("kafka.password_secret_key", "kafka_password")
	v.SetDefault("tenant_rate_limit.requests", 100)
	v.SetDefault("tenant_rate_limit.window", "1m")
	v.SetDefault("tenant_rate_limit.redis_address", "127.0.0.1:6380")
	v.SetDefault("tenant_rate_limit.redis_username_env", "OBSERVA_TENANT_RATE_LIMIT_REDIS_USERNAME")
	v.SetDefault("tenant_rate_limit.redis_password_env", "OBSERVA_TENANT_RATE_LIMIT_REDIS_PASSWORD")
	v.SetDefault("tenant_rate_limit.redis_db", 0)
	v.SetDefault("tenant_rate_limit.redis_tls", false)
	v.SetDefault("tenant_rate_limit.dial_timeout", "2s")

	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Environment) == "" {
		return errors.New("environment is required")
	}
	if c.Development.AllowUnauthenticated {
		if c.Environment != "development" {
			return errors.New("development.allow_unauthenticated may be enabled only when environment=development")
		}
		if strings.TrimSpace(c.Development.TenantID) == "" || strings.TrimSpace(c.Development.SourceID) == "" {
			return errors.New("development tenant_id and source_id are required when unauthenticated development mode is enabled")
		}
		host, _, err := net.SplitHostPort(c.HTTP.Address)
		if err != nil || !isExplicitLoopbackIP(host) {
			return errors.New("unauthenticated development mode must bind explicitly to 127.0.0.1 or ::1")
		}
	}
	if len(c.Kafka.Brokers) == 0 {
		return errors.New("at least one Kafka broker is required")
	}
	for i, broker := range c.Kafka.Brokers {
		broker = strings.TrimSpace(broker)
		if broker == "" {
			return fmt.Errorf("kafka.brokers[%d] must not be empty", i)
		}
		if _, _, err := net.SplitHostPort(broker); err != nil {
			return fmt.Errorf("kafka.brokers[%d] must be a host:port address: %w", i, err)
		}
	}
	if strings.TrimSpace(c.Kafka.Topic) == "" {
		return errors.New("Kafka topic is required")
	}
	if c.Kafka.SASLMechanism == "PLAIN" && !c.Kafka.TLS {
		return errors.New("Kafka SASL PLAIN requires kafka.tls=true")
	}
	if c.HTTP.MaxBodyBytes < 1 || c.HTTP.MaxBodyBytes > 16<<20 {
		return errors.New("http.max_body_bytes must be between 1 byte and 16 MiB")
	}

	if c.HTTP.ReadHeaderTimeout <= 0 || c.HTTP.ReadTimeout <= 0 ||
		c.HTTP.WriteTimeout <= 0 || c.HTTP.IdleTimeout <= 0 ||
		c.HTTP.ShutdownTimeout <= 0 || c.HTTP.PublishTimeout <= 0 ||
		c.Kafka.DeliveryTimeout <= 0 || c.Vault.RequestTimeout <= 0 ||
		c.TenantRateLimit.DialTimeout <= 0 {
		return errors.New("all configured timeouts must be positive")
	}

	if c.Vault.SkipTLSVerify && c.Environment != "development" {
		return errors.New("vault.skip_tls_verify is permitted only in development")
	}
	if c.Vault.Enabled {
		if strings.TrimSpace(c.Vault.Address) == "" || strings.TrimSpace(c.Vault.KVV2Path) == "" {
			return errors.New("vault.address and vault.kv_v2_path are required when Vault is enabled")
		}
		if strings.HasPrefix(c.Vault.KVV2Path, "secret/") || strings.Contains(c.Vault.KVV2Path, "/data/") {
			return errors.New("vault.kv_v2_path must be relative to the KV v2 mount (for example, observa/ingestion), not include secret/data/")
		}
		if c.Kafka.SASLMechanism == "" {
			return errors.New("Vault is used to load Kafka SASL credentials; configure kafka.sasl_mechanism or disable Vault")
		}
		if strings.TrimSpace(c.Kafka.UsernameSecretKey) == "" || strings.TrimSpace(c.Kafka.PasswordSecretKey) == "" {
			return errors.New("Kafka username_secret_key and password_secret_key are required when Vault is enabled")
		}
	}

	if c.Kafka.SASLMechanism != "" && !c.Vault.Enabled {
		return errors.New("Kafka SASL is configured but Vault is disabled; credentials must be loaded from Vault")
	}
	if strings.TrimSpace(c.TenantRateLimit.RedisAddress) == "" {
		return errors.New("tenant_rate_limit.redis_address is required")
	}
	if c.TenantRateLimit.RedisDB < 0 {
		return errors.New("tenant_rate_limit.redis_db must not be negative")
	}
	if c.TenantRateLimit.Requests < 1 || c.TenantRateLimit.Window <= 0 {
		return errors.New(
			"tenant_rate_limit.requests and tenant_rate_limit.window must be positive",
		)
	}
	if c.TenantRateLimit.Window.Milliseconds() < 1 {
		return errors.New("tenant_rate_limit.window must be at least 1ms")
	}

	switch c.Kafka.SASLMechanism {
	case "", "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512":
	default:
		return fmt.Errorf("unsupported Kafka SASL mechanism %q", c.Kafka.SASLMechanism)
	}
	switch strings.ToLower(strings.TrimSpace(c.Logger.Level)) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logger.level must be one of debug, info, warn, error")
	}
	switch strings.ToLower(strings.TrimSpace(c.Logger.Format)) {
	case "json", "text":
	default:
		return fmt.Errorf("logger.format must be json or text")
	}
	return nil
}

// isExplicitLoopbackIP is intentionally stricter than resolving the hostname
// "localhost": the unauthenticated dev-mode guard only accepts literal loopback IPs.
func isExplicitLoopbackIP(host string) bool {
	ip := net.ParseIP(strings.TrimSpace(host))
	return ip != nil && ip.IsLoopback() && (ip.Equal(net.ParseIP("127.0.0.1")) || ip.Equal(net.ParseIP("::1")))
}

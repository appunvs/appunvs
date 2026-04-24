// Package config loads the relay configuration from YAML + environment
// variables via viper.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the top-level runtime configuration for the relay.
type Config struct {
	Listen  string  `mapstructure:"listen"`
	Redis   Redis   `mapstructure:"redis"`
	Auth    Auth    `mapstructure:"auth"`
	Stream  Stream  `mapstructure:"stream"`
	Log     Log     `mapstructure:"log"`
	DB      DB      `mapstructure:"db"`
	Billing Billing `mapstructure:"billing"`
}

// Billing holds Stripe credentials. Leaving both blank flips the billing
// layer into mock mode, where /billing/checkout returns a synthetic URL
// and /billing/webhook accepts unsigned POSTs carrying X-Mock-Upgrade.
type Billing struct {
	StripeSecretKey     string `mapstructure:"stripe_secret_key"`
	StripeWebhookSecret string `mapstructure:"stripe_webhook_secret"`
	CheckoutSuccessURL  string `mapstructure:"checkout_success_url"`
	CheckoutCancelURL   string `mapstructure:"checkout_cancel_url"`
}

// DB points at the relay's SQLite file holding accounts, devices, api keys,
// plans, subscriptions, and app schema.
type DB struct {
	Path string `mapstructure:"path"`
}

// Redis points at a Redis instance used for seq + stream.
type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Auth holds JWT signing material and issuer/audience for RS256 tokens.
type Auth struct {
	PrivateKeyPath string `mapstructure:"private_key_path"`
	PublicKeyPath  string `mapstructure:"public_key_path"`
	Issuer         string `mapstructure:"issuer"`
	Audience       string `mapstructure:"audience"`
	SessionHours   int    `mapstructure:"session_hours"`
	DeviceDays     int    `mapstructure:"device_days"`
}

// Stream controls Redis Stream retention.
type Stream struct {
	MaxLen int64 `mapstructure:"max_len"`
}

// Log controls logger output.
type Log struct {
	Level string `mapstructure:"level"`
}

// Load reads config from the given file path, then merges APPUNVS_* env vars.
// Passing path="" falls back to ./config/config.yaml.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetDefault("listen", ":8080")
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("auth.issuer", "appunvs-relay")
	v.SetDefault("auth.audience", "appunvs-clients")
	v.SetDefault("auth.session_hours", 24)
	v.SetDefault("auth.device_days", 30)
	v.SetDefault("stream.max_len", 100000)
	v.SetDefault("log.level", "info")
	v.SetDefault("db.path", "data/relay.db")
	v.SetDefault("billing.stripe_secret_key", "")
	v.SetDefault("billing.stripe_webhook_secret", "")
	v.SetDefault("billing.checkout_success_url", "https://appunvs.local/billing/success")
	v.SetDefault("billing.checkout_cancel_url", "https://appunvs.local/billing/cancel")

	if path == "" {
		path = "config/config.yaml"
	}
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetEnvPrefix("APPUNVS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		// Missing file is not fatal in dev; env + defaults still apply.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// If a path was explicitly given but unreadable, surface it.
			// (We swallow "file doesn't exist" silently.)
			if !isNotFound(err) {
				return nil, fmt.Errorf("config: read %s: %w", path, err)
			}
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}
	return cfg, nil
}

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "no such file")
}

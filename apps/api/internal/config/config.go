package config

import (
	"os"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
// No hardcoded app names or domains — everything comes from env.
type Config struct {
	DatabaseURL      string
	RedisURL         string
	Port             string
	AppName          string
	AppDomain        string
	JWTSecret        string // Phase 3 — auth
	AnthropicAPIKey  string // required for AI search enrichment and flow interpretation
	VoyageAPIKey     string // required for reach embeddings and /ask endpoint
	USGSAPIKey       string // optional, raises rate limits
	USGSPollInterval string
	DWRPollInterval  string
	MigrationsPath   string
}

func Load() Config {
	return Config{
		DatabaseURL:      mustEnv("DATABASE_URL"),
		RedisURL:         env("REDIS_URL", "redis://localhost:6379"),
		Port:             env("APP_PORT", "8080"),
		AppName:          env("APP_NAME", "H2OFlows"),
		AppDomain:        env("APP_DOMAIN", "localhost"),
		JWTSecret:        env("JWT_SECRET", ""),
		AnthropicAPIKey:  env("ANTHROPIC_API_KEY", ""),
		VoyageAPIKey:     env("VOYAGE_API_KEY", ""),
		USGSAPIKey:       env("USGS_API_KEY", ""),
		USGSPollInterval: env("USGS_POLL_INTERVAL", "15m"),
		DWRPollInterval:  env("DWR_POLL_INTERVAL", "15m"),
		MigrationsPath:   env("MIGRATIONS_PATH", "migrations"),
	}
}

// PollIntervals holds parsed durations for each gauge source.
type PollIntervals struct {
	USGS time.Duration
	DWR  time.Duration
}

// ParsePollInterval parses the string interval fields into durations.
// Falls back to 15 minutes if a value is missing or unparseable.
func (c Config) ParsePollInterval() PollIntervals {
	parse := func(s string) time.Duration {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			return d
		}
		return 15 * time.Minute
	}
	return PollIntervals{
		USGS: parse(c.USGSPollInterval),
		DWR:  parse(c.DWRPollInterval),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var not set: " + key)
	}
	return v
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

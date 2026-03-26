package config

import "os"

// Config holds all runtime configuration loaded from environment variables.
// No hardcoded app names or domains — everything comes from env.
type Config struct {
	DatabaseURL      string
	RedisURL         string
	Port             string
	AppName          string
	AppDomain        string
	JWTSecret        string // Phase 3 — auth
	USGSAPIKey       string // optional, raises rate limits
	USGSPollInterval string
	DWRPollInterval  string
}

func Load() Config {
	return Config{
		DatabaseURL:      mustEnv("DATABASE_URL"),
		RedisURL:         env("REDIS_URL", "redis://localhost:6379"),
		Port:             env("APP_PORT", "8080"),
		AppName:          env("APP_NAME", "H2OFlow"),
		AppDomain:        env("APP_DOMAIN", "localhost"),
		JWTSecret:        env("JWT_SECRET", ""),
		USGSAPIKey:       env("USGS_API_KEY", ""),
		USGSPollInterval: env("USGS_POLL_INTERVAL", "15m"),
		DWRPollInterval:  env("DWR_POLL_INTERVAL", "15m"),
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

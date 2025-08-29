package config

import (
	"os"
)

type Config struct {
	Env         string
	HTTPPort    string
	DatabaseURL string
	JWTSecret   string
	JWTIssuer   string
	RateRPS     int
}

func Load() Config {
	cfg := Config{
		Env:         get("APP_ENV", "dev"),
		HTTPPort:    get("HTTP_PORT", "8080"),
		DatabaseURL: get("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/insider?sslmode=disable"),
		JWTSecret:   get("JWT_SECRET", "changeme-secret"),
		JWTIssuer:   get("JWT_ISSUER", "insider-backend"),
	}
	return cfg
}

func get(key, def string) string { v := os.Getenv(key); if v == "" { return def }; return v }
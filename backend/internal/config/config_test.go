package config_test

import (
	"os"
	"testing"

	"github.com/thesct22/ezyshare/backend/internal/config"
)

func TestLoadConfigDefaults(t *testing.T) {
	os.Unsetenv("APP_ENV")
	os.Unsetenv("ENV")
	os.Unsetenv("PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("LOG_FORMAT")
	os.Unsetenv("ALLOWED_ORIGINS")

	cfg := config.LoadConfig()
	if cfg.AppEnv != "dev" {
		t.Fatalf("expected AppEnv dev, got %s", cfg.AppEnv)
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected Port 8080, got %s", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected LogLevel debug, got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Fatalf("expected LogFormat text, got %s", cfg.LogFormat)
	}
}

func TestLoadConfigProd(t *testing.T) {
	os.Setenv("APP_ENV", "prod")
	defer os.Unsetenv("APP_ENV")

	cfg := config.LoadConfig()
	if cfg.AppEnv != "prod" {
		t.Fatalf("expected AppEnv prod, got %s", cfg.AppEnv)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected LogLevel info, got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Fatalf("expected LogFormat json, got %s", cfg.LogFormat)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "https://sharath.is-a.dev" {
		t.Fatalf("expected prod allowed origin https://sharath.is-a.dev, got %v", cfg.AllowedOrigins)
	}
}

func TestIsOriginAllowed(t *testing.T) {
	origins := []string{"https://sharath.is-a.dev", "http://localhost:*"}

	if !config.IsOriginAllowed("https://sharath.is-a.dev", origins) {
		t.Fatalf("expected https://sharath.is-a.dev to be allowed")
	}
	if !config.IsOriginAllowed("http://localhost:3000", origins) {
		t.Fatalf("expected http://localhost:3000 to be allowed")
	}
	if config.IsOriginAllowed("https://malicious-website.com", origins) {
		t.Fatalf("expected https://malicious-website.com to be rejected")
	}
}

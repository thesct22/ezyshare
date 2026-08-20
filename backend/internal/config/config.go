package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	AppEnv         string
	Port           string
	LogLevel       string
	LogFormat      string
	AllowedOrigins []string
}

func LoadConfig() *Config {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env == "" {
		env = "dev"
	}
	env = strings.ToLower(env)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		if env == "dev" {
			logLevel = "debug"
		} else {
			logLevel = "info"
		}
	}

	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		if env == "dev" {
			logFormat = "text"
		} else {
			logFormat = "json"
		}
	}

	var origins []string
	if custom := os.Getenv("ALLOWED_ORIGINS"); custom != "" {
		for _, o := range strings.Split(custom, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	} else {
		switch env {
		case "prod":
			origins = []string{"https://sharath.is-a.dev"}
		case "staging":
			origins = []string{"https://sharath.is-a.dev", "http://localhost:*", "http://127.0.0.1:*"}
		default: // dev
			origins = []string{"http://localhost:*", "http://127.0.0.1:*", "https://sharath.is-a.dev"}
		}
	}

	return &Config{
		AppEnv:         env,
		Port:           port,
		LogLevel:       logLevel,
		LogFormat:      logFormat,
		AllowedOrigins: origins,
	}
}

// IsOriginAllowed checks if an incoming origin matches the allowed origin patterns.
func IsOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}

	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}

	reqHost := u.Hostname()
	reqPort := u.Port()
	reqScheme := u.Scheme

	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}

		var allowedScheme, allowedHostPort string
		if idx := strings.Index(allowed, "://"); idx != -1 {
			allowedScheme = allowed[:idx]
			allowedHostPort = allowed[idx+3:]
		} else {
			allowedHostPort = allowed
		}

		if allowedScheme != "" && allowedScheme != reqScheme {
			continue
		}

		var allowedHost, allowedPort string
		if idx := strings.LastIndex(allowedHostPort, ":"); idx != -1 {
			allowedHost = allowedHostPort[:idx]
			allowedPort = allowedHostPort[idx+1:]
		} else {
			allowedHost = allowedHostPort
		}

		// Match host
		if matched, _ := filepath.Match(allowedHost, reqHost); !matched && allowedHost != reqHost {
			continue
		}

		// Match port
		if allowedPort == "*" || allowedPort == "" || allowedPort == reqPort {
			return true
		}
	}

	return false
}

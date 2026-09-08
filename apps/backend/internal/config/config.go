package config

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL    string
	Port           int
	FrontendOrigin string
	RequestTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		Port:           8080,
		FrontendOrigin: "http://localhost:3000",
		RequestTimeout: 10 * time.Second,
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	if value := strings.TrimSpace(os.Getenv("PORT")); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, errors.New("PORT must be an integer between 1 and 65535")
		}
		cfg.Port = port
	}
	if value := strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN")); value != "" {
		cfg.FrontendOrigin = value
	}
	if value := strings.TrimSpace(os.Getenv("REQUEST_TIMEOUT")); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil || timeout <= 0 {
			return Config{}, errors.New("REQUEST_TIMEOUT must be a positive duration")
		}
		cfg.RequestTimeout = timeout
	}

	origin, err := url.Parse(cfg.FrontendOrigin)
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Host == "" || origin.User != nil || (origin.Path != "" && origin.Path != "/") || origin.RawQuery != "" || origin.Fragment != "" {
		return Config{}, errors.New("FRONTEND_ORIGIN must be an HTTP(S) origin")
	}
	cfg.FrontendOrigin = strings.TrimSuffix(cfg.FrontendOrigin, "/")

	return cfg, nil
}

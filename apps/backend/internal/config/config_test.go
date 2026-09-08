package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/searchlens")
	t.Setenv("PORT", "")
	t.Setenv("FRONTEND_ORIGIN", "")
	t.Setenv("REQUEST_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 8080 || cfg.FrontendOrigin != "http://localhost:3000" || cfg.RequestTimeout != 10*time.Second {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		database string
	}{
		{name: "missing database URL"},
		{name: "bad port", key: "PORT", value: "0", database: "postgres://localhost/searchlens"},
		{name: "bad timeout", key: "REQUEST_TIMEOUT", value: "0s", database: "postgres://localhost/searchlens"},
		{name: "bad origin", key: "FRONTEND_ORIGIN", value: "http://localhost:3000/path", database: "postgres://localhost/searchlens"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", test.database)
			t.Setenv("PORT", "")
			t.Setenv("FRONTEND_ORIGIN", "")
			t.Setenv("REQUEST_TIMEOUT", "")
			if test.key != "" {
				t.Setenv(test.key, test.value)
			}
			if _, err := Load(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

func TestDefaults(t *testing.T) {
	c, err := Load("demo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Env != "development" {
		t.Errorf("env = %q, want development", c.Env)
	}
	if c.Server.Port != 8080 {
		t.Errorf("port = %d, want 8080", c.Server.Port)
	}
	if c.DB.Name != "demo" {
		t.Errorf("db name = %q, want demo", c.DB.Name)
	}
	if c.HTTP.ShutdownTimeout != 10*time.Second {
		t.Errorf("shutdown timeout = %v, want 10s", c.HTTP.ShutdownTimeout)
	}
}

func TestEnvOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := []byte("server:\n  port: 9000\ndb:\n  name: from_yaml\n")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}

	setEnv(t, map[string]string{"SERVER_PORT": "9100"})

	c, err := Load("demo", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Server.Port != 9100 {
		t.Errorf("port = %d, want env value 9100", c.Server.Port)
	}
	if c.DB.Name != "from_yaml" {
		t.Errorf("db name = %q, want yaml value from_yaml", c.DB.Name)
	}
}

func TestProductionRequiresPassword(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":              "production",
		"DB_SSLMODE":           "require",
		"CORS_ALLOWED_ORIGINS": "https://example.com",
	})
	if _, err := Load("demo", ""); err == nil {
		t.Fatal("expected fail-fast when DB_PASSWORD is empty in production")
	}
}

func TestProductionRejectsWildcardCORS(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":              "production",
		"DB_PASSWORD":          "secret",
		"DB_SSLMODE":           "require",
		"CORS_ALLOWED_ORIGINS": "*",
	})
	if _, err := Load("demo", ""); err == nil {
		t.Fatal("expected fail-fast when CORS is wildcard in production")
	}
}

func TestProductionRejectsDisabledSSL(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":              "production",
		"DB_PASSWORD":          "secret",
		"DB_SSLMODE":           "disable",
		"CORS_ALLOWED_ORIGINS": "https://example.com",
	})
	if _, err := Load("demo", ""); err == nil {
		t.Fatal("expected fail-fast when DB_SSLMODE=disable in production")
	}
}

func TestProductionAcceptsSafeConfig(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":              "production",
		"DB_PASSWORD":          "secret",
		"DB_SSLMODE":           "require",
		"CORS_ALLOWED_ORIGINS": "https://example.com, https://app.example.com",
		"JWT_ACCESS_SECRET":    "0123456789abcdef0123456789abcdef",
		"JWT_REFRESH_SECRET":   "fedcba9876543210fedcba9876543210",
	})
	c, err := Load("demo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Security.AllowedOrigins) != 2 {
		t.Errorf("origins = %v, want 2 entries", c.Security.AllowedOrigins)
	}
	if !c.Identity.CookieSecure {
		t.Error("production cookies must be Secure")
	}
}

func TestProductionRequiresJWSecrets(t *testing.T) {
	base := map[string]string{
		"APP_ENV":              "production",
		"DB_PASSWORD":          "secret",
		"DB_SSLMODE":           "require",
		"CORS_ALLOWED_ORIGINS": "https://example.com",
	}

	t.Run("missing access secret", func(t *testing.T) {
		setEnv(t, base)
		if _, err := Load("demo", ""); err == nil {
			t.Fatal("expected failure when JWT_ACCESS_SECRET is empty")
		}
	})

	t.Run("identical secrets", func(t *testing.T) {
		setEnv(t, base)
		setEnv(t, map[string]string{
			"JWT_ACCESS_SECRET":  "0123456789abcdef0123456789abcdef",
			"JWT_REFRESH_SECRET": "0123456789abcdef0123456789abcdef",
		})
		if _, err := Load("demo", ""); err == nil {
			t.Fatal("expected failure when the two secrets are identical")
		}
	})

	t.Run("short secret", func(t *testing.T) {
		setEnv(t, base)
		setEnv(t, map[string]string{
			"JWT_ACCESS_SECRET":  "short",
			"JWT_REFRESH_SECRET": "fedcba9876543210fedcba9876543210",
		})
		if _, err := Load("demo", ""); err == nil {
			t.Fatal("expected failure for a short access secret")
		}
	})
}

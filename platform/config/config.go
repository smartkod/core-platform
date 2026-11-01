// Package config loads and validates platform configuration.
//
// Precedence is env > yaml > default (L0.md 4.3). Validation is fail-fast in
// production: the process refuses to start with a missing secret or an unsafe
// value rather than running misconfigured.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the fully resolved platform configuration.
type Config struct {
	Env        string
	Server     Server
	DB         DB
	HTTP       HTTP
	Log        Log
	Security   Security
	Identity   Identity
	AI         AI
	AppName    string
	ConfigPath string
	Modules    map[string]string
}

type Server struct {
	Host string
	Port int
}

type DB struct {
	Host         string
	Port         int
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	SlowQuery    time.Duration
}

type HTTP struct {
	BodyLimit       string
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

type Log struct {
	Level  string
	Format string
}

type Security struct {
	AllowedOrigins []string
}

// AI holds the model-provider settings.
//
// The API key is read from the environment only. Keeping it out of the YAML file
// is deliberate: the YAML is tracked in git, the environment is not.
type AI struct {
	Enabled        bool
	BaseURL        string
	APIKey         string
	DefaultModel   string
	RequestTimeout time.Duration
	MaxRetries     int
}

// Identity holds the authentication settings.
//
// Secrets live here so a single fail-fast check can refuse to boot a
// production process without them (L0.md 4.3, L1.md 3.5).
type Identity struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	PasswordMin   int
	// CookieSecure must be true in production; derived from Env when unset.
	CookieSecure bool
}

// IsProduction reports whether production-only rules apply.
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// DSN builds the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s default_query_exec_mode=simple_protocol",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Name, c.DB.SSLMode,
	)
}

// Address is the listen address for the HTTP server.
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// fileShape mirrors the YAML file so a missing key is distinguishable from a
// zero value. Pointers are used only where "unset" matters.
type fileShape struct {
	Env    string `yaml:"env"`
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
	DB struct {
		Host         string `yaml:"host"`
		Port         int    `yaml:"port"`
		User         string `yaml:"user"`
		Password     string `yaml:"password"`
		Name         string `yaml:"name"`
		SSLMode      string `yaml:"sslmode"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		SlowQuery    string `yaml:"slow_query"`
	} `yaml:"db"`
	HTTP struct {
		BodyLimit       string `yaml:"body_limit"`
		RequestTimeout  string `yaml:"request_timeout"`
		ShutdownTimeout string `yaml:"shutdown_timeout"`
	} `yaml:"http"`
	Log struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"log"`
	Security struct {
		AllowedOrigins []string `yaml:"allowed_origins"`
	} `yaml:"security"`
	Identity struct {
		AccessSecret  string `yaml:"access_secret"`
		RefreshSecret string `yaml:"refresh_secret"`
		AccessTTL     string `yaml:"access_ttl"`
		RefreshTTL    string `yaml:"refresh_ttl"`
		PasswordMin   int    `yaml:"password_min_length"`
		CookieSecure  *bool  `yaml:"cookie_secure"`
	} `yaml:"identity"`
	AI struct {
		Enabled        bool   `yaml:"enabled"`
		BaseURL        string `yaml:"base_url"`
		DefaultModel   string `yaml:"default_model"`
		RequestTimeout string `yaml:"request_timeout"`
		MaxRetries     int    `yaml:"max_retries"`
	} `yaml:"ai"`
	Modules map[string]string `yaml:"modules"`
}

// Load resolves configuration from defaults, then the YAML file, then the
// environment, and validates the result.
func Load(appName, path string) (*Config, error) {
	c := defaults(appName)

	if path != "" {
		if f, err := os.ReadFile(path); err == nil {
			var fs fileShape
			if err := yaml.Unmarshal(f, &fs); err != nil {
				return nil, fmt.Errorf("config: parse %s: %w", path, err)
			}
			applyFile(c, &fs)
			c.ConfigPath = path
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("config: read %s: %w", path, err)
		}

		// Module states live in a sibling file so operations can change them
		// without touching the main config (L0.md 4.2).
		if err := loadModulesFile(c, path); err != nil {
			return nil, err
		}
	}

	applyEnv(c)

	// Cookie security follows the environment unless explicitly set: a
	// production cookie that is not Secure is a real vulnerability, while a
	// development cookie that is Secure simply never gets stored over HTTP.
	if c.IsProduction() {
		c.Identity.CookieSecure = true
	}

	if err := validate(c); err != nil {
		return nil, err
	}
	return c, nil
}

func defaults(appName string) *Config {
	return &Config{
		Env:     "development",
		AppName: appName,
		Server:  Server{Host: "0.0.0.0", Port: 8080},
		DB: DB{
			Host: "localhost", Port: 5432, User: "postgres",
			Name: appName, SSLMode: "disable", MaxOpenConns: 25,
			SlowQuery: 200 * time.Millisecond,
		},
		HTTP: HTTP{
			BodyLimit:       "8M",
			RequestTimeout:  30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		Log: Log{Level: "info", Format: "json"},
		Identity: Identity{
			AccessTTL:   15 * time.Minute,
			RefreshTTL:  7 * 24 * time.Hour,
			PasswordMin: 10,
		},
		AI: AI{
			RequestTimeout: 60 * time.Second,
			MaxRetries:     2,
		},
		Modules: map[string]string{},
	}
}

func applyFile(c *Config, f *fileShape) {
	if f.Env != "" {
		c.Env = f.Env
	}
	if f.Server.Host != "" {
		c.Server.Host = f.Server.Host
	}
	if f.Server.Port != 0 {
		c.Server.Port = f.Server.Port
	}
	if f.DB.Host != "" {
		c.DB.Host = f.DB.Host
	}
	if f.DB.Port != 0 {
		c.DB.Port = f.DB.Port
	}
	if f.DB.User != "" {
		c.DB.User = f.DB.User
	}
	if f.DB.Password != "" {
		c.DB.Password = f.DB.Password
	}
	if f.DB.Name != "" {
		c.DB.Name = f.DB.Name
	}
	if f.DB.SSLMode != "" {
		c.DB.SSLMode = f.DB.SSLMode
	}
	if f.DB.MaxOpenConns != 0 {
		c.DB.MaxOpenConns = f.DB.MaxOpenConns
	}
	if d, err := time.ParseDuration(f.DB.SlowQuery); err == nil && f.DB.SlowQuery != "" {
		c.DB.SlowQuery = d
	}
	if f.HTTP.BodyLimit != "" {
		c.HTTP.BodyLimit = f.HTTP.BodyLimit
	}
	if d, err := time.ParseDuration(f.HTTP.RequestTimeout); err == nil && f.HTTP.RequestTimeout != "" {
		c.HTTP.RequestTimeout = d
	}
	if d, err := time.ParseDuration(f.HTTP.ShutdownTimeout); err == nil && f.HTTP.ShutdownTimeout != "" {
		c.HTTP.ShutdownTimeout = d
	}
	if f.Log.Level != "" {
		c.Log.Level = f.Log.Level
	}
	if f.Log.Format != "" {
		c.Log.Format = f.Log.Format
	}
	if len(f.Security.AllowedOrigins) > 0 {
		c.Security.AllowedOrigins = f.Security.AllowedOrigins
	}
	if f.Identity.AccessSecret != "" {
		c.Identity.AccessSecret = f.Identity.AccessSecret
	}
	if f.Identity.RefreshSecret != "" {
		c.Identity.RefreshSecret = f.Identity.RefreshSecret
	}
	if d, err := time.ParseDuration(f.Identity.AccessTTL); err == nil && f.Identity.AccessTTL != "" {
		c.Identity.AccessTTL = d
	}
	if d, err := time.ParseDuration(f.Identity.RefreshTTL); err == nil && f.Identity.RefreshTTL != "" {
		c.Identity.RefreshTTL = d
	}
	if f.Identity.PasswordMin != 0 {
		c.Identity.PasswordMin = f.Identity.PasswordMin
	}
	if f.Identity.CookieSecure != nil {
		c.Identity.CookieSecure = *f.Identity.CookieSecure
	}
	if f.AI.Enabled {
		c.AI.Enabled = true
	}
	if f.AI.BaseURL != "" {
		c.AI.BaseURL = f.AI.BaseURL
	}
	if f.AI.DefaultModel != "" {
		c.AI.DefaultModel = f.AI.DefaultModel
	}
	if d, err := time.ParseDuration(f.AI.RequestTimeout); err == nil && f.AI.RequestTimeout != "" {
		c.AI.RequestTimeout = d
	}
	if f.AI.MaxRetries != 0 {
		c.AI.MaxRetries = f.AI.MaxRetries
	}
	for k, v := range f.Modules {
		c.Modules[strings.ToUpper(strings.TrimSpace(k))] = strings.ToUpper(strings.TrimSpace(v))
	}
}

func applyEnv(c *Config) {
	str := func(key string, dst *string) {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			*dst = v
		}
	}
	num := func(key string, dst *int) {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
			}
		}
	}
	dur := func(key string, dst *time.Duration) {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				*dst = d
			}
		}
	}

	str("APP_ENV", &c.Env)
	str("SERVER_HOST", &c.Server.Host)
	num("SERVER_PORT", &c.Server.Port)
	str("DB_HOST", &c.DB.Host)
	num("DB_PORT", &c.DB.Port)
	str("DB_USER", &c.DB.User)
	str("DB_PASSWORD", &c.DB.Password)
	str("DB_NAME", &c.DB.Name)
	str("DB_SSLMODE", &c.DB.SSLMode)
	num("DB_MAX_OPEN_CONNS", &c.DB.MaxOpenConns)
	dur("DB_SLOW_QUERY", &c.DB.SlowQuery)
	str("HTTP_BODY_LIMIT", &c.HTTP.BodyLimit)
	dur("HTTP_REQUEST_TIMEOUT", &c.HTTP.RequestTimeout)
	dur("HTTP_SHUTDOWN_TIMEOUT", &c.HTTP.ShutdownTimeout)
	str("LOG_LEVEL", &c.Log.Level)
	str("LOG_FORMAT", &c.Log.Format)
	str("JWT_ACCESS_SECRET", &c.Identity.AccessSecret)
	str("JWT_REFRESH_SECRET", &c.Identity.RefreshSecret)
	dur("JWT_ACCESS_TTL", &c.Identity.AccessTTL)
	dur("JWT_REFRESH_TTL", &c.Identity.RefreshTTL)

	// AI settings. The key is env-only by design; there is no YAML field for it,
	// so it cannot be committed by accident.
	if v, ok := os.LookupEnv("AI_ENABLED"); ok {
		c.AI.Enabled = v == "true" || v == "1"
	}
	str("AI_BASE_URL", &c.AI.BaseURL)
	str("AI_API_KEY", &c.AI.APIKey)
	str("AI_DEFAULT_MODEL", &c.AI.DefaultModel)
	dur("AI_REQUEST_TIMEOUT", &c.AI.RequestTimeout)
	num("AI_MAX_RETRIES", &c.AI.MaxRetries)

	if v, ok := os.LookupEnv("CORS_ALLOWED_ORIGINS"); ok {
		var out []string
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		c.Security.AllowedOrigins = out
	}
}

// loadModulesFile reads modules.yaml from the same directory as the main
// config. A missing file is not an error: every module then defaults to
// enabled.
func loadModulesFile(c *Config, mainPath string) error {
	dir := filepath.Dir(mainPath)
	path := filepath.Join(dir, "modules.yaml")

	f, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}

	var mf struct {
		Modules map[string]string `yaml:"modules"`
	}
	if err := yaml.Unmarshal(f, &mf); err != nil {
		return fmt.Errorf("config: parse %s: %w", path, err)
	}
	for k, v := range mf.Modules {
		c.Modules[strings.ToUpper(strings.TrimSpace(k))] = strings.ToUpper(strings.TrimSpace(v))
	}
	return nil
}

// validate enforces production rules. Every failure stops the process before
// it can serve traffic.
func validate(c *Config) error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("config: SERVER_PORT out of range: %d", c.Server.Port)
	}
	if c.DB.Name == "" {
		return fmt.Errorf("config: DB_NAME is required")
	}
	if !c.IsProduction() {
		return nil
	}
	if c.DB.Password == "" {
		return fmt.Errorf("config: DB_PASSWORD is required in production")
	}
	if c.DB.SSLMode == "disable" {
		return fmt.Errorf("config: DB_SSLMODE=disable is not allowed in production")
	}
	if len(c.Security.AllowedOrigins) == 0 {
		return fmt.Errorf("config: CORS_ALLOWED_ORIGINS is required in production")
	}
	for _, o := range c.Security.AllowedOrigins {
		if o == "*" {
			return fmt.Errorf("config: CORS_ALLOWED_ORIGINS=* is not allowed in production")
		}
	}
	if c.Identity.AccessSecret == "" {
		return fmt.Errorf("config: JWT_ACCESS_SECRET is required in production")
	}
	if c.Identity.RefreshSecret == "" {
		return fmt.Errorf("config: JWT_REFRESH_SECRET is required in production")
	}
	if c.Identity.AccessSecret == c.Identity.RefreshSecret {
		return fmt.Errorf("config: JWT_ACCESS_SECRET and JWT_REFRESH_SECRET must differ")
	}
	if len(c.Identity.AccessSecret) < 32 {
		return fmt.Errorf("config: JWT_ACCESS_SECRET must be at least 32 characters")
	}
	// An enabled AI provider with no key would fail on the first user request;
	// failing at boot makes the misconfiguration obvious instead.
	if c.AI.Enabled && c.AI.APIKey == "" {
		return fmt.Errorf("config: AI_ENABLED is true but AI_API_KEY is empty")
	}
	if c.AI.Enabled && c.AI.BaseURL == "" {
		return fmt.Errorf("config: AI_ENABLED is true but AI_BASE_URL is empty")
	}
	return nil
}

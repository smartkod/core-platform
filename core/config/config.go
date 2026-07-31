package config

import (
"encoding/json"
"fmt"
"os"
"path/filepath"

"gopkg.in/yaml.v3"
)

// Config ana yapılandırma
type Config struct {
Server   ServerConfig   `json:"server" yaml:"server"`
Database DatabaseConfig `json:"database" yaml:"database"`
Logger   LoggerConfig   `json:"logger" yaml:"logger"`
Modules  ModulesConfig  `json:"modules" yaml:"modules"`
}

// ServerConfig sunucu ayarları
type ServerConfig struct {
Host string `json:"host" yaml:"host" env:"SERVER_HOST"`
Port string `json:"port" yaml:"port" env:"SERVER_PORT"`
}

// DatabaseConfig veritabanı ayarları
type DatabaseConfig struct {
Host     string `json:"host" yaml:"host" env:"DB_HOST"`
Port     string `json:"port" yaml:"port" env:"DB_PORT"`
User     string `json:"user" yaml:"user" env:"DB_USER"`
Password string `json:"password" yaml:"password" env:"DB_PASSWORD"`
Name     string `json:"name" yaml:"name" env:"DB_NAME"`
SSLMode  string `json:"sslmode" yaml:"sslmode" env:"DB_SSLMODE"`
}

// LoggerConfig log ayarları
type LoggerConfig struct {
Level  string `json:"level" yaml:"level" env:"LOG_LEVEL"`
Format string `json:"format" yaml:"format" env:"LOG_FORMAT"`
}

// ModulesConfig modül ayarları
type ModulesConfig struct {
ConfigPath string            `json:"config_path" yaml:"config_path" env:"MODULES_CONFIG"`
States     map[string]string `json:"states" yaml:"states"`
}

// Loader konfigürasyon yükleyici
type Loader struct {
configPath string
config     *Config
}

// NewLoader yeni bir config loader oluşturur
func NewLoader(configPath string) *Loader {
return &Loader{
configPath: configPath,
}
}

// Load konfigürasyonu yükler
func (l *Loader) Load() (*Config, error) {
cfg := &Config{}

// 1. Varsayılan değerleri ayarla
l.setDefaults(cfg)

// 2. Dosyadan oku (eğer varsa)
if l.configPath != "" {
if err := l.loadFromFile(cfg); err != nil {
return nil, fmt.Errorf("failed to load config file: %w", err)
}
}

// 3. Environment variable'lardan oku (override)
if err := l.loadFromEnv(cfg); err != nil {
return nil, fmt.Errorf("failed to load from env: %w", err)
}

l.config = cfg
return cfg, nil
}

// setDefaults varsayılan değerleri ayarlar
func (l *Loader) setDefaults(cfg *Config) {
cfg.Server.Host = "0.0.0.0"
cfg.Server.Port = "8080"

cfg.Database.Host = "localhost"
cfg.Database.Port = "5432"
cfg.Database.User = "postgres"
cfg.Database.Password = "postgres"
cfg.Database.Name = "core_platform"
cfg.Database.SSLMode = "disable"

cfg.Logger.Level = "info"
cfg.Logger.Format = "json"

cfg.Modules.ConfigPath = "config/modules.yaml"
cfg.Modules.States = make(map[string]string)
}

// loadFromFile dosyadan konfigürasyon yükler
func (l *Loader) loadFromFile(cfg *Config) error {
data, err := os.ReadFile(l.configPath)
if err != nil {
if os.IsNotExist(err) {
return nil // Dosya yoksa sorun değil
}
return err
}

// Dosya uzantısına göre parse et
ext := filepath.Ext(l.configPath)
switch ext {
case ".json":
return json.Unmarshal(data, cfg)
case ".yaml", ".yml":
return yaml.Unmarshal(data, cfg)
default:
return fmt.Errorf("unsupported config format: %s", ext)
}
}

// loadFromEnv environment variable'lardan yükler
func (l *Loader) loadFromEnv(cfg *Config) error {
// Server
if val := os.Getenv("SERVER_HOST"); val != "" {
cfg.Server.Host = val
}
if val := os.Getenv("SERVER_PORT"); val != "" {
cfg.Server.Port = val
}

// Database
if val := os.Getenv("DB_HOST"); val != "" {
cfg.Database.Host = val
}
if val := os.Getenv("DB_PORT"); val != "" {
cfg.Database.Port = val
}
if val := os.Getenv("DB_USER"); val != "" {
cfg.Database.User = val
}
if val := os.Getenv("DB_PASSWORD"); val != "" {
cfg.Database.Password = val
}
if val := os.Getenv("DB_NAME"); val != "" {
cfg.Database.Name = val
}
if val := os.Getenv("DB_SSLMODE"); val != "" {
cfg.Database.SSLMode = val
}

// Logger
if val := os.Getenv("LOG_LEVEL"); val != "" {
cfg.Logger.Level = val
}
if val := os.Getenv("LOG_FORMAT"); val != "" {
cfg.Logger.Format = val
}

// Modules
if val := os.Getenv("MODULES_CONFIG"); val != "" {
cfg.Modules.ConfigPath = val
}

return nil
}

// GetConfig konfigürasyonu döndürür
func (l *Loader) GetConfig() *Config {
return l.config
}

// Validate konfigürasyonu doğrular
func (c *Config) Validate() error {
// Server validasyonu
if c.Server.Port == "" {
return fmt.Errorf("server port is required")
}

// Database validasyonu
if c.Database.Host == "" {
return fmt.Errorf("database host is required")
}
if c.Database.User == "" {
return fmt.Errorf("database user is required")
}
if c.Database.Name == "" {
return fmt.Errorf("database name is required")
}

// Logger validasyonu
validLevels := []string{"debug", "info", "warn", "error"}
if !contains(validLevels, c.Logger.Level) {
return fmt.Errorf("invalid log level: %s", c.Logger.Level)
}

validFormats := []string{"json", "console"}
if !contains(validFormats, c.Logger.Format) {
return fmt.Errorf("invalid log format: %s", c.Logger.Format)
}

return nil
}

// contains helper
func contains(slice []string, item string) bool {
for _, s := range slice {
if s == item {
return true
}
}
return false
}

// String konfigürasyonu string olarak döndürür
func (c *Config) String() string {
data, _ := json.MarshalIndent(c, "", "  ")
return string(data)
}

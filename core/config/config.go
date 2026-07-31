package config

import (
"os"
)

type Config struct {
Server   ServerConfig   `json:"server" yaml:"server"`
Database DatabaseConfig `json:"database" yaml:"database"`
Logger   LoggerConfig   `json:"logger" yaml:"logger"`
JWT      JWTConfig      `json:"jwt" yaml:"jwt"`
External ExternalConfig `json:"external" yaml:"external"`
Redis    RedisConfig    `json:"redis" yaml:"redis"`
Modules  ModulesConfig  `json:"modules" yaml:"modules"`
Features FeaturesConfig `json:"features" yaml:"features"`
}

type ServerConfig struct {
Host string `json:"host" yaml:"host" env:"SERVER_HOST"`
Port string `json:"port" yaml:"port" env:"SERVER_PORT"`
}

type DatabaseConfig struct {
Host     string `json:"host" yaml:"host" env:"DB_HOST"`
Port     string `json:"port" yaml:"port" env:"DB_PORT"`
User     string `json:"user" yaml:"user" env:"DB_USER"`
Password string `json:"password" yaml:"password" env:"DB_PASSWORD"` // 🔐 Environment'dan
Name     string `json:"name" yaml:"name" env:"DB_NAME"`
SSLMode  string `json:"sslmode" yaml:"sslmode" env:"DB_SSLMODE"`
}

type LoggerConfig struct {
Level  string `json:"level" yaml:"level" env:"LOG_LEVEL"`
Format string `json:"format" yaml:"format" env:"LOG_FORMAT"`
}

type JWTConfig struct {
Secret        string `json:"secret" yaml:"secret" env:"JWT_SECRET"`                 // 🔐 Environment'dan
Expiry        int64  `json:"expiry" yaml:"expiry" env:"JWT_EXPIRY"`
RefreshSecret string `json:"refresh_secret" yaml:"refresh_secret" env:"REFRESH_SECRET"` // 🔐 Environment'dan
RefreshExpiry int64  `json:"refresh_expiry" yaml:"refresh_expiry" env:"REFRESH_EXPIRY"`
}

type ExternalConfig struct {
OpenAI     OpenAIConfig     `json:"openai" yaml:"openai"`
Anthropic  AnthropicConfig  `json:"anthropic" yaml:"anthropic"`
Gemini     GeminiConfig     `json:"gemini" yaml:"gemini"`
Supabase   SupabaseConfig   `json:"supabase" yaml:"supabase"`
Cloudflare CloudflareConfig `json:"cloudflare" yaml:"cloudflare"`
AWS        AWSConfig        `json:"aws" yaml:"aws"`
}

type OpenAIConfig struct {
APIKey  string `json:"api_key" yaml:"api_key" env:"OPENAI_API_KEY"` // 🔐 Environment'dan
BaseURL string `json:"base_url" yaml:"base_url" env:"OPENAI_BASE_URL"`
}

type AnthropicConfig struct {
APIKey string `json:"api_key" yaml:"api_key" env:"ANTHROPIC_API_KEY"` // 🔐 Environment'dan
}

type GeminiConfig struct {
APIKey string `json:"api_key" yaml:"api_key" env:"GEMINI_API_KEY"` // 🔐 Environment'dan
}

type SupabaseConfig struct {
URL string `json:"url" yaml:"url" env:"SUPABASE_URL"`
Key string `json:"key" yaml:"key" env:"SUPABASE_KEY"` // 🔐 Environment'dan
}

type CloudflareConfig struct {
APIToken  string `json:"api_token" yaml:"api_token" env:"CLOUDFLARE_API_TOKEN"` // 🔐 Environment'dan
AccountID string `json:"account_id" yaml:"account_id" env:"CLOUDFLARE_ACCOUNT_ID"`
}

type AWSConfig struct {
AccessKeyID     string `json:"access_key_id" yaml:"access_key_id" env:"AWS_ACCESS_KEY_ID"`         // 🔐 Environment'dan
SecretAccessKey string `json:"secret_access_key" yaml:"secret_access_key" env:"AWS_SECRET_ACCESS_KEY"` // 🔐 Environment'dan
Region          string `json:"region" yaml:"region" env:"AWS_REGION"`
S3Bucket        string `json:"s3_bucket" yaml:"s3_bucket" env:"AWS_S3_BUCKET"`
}

type RedisConfig struct {
Host     string `json:"host" yaml:"host" env:"REDIS_HOST"`
Port     string `json:"port" yaml:"port" env:"REDIS_PORT"`
Password string `json:"password" yaml:"password" env:"REDIS_PASSWORD"` // 🔐 Environment'dan
}

type ModulesConfig struct {
ConfigPath string            `json:"config_path" yaml:"config_path" env:"MODULES_CONFIG"`
States     map[string]string `json:"states" yaml:"states"`
}

type FeaturesConfig struct {
OpenAPI     bool `json:"openapi" yaml:"openapi" env:"ENABLE_OPENAPI"`
CSRF        bool `json:"csrf" yaml:"csrf" env:"ENABLE_CSRF"`
CORS        bool `json:"cors" yaml:"cors" env:"ENABLE_CORS"`
RateLimit   bool `json:"rate_limit" yaml:"rate_limit" env:"ENABLE_RATE_LIMIT"`
}

func New() *Config {
return &Config{}
}

func getEnv(key, defaultValue string) string {
if value := os.Getenv(key); value != "" {
return value
}
return defaultValue
}

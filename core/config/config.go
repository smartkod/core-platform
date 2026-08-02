package config

import (
"os"
)

type Config struct {
Server   ServerConfig   `json:"server" yaml:"server"`
Database DatabaseConfig `json:"database" yaml:"database"`
Logger   LoggerConfig   `json:"logger" yaml:"logger"`
}

type ServerConfig struct {
Host string `json:"host" yaml:"host" env:"SERVER_HOST"`
Port string `json:"port" yaml:"port" env:"SERVER_PORT"`
}

type DatabaseConfig struct {
Host     string `json:"host" yaml:"host" env:"DB_HOST"`
Port     string `json:"port" yaml:"port" env:"DB_PORT"`
User     string `json:"user" yaml:"user" env:"DB_USER"`
Password string `json:"password" yaml:"password" env:"DB_PASSWORD"`
Name     string `json:"name" yaml:"name" env:"DB_NAME"`
SSLMode  string `json:"sslmode" yaml:"sslmode" env:"DB_SSLMODE"`
}

type LoggerConfig struct {
Level  string `json:"level" yaml:"level" env:"LOG_LEVEL"`
Format string `json:"format" yaml:"format" env:"LOG_FORMAT"`
}

func New() *Config {
return &Config{
Server: ServerConfig{
Host: getEnv("SERVER_HOST", "0.0.0.0"),
Port: getEnv("SERVER_PORT", "8080"),
},
Database: DatabaseConfig{
Host:     getEnv("DB_HOST", "localhost"),
Port:     getEnv("DB_PORT", "5432"),
User:     getEnv("DB_USER", "postgres"),
Password: getEnv("DB_PASSWORD", "postgres"),
Name:     getEnv("DB_NAME", "core_platform"),
SSLMode:  getEnv("DB_SSLMODE", "disable"),
},
Logger: LoggerConfig{
Level:  getEnv("LOG_LEVEL", "info"),
Format: getEnv("LOG_FORMAT", "json"),
},
}
}

func getEnv(key, defaultValue string) string {
if value := os.Getenv(key); value != "" {
return value
}
return defaultValue
}
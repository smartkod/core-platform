package logger

import (
"go.uber.org/zap"
"go.uber.org/zap/zapcore"
)

type Logger struct {
*zap.Logger
}

func New(level, format string) *Logger {
var config zap.Config

if format == "json" {
config = zap.NewProductionConfig()
} else {
config = zap.NewDevelopmentConfig()
}

// Set log level
var zapLevel zapcore.Level
switch level {
case "debug":
zapLevel = zapcore.DebugLevel
case "info":
zapLevel = zapcore.InfoLevel
case "warn":
zapLevel = zapcore.WarnLevel
case "error":
zapLevel = zapcore.ErrorLevel
default:
zapLevel = zapcore.InfoLevel
}
config.Level = zap.NewAtomicLevelAt(zapLevel)

logger, err := config.Build()
if err != nil {
panic("Failed to initialize logger: " + err.Error())
}
return &Logger{logger}
}

// Package logger wraps zap behind the narrow core.Logger interface so modules
// never depend on a logging library.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"core-platform/contract/core"
)

// New builds a logger from the resolved config.
func New(level, format string) (core.Logger, error) {
	lvl := zapcore.InfoLevel
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = zapcore.InfoLevel
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.Encoding = "json"
	if format == "console" {
		cfg = zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(lvl)
		cfg.Encoding = "console"
	}
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	z, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return &wrapper{z: z}, nil
}

type wrapper struct{ z *zap.Logger }

func (w *wrapper) Info(msg string, kv ...any)  { w.z.Sugar().Infow(msg, kv...) }
func (w *wrapper) Warn(msg string, kv ...any)  { w.z.Sugar().Warnw(msg, kv...) }
func (w *wrapper) Error(msg string, kv ...any) { w.z.Sugar().Errorw(msg, kv...) }

func (w *wrapper) With(kv ...any) core.Logger {
	return &wrapper{z: w.z.Sugar().With(kv...).Desugar()}
}

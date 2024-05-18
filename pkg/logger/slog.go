package logger

import (
	"context"
	"log/slog"
	"os"
)

type SlogLogger struct {
	logger *slog.Logger
}

type SlogEnvironment string

const (
	EnvLocal SlogEnvironment = "local"
	EnvDev   SlogEnvironment = "dev"
)

func NewSlogLogger(env SlogEnvironment) *SlogLogger {
	var slogger *slog.Logger

	switch env {
	case EnvLocal:
		slogger = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case EnvDev:
		slogger = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	}

	return &SlogLogger{
		logger: slogger,
	}
}

func (s *SlogLogger) Debug(msg string, fields ...any) {
	s.logger.Debug(msg, fields)
}

func (s *SlogLogger) Info(msg string, fields ...any) {
	s.logger.Info(msg, fields)
}

func (s *SlogLogger) Warn(msg string, fields ...any) {
	s.logger.Warn(msg, fields)
}

func (s *SlogLogger) Error(msg string, fields ...any) {
	s.logger.Error(msg, fields)
}

func (s *SlogLogger) DebugContext(ctx context.Context, msg string, fields ...any) {
	s.logger.DebugContext(ctx, msg, fields)
	s.logger.With()
}

func (s *SlogLogger) InfoContext(ctx context.Context, msg string, fields ...any) {
	s.logger.InfoContext(ctx, msg, fields)
}

func (s *SlogLogger) WarnContext(ctx context.Context, msg string, fields ...any) {
	s.logger.WarnContext(ctx, msg, fields)
}

func (s *SlogLogger) ErrorContext(ctx context.Context, msg string, fields ...any) {
	s.logger.ErrorContext(ctx, msg, fields)
}

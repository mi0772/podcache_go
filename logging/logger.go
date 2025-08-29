package logging

import (
	"log/slog"
	"os"
)

// Logger interface for dependency injection and testing
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Fatal(msg string, args ...any)
}

// SlogLogger wraps slog.Logger to implement our Logger interface
type SlogLogger struct {
	logger *slog.Logger
}

// NewLogger creates a new configured logger
func NewLogger(level slog.Level) Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	return &SlogLogger{logger: logger}
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger() Logger {
	return NewLogger(slog.LevelInfo)
}

// NewDebugLogger creates a logger with debug level enabled
func NewDebugLogger() Logger {
	return NewLogger(slog.LevelDebug)
}

func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *SlogLogger) Fatal(msg string, args ...any) {
	l.logger.Error(msg, args...)
	os.Exit(1)
}

// NoOpLogger for performance-critical sections or testing
type NoOpLogger struct{}

func NewNoOpLogger() Logger {
	return &NoOpLogger{}
}

func (l *NoOpLogger) Debug(msg string, args ...any) {}
func (l *NoOpLogger) Info(msg string, args ...any)  {}
func (l *NoOpLogger) Warn(msg string, args ...any)  {}
func (l *NoOpLogger) Error(msg string, args ...any) {}
func (l *NoOpLogger) Fatal(msg string, args ...any) {
	os.Exit(1)
}

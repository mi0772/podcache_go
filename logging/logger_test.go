package logging

import (
	"log/slog"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger(slog.LevelInfo)
	if logger == nil {
		t.Fatal("Expected logger to be created")
	}
}

func TestNoOpLogger(t *testing.T) {
	logger := NewNoOpLogger()

	// These should not panic or cause issues
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
}

func TestLoggerMethods(t *testing.T) {
	// Test that all methods are available and callable
	logger := NewDefaultLogger()

	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}

func TestStructuredLogging(t *testing.T) {
	logger := NewDebugLogger()

	// Test helper functions don't panic
	LogServerPhase(logger, OpStarting)
	LogServerPhase(logger, OpStarted, "port", 8080)
	LogCacheStats(logger, "general", "capacity", 1000)
}

package logging

import (
	"encoding/json"
)

// Component names for structured logging
const (
	ComponentServer    = "TCP Server"
	ComponentCache     = "Cache"
	ComponentMemCache  = "Memory Cache"
	ComponentDiskCache = "Disk Cache"
	ComponentMain      = "Main"
)

// Operation names for structured logging
const (
	OpStarting = "starting"
	OpStarted  = "started"
	OpStopping = "stopping"
	OpStopped  = "stopped"
	OpGet      = "get"
	OpPut      = "put"
	OpEvict    = "evict"
	OpDelete   = "delete"
	OpShutdown = "shutdown"
)

// Results for structured logging
const (
	ResultFound    = "found"
	ResultNotFound = "not found"
	ResultInserted = "inserted"
	ResultUpdated  = "updated"
	ResultEvicted  = "evicted"
)

// Helper functions for common logging patterns
func LogServerPhase(logger Logger, phase string, args ...any) {
	logger.Info(ComponentServer, append([]any{"phase", phase}, args...)...)
}

func LogServerError(logger Logger, phase string, err error, args ...any) {
	logger.Error(ComponentServer, append([]any{"phase", phase, "error", err}, args...)...)
}

func LogCacheOperation(logger Logger, operation, key, result string, args ...any) {
	logger.Debug(ComponentMemCache, append([]any{"operation", operation, "key", key, "result", result}, args...)...)
}

func LogCacheStats(logger Logger, component string, args ...any) {
	logger.Info(component+" statistics", args...)
}

// LogStatsAsJSON logs statistics as a single JSON line
func LogStatsAsJSON(logger Logger, stats interface{}) {
	if statsJSON, err := json.Marshal(stats); err != nil {
		logger.Error("Failed to marshal stats to JSON", "error", err)
	} else {
		logger.Info("Cache statistics", "stats_json", string(statsJSON))
	}
}

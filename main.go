package main

import (
	"context"
	"fmt"
	"mi0772/podcache/cache"
	"mi0772/podcache/logging"
	"mi0772/podcache/server"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	AppVersion          = "0.0.1"
	DefaultPartitions   = 3
	DefaultCapacityMB   = 100
	ShutdownTimeoutSecs = 10
)

var ticker *time.Ticker
var tickerShrink *time.Ticker

var podcache *cache.PodCache
var logger logging.Logger

type CacheConfiguration struct {
	partition uint8
	capacity  uint64
}

func main() {
	// Initialize logger
	logger = logging.NewDebugLogger()

	logger.Info("Welcome to PodCache", "version", AppVersion)
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	setupGracefulShutdown(cancel)

	setupTickerCacheStatistics()
	setupTickerCacheShrink()

	// Read configuration
	config, err := readCacheConfiguration()
	if err != nil {
		logger.Fatal("Failed to read configuration", "error", err)
	}

	displayConfiguration(config)

	// Initialize cache
	podcache, err = initializeCache(config)
	if err != nil {
		logger.Fatal("Failed to initialize cache", "error", err)
	}

	// Start server
	if err := startServer(ctx, podcache); err != nil {
		logger.Fatal("Server failed", "error", err)
	}

	logger.Info("PodCache server shutdown complete")
}

func setupTickerCacheStatistics() {
	ticker = time.NewTicker(60 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-ticker.C:
				if podcache != nil {
					var stat = podcache.Stats()
					logging.LogStatsAsJSON(logger, stat)
				}
			}
		}
	}()
}

func setupTickerCacheShrink() {
	tickerShrink = time.NewTicker(300 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case _ = <-tickerShrink.C:
				if podcache != nil {
					podcache.Shrink()
				}
			}
		}
	}()
}

func setupGracefulShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {

		sig := <-sigChan
		ticker.Stop()
		logger.Info("Graceful shutdown initiated", "signal", sig)
		cancel()

		// Force exit after timeout
		time.AfterFunc(ShutdownTimeoutSecs*time.Second, func() {
			logger.Error("Shutdown timeout exceeded, forcing exit", "timeout_seconds", ShutdownTimeoutSecs)
			os.Exit(1)
		})
	}()
}

func readCacheConfiguration() (*CacheConfiguration, error) {
	config := &CacheConfiguration{}

	// Read partitions
	partition, err := readEnvUint8("PODCACHE_PARTITIONS", DefaultPartitions)
	if err != nil {
		return nil, fmt.Errorf("invalid PODCACHE_PARTITIONS: %w", err)
	}
	config.partition = partition

	// Read capacity
	capacityMB, err := readEnvUint64("PODCACHE_CAPACITY_MB", DefaultCapacityMB)
	if err != nil {
		return nil, fmt.Errorf("invalid PODCACHE_CAPACITY_MB: %w", err)
	}
	config.capacity = capacityMB * 1024 * 1024

	return config, nil
}

func readEnvUint8(key string, defaultValue uint8) (uint8, error) {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue, nil
	}

	value, err := strconv.ParseUint(valueStr, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("must be a valid 8-bit unsigned integer: %s", valueStr)
	}

	return uint8(value), nil
}

func readEnvUint64(key string, defaultValue uint64) (uint64, error) {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue, nil
	}

	value, err := strconv.ParseUint(valueStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("must be a valid positive integer: %s", valueStr)
	}

	return value, nil
}

func displayConfiguration(config *CacheConfiguration) {
	logger.Info("Cache Configuration",
		"partitions", config.partition,
		"capacity_mb", config.capacity/(1024*1024),
		"capacity_bytes", config.capacity,
	)
}

func initializeCache(config *CacheConfiguration) (*cache.PodCache, error) {
	cache, err := cache.NewPodCache(config.partition, config.capacity, logger)
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func startServer(ctx context.Context, cache *cache.PodCache) error {
	server := server.NewPodCacheServer(cache, logger)

	// Start server in the context (blocking call)
	if err := server.Start(ctx); err != nil {
		// Don't treat context cancellation as an error
		if err == context.Canceled {
			logger.Info("Server stopped gracefully")
			return nil
		}
		logging.LogServerError(logger, "server error", err)
		return err
	}

	return nil
}

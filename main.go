package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	cache2 "mi0772/podcache/cache"
	"mi0772/podcache/server"
)

const (
	AppVersion          = "0.0.1"
	DefaultPartitions   = 3
	DefaultCapacityMB   = 100
	ShutdownTimeoutSecs = 10
)

type CacheConfiguration struct {
	partition uint8
	capacity  uint64
}

func main() {
	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Printf("PodCache v%s\n", AppVersion)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	setupGracefulShutdown(cancel)

	// Read configuration
	config, err := readCacheConfiguration()
	if err != nil {
		log.Fatalf("Failed to read configuration: %v", err)
	}

	displayConfiguration(config)

	// Initialize cache
	cache, err := initializeCache(config)
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}

	// Start server
	if err := startServer(ctx, cache); err != nil {
		log.Fatalf("Server failed: %v", err)
	}

	fmt.Println("PodCache server shutdown complete")
}

func setupGracefulShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, initiating graceful shutdown...", sig)
		cancel()

		// Force exit after timeout
		time.AfterFunc(ShutdownTimeoutSecs*time.Second, func() {
			log.Printf("Shutdown timeout exceeded (%ds), forcing exit", ShutdownTimeoutSecs)
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
	fmt.Println("\nCache Configuration:")
	fmt.Printf("  • Partitions: %d\n", config.partition)
	fmt.Printf("  • Capacity: %d MB (%d bytes)\n",
		config.capacity/(1024*1024), config.capacity)
	fmt.Println()
}

func initializeCache(config *CacheConfiguration) (*cache2.PodCache, error) {
	fmt.Println("Initializing cache layer...")

	cache, err := cache2.NewPodCache(config.partition, config.capacity)
	if err != nil {
		return nil, err
	}

	fmt.Println("Cache layer created successfully")
	return cache, nil
}

func startServer(ctx context.Context, cache *cache2.PodCache) error {
	fmt.Println("Starting PodCache server...")

	server := server.NewPodCacheServer(cache)

	// Start server in the context (blocking call)
	if err := server.Start(ctx); err != nil {
		// Don't treat context cancellation as an error
		if err == context.Canceled {
			fmt.Println("Server stopped gracefully")
			return nil
		}
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

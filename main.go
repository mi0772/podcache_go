package main

import (
	"fmt"
	"log"
	cache2 "mi0772/podcache/cache"
	"mi0772/podcache/server"
	"os"
	"strconv"
)

type CacheConfiguration struct {
	partition uint8
	capacity  uint64
}

func main() {

	fmt.Println("PodCache v0.0.1")
	cacheConfig := readCacheConfiguration()

	fmt.Println("Creating new cache layer with this configuration:")
	fmt.Println("Partition:", cacheConfig.partition)
	fmt.Printf("Capacity: %d Mb, %d bytes", cacheConfig.capacity/(1024*1024), cacheConfig.capacity)

	cache, err := cache2.NewPodCache(cacheConfig.partition, cacheConfig.capacity)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Cache layer created successfully")
	s := server.NewPodCacheServer(cache)

	s.Bootstrap()

}

func readCacheConfiguration() CacheConfiguration {
	configuration := CacheConfiguration{}

	if v, found := os.LookupEnv("PODCACHE_PARTITIONS"); !found {
		configuration.partition = 3
	} else {
		c, err := strconv.Atoi(v)
		if err != nil {
			panic("PODCACHE_PARTITIONS must be an integer")
		} else {
			configuration.partition = uint8(c)
		}

	}

	if v, found := os.LookupEnv("PODCACHE_CAPACITY_MB"); !found {
		configuration.capacity = 100 * 1024 * 1024
	} else {
		c, err := strconv.Atoi(v)
		if err != nil {
			panic("PODCACHE_CAPACITY_MB must be an integer")
		} else {
			configuration.capacity = uint64(c * 1024 * 1024)
		}
	}
	return configuration
}

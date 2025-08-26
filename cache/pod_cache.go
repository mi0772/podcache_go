package cache

import (
	"errors"
	"fmt"
	"log"
	"mi0772/podcache/disk"
	"mi0772/podcache/ram"
)

type PodCache struct {
	partitions      []*ram.Cache[[]byte]
	disk_cache      *disk.Cache
	partition_count uint8
	capacity        uint64
}

func NewPodCache(partitions uint8, capacity uint64) (*PodCache, error) {
	partition_capacity := capacity / uint64(partitions)

	p := make([]*ram.Cache[[]byte], int(partitions))
	for i := 0; i < int(partitions); i++ {
		p[i] = ram.New[[]byte](partition_capacity)
		if p[i] == nil {
			panic("ram.New() returned nil")
		}
	}

	dc := disk.NewCache()

	return &PodCache{
		partitions:      p,
		disk_cache:      dc,
		capacity:        capacity,
		partition_count: partitions,
	}, nil
}

func (c *PodCache) Put(key string, value []byte) error {
	partitionIndex := partitionIndex(key, c.partition_count)
	var partition = c.partitions[partitionIndex]

	var sentinelError = ram.ErrMemoryFull
	for sentinelError == ram.ErrMemoryFull {
		err := partition.Put(key, value, uint64(len(value)))
		if err != nil && errors.Is(err, ram.ErrMemoryFull) {
			tailNode := partition.Tail
			log.Printf("Evicting key %s to disk due to memory pressure, %d bytes left on partition", tailNode.Key, partition.MaxCapacity-partition.CurrentCapacity)
			if tailNode == nil {
				return errors.New("ram.Tail() returned nil, memory full but tail is empty, do you create a cache with 0 bytes of capacity ?")
			}
			//salvo su disco e poi faccio evict dalla memoria
			if err := c.disk_cache.Put(tailNode.Key, tailNode.Value); err != nil {
				return fmt.Errorf("failed to save to disk cache: %w", err)
			}

			if err := partition.Evict(tailNode.Key); err {
				return fmt.Errorf("Eviction of tail node failed, this is strange: %w", err)
			}
		} else {
			sentinelError = nil
		}
	}
	return nil
}

func (c *PodCache) Get(key string) ([]byte, error) {
	partitionIndex := partitionIndex(key, c.partition_count)
	v, found := c.partitions[partitionIndex].Get(key)
	if !found {
		log.Printf("ram.Get: key %s not found on partition %d , try looking into disk", key, partitionIndex)
		v, found, err := c.disk_cache.Get(key)
		if err != nil {
			return nil, err
		}
		if !found {
			log.Printf("ram.Get: key %s not found on disk", key)
			return nil, nil
		}

		return v, nil
	}
	log.Printf("ram.Get: key %s found on partition %d", key, partitionIndex)
	return v, nil
}

func partitionIndex(key string, partition_count uint8) uint8 {
	return uint8(hash(key) % uint32(partition_count))
}

func hash(key string) uint32 {
	var hash uint32 = 5381
	for _, c := range []byte(key) {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return hash
}

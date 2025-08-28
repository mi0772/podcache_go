package cache

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"mi0772/podcache/disk"
	"mi0772/podcache/hash"
	"mi0772/podcache/ram"
)

type PodCache struct {
	partitions      []*ram.Cache[[]byte]
	disk_cache      *disk.Cache
	partition_count uint8
	capacity        uint64
}

type PodCacheStats struct {
	Capacity   uint64
	Used       uint64
	Free       uint64
	Partitions []PartitionStats
	Disk       DiskStats
}

type PartitionStats struct {
	Entries  uint64
	Capacity uint64
	Used     uint64
	Free     uint64
}

type DiskStats struct {
	Entries uint64
	Used    uint64
}

func (pc *PodCache) Stats() PodCacheStats {
	result := PodCacheStats{}
	result.Capacity = pc.capacity
	var totalUsed uint64 = 0

	for _, partition := range pc.partitions {
		pstat := PartitionStats{}
		pstat.Capacity = partition.MaxCapacity
		pstat.Entries = uint64(partition.ItemCount())
		pstat.Used = partition.CurrentCapacity
		pstat.Free = partition.MaxCapacity - partition.CurrentCapacity
		totalUsed += pstat.Used
		result.Partitions = append(result.Partitions, pstat)
	}
	result.Disk.Used = pc.disk_cache.Capacity
	result.Disk.Entries = pc.disk_cache.Entries_count
	result.Used = totalUsed
	result.Free = result.Capacity - totalUsed
	return result
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
	slog.Info("Creating Cache", "partitions number", partitions, "partition capacity", partition_capacity)

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

			if !partition.Evict(tailNode.Key) {
				return fmt.Errorf("Eviction of tail node failed, this is strange")
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
		v, found, err := c.disk_cache.Get(key)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, nil
		}

		return v, nil
	}
	return v, nil
}

func (c *PodCache) Evict(key string) bool {
	partitionIndex := partitionIndex(key, c.partition_count)

	if _, found := c.partitions[partitionIndex].Get(key); found {
		return c.partitions[partitionIndex].Evict(key)
	}

	log.Printf("ram.Get: key %s not found on partition %d, try looking into disk", key, partitionIndex)

	_, found, err := c.disk_cache.Get(key)
	if err != nil || !found {
		log.Printf("disk.Get: key %s not found or error: %v", key, err)
		return false
	}

	ok, err := c.disk_cache.Evict(key)
	if err != nil {
		log.Printf("disk.Evict error for key %s: %v", key, err)
		return false
	}

	return ok
}

func partitionIndex(key string, partition_count uint8) uint8 {
	return uint8(hash.CalculateDJB2(key) % uint32(partition_count))
}

package cache

import (
	"errors"
	"fmt"
	"mi0772/podcache/disk"
	"mi0772/podcache/hash"
	"mi0772/podcache/logging"
	"mi0772/podcache/ram"
	"time"
)

type PodCache struct {
	partitions      []*ram.Cache[[]byte]
	disk_cache      *disk.Cache
	partition_count uint8
	capacity        uint64
	logger          logging.Logger
}

type PodCacheStats struct {
	Timestamp  time.Time        `json:"timestamp"`
	Capacity   uint64           `json:"capacity"`
	Used       uint64           `json:"used"`
	Free       uint64           `json:"free"`
	Partitions []PartitionStats `json:"partitions"`
	Disk       DiskStats        `json:"disk"`
}

type PartitionStats struct {
	Entries  uint64  `json:"entries"`
	Capacity uint64  `json:"capacity"`
	Used     uint64  `json:"used"`
	Free     uint64  `json:"free"`
	Hits     uint64  `json:"hits"`
	Misses   uint64  `json:"misses"`
	HitRatio float64 `json:"hit_ratio"`
}

type DiskStats struct {
	Entries uint64 `json:"entries"`
	Used    uint64 `json:"used"`
}

func (pc *PodCache) Stats() PodCacheStats {
	result := PodCacheStats{
		Timestamp: time.Now(),
	}
	result.Capacity = pc.capacity
	var totalUsed uint64 = 0

	for _, partition := range pc.partitions {
		pstat := PartitionStats{}
		pstat.Capacity = partition.MaxCapacity
		pstat.Entries = uint64(partition.ItemCount())
		pstat.Used = partition.CurrentCapacity
		pstat.Free = partition.MaxCapacity - partition.CurrentCapacity
		totalUsed += pstat.Used
		pstat.Hits = partition.Hits
		pstat.Misses = partition.Misses

		// Calculate hit ratio
		total := pstat.Hits + pstat.Misses
		if total > 0 {
			pstat.HitRatio = float64(pstat.Hits) / float64(total)
		}

		result.Partitions = append(result.Partitions, pstat)
	}
	result.Disk.Used = pc.disk_cache.Capacity
	result.Disk.Entries = pc.disk_cache.Entries_count
	result.Used = totalUsed
	result.Free = result.Capacity - totalUsed
	return result
}

func NewPodCache(partitions uint8, capacity uint64, logger logging.Logger) (*PodCache, error) {
	partition_capacity := capacity / uint64(partitions)

	p := make([]*ram.Cache[[]byte], int(partitions))
	for i := 0; i < int(partitions); i++ {
		p[i] = ram.New[[]byte](partition_capacity)
		if p[i] == nil {
			panic("ram.New() returned nil")
		}
	}
	logger.Info("Creating Cache", "partitions_number", partitions, "partition_capacity", partition_capacity)

	dc := disk.NewCache()

	return &PodCache{
		partitions:      p,
		disk_cache:      dc,
		capacity:        capacity,
		partition_count: partitions,
		logger:          logger,
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
			var m = fmt.Sprintf("Evicting key %s to disk due to memory pressure, %d bytes left on partition", tailNode.Key, partition.MaxCapacity-partition.CurrentCapacity)
			c.logger.Debug("Cache proxy", "operation", "put", "event", m)

			if tailNode == nil {
				return errors.New("ram.Tail() returned nil, memory full but tail is empty, do you create a cache with 0 bytes of capacity ")
			}
			//salvo su disco e poi faccio evict dalla memoria
			if err := c.disk_cache.Put(tailNode.Key, tailNode.Value); err != nil {
				return fmt.Errorf("failed to save to disk cache: %w", err)
			}

			if !partition.Evict(tailNode.Key) {
				return fmt.Errorf("eviction of tail node failed, this is abnormal condition")
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

	m := fmt.Sprintf("ram.Get: key %s not found on partition %d, try looking into disk", key, partitionIndex)
	c.logger.Debug("Cache proxy", "operation", "evict", "event", m)

	_, found, err := c.disk_cache.Get(key)
	if err != nil || !found {

		c.logger.Debug("Cache proxy", "operation", "evict", "event", fmt.Sprintf("disk.Get: key %s not found or error: %v", key, err))
		return false
	}

	ok, err := c.disk_cache.Evict(key)
	if err != nil {

		c.logger.Debug("Cache proxy", "operation", "evict", "event", fmt.Sprintf("disk.Evict error for key %s: %v", key, err))
		return false
	}

	return ok
}

func (pc *PodCache) Shrink() {
	pc.logger.Info("Cache shrink operation", "status", "initiated")
	for i, partition := range pc.partitions {
		pc.logger.Debug("Cache shrink", "partition", i)
		partition.Shrink()
	}
	pc.logger.Info("Cache shrink operation", "status", "completed")
}

func partitionIndex(key string, partition_count uint8) uint8 {
	return uint8(hash.CalculateDJB2(key) % uint32(partition_count))
}

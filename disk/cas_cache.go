package disk

import (
	"fmt"
	"mi0772/podcache/disk/hashpath"
	"mi0772/podcache/util"
	"os"
	"path/filepath"
)

type Cache struct {
	entries       map[string]bool
	basePath      string
	entries_count uint64
	capacity      uint64
}

func NewCache() *Cache {
	//creo la directory di base, con un hash casuale di 8 bytes
	bpath, ok := os.LookupEnv("CAS_BASE_PATH")
	if !ok {
		panic("CAS_BASE_PATH environment variable not set")
	}

	finalPath, err := createBasePath(bpath)
	if err != nil {
		panic(err)
	}

	return &Cache{
		basePath:      finalPath,
		entries_count: 0,
		capacity:      0,
		entries:       make(map[string]bool, 0),
	}
}

func (c *Cache) Get(key string) ([]byte, bool, error) {
	if _, exist := c.entries[key]; !exist {
		return nil, false, nil
	}

	entryPath := filepath.Join(c.basePath, hashpath.PathFromKey(key))
	valuePath := filepath.Join(entryPath, "value.dat")

	v, err := os.ReadFile(valuePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load entry value: %w", err)
	}
	return v, true, nil
}

func (c *Cache) Evict(key string) (bool, error) {
	if _, exist := c.entries[key]; !exist {
		return false, nil
	}
	entryPath := filepath.Join(c.basePath, hashpath.PathFromKey(key))
	valuePath := filepath.Join(entryPath, "value.dat")

	if err := os.RemoveAll(valuePath); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Cache) Put(key string, value []byte) error {
	if _, exists := c.entries[key]; exists {
		return fmt.Errorf("entry with key %q already present in disk cache", key)
	}

	entryPath := filepath.Join(c.basePath, hashpath.PathFromKey(key))
	valuePath := filepath.Join(entryPath, "value.dat")

	if err := os.MkdirAll(entryPath, 0755); err != nil {
		return fmt.Errorf("failed to create entry dir: %w", err)
	}

	if err := os.WriteFile(valuePath, value, 0644); err != nil {
		return fmt.Errorf("failed to write value file: %w", err)
	}

	c.entries[key] = true
	c.entries_count++
	c.capacity += uint64(len(value))

	return nil
}

func createBasePath(basePath string) (string, error) {
	var e = os.ErrExist
	var path string

	for e == os.ErrExist {
		randomPath, err := util.RandomString(8)
		if err != nil {
			panic(err)
		}
		path = fmt.Sprintf("%s/%s", basePath, randomPath)
		e = os.MkdirAll(path, 0755)
	}
	return path, nil
}

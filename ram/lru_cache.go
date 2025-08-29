package ram

import (
	"errors"
	"sync"
	"time"
)

type Node[T any] struct {
	Key           string
	Value         T
	ValueSize     uint64
	InsertionTime time.Time
	Next          *Node[T]
	Previous      *Node[T]
}

type Cache[T any] struct {
	Head            *Node[T]
	Tail            *Node[T]
	buckets         map[string]*Node[T]
	MaxCapacity     uint64
	CurrentCapacity uint64
	mutex           sync.RWMutex

	// Stats opzionali per monitoring
	Hits   uint64
	Misses uint64
}

var (
	ErrMemoryFull = errors.New("memory full")
)

func New[T any](maxCapacity uint64) *Cache[T] {
	// Stima più intelligente della dimensione iniziale
	initialSize := 1000
	if maxCapacity < 100 {
		initialSize = 100
	} else if maxCapacity > 10000 {
		initialSize = int(maxCapacity / 10) // ~10% della capacità massima
	}

	return &Cache[T]{
		buckets:     make(map[string]*Node[T], initialSize),
		MaxCapacity: maxCapacity,
		mutex:       sync.RWMutex{},
	}
}

func (c *Cache[T]) ItemCount() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.buckets)
}

func (c *Cache[T]) Stats() (hits, misses uint64, hitRatio float64) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	total := c.Hits + c.Misses
	if total == 0 {
		return c.Hits, c.Misses, 0.0
	}
	return c.Hits, c.Misses, float64(c.Hits) / float64(total)
}

func (c *Cache[T]) Capacity() (current, max uint64) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.CurrentCapacity, c.MaxCapacity
}

func (c *Cache[T]) Shrink() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Ricrea la mappa per ottimizzare l'allocazione interna
	// Utile dopo molte operazioni di delete
	newBuckets := make(map[string]*Node[T], len(c.buckets))
	for k, v := range c.buckets {
		newBuckets[k] = v
	}
	c.buckets = newBuckets
}

func (c *Cache[T]) Get(key string) (T, bool) {
	var zero T

	c.mutex.Lock()
	defer c.mutex.Unlock()

	v, ok := c.buckets[key]
	if !ok {
		c.Misses++
		return zero, false
	}

	c.Hits++
	c.moveToHead(v)
	return v.Value, true
}

func (c *Cache[T]) Evict(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	v, ok := c.buckets[key]
	if !ok {
		return false
	}

	c.removeFromList(v)
	delete(c.buckets, key)
	c.CurrentCapacity -= v.ValueSize
	return true
}

func (c *Cache[T]) Put(key string, value T, valueSize uint64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if v, exists := c.buckets[key]; exists {
		// Aggiorna elemento esistente
		newCapacity := c.CurrentCapacity - v.ValueSize + valueSize
		if newCapacity > c.MaxCapacity {
			return ErrMemoryFull
		}

		c.CurrentCapacity = newCapacity
		v.InsertionTime = time.Now()
		v.Value = value
		v.ValueSize = valueSize
		c.moveToHead(v)
		return nil
	}

	// Nuovo elemento
	if c.CurrentCapacity+valueSize > c.MaxCapacity {
		return ErrMemoryFull
	}

	newNode := &Node[T]{
		Key:           key,
		Value:         value,
		ValueSize:     valueSize,
		InsertionTime: time.Now(),
	}

	c.buckets[key] = newNode
	c.CurrentCapacity += valueSize
	c.addToHead(newNode)
	return nil
}

// EvictLRU rimuove l'elemento meno recentemente usato
func (c *Cache[T]) EvictLRU() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.Tail == nil {
		return false
	}

	key := c.Tail.Key
	c.removeFromList(c.Tail)
	delete(c.buckets, key)
	c.CurrentCapacity -= c.buckets[key].ValueSize // Usa la reference prima di delete
	return true
}

// EvictOldest rimuove elementi più vecchi del tempo specificato
func (c *Cache[T]) EvictOldest(maxAge time.Duration) int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.Tail == nil {
		return 0
	}

	cutoff := time.Now().Add(-maxAge)
	evicted := 0

	// Parti dalla coda (elementi più vecchi per LRU)
	current := c.Tail
	for current != nil && current.InsertionTime.Before(cutoff) {
		prev := current.Previous

		c.removeFromList(current)
		delete(c.buckets, current.Key)
		c.CurrentCapacity -= current.ValueSize
		evicted++

		current = prev
	}

	return evicted
}

// PutWithEviction inserisce e fa auto-evict se necessario
func (c *Cache[T]) PutWithEviction(key string, value T, valueSize uint64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Se esiste, aggiorna
	if v, exists := c.buckets[key]; exists {
		newCapacity := c.CurrentCapacity - v.ValueSize + valueSize
		if newCapacity > c.MaxCapacity {
			// Anche aggiornando non ci sta
			return ErrMemoryFull
		}
		c.CurrentCapacity = newCapacity
		v.Value = value
		v.ValueSize = valueSize
		v.InsertionTime = time.Now()
		c.moveToHead(v)
		return nil
	}

	// Nuovo elemento - fai spazio se necessario
	for c.CurrentCapacity+valueSize > c.MaxCapacity && c.Tail != nil {
		tail := c.Tail
		c.removeFromList(tail)
		delete(c.buckets, tail.Key)
		c.CurrentCapacity -= tail.ValueSize
	}

	if c.CurrentCapacity+valueSize > c.MaxCapacity {
		return ErrMemoryFull // Cache troppo piccola per questo elemento
	}

	newNode := &Node[T]{
		Key:           key,
		Value:         value,
		ValueSize:     valueSize,
		InsertionTime: time.Now(),
	}

	c.buckets[key] = newNode
	c.CurrentCapacity += valueSize
	c.addToHead(newNode)
	return nil
}

func (c *Cache[T]) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Head = nil
	c.Tail = nil
	c.CurrentCapacity = 0
	c.Hits = 0
	c.Misses = 0

	// Mantieni la capacità della mappa per evitare riallocazioni
	for k := range c.buckets {
		delete(c.buckets, k)
	}
}

/* ************************************************************************
   Metodi privati - assumono che il caller abbia già acquisito c.mutex.Lock()
 * ************************************************************************ */

func (c *Cache[T]) removeFromList(node *Node[T]) {
	if node == nil {
		return
	}

	// Caso speciale: unico elemento
	if c.Head == node && c.Tail == node {
		c.Head = nil
		c.Tail = nil
		return
	}

	// Aggiorna i collegamenti
	if node.Previous != nil {
		node.Previous.Next = node.Next
	} else {
		// node è head
		c.Head = node.Next
	}

	if node.Next != nil {
		node.Next.Previous = node.Previous
	} else {
		// node è tail
		c.Tail = node.Previous
	}

	// Pulisci i riferimenti (importante per GC)
	node.Next = nil
	node.Previous = nil
}

func (c *Cache[T]) moveToHead(node *Node[T]) {
	if node == nil || c.Head == node {
		return
	}

	// Prima rimuovi dalla posizione corrente
	if node.Previous != nil {
		node.Previous.Next = node.Next
	}
	if node.Next != nil {
		node.Next.Previous = node.Previous
	} else {
		// Era tail
		c.Tail = node.Previous
	}

	// Poi aggiungi in testa
	node.Previous = nil
	node.Next = c.Head
	if c.Head != nil {
		c.Head.Previous = node
	} else {
		// Lista era vuota
		c.Tail = node
	}
	c.Head = node
}

func (c *Cache[T]) addToHead(node *Node[T]) {
	if node == nil {
		return
	}

	node.Previous = nil

	if c.Head == nil {
		// Lista vuota
		c.Head = node
		c.Tail = node
		node.Next = nil
	} else {
		// Lista non vuota
		node.Next = c.Head
		c.Head.Previous = node
		c.Head = node
	}
}

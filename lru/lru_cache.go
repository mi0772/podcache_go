package lru

import (
	"errors"
	"log"
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
	maxCapacity     uint64
	currentCapacity uint64
	mutex           sync.RWMutex
}

var (
	ErrMemoryFull = errors.New("memory full")
)

func New[T any](maxCapacity uint64) *Cache[T] {
	log.Printf("creating lru cache with max capacity %d", maxCapacity)
	bucketsSize := estimateBucketSize(maxCapacity)
	buckets := make(map[string]*Node[T], bucketsSize)
	log.Printf("esimated bucket size %d ", bucketsSize)

	return &Cache[T]{
		Head:            nil,
		Tail:            nil,
		buckets:         buckets,
		maxCapacity:     maxCapacity,
		currentCapacity: 0,
		mutex:           sync.RWMutex{},
	}
}

func (c *Cache[T]) Get(key string) (T, bool) {
	var zero T
	c.mutex.Lock()
	defer c.mutex.Unlock()

	v, ok := c.buckets[key]
	if !ok {
		return zero, false
	}

	// prima di restituire, sposto il nodo trovato come head
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

	// trovato, devo rimuoverlo ma spostare i riferimenti
	if c.Head == c.Tail {
		c.Head = nil
		c.Tail = nil
	} else if v == c.Head { //sto cancellando elemento in testa
		c.Head = v.Next
		if c.Head != nil {
			c.Head.Previous = nil
		}
	} else if v == c.Tail { // sto cancellando elemento di coda
		c.Tail = v.Previous
		c.Tail.Next = nil
	} else { // elemento centrale
		v.Previous.Next = v.Next
		v.Next.Previous = v.Previous
	}

	delete(c.buckets, key)
	c.currentCapacity -= v.ValueSize
	return true
}

func (c *Cache[T]) Put(key string, value T, valueSize uint64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	v, ok := c.buckets[key]
	if ok {
		// Elemento esistente - controlla se il nuovo size è accettabile
		newCapacity := c.currentCapacity - v.ValueSize + valueSize
		if newCapacity > c.maxCapacity {
			return ErrMemoryFull
		}

		c.currentCapacity = newCapacity
		v.InsertionTime = time.Now()
		v.Value = value
		v.ValueSize = valueSize
		c.moveToHead(v)
	} else {
		// Elemento nuovo - controlla se c'è spazio
		if c.currentCapacity+valueSize > c.maxCapacity {
			return ErrMemoryFull
		}

		v := &Node[T]{
			Key:           key,
			Value:         value,
			ValueSize:     valueSize,
			InsertionTime: time.Now(),
			Next:          nil,
			Previous:      nil,
		}
		c.buckets[key] = v
		c.currentCapacity += valueSize
		c.addToHead(v)
	}
	return nil
}

/* ************************************************************************
   private methods
 * ************************************************************************ */

func (c *Cache[T]) moveToHead(node *Node[T]) {
	if node == nil || c.Head == node {
		return
	}

	// Rimuovi il nodo dalla sua posizione attuale
	if node.Previous != nil {
		node.Previous.Next = node.Next
	}
	if node.Next != nil {
		node.Next.Previous = node.Previous
	}

	// Aggiorna tail se necessario
	if c.Tail == node {
		c.Tail = node.Previous
	}

	// Aggiungi in testa
	node.Previous = nil
	node.Next = c.Head
	if c.Head != nil {
		c.Head.Previous = node
	}
	c.Head = node

	// Se era l'unico nodo, aggiorna anche tail
	if c.Tail == nil {
		c.Tail = node
	}
}

func (c *Cache[T]) addToHead(node *Node[T]) {
	if node == nil {
		return
	}
	node.Previous = nil
	if c.Head == nil {
		c.Head = node
		c.Tail = node
		node.Next = nil
		return
	}
	node.Next = c.Head
	c.Head.Previous = node
	c.Head = node
}

func estimateBucketSize(maxCapacity uint64) int {
	estimated_elements := maxCapacity / 1024
	target_size := int(float64(estimated_elements) / 0.75)

	size := 16
	for size < target_size && size < 65536 { // max 64K
		size <<= 1
	}

	return size
}

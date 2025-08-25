package ram

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestCreation(t *testing.T) {

	cache := New[string](100_000_000)

	if cache == nil {
		t.Errorf("cache is nil")
	}
}

func TestPut(t *testing.T) {
	cache := New[string](8192) //8kb
	if cache == nil {
		t.Errorf("cache is nil")
	}

	v := "Bella zio questa si che è una stringa"

	if e := cache.Put("carlo", v, uint64(len(v))); e != nil {
		t.Errorf("put fail")
	}

}

func TestGet(t *testing.T) {
	cache := New[string](8192)
	if cache == nil {
		t.Errorf("cache is nil")
		return
	}
	v := "Bella zio questa si che una stringa"

	r := cache.Put("carlo", v, uint64(len(v)))
	if r != nil {
		t.Errorf("put fail")
	}

	value, ok := cache.Get("carlo")
	if !ok {
		t.Errorf("get fail")
	}
	if value != v {
		t.Errorf("get fail")
	}

}

func TestMany(t *testing.T) {
	cache := New[string](1024 * 1024)
	if cache == nil {
		t.Errorf("cache is nil")
	}

	for i := 0; i < 100; i++ {
		if v, err := secureRandomString(100); err != nil {
			t.Errorf("%v", err)
		} else {
			cache.Put(fmt.Sprintf("k-%d", i), v, uint64(len(v)))
		}
	}

	for i := 0; i < 100; i++ {
		_, ok := cache.Get(fmt.Sprintf("k-%d", i))
		if !ok {
			t.Errorf("get fail")
		}
	}
}

func TestEvict(t *testing.T) {
	cache := New[string](1024 * 1024)
	if cache == nil {
		t.Errorf("cache is nil")
		return
	}

	for i := 0; i < 100; i++ {
		if v, err := secureRandomString(100); err != nil {
			t.Errorf("%v", err)
		} else {
			cache.Put(fmt.Sprintf("k-%d", i), v, uint64(len(v)))
		}
	}

	for i := 0; i < 100; i++ {
		if ok := cache.Evict(fmt.Sprintf("k-%d", i)); !ok {
			t.Errorf("evict fail")
		}
	}
}

func secureRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Converti in stringa base64 (sarà più lunga del length originale)
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

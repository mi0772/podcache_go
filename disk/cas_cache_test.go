package disk

import (
	"os"
	"path/filepath"
	"testing"
)

// helper per creare una nuova cache su una directory temporanea
func newTestCache(t *testing.T) *Cache {
	t.Helper()

	base := t.TempDir()
	if err := os.Setenv("CAS_BASE_PATH", base); err != nil {
		t.Fatalf("failed to set CAS_BASE_PATH: %v", err)
	}

	c := NewCache()
	if c == nil {
		t.Fatal("NewCache() returned nil")
	}

	c.basePath = filepath.Join(base)
	return c
}

func TestCache(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		_ = newTestCache(t) // se non crasha, passa
	})

	t.Run("Put", func(t *testing.T) {
		c := newTestCache(t)
		sv := "Ciao sono una stringa"

		if err := c.Put("carlo", []byte(sv)); err != nil {
			t.Fatalf("Put() returned an error: %v", err)
		}
	})

	t.Run("Get", func(t *testing.T) {
		c := newTestCache(t)
		sv := "Ciao sono una stringa"

		if err := c.Put("carlo", []byte(sv)); err != nil {
			t.Fatalf("Put() returned an error: %v", err)
		}

		v, found, err := c.Get("carlo")
		if err != nil {
			t.Fatalf("Get() returned an error: %v", err)
		}
		if !found {
			t.Fatal("Get() did not find the value")
		}
		if string(v) != sv {
			t.Fatalf("Get() returned wrong value: got %q, want %q", string(v), sv)
		}
	})

	t.Run("Evict", func(t *testing.T) {
		c := newTestCache(t)
		sv := "Ciao sono una stringa"

		if err := c.Put("carlo", []byte(sv)); err != nil {
			t.Fatalf("Put() returned an error: %v", err)
		}

		ok, err := c.Evict("carlo")
		if err != nil {
			t.Fatalf("Evict() returned an error: %v", err)
		}
		if !ok {
			t.Fatal("Evict() returned false")
		}
	})
}

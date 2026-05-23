package cache

import (
	"sync"
	"testing"
	"time"
)

func TestNewMetadataCache(t *testing.T) {
	dir := t.TempDir()
	c, err := NewMetadataCache(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
	if c.Entries == nil {
		t.Error("expected non-nil Entries map")
	}
	if len(c.Entries) != 0 {
		t.Errorf("expected empty cache, got %d entries", len(c.Entries))
	}
}

func TestSetAndGet(t *testing.T) {
	dir := t.TempDir()
	c, err := NewMetadataCache(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := &CacheEntry{
		Weight:    42,
		Resolved:  "https://example.com/pkg.tgz",
		Integrity: "sha512-abc123",
		CachedAt:  time.Now().Truncate(time.Second),
	}

	c.Set("test-pkg", entry)

	got := c.Get("test-pkg")
	if got == nil {
		t.Fatal("expected non-nil entry")
	}
	if got.Weight != entry.Weight {
		t.Errorf("expected weight %d, got %d", entry.Weight, got.Weight)
	}
	if got.Resolved != entry.Resolved {
		t.Errorf("expected resolved %s, got %s", entry.Resolved, got.Resolved)
	}
	if got.Integrity != entry.Integrity {
		t.Errorf("expected integrity %s, got %s", entry.Integrity, got.Integrity)
	}
	if !got.CachedAt.Equal(entry.CachedAt) {
		t.Errorf("expected cached_at %v, got %v", entry.CachedAt, got.CachedAt)
	}

	// non-existent key returns nil
	if c.Get("nonexistent") != nil {
		t.Error("expected nil for nonexistent key")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	c, err := NewMetadataCache(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := &CacheEntry{
		Weight:    99,
		Resolved:  "https://example.com/bar.tgz",
		Integrity: "sha512-def456",
		CachedAt:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	c.Set("persist-pkg", entry)

	if err := c.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// create a fresh cache from the same directory
	c2, err := NewMetadataCache(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := c2.Get("persist-pkg")
	if got == nil {
		t.Fatal("expected persisted entry to be loaded")
	}
	if got.Weight != 99 {
		t.Errorf("expected weight 99, got %d", got.Weight)
	}
	if got.Resolved != "https://example.com/bar.tgz" {
		t.Errorf("expected resolved https://example.com/bar.tgz, got %s", got.Resolved)
	}
}

func TestConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	c, err := NewMetadataCache(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	const goroutines = 20

	// concurrent writers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			entry := &CacheEntry{
				Weight:    int64(n),
				Resolved:  "https://example.com/pkg.tgz",
				Integrity: "sha512-abc",
				CachedAt:  time.Now(),
			}
			c.Set("shared", entry)
		}(i)
	}

	// concurrent readers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Get("shared")
			_ = c.Get("nonexistent")
		}()
	}

	wg.Wait()

	// verify at least one writer succeeded
	got := c.Get("shared")
	if got == nil {
		t.Error("expected a value for key 'shared' after concurrent writes")
	}
}

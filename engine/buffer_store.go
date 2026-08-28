package engine

import (
	"fmt"
	"sync"
	"time"
)

type bufferItem struct {
	data      any
	expiresAt time.Time
}

type BufferStore struct {
	mu    sync.RWMutex
	items map[string]bufferItem
}

func NewBufferStore(cleanupInterval time.Duration) *BufferStore {
	store := &BufferStore{
		items: make(map[string]bufferItem),
	}
	if cleanupInterval > 0 {
		go store.startCleanupLoop(cleanupInterval)
	}
	return store
}

func (s *BufferStore) Set(key string, val any, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = bufferItem{
		data:      val,
		expiresAt: time.Now().Add(ttl),
	}
}

func (s *BufferStore) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[key]
	if !exists || time.Now().After(item.expiresAt) {
		return nil, false
	}
	return item.data, true
}

func (s *BufferStore) MakeKey(planID, stepID string) string {
	return fmt.Sprintf("%s:%s", planID, stepID)
}

func (s *BufferStore) startCleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, v := range s.items {
			if now.After(v.expiresAt) {
				delete(s.items, k)
			}
		}
		s.mu.Unlock()
	}
}

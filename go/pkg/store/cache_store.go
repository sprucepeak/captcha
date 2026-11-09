package store

import (
	"sync"
	"time"
)

// CacheStore defines cache interface
type CacheStore interface {
	Set(key string, val interface{}, expire time.Duration)
	Get(key string) (interface{}, bool)
	GetAndRemove(key string) (interface{}, bool)
	Delete(key string)
}

type item struct {
	value      interface{}
	expireTime time.Time
}

// MemoryCacheStore implements CacheStore in memory using standard library sync.Map / Mutex
type MemoryCacheStore struct {
	items sync.Map
}

func NewMemoryCacheStore() *MemoryCacheStore {
	store := &MemoryCacheStore{}
	// periodic cleanup goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			store.items.Range(func(k, v interface{}) bool {
				it := v.(*item)
				if !it.expireTime.IsZero() && now.After(it.expireTime) {
					store.items.Delete(k)
				}
				return true
			})
		}
	}()
	return store
}

func (m *MemoryCacheStore) Set(key string, val interface{}, expire time.Duration) {
	var exp time.Time
	if expire > 0 {
		exp = time.Now().Add(expire)
	}
	m.items.Store(key, &item{value: val, expireTime: exp})
}

func (m *MemoryCacheStore) Get(key string) (interface{}, bool) {
	v, ok := m.items.Load(key)
	if !ok {
		return nil, false
	}
	it := v.(*item)
	if !it.expireTime.IsZero() && time.Now().After(it.expireTime) {
		m.items.Delete(key)
		return nil, false
	}
	return it.value, true
}

func (m *MemoryCacheStore) GetAndRemove(key string) (interface{}, bool) {
	v, ok := m.items.LoadAndDelete(key)
	if !ok {
		return nil, false
	}
	it := v.(*item)
	if !it.expireTime.IsZero() && time.Now().After(it.expireTime) {
		return nil, false
	}
	return it.value, true
}

func (m *MemoryCacheStore) Delete(key string) {
	m.items.Delete(key)
}


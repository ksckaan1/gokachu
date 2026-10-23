package gokachu

import (
	"cmp"
	"container/list"
	"hash/maphash"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

var shardSeed = maphash.MakeSeed()

type Gokachu[K comparable, V any] struct {
	shards       []shard[K, V]
	pollInterval time.Duration
	pollCancel   chan struct{}
	closeOnce    sync.Once
	closed       atomic.Bool
	wg           *sync.WaitGroup

	// Hooks
	inc           atomic.Uint64
	hooksCount    atomic.Int64
	hooksMut      sync.RWMutex
	onSetHooks    map[uint64]func(key K, value V, ttl time.Duration)
	onGetHooks    map[uint64]func(key K, value V)
	onMissHooks   map[uint64]func(key K)
	onDeleteHooks map[uint64]func(key K, value V)
}

type Config struct {
	ReplacementStrategy ReplacementStrategy // default: ReplacementStrategyNone
	MaxRecordThreshold  int                 // This parameter is used to control the maximum number of records in the cache. If the number of records exceeds this threshold, records will be deleted according to the replacement strategy.
	ClearNum            int                 // This parameter is used to control the number of records to be deleted.
	PollInterval        time.Duration       // This parameter is used to control the polling interval. If value is 0, uses default = 1 second.
}

// New creates a new Gokachu instance with the given configuration. Do not forgot call Close() function before exit.
func New[K comparable, V any](cfg Config) *Gokachu[K, V] {
	return newGokachu[K, V](cfg, 1)
}

// NewSharded creates a new Gokachu instance split across shardCount independent shards.
// Keys are distributed with hash/maphash.Comparable, so any comparable key type works.
// MaxRecordThreshold and ClearNum are applied per shard, so the total number of records is approximate.
func NewSharded[K comparable, V any](cfg Config, shardCount int) *Gokachu[K, V] {
	if shardCount < 1 {
		shardCount = 1
	}

	return newGokachu[K, V](cfg, shardCount)
}

func newGokachu[K comparable, V any](cfg Config, shardCount int) *Gokachu[K, V] {
	maxRecordThreshold := cfg.MaxRecordThreshold
	clearNum := cfg.ClearNum

	if shardCount > 1 {
		// Keep the total threshold close to the configured one.
		if maxRecordThreshold > 0 {
			maxRecordThreshold = (maxRecordThreshold + shardCount - 1) / shardCount
		}

		if clearNum > 0 {
			clearNum = (clearNum + shardCount - 1) / shardCount
		}
	}

	shards := make([]shard[K, V], shardCount)

	for i := range shards {
		shards[i] = shard[K, V]{
			elems:               list.New(),
			store:               make(map[K]*list.Element),
			maxRecordThreshold:  maxRecordThreshold,
			clearNum:            clearNum,
			replacementStrategy: cfg.ReplacementStrategy,
		}
	}

	g := &Gokachu[K, V]{
		shards:       shards,
		pollInterval: cmp.Or(cfg.PollInterval, time.Second), // Default poll interval is 1 second
		pollCancel:   make(chan struct{}),
		wg:           new(sync.WaitGroup),

		// Hooks
		onSetHooks:    make(map[uint64]func(key K, value V, ttl time.Duration)),
		onGetHooks:    make(map[uint64]func(key K, value V)),
		onMissHooks:   make(map[uint64]func(key K)),
		onDeleteHooks: make(map[uint64]func(key K, value V)),
	}

	g.wg.Add(1)

	go g.poll()

	return g
}

func (g *Gokachu[K, V]) shardFor(key K) *shard[K, V] {
	if len(g.shards) == 1 {
		return &g.shards[0]
	}

	return &g.shards[maphash.Comparable(shardSeed, key)%uint64(len(g.shards))]
}

// Set sets a value in the cache with a TTL. If the TTL is 0, the value will not expire.
func (g *Gokachu[K, V]) Set(key K, v V, ttl time.Duration, hooks ...Hook) {
	s := g.shardFor(key)

	defer s.lock()()

	if g.closed.Load() {
		return
	}

	g.runOnSetHooks(key, v, ttl)

	exp := time.Time{}
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}

	// if exists
	if oldElem, ok := s.store[key]; ok {
		oldElem.Value.(*valueWithTTL[K, V]).value = v
		oldElem.Value.(*valueWithTTL[K, V]).expireTime = exp

		// set individual hooks
		for _, hook := range hooks {
			if hook.OnGet != nil {
				oldElem.Value.(*valueWithTTL[K, V]).hook.OnGet = hook.OnGet
			}

			if hook.OnDelete != nil {
				oldElem.Value.(*valueWithTTL[K, V]).hook.OnDelete = hook.OnDelete
			}
		}

		switch s.replacementStrategy {
		case ReplacementStrategyLRU:
			s.elems.MoveToBack(oldElem)
		case ReplacementStrategyMRU:
			s.elems.MoveToFront(oldElem)
		}

		return
	}

	// if not exists

	// clear if cache is full
	if s.maxRecordThreshold > 0 && s.clearNum > 0 && s.replacementStrategy > ReplacementStrategyNone && len(s.store) >= s.maxRecordThreshold {
		s.clear()
	}

	value := &valueWithTTL[K, V]{
		key:        key,
		value:      v,
		expireTime: exp,
	}

	// set individual hooks
	for _, hook := range hooks {
		if hook.OnGet != nil {
			value.hook.OnGet = hook.OnGet
		}

		if hook.OnDelete != nil {
			value.hook.OnDelete = hook.OnDelete
		}
	}

	switch s.replacementStrategy {
	case ReplacementStrategyFIFO, ReplacementStrategyLRU, ReplacementStrategyLFU, ReplacementStrategyMFU, ReplacementStrategyNone:
		s.store[key] = s.elems.PushBack(value)
	case ReplacementStrategyLIFO, ReplacementStrategyMRU:
		s.store[key] = s.elems.PushFront(value)
	}
}

// Get gets a value from the cache. Returns false in second value if the key does not exist.
func (g *Gokachu[K, V]) Get(key K) (V, bool) {
	s := g.shardFor(key)

	defer s.lock()()

	item, ok := s.store[key]
	if !ok {
		g.runOnMissHooks(key)
		return *new(V), false
	}

	value := item.Value.(*valueWithTTL[K, V])

	switch s.replacementStrategy {
	case ReplacementStrategyLRU:
		s.elems.MoveToBack(item)
	case ReplacementStrategyMRU:
		s.elems.MoveToFront(item)
	case ReplacementStrategyMFU, ReplacementStrategyLFU:
		value.hitCount++

		s.moveByHits(item)
	}

	// run hooks before getting value
	g.runOnGetHooks(key, value.value)

	if value.hook.OnGet != nil {
		value.hook.OnGet()
	}

	return value.value, true
}

// GetAs retrieves a value from the cache and converts it to the specified type. Returns false in second value if the key does not exist.
func (g *Gokachu[K, V]) GetAs[V2 any](key K) (V2, bool) {
	value, ok := g.Get(key)
	if !ok {
		return *new(V2), false
	}

	switch v := any(value).(type) {
	case V2:
		return v, true
	default:
		return *new(V2), false
	}
}

// GetFunc retrieves a first matching value from the cache using a callback function. If all matches return false, the second value also returns false.
func (g *Gokachu[K, V]) GetFunc(cb func(key K, value V) bool) (V, bool) {
	var foundKey K

	found := false

	for i := range g.shards {
		s := &g.shards[i]

		unlock := s.rlock()

		for k, v := range s.store {
			if cb(k, v.Value.(*valueWithTTL[K, V]).value) {
				foundKey = k
				found = true

				break
			}
		}

		unlock()

		if found {
			break
		}
	}

	if found {
		return g.Get(foundKey)
	}

	var zeroV V

	return zeroV, false
}

// HasKey reports whether key exists in the cache without updating access order or running hooks.
func (g *Gokachu[K, V]) HasKey(key K) bool {
	s := g.shardFor(key)

	unlock := s.rlock()
	defer unlock()

	_, ok := s.store[key]

	return ok
}

// Delete deletes a value from the cache and returns true if the key existed.
func (g *Gokachu[K, V]) Delete(key K) bool {
	s := g.shardFor(key)

	defer s.lock()()

	value, ok := s.store[key]
	if ok {
		// run hooks before delete
		g.runOnDeleteHooks(key, value.Value.(*valueWithTTL[K, V]).value)

		if value.Value.(*valueWithTTL[K, V]).hook.OnDelete != nil {
			value.Value.(*valueWithTTL[K, V]).hook.OnDelete()
		}

		// delete
		s.elems.Remove(value)
		delete(s.store, key)
	}

	return ok
}

// DeleteFunc deletes values from the cache for which the callback returns true and returns the number of deleted values.
func (g *Gokachu[K, V]) DeleteFunc(cb func(key K, value V) bool) int {
	count := 0 // deleted count

	for i := range g.shards {
		s := &g.shards[i]

		func() {
			defer s.lock()()

			for key, value := range s.store {
				if cb(key, value.Value.(*valueWithTTL[K, V]).value) {
					// run hooks before delete
					g.runOnDeleteHooks(key, value.Value.(*valueWithTTL[K, V]).value)

					if value.Value.(*valueWithTTL[K, V]).hook.OnDelete != nil {
						value.Value.(*valueWithTTL[K, V]).hook.OnDelete()
					}

					// delete
					s.elems.Remove(value)
					delete(s.store, key)

					count++
				}
			}
		}()
	}

	return count
}

// Flush deletes all values from the cache and return the number of deleted values.
func (g *Gokachu[K, V]) Flush() int {
	count := 0

	for i := range g.shards {
		s := &g.shards[i]

		s.mut.Lock()
		s.elems.Init()
		count += len(s.store)
		clear(s.store)
		s.mut.Unlock()
	}

	return count
}

// Keys returns all keys in the cache.
func (g *Gokachu[K, V]) Keys() []K {
	keys := make([]K, 0)

	for i := range g.shards {
		s := &g.shards[i]

		unlock := s.rlock()

		for e := s.elems.Front(); e != nil; e = e.Next() {
			keys = append(keys, e.Value.(*valueWithTTL[K, V]).key)
		}

		unlock()
	}

	return keys
}

// KeysFunc returns all keys in the cache for which the callback returns true.
func (g *Gokachu[K, V]) KeysFunc(cb func(key K, value V) bool) []K {
	keys := make([]K, 0)

	for i := range g.shards {
		s := &g.shards[i]

		unlock := s.rlock()

		for e := s.elems.Front(); e != nil; e = e.Next() {
			if cb(e.Value.(*valueWithTTL[K, V]).key, e.Value.(*valueWithTTL[K, V]).value) {
				keys = append(keys, e.Value.(*valueWithTTL[K, V]).key)
			}
		}

		unlock()
	}

	return slices.Clip(keys)
}

// Count returns the number of values in the cache.
func (g *Gokachu[K, V]) Count() int {
	count := 0

	for i := range g.shards {
		s := &g.shards[i]

		unlock := s.rlock()
		count += s.elems.Len()

		unlock()
	}

	return count
}

// CountFunc returns the number of values in the cache for which the callback returns true.
func (g *Gokachu[K, V]) CountFunc(cb func(key K, value V) bool) int {
	count := 0

	for i := range g.shards {
		s := &g.shards[i]

		unlock := s.rlock()

		for key, value := range s.store {
			if cb(key, value.Value.(*valueWithTTL[K, V]).value) {
				count++
			}
		}

		unlock()
	}

	return count
}

// Close closes the cache and all associated resources.
func (g *Gokachu[K, V]) Close() {
	g.closeOnce.Do(func() {
		g.closed.Store(true)
		close(g.pollCancel)
	})

	for i := range g.shards {
		s := &g.shards[i]

		s.mut.Lock()
		clear(s.store)
		s.elems.Init()
		s.mut.Unlock()
	}

	// clear hooks
	g.hooksMut.Lock()
	g.onSetHooks = nil
	g.onGetHooks = nil
	g.onDeleteHooks = nil
	g.onMissHooks = nil
	g.hooksCount.Store(0)
	g.hooksMut.Unlock()

	g.wg.Wait()
}

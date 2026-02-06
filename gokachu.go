package gokachu

import (
	"cmp"
	"container/list"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type Gokachu[K comparable, V any] struct {
	elems               *list.List // front of list == greater risk of deletion <---------list---------> back of list == less risk of deletion
	store               map[K]*list.Element
	mut                 *sync.Mutex
	maxRecordThreshold  int
	cleanNum            int
	replacementStrategy ReplacementStrategy
	pollInterval        time.Duration
	pollCancel          chan struct{}
	wg                  *sync.WaitGroup

	// Hooks
	inc           atomic.Uint64
	onSetHooks    map[uint64]func(key K, value V, ttl time.Duration)
	onGetHooks    map[uint64]func(key K, value V)
	onMissHooks   map[uint64]func(key K)
	onDeleteHooks map[uint64]func(key K, value V)
}

type Config struct {
	ReplacementStrategy ReplacementStrategy // default: ReplacementStrategyNone
	MaxRecordThreshold  int                 // This parameter is used to control the maximum number of records in the cache. If the number of records exceeds this threshold, records will be deleted according to the replacement strategy.
	CleanNum            int                 // This parameter is used to control the number of records to be deleted.
	PollInterval        time.Duration       // This parameter is used to control the polling interval. If value is 0, uses default = 1 second.
}

// New creates a new Gokachu instance with the given configuration. Do not forgot call Close() function before exit.
func New[K comparable, V any](cfg Config) *Gokachu[K, V] {
	g := &Gokachu[K, V]{
		elems:               list.New(),
		store:               make(map[K]*list.Element),
		mut:                 new(sync.Mutex),
		maxRecordThreshold:  cfg.MaxRecordThreshold,
		cleanNum:            cfg.CleanNum,
		replacementStrategy: cfg.ReplacementStrategy,
		pollInterval:        cmp.Or(cfg.PollInterval, time.Second), // Default poll interval is 1 second
		pollCancel:          make(chan struct{}),
		wg:                  new(sync.WaitGroup),

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

// Set sets a value in the cache with a TTL. If the TTL is 0, the value will not expire.
func (g *Gokachu[K, V]) Set(key K, v V, ttl time.Duration, hooks ...Hook) {
	defer g.lock()()

	if g.pollCancel == nil {
		return
	}

	g.runOnSetHooks(key, v, ttl)

	if g.maxRecordThreshold > 0 && g.cleanNum > 0 && g.replacementStrategy > ReplacementStrategyNone && len(g.store) >= g.maxRecordThreshold {
		g.clear()
	}

	exp := time.Time{}
	if ttl > 0 {
		exp = time.Now().Add(ttl)
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

	// if exists
	if oldElem, ok := g.store[key]; ok {
		oldElem.Value = value
		switch g.replacementStrategy {
		case ReplacementStrategyLRU:
			g.elems.MoveToBack(oldElem)
		case ReplacementStrategyMRU:
			g.elems.MoveToFront(oldElem)
		}
		return
	}

	// if not exists
	switch g.replacementStrategy {
	case ReplacementStrategyFIFO, ReplacementStrategyLRU, ReplacementStrategyLFU, ReplacementStrategyMFU, ReplacementStrategyNone:
		g.store[key] = g.elems.PushBack(value)
	case ReplacementStrategyLIFO, ReplacementStrategyMRU:
		g.store[key] = g.elems.PushFront(value)
	}
}

// Get gets a value from the cache. Returns false in second value if the key does not exist.
func (g *Gokachu[K, V]) Get(key K) (V, bool) {
	defer g.lock()()

	item, ok := g.store[key]
	if !ok {
		g.runOnMissHooks(key)
		return *new(V), false
	}

	value := item.Value.(*valueWithTTL[K, V])

	switch g.replacementStrategy {
	case ReplacementStrategyLRU:
		g.elems.MoveToBack(item)
	case ReplacementStrategyMRU:
		g.elems.MoveToFront(item)
	case ReplacementStrategyMFU, ReplacementStrategyLFU:
		value.hitCount++
		g.moveByHits(item)
	}

	// run hooks before getting value
	g.runOnGetHooks(key, value.value)
	if value.hook.OnGet != nil {
		value.hook.OnGet()
	}

	return value.value, true
}

// GetFunc retrieves a first matching value from the cache using a callback function. If all matches return false, the second value also returns false.
func (g *Gokachu[K, V]) GetFunc(cb func(key K, value V) bool) (V, bool) {
	unlock := g.lock()
	val := *new(V)
	key := *new(K)
	found := false

	for key, value := range g.store {
		if !cb(key, value.Value.(*valueWithTTL[K, V]).value) {
			continue
		}
		found = true
		break
	}
	unlock()

	if found {
		return g.Get(key)
	}

	return val, false
}

// Delete deletes a value from the cache and returns true if the key existed.
func (g *Gokachu[K, V]) Delete(key K) bool {
	defer g.lock()()

	value, ok := g.store[key]
	if ok {
		// run hooks before delete
		g.runOnDeleteHooks(key, value.Value.(*valueWithTTL[K, V]).value)
		if value.Value.(*valueWithTTL[K, V]).hook.OnDelete != nil {
			value.Value.(*valueWithTTL[K, V]).hook.OnDelete()
		}

		// delete
		g.elems.Remove(value)
		delete(g.store, key)
	}
	return ok
}

// DeleteFunc deletes values from the cache for which the callback returns true and returns the number of deleted values.
func (g *Gokachu[K, V]) DeleteFunc(cb func(key K, value V) bool) int {
	defer g.lock()()

	count := 0 // deleted count

	for key, value := range g.store {
		if cb(key, value.Value.(*valueWithTTL[K, V]).value) {
			// run hooks before delete
			g.runOnDeleteHooks(key, value.Value.(*valueWithTTL[K, V]).value)
			if value.Value.(*valueWithTTL[K, V]).hook.OnDelete != nil {
				value.Value.(*valueWithTTL[K, V]).hook.OnDelete()
			}

			// delete
			g.elems.Remove(value)
			delete(g.store, key)
			count++
		}
	}
	return count
}

// Flush deletes all values from the cache and return the number of deleted values.
func (g *Gokachu[K, V]) Flush() int {
	defer g.lock()()

	g.elems.Init()
	count := len(g.store)
	clear(g.store)

	return count
}

// Keys returns all keys in the cache.
func (g *Gokachu[K, V]) Keys() []K {
	defer g.lock()()

	keys := make([]K, 0, len(g.store))

	for e := g.elems.Front(); e != nil; e = e.Next() {
		keys = append(keys, e.Value.(*valueWithTTL[K, V]).key)
	}

	return keys
}

// KeysFunc returns all keys in the cache for which the callback returns true.
func (g *Gokachu[K, V]) KeysFunc(cb func(key K, value V) bool) []K {
	defer g.lock()()

	keys := make([]K, 0, len(g.store))

	for e := g.elems.Front(); e != nil; e = e.Next() {
		if cb(e.Value.(*valueWithTTL[K, V]).key, e.Value.(*valueWithTTL[K, V]).value) {
			keys = append(keys, e.Value.(*valueWithTTL[K, V]).key)
		}
	}

	return slices.Clip(keys)
}

// Count returns the number of values in the cache.
func (g *Gokachu[K, V]) Count() int {
	defer g.lock()()

	return g.elems.Len()
}

// CountFunc returns the number of values in the cache for which the callback returns true.
func (g *Gokachu[K, V]) CountFunc(cb func(key K, value V) bool) int {
	defer g.lock()()

	count := 0

	for key, value := range g.store {
		if cb(key, value.Value.(*valueWithTTL[K, V]).value) {
			count++
		}
	}

	return count
}

// Close closes the cache and all associated resources.
func (g *Gokachu[K, V]) Close() {
	if g.pollCancel == nil {
		return
	}

	g.lock()()
	close(g.pollCancel)
	clear(g.store)

	// clear hooks
	g.onSetHooks = nil
	g.onGetHooks = nil
	g.onDeleteHooks = nil
	g.onMissHooks = nil

	g.elems.Init()
	g.wg.Wait()
}

func (k *Gokachu[K, V]) lock() func() {
	k.mut.Lock()

	return k.mut.Unlock
}

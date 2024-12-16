package gokachu

import (
	"container/list"
	"sync"
	"time"
)

type valueWithTTL[K comparable, V any] struct {
	key        K
	value      V
	hitCount   uint
	expireTime time.Time
}

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
}

type Config struct {
	ReplacementStrategy ReplacementStrategy // default: ReplacementStrategyNone
	MaxRecordTreshold   int                 // This parameter is used to control the maximum number of records in the cache. If the number of records exceeds this threshold, records will be deleted according to the replacement strategy.
	CleanNum            int                 // This parameter is used to control the number of records to be deleted.
	PollInterval        time.Duration       // This parameter is used to control the polling interval. If value is 0, uses default = 1 second.
}

func New[K comparable, V any](cfg Config) *Gokachu[K, V] {
	pollInterval := time.Second
	if cfg.PollInterval > 0 {
		pollInterval = cfg.PollInterval
	}

	g := &Gokachu[K, V]{
		elems:               list.New(),
		store:               make(map[K]*list.Element),
		mut:                 new(sync.Mutex),
		maxRecordThreshold:  cfg.MaxRecordTreshold,
		cleanNum:            cfg.CleanNum,
		replacementStrategy: cfg.ReplacementStrategy,
		pollInterval:        pollInterval,
		pollCancel:          make(chan struct{}),
		wg:                  new(sync.WaitGroup),
	}

	g.wg.Add(1)
	go g.poll()

	return g
}

// Set sets a value in the cache.
func (g *Gokachu[K, V]) Set(key K, v V) {
	g.set(key, v, 0)
}

// SetWithTTL sets a value in the cache with a TTL. If the TTL is 0, the value will not expire.
func (g *Gokachu[K, V]) SetWithTTL(key K, v V, ttl time.Duration) {
	g.set(key, v, ttl)
}

// Get gets a value from the cache. Returns false in second value if the key does not exist.
func (g *Gokachu[K, V]) Get(key K) (V, bool) {
	defer g.lock()()

	item, ok := g.store[key]
	if !ok {
		return *new(V), false
	}

	value := item.Value.(*valueWithTTL[K, V])

	if g.replacementStrategy == ReplacementStrategyMFU || g.replacementStrategy == ReplacementStrategyLFU {
		value.hitCount++
	}

	switch g.replacementStrategy {
	case ReplacementStrategyLRU:
		g.elems.MoveToBack(item)
	case ReplacementStrategyMRU:
		g.elems.MoveToFront(item)
	case ReplacementStrategyMFU, ReplacementStrategyLFU:
		g.moveByHits(item)
	}

	return value.value, true
}

// Delete deletes a value from the cache.
func (g *Gokachu[K, V]) Delete(key K) {
	defer g.lock()()
	g.elems.Remove(g.store[key])
	delete(g.store, key)
}

// Flush deletes all values from the cache.
func (g *Gokachu[K, V]) Flush() {
	defer g.lock()()
	g.elems.Init()
	clear(g.store)
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

// Count returns the number of values in the cache.
func (g *Gokachu[K, V]) Count() int {
	defer g.lock()()
	return g.elems.Len()
}

func (g *Gokachu[K, V]) Close() {
	if g.pollCancel == nil {
		return
	}
	g.lock()()
	close(g.pollCancel)
	clear(g.store)
	g.elems.Init()
	g.wg.Wait()
}

// Poll deletes expired values from the cache with the given poll interval. If context is cancelled, the polling stops.
func (g *Gokachu[K, V]) poll() {
	ticker := time.NewTicker(g.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-g.pollCancel:
			g.pollCancel = nil
			g.wg.Done()
			return
		case <-ticker.C:
			g.mut.Lock()
			now := time.Now()
			for key := range g.store {
				if g.store[key].Value.(*valueWithTTL[K, V]).expireTime.IsZero() {
					continue
				}
				if elem := g.store[key]; elem.Value.(*valueWithTTL[K, V]).expireTime.After(now) {
					g.elems.Remove(elem)
					delete(g.store, key)
				}
			}
			g.mut.Unlock()
		}
	}
}

func (k *Gokachu[K, V]) lock() func() {
	k.mut.Lock()
	return k.mut.Unlock
}

func (g *Gokachu[K, V]) set(key K, v V, ttl time.Duration) {
	if g.pollCancel == nil {
		return
	}
	defer g.lock()()
	if g.maxRecordThreshold > 0 && g.cleanNum > 0 && g.replacementStrategy > ReplacementStrategyNone && len(g.store) >= g.maxRecordThreshold {
		g.clean()
	}

	value := &valueWithTTL[K, V]{
		key:        key,
		value:      v,
		expireTime: time.Now().Add(ttl),
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

func (g *Gokachu[K, V]) moveByHits(elem *list.Element) {
	prev := elem.Prev()
	next := elem.Next()
	switch g.replacementStrategy {
	case ReplacementStrategyLFU:
		if prev != nil && prev.Value.(*valueWithTTL[K, V]).hitCount > elem.Value.(*valueWithTTL[K, V]).hitCount {
			g.elems.MoveBefore(elem, prev)
			g.moveByHits(elem)
			return
		}

		if next != nil && next.Value.(*valueWithTTL[K, V]).hitCount < elem.Value.(*valueWithTTL[K, V]).hitCount {
			g.elems.MoveAfter(elem, next)
			g.moveByHits(elem)
		}
	case ReplacementStrategyMFU:
		if prev != nil && prev.Value.(*valueWithTTL[K, V]).hitCount < elem.Value.(*valueWithTTL[K, V]).hitCount {
			g.elems.MoveBefore(elem, prev)
			g.moveByHits(elem)
			return
		}

		if next != nil && next.Value.(*valueWithTTL[K, V]).hitCount > elem.Value.(*valueWithTTL[K, V]).hitCount {
			g.elems.MoveAfter(elem, next)
			g.moveByHits(elem)
		}
	}
}

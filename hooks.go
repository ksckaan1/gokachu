package gokachu

import "time"

// Hooks must not add or remove hooks themselves: callbacks run while the hook registry is read-locked.

func (g *Gokachu[K, V]) AddOnSetHook(hook func(key K, value V, ttl time.Duration)) uint64 {
	id := g.inc.Add(1)

	g.hooksMut.Lock()
	g.hooksCount.Add(1)
	g.onSetHooks[id] = hook
	g.hooksMut.Unlock()

	return id
}

func (g *Gokachu[K, V]) RemoveOnSetHook(id uint64) bool {
	g.hooksMut.Lock()
	defer g.hooksMut.Unlock()

	_, ok := g.onSetHooks[id]
	if ok {
		delete(g.onSetHooks, id)
		g.hooksCount.Add(-1)
	}

	return ok
}

func (g *Gokachu[K, V]) runOnSetHooks(key K, value V, ttl time.Duration) {
	if g.hooksCount.Load() == 0 {
		return
	}

	g.hooksMut.RLock()
	defer g.hooksMut.RUnlock()

	for _, hook := range g.onSetHooks {
		hook(key, value, ttl)
	}
}

func (g *Gokachu[K, V]) AddOnGetHook(hook func(key K, value V)) uint64 {
	id := g.inc.Add(1)

	g.hooksMut.Lock()
	g.hooksCount.Add(1)
	g.onGetHooks[id] = hook
	g.hooksMut.Unlock()

	return id
}

func (g *Gokachu[K, V]) RemoveOnGetHook(id uint64) bool {
	g.hooksMut.Lock()
	defer g.hooksMut.Unlock()

	_, ok := g.onGetHooks[id]
	if ok {
		delete(g.onGetHooks, id)
		g.hooksCount.Add(-1)
	}

	return ok
}

func (g *Gokachu[K, V]) runOnGetHooks(key K, value V) {
	if g.hooksCount.Load() == 0 {
		return
	}

	g.hooksMut.RLock()
	defer g.hooksMut.RUnlock()

	for _, hook := range g.onGetHooks {
		hook(key, value)
	}
}

func (g *Gokachu[K, V]) AddOnMissHook(hook func(key K)) uint64 {
	id := g.inc.Add(1)

	g.hooksMut.Lock()
	g.hooksCount.Add(1)
	g.onMissHooks[id] = hook
	g.hooksMut.Unlock()

	return id
}

func (g *Gokachu[K, V]) RemoveOnMissHook(id uint64) bool {
	g.hooksMut.Lock()
	defer g.hooksMut.Unlock()

	_, ok := g.onMissHooks[id]
	if ok {
		delete(g.onMissHooks, id)
		g.hooksCount.Add(-1)
	}

	return ok
}

func (g *Gokachu[K, V]) runOnMissHooks(key K) {
	if g.hooksCount.Load() == 0 {
		return
	}

	g.hooksMut.RLock()
	defer g.hooksMut.RUnlock()

	for _, hook := range g.onMissHooks {
		hook(key)
	}
}

func (g *Gokachu[K, V]) AddOnDeleteHook(hook func(key K, value V)) uint64 {
	id := g.inc.Add(1)

	g.hooksMut.Lock()
	g.hooksCount.Add(1)
	g.onDeleteHooks[id] = hook
	g.hooksMut.Unlock()

	return id
}

func (g *Gokachu[K, V]) RemoveOnDeleteHook(id uint64) bool {
	g.hooksMut.Lock()
	defer g.hooksMut.Unlock()

	_, ok := g.onDeleteHooks[id]
	if ok {
		delete(g.onDeleteHooks, id)
		g.hooksCount.Add(-1)
	}

	return ok
}

func (g *Gokachu[K, V]) runOnDeleteHooks(key K, value V) {
	if g.hooksCount.Load() == 0 {
		return
	}

	g.hooksMut.RLock()
	defer g.hooksMut.RUnlock()

	for _, hook := range g.onDeleteHooks {
		hook(key, value)
	}
}

func WithOnGetHook(hook func()) Hook {
	return Hook{
		OnGet: hook,
	}
}

func WithOnDeleteHook(hook func()) Hook {
	return Hook{
		OnDelete: hook,
	}
}

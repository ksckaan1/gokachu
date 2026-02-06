package gokachu

import "time"

func (g *Gokachu[K, V]) AddOnSetHook(hook func(key K, value V, ttl time.Duration)) uint64 {
	id := g.inc.Add(1)
	g.onSetHooks[id] = hook

	return id
}

func (g *Gokachu[K, V]) RemoveOnSetHook(id uint64) bool {
	_, ok := g.onSetHooks[id]
	if ok {
		delete(g.onSetHooks, id)
	}

	return ok
}

func (g *Gokachu[K, V]) runOnSetHooks(key K, value V, ttl time.Duration) {
	for _, hook := range g.onSetHooks {
		hook(key, value, ttl)
	}
}

func (g *Gokachu[K, V]) AddOnGetHook(hook func(key K, value V)) uint64 {
	id := g.inc.Add(1)
	g.onGetHooks[id] = hook

	return id
}

func (g *Gokachu[K, V]) RemoveOnGetHook(id uint64) bool {
	_, ok := g.onGetHooks[id]
	if ok {
		delete(g.onGetHooks, id)
	}

	return ok
}

func (g *Gokachu[K, V]) runOnGetHooks(key K, value V) {
	for _, hook := range g.onGetHooks {
		hook(key, value)
	}
}

func (g *Gokachu[K, V]) AddOnMissHook(hook func(key K)) uint64 {
	id := g.inc.Add(1)
	g.onMissHooks[id] = hook

	return id
}

func (g *Gokachu[K, V]) RemoveOnMissHook(id uint64) bool {
	_, ok := g.onMissHooks[id]
	if ok {
		delete(g.onMissHooks, id)
	}

	return ok
}

func (g *Gokachu[K, V]) runOnMissHooks(key K) {
	for _, hook := range g.onMissHooks {
		hook(key)
	}
}

func (g *Gokachu[K, V]) AddOnDeleteHook(hook func(key K, value V)) uint64 {
	id := g.inc.Add(1)
	g.onDeleteHooks[id] = hook

	return id
}

func (g *Gokachu[K, V]) RemoveOnDeleteHook(id uint64) bool {
	_, ok := g.onDeleteHooks[id]
	if ok {
		delete(g.onDeleteHooks, id)
	}

	return ok
}

func (g *Gokachu[K, V]) runOnDeleteHooks(key K, value V) {
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

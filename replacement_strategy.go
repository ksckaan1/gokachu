package gokachu

import "container/list"

type ReplacementStrategy uint

const (
	ReplacementStrategyNone ReplacementStrategy = iota
	ReplacementStrategyLRU                      // Least Recently Used
	ReplacementStrategyMRU                      // Most Recently Used
	ReplacementStrategyFIFO                     // First In First Out
	ReplacementStrategyLIFO                     // Last In First Out
	ReplacementStrategyLFU                      // Least Frequently Used
	ReplacementStrategyMFU                      // Most Frequently Used
)

func (g *Gokachu[K, V]) clear() {
	currentElem := g.elems.Front()

	deletedCount := 0
	for deletedCount < g.clearNum && currentElem != nil {
		delete(g.store, currentElem.Value.(*valueWithTTL[K, V]).key)
		nextElem := currentElem.Next()
		g.elems.Remove(currentElem)

		deletedCount++
		currentElem = nextElem
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

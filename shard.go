package gokachu

import (
	"container/list"
	"sync"
)

type shard[K comparable, V any] struct {
	elems               *list.List // front of list == greater risk of deletion <---------list---------> back of list == less risk of deletion
	store               map[K]*list.Element
	mut                 sync.RWMutex
	maxRecordThreshold  int
	clearNum            int
	replacementStrategy ReplacementStrategy
}

func (s *shard[K, V]) lock() func() {
	s.mut.Lock()

	return s.mut.Unlock
}

func (s *shard[K, V]) rlock() func() {
	s.mut.RLock()

	return s.mut.RUnlock
}

func (s *shard[K, V]) clear() {
	currentElem := s.elems.Front()

	deletedCount := 0
	for deletedCount < s.clearNum && currentElem != nil {
		delete(s.store, currentElem.Value.(*valueWithTTL[K, V]).key)
		nextElem := currentElem.Next()
		s.elems.Remove(currentElem)

		deletedCount++
		currentElem = nextElem
	}
}

func (s *shard[K, V]) moveByHits(elem *list.Element) {
	prev := elem.Prev()
	next := elem.Next()

	switch s.replacementStrategy {
	case ReplacementStrategyLFU:
		if prev != nil && prev.Value.(*valueWithTTL[K, V]).hitCount > elem.Value.(*valueWithTTL[K, V]).hitCount {
			s.elems.MoveBefore(elem, prev)
			s.moveByHits(elem)

			return
		}

		if next != nil && next.Value.(*valueWithTTL[K, V]).hitCount < elem.Value.(*valueWithTTL[K, V]).hitCount {
			s.elems.MoveAfter(elem, next)
			s.moveByHits(elem)
		}

	case ReplacementStrategyMFU:
		if prev != nil && prev.Value.(*valueWithTTL[K, V]).hitCount < elem.Value.(*valueWithTTL[K, V]).hitCount {
			s.elems.MoveBefore(elem, prev)
			s.moveByHits(elem)

			return
		}

		if next != nil && next.Value.(*valueWithTTL[K, V]).hitCount > elem.Value.(*valueWithTTL[K, V]).hitCount {
			s.elems.MoveAfter(elem, next)
			s.moveByHits(elem)
		}
	}
}

package gokachu

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

func (g *Gokachu[K, V]) clean() {
	currentElem := g.elems.Front()
	deletedCount := 0
	for {
		if deletedCount >= g.cleanNum || currentElem == nil {
			break
		}
		delete(g.store, currentElem.Value.(*valueWithTTL[K, V]).key)
		nextElem := currentElem.Next()
		g.elems.Remove(currentElem)
		deletedCount++
		currentElem = nextElem
	}
}

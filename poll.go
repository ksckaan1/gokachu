package gokachu

import "time"

// poll deletes expired values from the cache with the given poll interval. If context is cancelled, the polling stops.
func (g *Gokachu[K, V]) poll() {
	ticker := time.NewTicker(g.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-g.pollCancel: // when Close method called, polling stops
			g.pollCancel = nil
			g.wg.Done()

			return

		case <-ticker.C:
			g.mut.Lock()

			now := time.Now()

			for key := range g.store {
				elem := g.store[key]

				// elem must be non-expired
				if elem.Value.(*valueWithTTL[K, V]).expireTime.IsZero() || elem.Value.(*valueWithTTL[K, V]).expireTime.After(now) {
					continue
				}

				// delete expired element
				g.runOnDeleteHooks(key, elem.Value.(*valueWithTTL[K, V]).value)
				g.elems.Remove(elem)
				delete(g.store, key)
			}

			g.mut.Unlock()
		}
	}
}

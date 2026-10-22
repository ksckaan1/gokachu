package gokachu

import "time"

// poll deletes expired values from the cache with the given poll interval. If context is cancelled, the polling stops.
func (g *Gokachu[K, V]) poll() {
	ticker := time.NewTicker(g.pollInterval)
	defer ticker.Stop()

	defer g.wg.Done()

	for {
		select {
		case <-g.pollCancel: // when Close method called, polling stops
			return

		case <-ticker.C:
			now := time.Now()

			for i := range g.shards {
				s := &g.shards[i]

				s.mut.Lock()

				for key := range s.store {
					elem := s.store[key]

					// elem must be non-expired
					if elem.Value.(*valueWithTTL[K, V]).expireTime.IsZero() || elem.Value.(*valueWithTTL[K, V]).expireTime.After(now) {
						continue
					}

					// delete expired element
					g.runOnDeleteHooks(key, elem.Value.(*valueWithTTL[K, V]).value)
					s.elems.Remove(elem)
					delete(s.store, key)
				}

				s.mut.Unlock()
			}
		}
	}
}

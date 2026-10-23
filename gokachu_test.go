package gokachu

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkGokachu_Update(b *testing.B) {
	k := New[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
		MaxRecordThreshold:  1000,
		ClearNum:            100,
	})
	defer k.Close()

	b.ResetTimer()

	for b.Loop() {
		k.Set(0, 0, 0)
	}
}

func BenchmarkGokachu_Get(b *testing.B) {
	k := New[string, string](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
		MaxRecordThreshold:  1000,
		ClearNum:            100,
	})
	defer k.Close()

	k.Set("key", "value", 30*time.Minute)

	b.ResetTimer()

	for i := 0; b.Loop(); i++ {
		k.Get("key")
	}
}

func BenchmarkGokachu_InsertEvict(b *testing.B) {
	const capacity = 1000

	k := New[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
		MaxRecordThreshold:  capacity,
		ClearNum:            1,
	})
	defer k.Close()

	for i := range capacity {
		k.Set(i, i, 0)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		k.Set(i+capacity, 0, 0)
	}
}

func BenchmarkGokachu_GetParallel(b *testing.B) {
	const capacity = 100_000

	k := New[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
		MaxRecordThreshold:  0,
		ClearNum:            0,
	})
	defer k.Close()

	for i := range capacity {
		k.Set(i, i, 0)
	}

	var goroutineID atomic.Int64

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := int(goroutineID.Add(1)-1) * 1000
		for pb.Next() {
			k.Get(i % capacity)
			i++
		}
	})
}

func BenchmarkGokachu_UpdateParallel(b *testing.B) {
	const capacity = 100_000

	k := New[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
		MaxRecordThreshold:  0,
		ClearNum:            0,
	})
	defer k.Close()

	for i := range capacity {
		k.Set(i, i, 0)
	}

	var goroutineID atomic.Int64

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := int(goroutineID.Add(1)-1) * 1000
		for pb.Next() {
			k.Set(i%capacity, i, 0)
			i++
		}
	})
}

func BenchmarkGokachu_GetParallelSharded(b *testing.B) {
	const (
		capacity = 100_000
		shards   = 64
	)

	k := NewSharded[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
		MaxRecordThreshold:  0,
		ClearNum:            0,
	}, shards)
	defer k.Close()

	for i := range capacity {
		k.Set(i, i, 0)
	}

	var goroutineID atomic.Int64

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := int(goroutineID.Add(1)-1) * 1000
		for pb.Next() {
			k.Get(i % capacity)
			i++
		}
	})
}

func BenchmarkGokachu_UpdateParallelSharded(b *testing.B) {
	const (
		capacity = 100_000
		shards   = 64
	)

	k := NewSharded[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
		MaxRecordThreshold:  0,
		ClearNum:            0,
	}, shards)
	defer k.Close()

	for i := range capacity {
		k.Set(i, i, 0)
	}

	var goroutineID atomic.Int64

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := int(goroutineID.Add(1)-1) * 1000
		for pb.Next() {
			k.Set(i%capacity, i, 0)
			i++
		}
	})
}

func BenchmarkGokachu_GetMiss(b *testing.B) {
	k := New[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
	})
	defer k.Close()

	b.ResetTimer()

	for i := 0; b.Loop(); i++ {
		k.Get(i)
	}
}

func BenchmarkGokachu_Delete(b *testing.B) {
	k := New[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
	})
	defer k.Close()

	b.ResetTimer()

	for i := 0; b.Loop(); i++ {
		k.Set(i, i, 0)
		k.Delete(i)
	}
}

func BenchmarkGokachu_GetStrategies(b *testing.B) {
	const capacity = 1000

	strategies := []struct {
		name  string
		value ReplacementStrategy
	}{
		{"None", ReplacementStrategyNone},
		{"LRU", ReplacementStrategyLRU},
		{"MRU", ReplacementStrategyMRU},
		{"FIFO", ReplacementStrategyFIFO},
		{"LIFO", ReplacementStrategyLIFO},
		{"LFU", ReplacementStrategyLFU},
		{"MFU", ReplacementStrategyMFU},
	}

	for _, strategy := range strategies {
		b.Run(strategy.name, func(b *testing.B) {
			k := New[int, int](Config{
				ReplacementStrategy: strategy.value,
			})
			defer k.Close()

			for i := range capacity {
				k.Set(i, i, 0)
			}

			b.ResetTimer()

			for i := 0; b.Loop(); i++ {
				k.Get(i % capacity)
			}
		})
	}
}

func BenchmarkGokachu_MixedParallel(b *testing.B) {
	const capacity = 100_000

	k := New[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
	})
	defer k.Close()

	for i := range capacity {
		k.Set(i, i, 0)
	}

	var goroutineID atomic.Int64

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := int(goroutineID.Add(1)-1) * 1000
		for pb.Next() {
			key := i % capacity
			if i%10 == 0 {
				k.Set(key, i, 0)
			} else {
				k.Get(key)
			}

			i++
		}
	})
}

func BenchmarkGokachu_MixedParallelSharded(b *testing.B) {
	const (
		capacity = 100_000
		shards   = 64
	)

	k := NewSharded[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
	}, shards)
	defer k.Close()

	for i := range capacity {
		k.Set(i, i, 0)
	}

	var goroutineID atomic.Int64

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := int(goroutineID.Add(1)-1) * 1000
		for pb.Next() {
			key := i % capacity
			if i%10 == 0 {
				k.Set(key, i, 0)
			} else {
				k.Get(key)
			}

			i++
		}
	})
}

func TestSharding(t *testing.T) {
	const shards = 8

	g := NewSharded[int, string](Config{}, shards)
	defer g.Close()

	for i := 0; i < 1000; i++ {
		g.Set(i, fmt.Sprint(i), 0)
	}

	if got := g.Count(); got != 1000 {
		t.Fatalf("expected count to be 1000, but got %d", got)
	}

	usedShards := 0

	for i := range g.shards {
		if len(g.shards[i].store) > 0 {
			usedShards++
		}
	}

	if usedShards != shards {
		t.Errorf("expected all %d shards to be used, but got %d", shards, usedShards)
	}

	for i := 0; i < 1000; i++ {
		v, ok := g.Get(i)
		if !ok || v != fmt.Sprint(i) {
			t.Fatalf("expected key %d to be %q, but got %q (ok=%v)", i, fmt.Sprint(i), v, ok)
		}
	}

	if got := len(g.Keys()); got != 1000 {
		t.Errorf("expected 1000 keys, but got %d", got)
	}

	if !g.Delete(500) || g.Delete(500) {
		t.Errorf("expected Delete to return true then false")
	}

	if got := g.Flush(); got != 999 {
		t.Errorf("expected 999 flushed values, but got %d", got)
	}

	if got := g.Count(); got != 0 {
		t.Errorf("expected count to be 0, but got %d", got)
	}
}

func TestShardingThreshold(t *testing.T) {
	const (
		shards = 8
		max    = 80
	)

	g := NewSharded[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
		MaxRecordThreshold:  max,
		ClearNum:            8,
	}, shards)
	defer g.Close()

	for i := 0; i < 1000; i++ {
		g.Set(i, i, 0)
	}

	if got := g.Count(); got > max {
		t.Errorf("expected at most %d records, but got %d", max, got)
	}
}

func TestShardingConcurrent(t *testing.T) {
	g := NewSharded[int, int](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
		MaxRecordThreshold:  10_000,
		ClearNum:            100,
	}, 16)
	defer g.Close()

	var wg sync.WaitGroup

	for w := 0; w < 8; w++ {
		wg.Add(1)

		go func(w int) {
			defer wg.Done()

			for i := 0; i < 1000; i++ {
				key := w*1000 + i

				g.Set(key, key, 0)
				g.Get(key)

				if i%3 == 0 {
					g.Delete(key)
				}
			}
		}(w)
	}

	wg.Wait()
}

func TestShardingConcurrentHooks(t *testing.T) {
	g := NewSharded[int, int](Config{}, 8)
	defer g.Close()

	var wg sync.WaitGroup

	for w := 0; w < 4; w++ {
		wg.Add(1)

		go func(w int) {
			defer wg.Done()

			for i := 0; i < 1000; i++ {
				g.Set(w*1000+i, i, 0)
				g.Get(w*1000 + i)
			}
		}(w)
	}

	wg.Add(1)

	go func() {
		defer wg.Done()

		for i := 0; i < 1000; i++ {
			id := g.AddOnGetHook(func(key, value int) {})
			g.RemoveOnGetHook(id)
		}
	}()

	wg.Wait()
}

func TestGokachuReplacementStrategies(t *testing.T) {
	t.Run("when reaches max record threshold, then clean", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyFIFO,
			MaxRecordThreshold:  1000,
			ClearNum:            100,
		})
		defer k.Close()

		for i := 0; i < 1001; i++ {
			k.Set(fmt.Sprint(i), "value", 0)
		}

		if k.Count() != 901 {
			t.Errorf("expected count to be 901, but got %d", k.Count())
		}
	})

	t.Run("when cleans, then check sort of keys by FIFO", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyFIFO,
			MaxRecordThreshold:  10,
			ClearNum:            6,
		})
		defer k.Close()

		for i := 0; i < 11; i++ {
			k.Set(fmt.Sprint(i), "value", 0)
		}

		if k.Count() != 5 {
			t.Errorf("expected count to be 5, but got %d", k.Count())
		}

		if !reflect.DeepEqual(k.Keys(), []string{"6", "7", "8", "9", "10"}) {
			t.Errorf("expected keys to be [6, 7, 8, 9, 10], but got %v", k.Keys())
		}
	})

	t.Run("when cleans, then check sort of keys by LIFO", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyLIFO,
			MaxRecordThreshold:  10,
			ClearNum:            6,
		})
		defer k.Close()

		for i := 0; i < 11; i++ {
			k.Set(fmt.Sprint(i), "value", 0)
		}

		if k.Count() != 5 {
			t.Errorf("expected count to be 5, but got %d", k.Count())
		}

		if !reflect.DeepEqual(k.Keys(), []string{"10", "3", "2", "1", "0"}) {
			t.Errorf("expected keys to be [10 3 2 1 0], but got %v", k.Keys())
		}
	})

	t.Run("when cleans, then check sort of keys by LRU", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyLRU,
			MaxRecordThreshold:  10,
			ClearNum:            6,
		})
		defer k.Close()

		for i := 0; i < 10; i++ {
			k.Set(fmt.Sprint(i), "value", 0)
		}

		for i := 0; i < 10; i++ {
			k.Get(fmt.Sprint(9 - i))
		}

		k.Set("10", "value", 0)

		if k.Count() != 5 {
			t.Errorf("expected count to be 5, but got %d", k.Count())
		}

		if !reflect.DeepEqual(k.Keys(), []string{"3", "2", "1", "0", "10"}) {
			t.Errorf("expected keys to be [3 2 1 0 10], but got %v", k.Keys())
		}
	})

	t.Run("when cleans, then check sort of keys by MRU", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyMRU,
			MaxRecordThreshold:  10,
			ClearNum:            6,
		})
		defer k.Close()

		for i := 0; i < 10; i++ {
			k.Set(fmt.Sprint(i), "value", 0)
		}

		for i := 0; i < 10; i++ {
			k.Get(fmt.Sprint(9 - i))
		}

		k.Set("10", "value", 0)

		if k.Count() != 5 {
			t.Errorf("expected count to be 5, but got %d", k.Count())
		}

		if !reflect.DeepEqual(k.Keys(), []string{"10", "6", "7", "8", "9"}) {
			t.Errorf("expected keys to be [10 6 7 8 9], but got %v", k.Keys())
		}
	})

	t.Run("when cleans, then check sort of keys by LFU", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyLFU,
			MaxRecordThreshold:  10,
			ClearNum:            6,
		})
		defer k.Close()

		for i := 0; i < 10; i++ {
			k.Set(fmt.Sprint(i), "value", 0)
		}

		for i := 0; i < 10; i++ {
			for j := 0; j < 10; j++ {
				if j < i {
					continue
				}
				k.Get(fmt.Sprint(9 - j))
			}
		}

		k.Set("10", "value", 0)

		if k.Count() != 5 {
			t.Errorf("expected count to be 5, but got %d", k.Count())
		}

		if !reflect.DeepEqual(k.Keys(), []string{"3", "2", "1", "0", "10"}) {
			t.Errorf("expected keys to be [3 2 1 0 10], but got %v", k.Keys())
		}
	})

	t.Run("when cleans, then check sort of keys by MFU", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyMFU,
			MaxRecordThreshold:  10,
			ClearNum:            6,
		})
		defer k.Close()

		for i := 0; i < 10; i++ {
			k.Set(fmt.Sprint(i), "value", 0)
		}

		for i := 0; i < 10; i++ {
			for j := 0; j < 10; j++ {
				if j < i {
					continue
				}
				k.Get(fmt.Sprint(9 - j))
			}
		}

		k.Set("10", "value", 0)

		for i := 0; i < 10; i++ {
			k.Get("10")
		}

		if k.Count() != 5 {
			t.Errorf("expected count to be 5, but got %d", k.Count())
		}

		if !reflect.DeepEqual(k.Keys(), []string{"10", "6", "7", "8", "9"}) {
			t.Errorf("expected keys to be [10 6 7 8 9], but got %v", k.Keys())
		}
	})
}

func TestSet(t *testing.T) {
	t.Run("set without replacement", func(t *testing.T) {
		k := New[string, string](Config{})
		defer k.Close()

		k.Set("key", "value", 0)
		k.Set("key", "value", 0) // already exists

		keys := k.Keys()
		if !reflect.DeepEqual(keys, []string{"key"}) {
			t.Errorf("expected keys to be [key], but got %v", keys)
		}
	})

	t.Run("set with lru replacement", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyLRU,
			MaxRecordThreshold:  100,
			ClearNum:            100,
		})
		defer k.Close()

		k.Set("key", "value", 0)
		k.Set("key", "value", 0) // already exists

		keys := k.Keys()
		if !reflect.DeepEqual(keys, []string{"key"}) {
			t.Errorf("expected keys to be [key], but got %v", keys)
		}
	})

	t.Run("set with mru replacement", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyMRU,
			MaxRecordThreshold:  100,
			ClearNum:            100,
		})
		defer k.Close()

		k.Set("key", "value", 0)
		k.Set("key", "value", 0) // already exists

		keys := k.Keys()
		if !reflect.DeepEqual(keys, []string{"key"}) {
			t.Errorf("expected keys to be [key], but got %v", keys)
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("get without replacement", func(t *testing.T) {
		k := New[string, string](Config{})
		defer k.Close()

		k.Set("key", "value", 0)

		v, ok := k.Get("key")
		if !ok {
			t.Errorf("expected key to exist")
		}

		if v != "value" {
			t.Errorf("expected value to be value, but got %s", v)
		}
	})

	t.Run("get with lru replacement", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyLRU,
			MaxRecordThreshold:  100,
			ClearNum:            100,
		})
		defer k.Close()

		k.Set("key", "value", 0)

		v, ok := k.Get("key")
		if !ok {
			t.Errorf("expected key to exist")
		}

		if v != "value" {
			t.Errorf("expected value to be value, but got %s", v)
		}
	})

	t.Run("get with mru replacement", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyMRU,
			MaxRecordThreshold:  100,
			ClearNum:            100,
		})
		defer k.Close()

		k.Set("key", "value", 0)

		v, ok := k.Get("key")
		if !ok {
			t.Errorf("expected key to exist")
		}

		if v != "value" {
			t.Errorf("expected value to be value, but got %s", v)
		}
	})

	t.Run("get with lfu replacement", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyLFU,
			MaxRecordThreshold:  100,
			ClearNum:            100,
		})
		defer k.Close()

		k.Set("key", "value", 0)

		v, ok := k.Get("key")
		if !ok {
			t.Errorf("expected key to exist")
		}

		if v != "value" {
			t.Errorf("expected value to be value, but got %s", v)
		}
	})

	t.Run("get with mfu replacement", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyMFU,
			MaxRecordThreshold:  100,
			ClearNum:            100,
		})
		defer k.Close()

		k.Set("key1", "value", 0)
		k.Set("key2", "value", 0)

		k.Get("key1")
		k.Get("key2")
		k.Get("key2")

		v, ok := k.Get("key1")
		if !ok {
			t.Errorf("expected key to exist")
		}

		if v != "value" {
			t.Errorf("expected value to be value, but got %s", v)
		}
	})
}

func TestTTL(t *testing.T) {
	t.Run("with TTL", func(t *testing.T) {
		k := New[string, string](Config{
			PollInterval: 100 * time.Millisecond,
		})
		defer k.Close()

		k.Set("key", "value", 300*time.Millisecond)

		if _, ok := k.Get("key"); !ok {
			t.Errorf("expected key to exist")
		}

		time.Sleep(400 * time.Millisecond)

		if _, ok := k.Get("key"); ok {
			t.Errorf("expected key to be deleted")
		}
	})

	t.Run("skip if no TTL", func(t *testing.T) {
		k := New[string, string](Config{
			PollInterval: 100 * time.Millisecond,
		})

		k.Set("key1", "value", 300*time.Millisecond)
		k.Set("key2", "value", 5*time.Second)
		k.Set("key3", "value", 0)
		k.Set("key3", "value", 0)

		keys := k.Keys()
		if !reflect.DeepEqual(keys, []string{"key1", "key2", "key3"}) {
			t.Errorf("expected keys to be [key1, key2, key3], but got %v", keys)
		}

		// wait until key1 is expired
		time.Sleep(400 * time.Millisecond)

		keys = k.Keys()
		if !reflect.DeepEqual(keys, []string{"key2", "key3"}) {
			t.Errorf("expected keys to be [key2, key3], but got %v", keys)
		}

		k.Close()
	})
}

func TestClose(t *testing.T) {
	t.Run("check goroutine is closed", func(t *testing.T) {
		start := runtime.NumGoroutine()

		k := New[string, string](Config{})
		k.Close()

		if start != runtime.NumGoroutine() {
			t.Errorf("expected %d, but got %d", start, runtime.NumGoroutine())
		}
	})

	t.Run("check if usable after close", func(t *testing.T) {
		k := New[string, string](Config{})
		k.Close()

		k.Set("key", "value", 0)

		if _, ok := k.Get("key"); ok {
			t.Errorf("expected key to be not set")
		}
	})

	t.Run("close again", func(t *testing.T) {
		k := New[string, string](Config{})
		k.Close()
		k.Close()
	})
}

func TestHasKey(t *testing.T) {
	g := New[string, string](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
	})
	defer g.Close()

	var gets atomic.Int64
	g.AddOnGetHook(func(key, value string) { gets.Add(1) })

	var misses atomic.Int64
	g.AddOnMissHook(func(key string) { misses.Add(1) })

	g.Set("a", "1", 0)
	g.Set("b", "2", 0)

	if !g.HasKey("a") {
		t.Errorf("expected key a to exist")
	}

	if g.HasKey("missing") {
		t.Errorf("expected key to not exist")
	}

	if got := gets.Load(); got != 0 {
		t.Errorf("expected no get hook call, but got %d", got)
	}

	if got := misses.Load(); got != 0 {
		t.Errorf("expected no miss hook call, but got %d", got)
	}

	// HasKey must not touch the LRU order: a is still the oldest entry.
	front := g.shards[0].elems.Front().Value.(*valueWithTTL[string, string]).key
	if front != "a" {
		t.Errorf("expected front key to be a, but got %s", front)
	}
}

func TestDelete(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("key1", "value", 0)
	g.Delete("key1")
	if _, ok := g.Get("key1"); ok {
		t.Errorf("expected key to be not set")
	}
	g.Close()
}

func TestDeleteFunc(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("a1", "a1", 0)
	g.Set("a2", "a2", 0)
	g.Set("a3", "a3", 0)
	g.Set("b1", "b1", 0)
	g.Set("b2", "b2", 0)
	g.Set("b3", "b3", 0)
	g.Set("c1", "c1", 0)
	// delete all keys start with "a"
	g.DeleteFunc(func(key, _ string) bool {
		return strings.HasPrefix(key, "a")
	})
	// delete all values start with "b"
	g.DeleteFunc(func(_, value string) bool {
		return strings.HasPrefix(value, "b")
	})
	if g.Count() != 1 {
		t.Errorf("expected elems count to be 1, but got %d", g.Count())
	}
	g.Close()
}

func TestKeys(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("a1", "a1", 0)
	g.Set("a2", "a2", 0)
	g.Set("a3", "a3", 0)
	g.Set("b1", "b1", 0)
	g.Set("b2", "b2", 0)
	g.Set("b3", "b3", 0)
	g.Set("c1", "c1", 0)
	keys := g.Keys()
	if !reflect.DeepEqual(keys, []string{"a1", "a2", "a3", "b1", "b2", "b3", "c1"}) {
		t.Errorf("expected keys to be [a1, a2, a3, b1, b2, b3, c1], but got %v", keys)
	}
	g.Close()
}

func TestKeysFunc(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("a1", "a1", 0)
	g.Set("a2", "a2", 0)
	g.Set("a3", "a3", 0)
	g.Set("b1", "b1", 0)
	g.Set("b2", "b2", 0)
	g.Set("b3", "b3", 0)
	g.Set("c1", "c1", 0)
	keys := g.KeysFunc(func(key, _ string) bool {
		return strings.HasPrefix(key, "a")
	})
	if !reflect.DeepEqual(keys, []string{"a1", "a2", "a3"}) {
		t.Errorf("expected keys to be [a1, a2, a3], but got %v", keys)
	}
	g.Close()
}

func TestCount(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("a1", "a1", 0)
	g.Set("a2", "a2", 0)
	g.Set("a3", "a3", 0)
	g.Set("b1", "b1", 0)
	g.Set("b2", "b2", 0)
	g.Set("b3", "b3", 0)
	g.Set("c1", "c1", 0)
	count := g.Count()
	if count != 7 {
		t.Errorf("expected count to be 7, but got %d", count)
	}
	g.Close()
}

func TestCountFunc(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("a1", "a1", 0)
	g.Set("a2", "a2", 0)
	g.Set("a3", "a3", 0)
	g.Set("b1", "b1", 0)
	g.Set("b2", "b2", 0)
	g.Set("b3", "b3", 0)
	g.Set("c1", "c1", 0)
	count := g.CountFunc(func(key, _ string) bool {
		return strings.HasPrefix(key, "a")
	})
	if count != 3 {
		t.Errorf("expected count to be 3, but got %d", count)
	}
	g.Close()
}

func TestFlush(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("a1", "a1", 0)
	g.Set("a2", "a2", 0)
	g.Set("a3", "a3", 0)
	g.Set("b1", "b1", 0)
	g.Set("b2", "b2", 0)
	g.Set("b3", "b3", 0)
	g.Set("c1", "c1", 0)
	g.Flush()
	if g.Count() != 0 {
		t.Errorf("expected elems count to be 0, but got %d", g.Count())
	}
	g.Close()
}

func TestGetAs(t *testing.T) {
	type Example struct {
		Name string
		Age  int
	}

	g := New[string, any](Config{})
	g.Set("a1", Example{Name: "John", Age: 30}, 0)

	v, ok := g.Get("a1")
	if !ok {
		t.Errorf("expected ok to be true, but got false")
	}
	if _, ok := v.(Example); !ok {
		t.Errorf("expected v to be of type Example, but got %T", v)
	}

	v2, ok := g.GetAs[Example]("a1")
	if !ok {
		t.Errorf("expected ok to be true, but got false")
	}
	if v2 != (Example{Name: "John", Age: 30}) {
		t.Errorf("expected v2 to be of type Example, but got %T", v2)
	}
}

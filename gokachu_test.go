package gokachu

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func BenchmarkGokachuSetWithTTL(b *testing.B) {
	k := New[string, string](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
		MaxRecordTreshold:   1000,
		CleanNum:            100,
	})
	defer k.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		k.SetWithTTL(fmt.Sprint(i), "value", 30*time.Minute)
	}
}

func BenchmarkGokachuGet(b *testing.B) {
	k := New[string, string](Config{
		ReplacementStrategy: ReplacementStrategyLRU,
		MaxRecordTreshold:   1000,
		CleanNum:            100,
	})
	defer k.Close()

	k.SetWithTTL("key", "value", 30*time.Minute)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		k.Get("key")
	}
}

func TestGokachuReplacementStrategies(t *testing.T) {
	t.Run("when reaches max record threshold, then clean", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyFIFO,
			MaxRecordTreshold:   1000,
			CleanNum:            100,
		})
		defer k.Close()

		for i := 0; i < 1001; i++ {
			k.Set(fmt.Sprint(i), "value")
		}

		if k.Count() != 901 {
			t.Errorf("expected count to be 901, but got %d", k.Count())
		}
	})

	t.Run("when cleans, then check sort of keys by FIFO", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyFIFO,
			MaxRecordTreshold:   10,
			CleanNum:            6,
		})
		defer k.Close()

		for i := 0; i < 11; i++ {
			k.Set(fmt.Sprint(i), "value")
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
			MaxRecordTreshold:   10,
			CleanNum:            6,
		})
		defer k.Close()

		for i := 0; i < 11; i++ {
			k.Set(fmt.Sprint(i), "value")
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
			MaxRecordTreshold:   10,
			CleanNum:            6,
		})
		defer k.Close()

		for i := 0; i < 10; i++ {
			k.Set(fmt.Sprint(i), "value")
		}

		for i := 0; i < 10; i++ {
			k.Get(fmt.Sprint(9 - i))
		}

		k.Set("10", "value")

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
			MaxRecordTreshold:   10,
			CleanNum:            6,
		})
		defer k.Close()

		for i := 0; i < 10; i++ {
			k.Set(fmt.Sprint(i), "value")
		}

		for i := 0; i < 10; i++ {
			k.Get(fmt.Sprint(9 - i))
		}

		k.Set("10", "value")

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
			MaxRecordTreshold:   10,
			CleanNum:            6,
		})
		defer k.Close()

		for i := 0; i < 10; i++ {
			k.Set(fmt.Sprint(i), "value")
		}

		for i := 0; i < 10; i++ {
			for j := 0; j < 10; j++ {
				if j < i {
					continue
				}
				k.Get(fmt.Sprint(9 - j))
			}
		}

		k.Set("10", "value")

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
			MaxRecordTreshold:   10,
			CleanNum:            6,
		})
		defer k.Close()

		for i := 0; i < 10; i++ {
			k.Set(fmt.Sprint(i), "value")
		}

		for i := 0; i < 10; i++ {
			for j := 0; j < 10; j++ {
				if j < i {
					continue
				}
				k.Get(fmt.Sprint(9 - j))
			}
		}

		k.Set("10", "value")

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

		k.Set("key", "value")
		k.Set("key", "value") // already exists

		keys := k.Keys()
		if !reflect.DeepEqual(keys, []string{"key"}) {
			t.Errorf("expected keys to be [key], but got %v", keys)
		}
	})

	t.Run("set with lru replacement", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyLRU,
			MaxRecordTreshold:   100,
			CleanNum:            100,
		})
		defer k.Close()

		k.Set("key", "value")
		k.Set("key", "value") // already exists

		keys := k.Keys()
		if !reflect.DeepEqual(keys, []string{"key"}) {
			t.Errorf("expected keys to be [key], but got %v", keys)
		}
	})

	t.Run("set with mru replacement", func(t *testing.T) {
		k := New[string, string](Config{
			ReplacementStrategy: ReplacementStrategyMRU,
			MaxRecordTreshold:   100,
			CleanNum:            100,
		})
		defer k.Close()

		k.Set("key", "value")
		k.Set("key", "value") // already exists

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

		k.Set("key", "value")

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
			MaxRecordTreshold:   100,
			CleanNum:            100,
		})
		defer k.Close()

		k.Set("key", "value")

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
			MaxRecordTreshold:   100,
			CleanNum:            100,
		})
		defer k.Close()

		k.Set("key", "value")

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
			MaxRecordTreshold:   100,
			CleanNum:            100,
		})
		defer k.Close()

		k.Set("key", "value")

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
			MaxRecordTreshold:   100,
			CleanNum:            100,
		})
		defer k.Close()

		k.Set("key1", "value")
		k.Set("key2", "value")

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

		k.SetWithTTL("key", "value", 300*time.Millisecond)

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

		k.SetWithTTL("key1", "value", 300*time.Millisecond)
		k.SetWithTTL("key2", "value", 5*time.Second)
		k.Set("key3", "value")
		k.SetWithTTL("key3", "value", 0)

		keys := k.Keys()
		if !reflect.DeepEqual(keys, []string{"key1", "key2", "key3"}) {
			t.Errorf("expected keys to be [key1, key2, key3], but got %v", keys)
		}

		// wait until key1 is expired
		time.Sleep(400 * time.Millisecond)

		keys = k.Keys()
		if !reflect.DeepEqual(keys, []string{"key3"}) {
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

		k.Set("key", "value")

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

func TestDelete(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("key1", "value")
	g.Delete("key1")
	if _, ok := g.Get("key1"); ok {
		t.Errorf("expected key to be not set")
	}
	g.Close()
}

func TestDeleteFunc(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("a1", "a1")
	g.Set("a2", "a2")
	g.Set("a3", "a3")
	g.Set("b1", "b1")
	g.Set("b2", "b2")
	g.Set("b3", "b3")
	g.Set("c1", "c1")
	// delete all keys start with "a"
	g.DeleteFunc(func(key, _ string) bool {
		return strings.HasPrefix(key, "a")
	})
	// delete all values start with "b"
	g.DeleteFunc(func(_, value string) bool {
		return strings.HasPrefix(value, "b")
	})
	if g.elems.Len() != 1 {
		t.Errorf("expected elems count to be 1, but got %d", g.elems.Len())
	}
	if len(g.store) != 1 {
		t.Errorf("expected store count to be 1, but got %d", g.elems.Len())
	}
	g.Close()
}

func TestKeys(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("a1", "a1")
	g.Set("a2", "a2")
	g.Set("a3", "a3")
	g.Set("b1", "b1")
	g.Set("b2", "b2")
	g.Set("b3", "b3")
	g.Set("c1", "c1")
	keys := g.Keys()
	if !reflect.DeepEqual(keys, []string{"a1", "a2", "a3", "b1", "b2", "b3", "c1"}) {
		t.Errorf("expected keys to be [a1, a2, a3, b1, b2, b3, c1], but got %v", keys)
	}
	g.Close()
}

func TestKeysFunc(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("a1", "a1")
	g.Set("a2", "a2")
	g.Set("a3", "a3")
	g.Set("b1", "b1")
	g.Set("b2", "b2")
	g.Set("b3", "b3")
	g.Set("c1", "c1")
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
	g.Set("a1", "a1")
	g.Set("a2", "a2")
	g.Set("a3", "a3")
	g.Set("b1", "b1")
	g.Set("b2", "b2")
	g.Set("b3", "b3")
	g.Set("c1", "c1")
	count := g.Count()
	if count != 7 {
		t.Errorf("expected count to be 7, but got %d", count)
	}
	g.Close()
}

func TestCountFunc(t *testing.T) {
	g := New[string, string](Config{})
	g.Set("a1", "a1")
	g.Set("a2", "a2")
	g.Set("a3", "a3")
	g.Set("b1", "b1")
	g.Set("b2", "b2")
	g.Set("b3", "b3")
	g.Set("c1", "c1")
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
	g.Set("a1", "a1")
	g.Set("a2", "a2")
	g.Set("a3", "a3")
	g.Set("b1", "b1")
	g.Set("b2", "b2")
	g.Set("b3", "b3")
	g.Set("c1", "c1")
	g.Flush()
	if g.elems.Len() != 0 {
		t.Errorf("expected elems count to be 0, but got %d", g.elems.Len())
	}
	if len(g.store) != 0 {
		t.Errorf("expected store count to be 0, but got %d", g.elems.Len())
	}
	g.Close()
}

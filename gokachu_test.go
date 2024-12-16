package gokachu

import (
	"fmt"
	"reflect"
	"runtime"
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
}

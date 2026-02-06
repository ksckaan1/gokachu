package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/ksckaan1/gokachu"
)

func main() {
	// 1. Initialization
	cache := gokachu.New[string, string](gokachu.Config{
		ReplacementStrategy: gokachu.ReplacementStrategyLRU,
		MaxRecordThreshold:  1_000, // When it reaches 1_000 records,
		CleanNum:            100,   // Cleans 100 records.
	})
	defer cache.Close()

	// 2. Global Hooks
	cache.AddOnSetHook(func(key, value string, ttl time.Duration) {
		fmt.Printf("[Global Hook] Set: key=%s, value=%s, ttl=%v\n", key, value, ttl)
	})

	cache.AddOnGetHook(func(key, value string) {
		fmt.Printf("[Global Hook] Get: key=%s, value=%s\n", key, value)
	})

	cache.AddOnDeleteHook(func(key, value string) {
		fmt.Printf("[Global Hook] Delete: key=%s, value=%s\n", key, value)
	})

	cache.AddOnMissHook(func(key string) {
		fmt.Printf("[Global Hook] Miss: key=%s\n", key)
	})

	// 3. Set with Individual Hooks
	cache.Set("user:1", "John Doe", 5*time.Minute,
		gokachu.WithOnGetHook(func() {
			fmt.Println("[Individual Hook] Got user:1!")
		}),
		gokachu.WithOnDeleteHook(func() {
			fmt.Println("[Individual Hook] Deleted user:1!")
		}),
	)

	cache.Set("user:2", "Jane Doe", 0)
	cache.Set("product:1", "Laptop", 0)
	cache.Set("product:2", "Mouse", 0)

	// Get a value to trigger the OnGet hooks
	fmt.Println("\n--- Getting user:1 ---")
	value, found := cache.Get("user:1")
	if found {
		fmt.Printf("Got value: %s\n", value)
	}

	// 4. Demonstrate Delete return value
	fmt.Println("\n--- Deleting user:1 ---")
	if deleted := cache.Delete("user:1"); deleted {
		fmt.Println("`user:1` was successfully deleted.")
	} else {
		fmt.Println("`user:1` was not found in the cache.")
	}

	// Try to get the deleted key to trigger the OnMiss hook
	fmt.Println("\n--- Getting user:1 again ---")
	cache.Get("user:1")

	// 5. Demonstrate DeleteFunc return value
	fmt.Println("\n--- Deleting all products ---")
	deletedCount := cache.DeleteFunc(func(key, value string) bool {
		return strings.HasPrefix(key, "product:")
	})
	fmt.Printf("%d products were deleted.\n", deletedCount)

	// 6. Other operations
	fmt.Println("\n--- Final Cache State ---")
	fmt.Println("Keys:", cache.Keys())
	fmt.Println("Count:", cache.Count())

	// 7. Flush the cache
	fmt.Println("\n--- Flushing the cache ---")
	flushedCount := cache.Flush()
	fmt.Printf("%d items were flushed from the cache.\n", flushedCount)
	fmt.Println("Keys after flush:", cache.Keys())
	fmt.Println("Count after flush:", cache.Count())
}

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/ksckaan1/gokachu"
)

func main() {
	cache := gokachu.New[string, string](gokachu.Config{
		ReplacementStrategy: gokachu.ReplacementStrategyLRU,
		MaxRecordTreshold:   1_000, // When it reaches 1_000 records,
		CleanNum:            100,   // Cleans 100 records.
	})
	defer cache.Close()

	// Set with TTL
	cache.SetWithTTL("token/user_id:1", "eyJhbGciOiJ...", 30*time.Minute)

	// Set without TTL
	cache.Set("get_user_response/user_id:1", "John Doe")
	cache.Set("get_user_response/user_id:2", "Jane Doe")

	// Delete specific key
	cache.Delete("get_user_response/user_id:1")

	// Delete keys with "token" prefix
	cache.DeleteFunc(func(key, _ string) bool {
		return strings.HasPrefix(key, "token")
	})

	// Get (uses comma ok idiom)
	fmt.Println(cache.Get("get_user_response/user_id:2")) // eyJhbGciOiJ..., true
	fmt.Println(cache.Get("get_user_response/user_id:1")) // "", false

	fmt.Println("keys", cache.Keys()) // List of keys

	// List only keys start with "token"
	filteredKeys := cache.KeysFunc(func(key, _ string) bool {
		return strings.HasPrefix(key, "token")
	})
	fmt.Println("filteredKeys", filteredKeys)

	fmt.Println("count", cache.Count()) // Number of keys

	// Count only keys start with "token"
	filteredCount := cache.CountFunc(func(key, _ string) bool {
		return strings.HasPrefix(key, "token")
	})
	fmt.Println("filteredCount", filteredCount)

	cache.Flush() // Deletes all keys
}

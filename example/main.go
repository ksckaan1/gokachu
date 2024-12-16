package main

import (
	"fmt"
	"time"

	"github.com/ksckaan1/gokachu"
)

func main() {
	cache := gokachu.New[string, string](gokachu.Config{
		ReplacementStrategy: gokachu.ReplacementStrategyLRU,
		MaxRecordTreshold:   1000,
		CleanNum:            100,
	})
	defer cache.Close()

	// Set with TTL
	cache.SetWithTTL("token/user_id:1", "eyJhbGciOiJ...", 30*time.Minute)

	// Set without TTL
	cache.Set("get_user_response/user_id:1", "John Doe")
	cache.Set("get_user_response/user_id:2", "Jane Doe")
	cache.Set("get_user_response/user_id:3", "Walter White")
	cache.Set("get_user_response/user_id:4", "Jesse Pinkman")

	cache.Delete("get_user_response/user_id:1")

	fmt.Println(cache.Get("token/user_id:1"))             // eyJhbGciOiJ..., true
	fmt.Println(cache.Get("get_user_response/user_id:1")) // "", false

	fmt.Println("keys", cache.Keys())   // List of keys
	fmt.Println("count", cache.Count()) // Number of keys

	cache.Flush() // Deletes all keys
}

![gokachu](./doc/gokachu.png)

[![release](https://img.shields.io/github/release/ksckaan1/gokachu.svg)](https://github.com/ksckaan1/gokachu/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-%23007d9c)
[![GoDoc](https://godoc.org/github.com/ksckaan1/gokachu?status.svg)](https://pkg.go.dev/github.com/ksckaan1/gokachu)
[![Go report](https://goreportcard.com/badge/github.com/ksckaan1/gokachu)](https://goreportcard.com/report/github.com/ksckaan1/gokachu)
![m2s](https://img.shields.io/badge/coverage-95.9%25-green?style=flat)
[![Contributors](https://img.shields.io/github/contributors/ksckaan1/gokachu)](https://github.com/ksckaan1/gokachu/graphs/contributors)
[![LICENSE](https://img.shields.io/badge/LICENCE-MIT-orange?style=flat)](./LICENSE)

In-memory cache with TTL and generics support.

## Features
- TTL support
- Generics support
- Supported Cache Replacement Strategies
  - LRU (Least Recently Used)
  - MRU (Most Recently Used)
  - LFU (Least Frequently Used)
  - MFU (Most Frequently Used)
  - FIFO (First In First Out)
  - LIFO (Last In First Out)
  - NONE (no replacement)

## Installation

```bash
go get -u github.com/ksckaan1/gokachu@latest
```

## Example

```go
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

```

## Benchmark Tests

### Set With TTL / Set Without TTL
```bash
goos: darwin
goarch: arm64
pkg: github.com/ksckaan1/gokachu
BenchmarkGokachuSetWithTTL
BenchmarkGokachuSetWithTTL-8   	 4838650	       236.8 ns/op	     129 B/op	       4 allocs/op
PASS
ok  	github.com/ksckaan1/gokachu	1.842s
```

### Get
```bash
goos: darwin
goarch: arm64
pkg: github.com/ksckaan1/gokachu
BenchmarkGokachuGet
BenchmarkGokachuGet-8   	83910825	        13.98 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/ksckaan1/gokachu	2.094s
```


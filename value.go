package gokachu

import "time"

type valueWithTTL[K comparable, V any] struct {
	key        K
	value      V
	hitCount   uint
	expireTime time.Time

	// Hooks
	hook Hook
}

type Hook struct {
	OnGet    func()
	OnDelete func()
}

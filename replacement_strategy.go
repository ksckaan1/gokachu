package gokachu

type ReplacementStrategy uint

const (
	ReplacementStrategyNone ReplacementStrategy = iota
	ReplacementStrategyLRU                      // Least Recently Used
	ReplacementStrategyMRU                      // Most Recently Used
	ReplacementStrategyFIFO                     // First In First Out
	ReplacementStrategyLIFO                     // Last In First Out
	ReplacementStrategyLFU                      // Least Frequently Used
	ReplacementStrategyMFU                      // Most Frequently Used
)

# 📊 Benchmarks

All benchmarks below were run on the same machine with `-benchmem`.

## Environment

| | |
| --- | --- |
| CPU | Intel Core i5-1135G7 (4 cores / 8 threads) |
| OS | Linux (amd64) |
| Go | 1.27.1 |

## Methodology

Serial and parallel benchmarks were run in separate sessions so they do not interfere with each other. Every value is the **median of 5 runs**.

```bash
# Serial
go test -run '^$' -bench '^BenchmarkGokachu_(Update|Get|InsertEvict|GetMiss|Delete)$|^BenchmarkGokachu_GetStrategies$' \
  -benchmem -cpu=1,4,8 -count=5 -benchtime=1s .

# Parallel
go test -run '^$' -bench 'Parallel' \
  -benchmem -cpu=1,4,8 -count=5 -benchtime=2s .
```

> [!NOTE]
> These numbers come from a laptop and absolute values drift with temperature. Compare sharded and unsharded results from the same session rather than across sessions.
>
> `-cpu` on the serial benchmarks only changes `GOMAXPROCS`; the work is still single-goroutine, so those columns should be flat and their differences are noise.

## Serial benchmarks

| Benchmark | `-cpu=1` | `-cpu=4` | `-cpu=8` | B/op | allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| `BenchmarkGokachu_Update` | 37.7 ns/op | 30.3 ns/op | 30.2 ns/op | 0 | 0 |
| `BenchmarkGokachu_Get` | 33.5 ns/op | 33.3 ns/op | 33.5 ns/op | 0 | 0 |
| `BenchmarkGokachu_GetMiss` | 27.9 ns/op | 43.8 ns/op | 28.1 ns/op | 0 | 0 |
| `BenchmarkGokachu_Delete` | 159.7 ns/op | 116.1 ns/op | 127.6 ns/op | 112 | 2 |
| `BenchmarkGokachu_InsertEvict` | 190.8 ns/op | 143.1 ns/op | 143.8 ns/op | 112 | 2 |

- `Update`: `Set` on an existing key.
- `Get`: `Get` on an existing key.
- `GetMiss`: `Get` on a key that does not exist.
- `Delete`: a `Set` + `Delete` pair, since the key has to exist to be deleted.
- `InsertEvict`: inserting into a full cache, which evicts before inserting.

## Replacement strategies

`Get` with 1,000 pre-populated keys, `-cpu=1`:

| Strategy | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `None` | 35.1 | 0 | 0 |
| `LRU` | 37.5 | 0 | 0 |
| `MRU` | 37.0 | 0 | 0 |
| `FIFO` | 35.2 | 0 | 0 |
| `LIFO` | 39.8 | 0 | 0 |
| `LFU` | 38.2 | 0 | 0 |
| `MFU` | 38.3 | 0 | 0 |

All strategies stay in the same range for `Get`; none of them allocate.

## Parallel benchmarks

100,000 keys, LRU, eviction disabled, `-cpu` goroutines on distinct key ranges.

| Benchmark | `-cpu=1` | `-cpu=4` | `-cpu=8` |
| --- | ---: | ---: | ---: |
| `BenchmarkGokachu_GetParallel` (1 shard) | 47.7 ns/op | 95.1 ns/op | 127.7 ns/op |
| `BenchmarkGokachu_GetParallelSharded` (64 shards) | 77.4 ns/op | 72.7 ns/op | 60.1 ns/op |
| `BenchmarkGokachu_UpdateParallel` (1 shard) | 52.5 ns/op | 104.4 ns/op | 135.6 ns/op |
| `BenchmarkGokachu_UpdateParallelSharded` (64 shards) | 81.0 ns/op | 73.2 ns/op | 64.4 ns/op |
| `BenchmarkGokachu_MixedParallel` (1 shard, 90% get / 10% set) | 50.4 ns/op | 99.4 ns/op | 134.0 ns/op |
| `BenchmarkGokachu_MixedParallelSharded` (64 shards, 90% get / 10% set) | 80.6 ns/op | 71.5 ns/op | 62.7 ns/op |

All parallel benchmarks report 0 B/op and 0 allocs/op.

### Sharding speedup at `-cpu=8`

| Workload | 1 shard | 64 shards | Speedup |
| --- | ---: | ---: | ---: |
| `Get` | 127.7 ns/op | 60.1 ns/op | 2.1x |
| `Set` | 135.6 ns/op | 64.4 ns/op | 2.1x |
| Mixed (90% get / 10% set) | 134.0 ns/op | 62.7 ns/op | 2.1x |

Unsharded throughput gets **worse** as goroutines increase because every operation takes the same lock. Sharded throughput improves with cores because each shard has its own lock.

Sharding is not free at low concurrency: at `-cpu=1` it is slower than a single shard, because `hash/maphash.Comparable` randomizes key placement and shard lookup adds indirection. Use it when many goroutines access many different keys; a single hot key always lands on one shard.

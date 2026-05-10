# Benchmarks

The Go and C++ benchmark suites use the same workload names and dimensions so
results can be compared by shape:

- `fill_drain`: push all items, then pop all items
- `steady_state`: prefill, then repeatedly pop one item and push one item
- `sparse_reused`: reuse 16 active priorities across a 1,000,000-priority range

The shared fixed-range matrix is:

- `buckets={16,1024,100000}`
- `items_per_bucket={1,100}`

Every workload includes the `stdlib_heap` baseline. In Go, `stdlib_heap` is a
FIFO-preserving adapter over the standard `container/heap` package.

Run:

```bash
go test -run '^$' -bench 'Benchmark(FillDrain|SteadyState|SparseReused)' -benchmem
```

By default, fixed-range cases above `10_000_000` items are skipped. Override the
cutoff with:

```bash
FASTPQ_BENCH_MAX_ITEMS=100000 go test -run '^$' -bench 'Benchmark(FillDrain|SteadyState|SparseReused)' -benchmem
```

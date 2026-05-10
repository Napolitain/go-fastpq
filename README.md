# go-fastpq

`go-fastpq` is a family of bucket-based priority queues for Go with:

- a dynamic number of values
- priority `0` as the highest priority
- FIFO ordering within each priority bucket
- generic storage via `Queue[T]`, `BulkQueue[T]`, and `SparseQueue[T]`

## Queue selection

| Queue | Use when | Priority range | Push/Pop pattern | Goal |
| --- | --- | --- | --- | --- |
| `Queue[T]` | General live workload | Fixed at construction | Interleaved `Push` and `Pop` | O(1) push plus bitmap-assisted pop without heap ordering |
| `BulkQueue[T]` | Values are pushed before draining starts | Fixed at construction | Fill, then drain | Lower metadata overhead; total drain cost is O(items + priorities) |
| `SparseQueue[T]` | Priority values may be huge but active pages are few | Non-negative and sparse | Interleaved `Push` and `Pop` | Allocate buckets by 64-priority pages instead of the full range |

## Install

```bash
go get github.com/Napolitain/go-fastpq
```

## Usage

```go
package main

import (
	"fmt"
	"log"

	fastpq "github.com/Napolitain/go-fastpq"
)

func main() {
	q, err := fastpq.New[string](4)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range []struct {
		priority int
		value    string
	}{
		{priority: 2, value: "background-a"},
		{priority: 0, value: "urgent"},
		{priority: 1, value: "normal"},
		{priority: 2, value: "background-b"},
	} {
		if err := q.Push(item.priority, item.value); err != nil {
			log.Fatal(err)
		}
	}

	for !q.IsEmpty() {
		value, _ := q.Pop()
		fmt.Println(value)
	}
}
```

Output:

```text
urgent
normal
background-a
background-b
```

## API

All queue variants implement:

```go
type PriorityQueue[T any] interface {
	Push(priority int, value T) error
	Peek() (T, bool)
	Pop() (T, bool)
	Len() int
	IsEmpty() bool
}
```

```go
q, err := fastpq.New[T](numPriorities)
err = q.Push(priority, value)
value, ok := q.Peek()
value, ok := q.Pop()
n := q.Len()
empty := q.IsEmpty()
priorities := q.NumPriorities()
```

Specialized constructors:

```go
bulk, err := fastpq.NewBulk[T](numPriorities)
sparse := fastpq.NewSparse[T]()
```

`BulkQueue` rejects `Push` after `Peek` or `Pop` starts a drain phase while
values remain queued. Drain it fully or call `Clear` before pushing a new batch.

## Benchmarks

The benchmark suite uses the same workload names and dimensions as the C++
benchmark. It compares `Queue`, `BulkQueue`, `SparseQueue`, and the `stdlib_heap`
baseline implemented with Go's standard `container/heap`.

It includes three workload styles:

- `fill_drain`: push everything, then pop everything
- `steady_state`: prefill, then repeatedly `Pop` + `Push`
- `sparse_reused`: reuse 16 active priorities across a 1,000,000-priority range

Run it with:

```bash
go test -run '^$' -bench 'Benchmark(FillDrain|SteadyState|SparseReused)' -benchmem
```

The shared matrix uses `buckets={16,1024,100000}` and
`items_per_bucket={1,100}`. By default, combinations above `10_000_000` items
are skipped. Override that cutoff with `FASTPQ_BENCH_MAX_ITEMS`.

## Notes

- Priorities are `0`-based and valid in `[0, N)`.
- `SparseQueue` accepts any non-negative priority.
- Fixed queue priority counts are immutable after construction.
- The current implementations are not synchronized for concurrent use.

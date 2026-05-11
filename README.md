# go-fastpq

`go-fastpq` is a small family of bucket-based priority queues for Go.

Use it when priorities are bounded or sparse enough that bucket lookup beats a
comparison heap. Priority `0` is the highest priority, and values with the same
priority are popped in FIFO order.

## Install

```bash
go get github.com/Napolitain/go-fastpq
```

```go
import fastpq "github.com/Napolitain/go-fastpq"
```

The module currently targets Go 1.26.3 or newer.

## Choose A Queue

| Queue | Use when | Priority range | Push/pop pattern | Goal |
| --- | --- | --- | --- | --- |
| `Queue[T]` | General live workload | Fixed at construction | Interleaved `Push` and `Pop` | O(1) push plus bitmap-assisted pop without heap ordering |
| `BulkQueue[T]` | Values are pushed before draining starts | Fixed at construction | Fill, then drain | Lower metadata overhead; total drain cost is O(items + priorities) |
| `SparseQueue[T]` | Priority values may be huge but active pages are few | Non-negative and sparse | Interleaved `Push` and `Pop` | Allocate buckets by 64-priority pages instead of the full range |

Use `Queue[T]` as the default. Switch to `BulkQueue[T]` only for strict
fill-then-drain batches. Use `SparseQueue[T]` when a dense queue would allocate
too many unused buckets.

## Live Queue Example

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

## Bulk Queue Example

`BulkQueue` is for one fill phase followed by one drain phase. It rejects new
pushes while a non-empty drain is in progress.

```go
q, err := fastpq.NewBulk[int](128)
if err != nil {
	return err
}

for _, job := range jobs {
	if err := q.Push(job.Priority, job.ID); err != nil {
		return err
	}
}

for {
	id, ok := q.Pop()
	if !ok {
		break
	}
	runJob(id)
}
```

Drain it fully or call `Clear` before starting another batch.

## Sparse Queue Example

```go
q := fastpq.NewSparse[string]()
_ = q.Push(1_000_000, "rare")
_ = q.Push(5, "soon")

value, ok := q.Pop() // "soon", true
```

`SparseQueue` accepts any non-negative priority and allocates 64-priority pages
only when they become active.

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

Constructors:

```go
q, err := fastpq.New[T](numPriorities)
bulk, err := fastpq.NewBulk[T](numPriorities)
sparse := fastpq.NewSparse[T]()
```

Fixed queues return `ErrInvalidPriorityCount` for non-positive priority counts
and `ErrPriorityOutOfRange` for priorities outside `[0, N)`. `BulkQueue`
returns `ErrBulkQueueDraining` when a push violates the fill-then-drain
contract. Use `errors.Is` when checking these errors.

The queues are not synchronized. Protect them with a lock if multiple
goroutines access the same instance.

## Development Checks

Run the same checks as GitHub Actions:

```bash
gofmt -l .
go mod tidy
git diff --exit-code -- go.mod go.sum
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run ./...
go test -v -coverprofile=coverage.out -covermode=atomic ./...
go build ./...
go run golang.org/x/tools/cmd/deadcode@v0.45.0 -test ./...
```

## Benchmarks

The benchmark suite uses the same workload names and dimensions as the C++
benchmark. It compares `Queue`, `BulkQueue`, `SparseQueue`, the `stdlib_heap`
baseline implemented with Go's standard `container/heap`, and the
`static_buckets_budget` fill-drain performance-envelope endpoint.

Run it with:

```bash
go test -run '^$' -bench 'Benchmark(FillDrain|SteadyState|SparseReused)' -benchmem
```

The shared matrix uses `buckets={16,1024,100000}` and
`items_per_bucket={1,100}`. By default, combinations above `10_000_000` items
are skipped. Override that cutoff with `FASTPQ_BENCH_MAX_ITEMS`.

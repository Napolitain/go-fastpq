# go-fastpq

`go-fastpq` is a bucket-based priority queue for Go with:

- a fixed number of priorities chosen at construction time
- a dynamic number of values
- priority `0` as the highest priority
- FIFO ordering within each priority bucket
- generic storage via `Queue[T]`

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

```go
q, err := fastpq.New[T](numPriorities)
err = q.Push(priority, value)
value, ok := q.Peek()
value, ok := q.Pop()
n := q.Len()
empty := q.IsEmpty()
priorities := q.NumPriorities()
```

## Benchmarks

The benchmark suite compares `go-fastpq` against a FIFO-preserving reference
implementation built on Go's standard `container/heap`.

It includes two workload styles:

- push everything, then pop everything
- steady flow with a prefilled queue and repeated `Pop` + `Push`

Run it with:

```bash
go test -run '^$' -bench 'BenchmarkQueue' -benchmem
```

The requested matrix uses priorities `{10, 1000, 100000}` and items per bucket
`{100, 10000, 1000000}`. By default, only practical combinations up to
`10_000_000` live items are included. You can raise or lower that cutoff with
`FASTPQ_BENCH_MAX_LIVE_ITEMS`.

## Notes

- Priorities are `0`-based and valid in `[0, N)`.
- The priority count is immutable after `New`.
- The current implementation is not synchronized for concurrent use.

package fastpq_test

import (
	"container/heap"
	"fmt"
	"os"
	"strconv"
	"testing"

	fastpq "github.com/Napolitain/go-fastpq"
)

var (
	benchmarkPriorities     = []int{10, 1000, 100000}
	benchmarkItemsPerBucket = []int{100, 10000, 1000000}
)

const defaultBenchmarkMaxLiveItems int64 = 10_000_000

type benchmarkCase struct {
	priorities     int
	itemsPerBucket int
	totalItems     int64
}

type heapBenchmarkItem struct {
	value    int
	priority int
	seq      uint64
}

type stableHeapQueue struct {
	items   []heapBenchmarkItem
	nextSeq uint64
}

func BenchmarkQueuePushPop(b *testing.B) {
	for _, tc := range benchmarkCases(b) {
		tc := tc
		b.Run(tc.name(), func(b *testing.B) {
			b.Run("fastpq", func(b *testing.B) {
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					q, err := fastpq.New[int](tc.priorities)
					if err != nil {
						b.Fatalf("New(%d): %v", tc.priorities, err)
					}

					fillFastPQ(b, q, tc)
					drainFastPQ(b, q, tc.totalItems)
				}
			})

			b.Run("container_heap", func(b *testing.B) {
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					q := newStableHeapQueue(b, tc.totalItems)
					fillStableHeap(q, tc)
					drainStableHeap(b, q, tc.totalItems)
				}
			})
		})
	}
}

func BenchmarkQueueSteadyFlow(b *testing.B) {
	for _, tc := range benchmarkCases(b) {
		tc := tc
		b.Run(tc.name(), func(b *testing.B) {
			b.Run("fastpq", func(b *testing.B) {
				q, err := fastpq.New[int](tc.priorities)
				if err != nil {
					b.Fatalf("New(%d): %v", tc.priorities, err)
				}

				fillFastPQ(b, q, tc)
				nextValue := benchmarkCapacity(b, tc.totalItems)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					if _, ok := q.Pop(); !ok {
						b.Fatal("Pop(): queue unexpectedly empty")
					}
					if err := q.Push(steadyFlowPriority(i, tc.priorities), nextValue); err != nil {
						b.Fatalf("Push(): %v", err)
					}
					nextValue++
				}
			})

			b.Run("container_heap", func(b *testing.B) {
				q := newStableHeapQueue(b, tc.totalItems)
				fillStableHeap(q, tc)
				nextValue := benchmarkCapacity(b, tc.totalItems)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					if _, ok := q.Dequeue(); !ok {
						b.Fatal("Dequeue(): queue unexpectedly empty")
					}
					q.Enqueue(steadyFlowPriority(i, tc.priorities), nextValue)
					nextValue++
				}
			})
		})
	}
}

func benchmarkCases(b testing.TB) []benchmarkCase {
	b.Helper()

	maxLiveItems := benchmarkMaxLiveItems(b)
	cases := make([]benchmarkCase, 0, len(benchmarkPriorities)*len(benchmarkItemsPerBucket))

	for _, priorities := range benchmarkPriorities {
		for _, itemsPerBucket := range benchmarkItemsPerBucket {
			totalItems := int64(priorities) * int64(itemsPerBucket)
			if totalItems > maxLiveItems {
				continue
			}

			cases = append(cases, benchmarkCase{
				priorities:     priorities,
				itemsPerBucket: itemsPerBucket,
				totalItems:     totalItems,
			})
		}
	}

	if len(cases) == 0 {
		b.Fatalf("no benchmark cases remain under FASTPQ_BENCH_MAX_LIVE_ITEMS=%d", maxLiveItems)
	}

	return cases
}

func benchmarkMaxLiveItems(b testing.TB) int64 {
	b.Helper()

	raw := os.Getenv("FASTPQ_BENCH_MAX_LIVE_ITEMS")
	if raw == "" {
		return defaultBenchmarkMaxLiveItems
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		b.Fatalf("FASTPQ_BENCH_MAX_LIVE_ITEMS=%q: %v", raw, err)
	}
	if value <= 0 {
		b.Fatalf("FASTPQ_BENCH_MAX_LIVE_ITEMS=%q must be positive", raw)
	}

	return value
}

func benchmarkCapacity(b testing.TB, totalItems int64) int {
	b.Helper()

	maxInt := int64(^uint(0) >> 1)
	if totalItems > maxInt {
		b.Fatalf("benchmark requires %d items, exceeding max int %d", totalItems, maxInt)
	}

	return int(totalItems)
}

func fillFastPQ(b testing.TB, q *fastpq.Queue[int], tc benchmarkCase) {
	b.Helper()

	value := 0
	for bucketOffset := 0; bucketOffset < tc.itemsPerBucket; bucketOffset++ {
		for priority := 0; priority < tc.priorities; priority++ {
			if err := q.Push(priority, value); err != nil {
				b.Fatalf("Push(%d, %d): %v", priority, value, err)
			}
			value++
		}
	}
}

func drainFastPQ(b testing.TB, q *fastpq.Queue[int], totalItems int64) {
	b.Helper()

	for popped := int64(0); popped < totalItems; popped++ {
		if _, ok := q.Pop(); !ok {
			b.Fatalf("Pop() failed after %d/%d items", popped, totalItems)
		}
	}
}

func steadyFlowPriority(step, priorities int) int {
	return int((uint64(step) * 11400714819323198485) % uint64(priorities))
}

func (tc benchmarkCase) name() string {
	return fmt.Sprintf("priorities_%d/items_per_bucket_%d", tc.priorities, tc.itemsPerBucket)
}

func newStableHeapQueue(b testing.TB, totalItems int64) *stableHeapQueue {
	b.Helper()

	q := &stableHeapQueue{
		items: make([]heapBenchmarkItem, 0, benchmarkCapacity(b, totalItems)),
	}
	heap.Init(q)

	return q
}

func fillStableHeap(q *stableHeapQueue, tc benchmarkCase) {
	value := 0
	for bucketOffset := 0; bucketOffset < tc.itemsPerBucket; bucketOffset++ {
		for priority := 0; priority < tc.priorities; priority++ {
			q.Enqueue(priority, value)
			value++
		}
	}
}

func drainStableHeap(b testing.TB, q *stableHeapQueue, totalItems int64) {
	b.Helper()

	for popped := int64(0); popped < totalItems; popped++ {
		if _, ok := q.Dequeue(); !ok {
			b.Fatalf("Dequeue() failed after %d/%d items", popped, totalItems)
		}
	}
}

func (q *stableHeapQueue) Len() int {
	return len(q.items)
}

func (q *stableHeapQueue) Less(i, j int) bool {
	if q.items[i].priority != q.items[j].priority {
		return q.items[i].priority < q.items[j].priority
	}
	return q.items[i].seq < q.items[j].seq
}

func (q *stableHeapQueue) Swap(i, j int) {
	q.items[i], q.items[j] = q.items[j], q.items[i]
}

func (q *stableHeapQueue) Push(x any) {
	q.items = append(q.items, x.(heapBenchmarkItem))
}

func (q *stableHeapQueue) Pop() any {
	last := len(q.items) - 1
	item := q.items[last]
	q.items[last] = heapBenchmarkItem{}
	q.items = q.items[:last]
	return item
}

func (q *stableHeapQueue) Enqueue(priority, value int) {
	heap.Push(q, heapBenchmarkItem{
		value:    value,
		priority: priority,
		seq:      q.nextSeq,
	})
	q.nextSeq++
}

func (q *stableHeapQueue) Dequeue() (int, bool) {
	if len(q.items) == 0 {
		return 0, false
	}

	item := heap.Pop(q).(heapBenchmarkItem)
	return item.value, true
}

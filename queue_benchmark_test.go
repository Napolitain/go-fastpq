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
	benchmarkBucketCounts   = []int{16, 1024, 100000}
	benchmarkItemsPerBucket = []int{1, 100}
)

const defaultBenchmarkMaxItems int64 = 10_000_000

type benchmarkCase struct {
	bucketCount    int
	itemsPerBucket int
	itemCount      int64
}

type sparseBenchmarkCase struct {
	priorityRange    int
	activePriorities int
	items            int
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

func BenchmarkFillDrain(b *testing.B) {
	for _, tc := range benchmarkCases(b) {
		tc := tc
		b.Run(tc.name(), func(b *testing.B) {
			b.Run("queue", func(b *testing.B) {
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					q, err := fastpq.New[int](tc.bucketCount)
					if err != nil {
						b.Fatalf("New(%d): %v", tc.bucketCount, err)
					}

					fillPQ(b, q, tc)
					drainPQ(b, q, tc.itemCount)
				}
			})

			b.Run("bulk_queue", func(b *testing.B) {
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					q, err := fastpq.NewBulk[int](tc.bucketCount)
					if err != nil {
						b.Fatalf("NewBulk(%d): %v", tc.bucketCount, err)
					}

					fillPQ(b, q, tc)
					drainPQ(b, q, tc.itemCount)
				}
			})

			b.Run("stdlib_heap", func(b *testing.B) {
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					q := newStableHeapQueue(b, tc.itemCount)
					fillStableHeap(q, tc)
					drainStableHeap(b, q, tc.itemCount)
				}
			})
		})
	}
}

func BenchmarkSteadyState(b *testing.B) {
	for _, tc := range benchmarkCases(b) {
		tc := tc
		b.Run(tc.name(), func(b *testing.B) {
			b.Run("queue", func(b *testing.B) {
				q, err := fastpq.New[int](tc.bucketCount)
				if err != nil {
					b.Fatalf("New(%d): %v", tc.bucketCount, err)
				}

				fillPQ(b, q, tc)
				nextValue := benchmarkCapacity(b, tc.itemCount)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					if _, ok := q.Pop(); !ok {
						b.Fatal("Pop(): queue unexpectedly empty")
					}
					if err := q.Push(steadyStatePriority(i, tc.bucketCount), nextValue); err != nil {
						b.Fatalf("Push(): %v", err)
					}
					nextValue++
				}
			})

			b.Run("stdlib_heap", func(b *testing.B) {
				q := newStableHeapQueue(b, tc.itemCount)
				fillStableHeap(q, tc)
				nextValue := benchmarkCapacity(b, tc.itemCount)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					if _, ok := q.Dequeue(); !ok {
						b.Fatal("Dequeue(): queue unexpectedly empty")
					}
					q.Enqueue(steadyStatePriority(i, tc.bucketCount), nextValue)
					nextValue++
				}
			})
		})
	}
}

func BenchmarkSparseReused(b *testing.B) {
	tc := sparseBenchmarkCase{
		priorityRange:    1_000_000,
		activePriorities: 16,
		items:            100_000,
	}

	b.Run(tc.name(), func(b *testing.B) {
		b.Run("sparse_queue", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				q := fastpq.NewSparse[int]()
				fillSparsePQ(b, q, tc)
				drainPQ(b, q, int64(tc.items))
			}
		})

		b.Run("fixed_queue", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				q, err := fastpq.New[int](tc.priorityRange)
				if err != nil {
					b.Fatalf("New(%d): %v", tc.priorityRange, err)
				}
				fillSparsePQ(b, q, tc)
				drainPQ(b, q, int64(tc.items))
			}
		})

		b.Run("stdlib_heap", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				q := newStableHeapQueue(b, int64(tc.items))
				fillSparseHeap(q, tc)
				drainStableHeap(b, q, int64(tc.items))
			}
		})
	})
}

func benchmarkCases(b testing.TB) []benchmarkCase {
	b.Helper()

	maxItems := benchmarkMaxItems(b)
	cases := make([]benchmarkCase, 0, len(benchmarkBucketCounts)*len(benchmarkItemsPerBucket))

	for _, bucketCount := range benchmarkBucketCounts {
		for _, itemsPerBucket := range benchmarkItemsPerBucket {
			itemCount := int64(bucketCount) * int64(itemsPerBucket)
			if itemCount > maxItems {
				continue
			}

			cases = append(cases, benchmarkCase{
				bucketCount:    bucketCount,
				itemsPerBucket: itemsPerBucket,
				itemCount:      itemCount,
			})
		}
	}

	if len(cases) == 0 {
		b.Fatalf("no benchmark cases remain under FASTPQ_BENCH_MAX_ITEMS=%d", maxItems)
	}

	return cases
}

func benchmarkMaxItems(b testing.TB) int64 {
	b.Helper()

	raw := os.Getenv("FASTPQ_BENCH_MAX_ITEMS")
	if raw == "" {
		raw = os.Getenv("FASTPQ_BENCH_MAX_LIVE_ITEMS")
	}
	if raw == "" {
		return defaultBenchmarkMaxItems
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		b.Fatalf("FASTPQ_BENCH_MAX_ITEMS=%q: %v", raw, err)
	}
	if value <= 0 {
		b.Fatalf("FASTPQ_BENCH_MAX_ITEMS=%q must be positive", raw)
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

func fillPQ(b testing.TB, q fastpq.PriorityQueue[int], tc benchmarkCase) {
	b.Helper()

	value := 0
	for bucketOffset := 0; bucketOffset < tc.itemsPerBucket; bucketOffset++ {
		for priority := 0; priority < tc.bucketCount; priority++ {
			if err := q.Push(priority, value); err != nil {
				b.Fatalf("Push(%d, %d): %v", priority, value, err)
			}
			value++
		}
	}
}

func fillSparsePQ(b testing.TB, q fastpq.PriorityQueue[int], tc sparseBenchmarkCase) {
	b.Helper()

	priorities := sparseReusedPriorities(tc)
	for value := 0; value < tc.items; value++ {
		priority := priorities[value%len(priorities)]
		if err := q.Push(priority, value); err != nil {
			b.Fatalf("Push(%d, %d): %v", priority, value, err)
		}
	}
}

func drainPQ(b testing.TB, q fastpq.PriorityQueue[int], totalItems int64) {
	b.Helper()

	for popped := int64(0); popped < totalItems; popped++ {
		if _, ok := q.Pop(); !ok {
			b.Fatalf("Pop() failed after %d/%d items", popped, totalItems)
		}
	}
}

func fillSparseHeap(q *stableHeapQueue, tc sparseBenchmarkCase) {
	priorities := sparseReusedPriorities(tc)
	for value := 0; value < tc.items; value++ {
		q.Enqueue(priorities[value%len(priorities)], value)
	}
}

func sparseReusedPriorities(tc sparseBenchmarkCase) []int {
	if tc.activePriorities <= 1 {
		return []int{0}
	}

	priorities := make([]int, tc.activePriorities)
	for i := 0; i < tc.activePriorities; i++ {
		priorities[i] = (i * (tc.priorityRange - 1)) / (tc.activePriorities - 1)
	}

	return priorities
}

func steadyStatePriority(step, bucketCount int) int {
	return int((uint64(step) * 11400714819323198485) % uint64(bucketCount))
}

func (tc benchmarkCase) name() string {
	return fmt.Sprintf("buckets_%d/items_per_bucket_%d/items_%d", tc.bucketCount, tc.itemsPerBucket, tc.itemCount)
}

func (tc sparseBenchmarkCase) name() string {
	return fmt.Sprintf("buckets_%d/items_per_bucket_0/items_%d/active_priorities_%d", tc.priorityRange, tc.items, tc.activePriorities)
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
		for priority := 0; priority < tc.bucketCount; priority++ {
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

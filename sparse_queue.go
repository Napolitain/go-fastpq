package fastpq

import (
	"container/heap"
	"math/bits"
)

// SparseQueue is optimized for interleaved Push and Pop calls where priorities
// are non-negative but the possible priority range is very large and only a
// small number of pages are active. It allocates buckets by 64-priority pages
// instead of allocating one bucket for every possible priority.
//
// Lower numeric priorities are popped first, so priority 0 is the highest
// priority. Ordering within each priority bucket is FIFO.
type SparseQueue[T any] struct {
	pages       map[int]*sparsePage[T]
	activePages minPageHeap
	size        int
}

type sparsePage[T any] struct {
	buckets  [wordBits]bucket[T]
	nonEmpty uint64
}

// NewSparse creates a live sparse queue for non-negative priorities.
func NewSparse[T any]() *SparseQueue[T] {
	return &SparseQueue[T]{
		pages: make(map[int]*sparsePage[T]),
	}
}

// Len returns the number of queued values across all active priority buckets.
func (q *SparseQueue[T]) Len() int {
	return q.size
}

// IsEmpty reports whether the queue currently holds any values.
func (q *SparseQueue[T]) IsEmpty() bool {
	return q.size == 0
}

// ActivePageCount returns the number of 64-priority pages currently holding
// at least one value.
func (q *SparseQueue[T]) ActivePageCount() int {
	return len(q.activePages)
}

// Push inserts value into the FIFO bucket for priority.
func (q *SparseQueue[T]) Push(priority int, value T) error {
	if err := validateSparsePriority(priority); err != nil {
		return err
	}
	q.ensureInitialized()

	pageIndex := priority / wordBits
	bucketOffset := priority % wordBits
	page := q.ensurePage(pageIndex)
	pageWasEmpty := page.nonEmpty == 0
	bucketWasEmpty := page.buckets[bucketOffset].push(value)

	if bucketWasEmpty {
		page.nonEmpty |= uint64(1) << uint(bucketOffset)
	}
	if pageWasEmpty {
		heap.Push(&q.activePages, pageIndex)
	}

	q.size++
	return nil
}

// Peek returns the next value to be popped without removing it.
func (q *SparseQueue[T]) Peek() (T, bool) {
	var zero T
	if q.size == 0 {
		return zero, false
	}

	_, page, bucketOffset := q.head()
	return page.buckets[bucketOffset].front(), true
}

// Pop removes and returns the next value from the highest-priority non-empty
// bucket while preserving FIFO order inside that bucket.
func (q *SparseQueue[T]) Pop() (T, bool) {
	var zero T
	if q.size == 0 {
		return zero, false
	}

	pageIndex, page, bucketOffset := q.head()
	value, emptied := page.buckets[bucketOffset].popFront()
	q.size--

	if emptied {
		page.nonEmpty &^= uint64(1) << uint(bucketOffset)
		if page.nonEmpty == 0 {
			delete(q.pages, pageIndex)
			heap.Pop(&q.activePages)
		}
	}

	return value, true
}

// Clear removes all queued values and keeps the queue ready for another sparse
// live workload.
func (q *SparseQueue[T]) Clear() {
	q.pages = make(map[int]*sparsePage[T])
	q.activePages = q.activePages[:0]
	q.size = 0
}

func (q *SparseQueue[T]) ensureInitialized() {
	if q.pages == nil {
		q.pages = make(map[int]*sparsePage[T])
	}
}

func (q *SparseQueue[T]) ensurePage(pageIndex int) *sparsePage[T] {
	if page := q.pages[pageIndex]; page != nil {
		return page
	}

	page := &sparsePage[T]{}
	q.pages[pageIndex] = page
	return page
}

func (q *SparseQueue[T]) head() (int, *sparsePage[T], int) {
	pageIndex := q.activePages[0]
	page := q.pages[pageIndex]
	return pageIndex, page, bits.TrailingZeros64(page.nonEmpty)
}

type minPageHeap []int

func (h minPageHeap) Len() int {
	return len(h)
}

func (h minPageHeap) Less(i, j int) bool {
	return h[i] < h[j]
}

func (h minPageHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *minPageHeap) Push(x any) {
	value, ok := x.(int)
	if !ok {
		panic("fastpq: minPageHeap only accepts int values")
	}

	*h = append(*h, value)
}

func (h *minPageHeap) Pop() any {
	old := *h
	last := len(old) - 1
	value := old[last]
	*h = old[:last]
	return value
}

package fastpq

import "fmt"

// BulkQueue is optimized for the fill-then-drain case: push all values first,
// then pop until empty. It does not maintain live non-empty bitmap metadata, so
// it has lower overhead than Queue while draining work in O(items + priorities).
//
// Push returns ErrBulkQueueDraining if called after Peek or Pop has started a
// drain phase and the queue still contains values. Once the queue is fully
// drained, Push starts a new fill phase.
type BulkQueue[T any] struct {
	buckets  []bucket[T]
	size     int
	cursor   int
	draining bool
}

// NewBulk creates a fill-then-drain queue with a fixed number of priorities.
func NewBulk[T any](numPriorities int) (*BulkQueue[T], error) {
	if err := validatePriorityCount(numPriorities); err != nil {
		return nil, err
	}

	return &BulkQueue[T]{
		buckets: make([]bucket[T], numPriorities),
	}, nil
}

// NumPriorities returns the queue's fixed priority count.
func (q *BulkQueue[T]) NumPriorities() int {
	return len(q.buckets)
}

// Len returns the number of queued values across all priority buckets.
func (q *BulkQueue[T]) Len() int {
	return q.size
}

// IsEmpty reports whether the queue currently holds any values.
func (q *BulkQueue[T]) IsEmpty() bool {
	return q.size == 0
}

// Push inserts value into the FIFO bucket for priority during the fill phase.
func (q *BulkQueue[T]) Push(priority int, value T) error {
	if err := validateFixedPriority(priority, len(q.buckets)); err != nil {
		return err
	}
	if q.draining {
		if q.size > 0 {
			return fmt.Errorf("%w: drain the queue or call Clear before pushing again", ErrBulkQueueDraining)
		}

		q.draining = false
		q.cursor = 0
	}

	q.buckets[priority].push(value)
	q.size++
	return nil
}

// Peek returns the next value to be popped without removing it. Calling Peek
// starts the drain phase.
func (q *BulkQueue[T]) Peek() (T, bool) {
	var zero T
	if q.size == 0 {
		return zero, false
	}

	q.startDrain()
	return q.buckets[q.cursor].front(), true
}

// Pop removes and returns the next value. Calling Pop starts the drain phase.
func (q *BulkQueue[T]) Pop() (T, bool) {
	var zero T
	if q.size == 0 {
		return zero, false
	}

	q.startDrain()
	b := &q.buckets[q.cursor]
	value, emptied := b.popFront()
	q.size--

	if emptied {
		q.cursor++
		q.advanceCursor()
	}

	return value, true
}

// Clear removes all queued values and starts a fresh fill phase.
func (q *BulkQueue[T]) Clear() {
	for i := range q.buckets {
		q.buckets[i].clear()
	}

	q.size = 0
	q.cursor = 0
	q.draining = false
}

func (q *BulkQueue[T]) startDrain() {
	if !q.draining {
		q.draining = true
		q.cursor = 0
	}

	q.advanceCursor()
}

func (q *BulkQueue[T]) advanceCursor() {
	for q.cursor < len(q.buckets) && q.buckets[q.cursor].empty() {
		q.cursor++
	}
}

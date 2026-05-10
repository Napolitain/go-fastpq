package fastpq

import (
	"errors"
	"fmt"
	"math/bits"
)

const (
	wordBits       = 64
	minCompactHead = 64
)

var (
	// ErrInvalidPriorityCount reports that a queue was created with no priorities.
	ErrInvalidPriorityCount = errors.New("fastpq: priority count must be greater than zero")
	// ErrPriorityOutOfRange reports that a pushed priority falls outside [0, N).
	ErrPriorityOutOfRange = errors.New("fastpq: priority out of range")
	// ErrBulkQueueDraining reports that a bulk queue was pushed after draining began.
	ErrBulkQueueDraining = errors.New("fastpq: bulk queue cannot be pushed while draining")
)

// PriorityQueue is the common API implemented by Queue, BulkQueue, and
// SparseQueue.
type PriorityQueue[T any] interface {
	Push(priority int, value T) error
	Peek() (T, bool)
	Pop() (T, bool)
	Len() int
	IsEmpty() bool
}

// Queue is the default live bucket-based priority queue. It is optimized for
// interleaved Push and Pop calls with a fixed number of priorities and FIFO
// ordering within each priority bucket. Lower numeric priorities are popped
// first, so priority 0 is the highest priority.
type Queue[T any] struct {
	buckets      []bucket[T]
	nonEmpty     []uint64
	size         int
	headPriority int
}

type bucket[T any] struct {
	values []T
	head   int
}

// New creates the default live queue with a fixed number of priorities. The
// priority count is immutable for the lifetime of the queue.
func New[T any](numPriorities int) (*Queue[T], error) {
	if err := validatePriorityCount(numPriorities); err != nil {
		return nil, err
	}

	return &Queue[T]{
		buckets:      make([]bucket[T], numPriorities),
		nonEmpty:     make([]uint64, (numPriorities+wordBits-1)/wordBits),
		headPriority: -1,
	}, nil
}

// NumPriorities returns the queue's fixed priority count.
func (q *Queue[T]) NumPriorities() int {
	return len(q.buckets)
}

// Len returns the number of queued values across all priority buckets.
func (q *Queue[T]) Len() int {
	return q.size
}

// IsEmpty reports whether the queue currently holds any values.
func (q *Queue[T]) IsEmpty() bool {
	return q.size == 0
}

// Push inserts value into the FIFO bucket for priority.
func (q *Queue[T]) Push(priority int, value T) error {
	if err := validateFixedPriority(priority, len(q.buckets)); err != nil {
		return err
	}

	b := &q.buckets[priority]
	wasEmpty := b.push(value)
	q.size++

	if wasEmpty {
		q.setNonEmpty(priority)
		if q.headPriority == -1 || priority < q.headPriority {
			q.headPriority = priority
		}
	}

	return nil
}

// Peek returns the next value to be popped without removing it.
func (q *Queue[T]) Peek() (T, bool) {
	var zero T
	if q.size == 0 {
		return zero, false
	}

	b := &q.buckets[q.headPriority]
	return b.front(), true
}

// Pop removes and returns the next value from the highest-priority non-empty
// bucket while preserving FIFO order inside that bucket.
func (q *Queue[T]) Pop() (T, bool) {
	var zero T
	if q.size == 0 {
		return zero, false
	}

	priority := q.headPriority
	b := &q.buckets[priority]

	value, emptied := b.popFront()
	q.size--

	if emptied {
		q.clearNonEmpty(priority)
		if q.size == 0 {
			q.headPriority = -1
		} else {
			next := q.nextNonEmpty(priority + 1)
			if next < 0 {
				panic("fastpq: corrupted non-empty priority tracking")
			}
			q.headPriority = next
		}
		return value, true
	}

	return value, true
}

func validatePriorityCount(numPriorities int) error {
	if numPriorities <= 0 {
		return fmt.Errorf("%w: got %d", ErrInvalidPriorityCount, numPriorities)
	}

	return nil
}

func validateFixedPriority(priority, priorityCount int) error {
	if priority < 0 || priority >= priorityCount {
		return fmt.Errorf("%w: got %d, want [0,%d)", ErrPriorityOutOfRange, priority, priorityCount)
	}

	return nil
}

func validateSparsePriority(priority int) error {
	if priority < 0 {
		return fmt.Errorf("%w: got %d, want >= 0", ErrPriorityOutOfRange, priority)
	}

	return nil
}

func (q *Queue[T]) setNonEmpty(priority int) {
	word := priority / wordBits
	bit := uint(priority % wordBits)
	q.nonEmpty[word] |= 1 << bit
}

func (q *Queue[T]) clearNonEmpty(priority int) {
	word := priority / wordBits
	bit := uint(priority % wordBits)
	q.nonEmpty[word] &^= 1 << bit
}

func (q *Queue[T]) nextNonEmpty(start int) int {
	if start < 0 {
		start = 0
	}

	word := start / wordBits
	if word >= len(q.nonEmpty) {
		return -1
	}

	bit := uint(start % wordBits)
	mask := q.nonEmpty[word] & (^uint64(0) << bit)
	if mask != 0 {
		return word*wordBits + bits.TrailingZeros64(mask)
	}

	for word++; word < len(q.nonEmpty); word++ {
		mask = q.nonEmpty[word]
		if mask != 0 {
			return word*wordBits + bits.TrailingZeros64(mask)
		}
	}

	return -1
}

func (b *bucket[T]) empty() bool {
	return b.head >= len(b.values)
}

func (b *bucket[T]) push(value T) bool {
	wasEmpty := b.empty()
	b.values = append(b.values, value)
	return wasEmpty
}

func (b *bucket[T]) front() T {
	return b.values[b.head]
}

func (b *bucket[T]) popFront() (T, bool) {
	value := b.values[b.head]
	var zero T
	b.values[b.head] = zero
	b.head++

	if b.empty() {
		b.reset()
		return value, true
	}

	b.compactIfNeeded()
	return value, false
}

func (b *bucket[T]) reset() {
	b.values = b.values[:0]
	b.head = 0
}

func (b *bucket[T]) clear() {
	var zero T
	for i := b.head; i < len(b.values); i++ {
		b.values[i] = zero
	}

	b.values = b.values[:0]
	b.head = 0
}

func (b *bucket[T]) compactIfNeeded() {
	if b.head < minCompactHead || b.head*2 < len(b.values) {
		return
	}

	active := copy(b.values, b.values[b.head:])
	var zero T
	for i := active; i < len(b.values); i++ {
		b.values[i] = zero
	}

	b.values = b.values[:active]
	b.head = 0
}

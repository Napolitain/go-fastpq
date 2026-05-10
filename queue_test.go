package fastpq_test

import (
	"errors"
	"testing"

	fastpq "github.com/Napolitain/go-fastpq"
)

func TestNewRejectsInvalidPriorityCount(t *testing.T) {
	t.Parallel()

	_, err := fastpq.New[int](0)
	if !errors.Is(err, fastpq.ErrInvalidPriorityCount) {
		t.Fatalf("New(0) error = %v, want ErrInvalidPriorityCount", err)
	}
}

func TestNewInitializesQueue(t *testing.T) {
	t.Parallel()

	q := mustNewQueue[int](t, 4)

	if got := q.NumPriorities(); got != 4 {
		t.Fatalf("NumPriorities() = %d, want 4", got)
	}
	if got := q.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}
	if !q.IsEmpty() {
		t.Fatal("IsEmpty() = false, want true")
	}
}

func TestPushRejectsPriorityOutOfRange(t *testing.T) {
	t.Parallel()

	q := mustNewQueue[string](t, 2)

	for _, priority := range []int{-1, 2} {
		if err := q.Push(priority, "value"); !errors.Is(err, fastpq.ErrPriorityOutOfRange) {
			t.Fatalf("Push(%d, ...) error = %v, want ErrPriorityOutOfRange", priority, err)
		}
	}
}

func TestPeekAndPopOnEmptyQueue(t *testing.T) {
	t.Parallel()

	q := mustNewQueue[int](t, 3)

	if _, ok := q.Peek(); ok {
		t.Fatal("Peek() ok = true, want false")
	}
	if _, ok := q.Pop(); ok {
		t.Fatal("Pop() ok = true, want false")
	}
}

func TestQueueRespectsPriorityAndFIFOOrder(t *testing.T) {
	t.Parallel()

	q := mustNewQueue[string](t, 4)

	pushes := []struct {
		priority int
		value    string
	}{
		{priority: 2, value: "p2-a"},
		{priority: 3, value: "p3-a"},
		{priority: 0, value: "p0-a"},
		{priority: 2, value: "p2-b"},
		{priority: 1, value: "p1-a"},
		{priority: 0, value: "p0-b"},
	}

	for _, push := range pushes {
		if err := q.Push(push.priority, push.value); err != nil {
			t.Fatalf("Push(%d, %q) error = %v", push.priority, push.value, err)
		}
	}

	if got, ok := q.Peek(); !ok || got != "p0-a" {
		t.Fatalf("Peek() = (%q, %v), want (%q, true)", got, ok, "p0-a")
	}

	want := []string{"p0-a", "p0-b", "p1-a", "p2-a", "p2-b", "p3-a"}
	for i, wantValue := range want {
		if gotLen := q.Len(); gotLen != len(want)-i {
			t.Fatalf("Len() before pop %d = %d, want %d", i, gotLen, len(want)-i)
		}

		got, ok := q.Pop()
		if !ok || got != wantValue {
			t.Fatalf("Pop() at %d = (%q, %v), want (%q, true)", i, got, ok, wantValue)
		}
	}

	if !q.IsEmpty() {
		t.Fatal("IsEmpty() = false, want true")
	}
}

func TestBucketRemainsFIFOThroughCompaction(t *testing.T) {
	t.Parallel()

	q := mustNewQueue[int](t, 1)

	for i := 0; i < 256; i++ {
		if err := q.Push(0, i); err != nil {
			t.Fatalf("initial Push(%d) error = %v", i, err)
		}
	}

	for want := 0; want < 200; want++ {
		got, ok := q.Pop()
		if !ok || got != want {
			t.Fatalf("initial Pop() = (%d, %v), want (%d, true)", got, ok, want)
		}
	}

	for i := 256; i < 384; i++ {
		if err := q.Push(0, i); err != nil {
			t.Fatalf("follow-up Push(%d) error = %v", i, err)
		}
	}

	for want := 200; want < 384; want++ {
		got, ok := q.Pop()
		if !ok || got != want {
			t.Fatalf("final Pop() = (%d, %v), want (%d, true)", got, ok, want)
		}
	}

	if got := q.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}
}

func TestBulkQueueRespectsPriorityAndFIFOOrder(t *testing.T) {
	t.Parallel()

	q, err := fastpq.NewBulk[string](4)
	if err != nil {
		t.Fatalf("NewBulk(4) error = %v", err)
	}

	for _, push := range []struct {
		priority int
		value    string
	}{
		{priority: 2, value: "p2-a"},
		{priority: 3, value: "p3-a"},
		{priority: 0, value: "p0-a"},
		{priority: 2, value: "p2-b"},
		{priority: 1, value: "p1-a"},
		{priority: 0, value: "p0-b"},
	} {
		if err := q.Push(push.priority, push.value); err != nil {
			t.Fatalf("Push(%d, %q) error = %v", push.priority, push.value, err)
		}
	}

	if got, ok := q.Peek(); !ok || got != "p0-a" {
		t.Fatalf("Peek() = (%q, %v), want (%q, true)", got, ok, "p0-a")
	}

	want := []string{"p0-a", "p0-b", "p1-a", "p2-a", "p2-b", "p3-a"}
	for i, wantValue := range want {
		got, ok := q.Pop()
		if !ok || got != wantValue {
			t.Fatalf("Pop() at %d = (%q, %v), want (%q, true)", i, got, ok, wantValue)
		}
	}

	if !q.IsEmpty() {
		t.Fatal("IsEmpty() = false, want true")
	}
}

func TestBulkQueueRejectsPushDuringDrain(t *testing.T) {
	t.Parallel()

	q, err := fastpq.NewBulk[int](3)
	if err != nil {
		t.Fatalf("NewBulk(3) error = %v", err)
	}

	if err := q.Push(2, 20); err != nil {
		t.Fatalf("Push(2, 20) error = %v", err)
	}
	if err := q.Push(0, 0); err != nil {
		t.Fatalf("Push(0, 0) error = %v", err)
	}
	if _, ok := q.Pop(); !ok {
		t.Fatal("Pop() ok = false, want true")
	}
	if err := q.Push(1, 10); !errors.Is(err, fastpq.ErrBulkQueueDraining) {
		t.Fatalf("Push during drain error = %v, want ErrBulkQueueDraining", err)
	}
}

func TestBulkQueueAllowsNewFillPhaseAfterDrain(t *testing.T) {
	t.Parallel()

	q, err := fastpq.NewBulk[int](2)
	if err != nil {
		t.Fatalf("NewBulk(2) error = %v", err)
	}

	if err := q.Push(1, 10); err != nil {
		t.Fatalf("Push(1, 10) error = %v", err)
	}
	if got, ok := q.Pop(); !ok || got != 10 {
		t.Fatalf("Pop() = (%d, %v), want (10, true)", got, ok)
	}
	if err := q.Push(0, 20); err != nil {
		t.Fatalf("Push after drain error = %v", err)
	}
	if got, ok := q.Pop(); !ok || got != 20 {
		t.Fatalf("Pop() after refill = (%d, %v), want (20, true)", got, ok)
	}
}

func TestSparseQueueRespectsSparsePriorityAndFIFOOrder(t *testing.T) {
	t.Parallel()

	q := fastpq.NewSparse[string]()

	for _, push := range []struct {
		priority int
		value    string
	}{
		{priority: 10000, value: "p10000-a"},
		{priority: 2, value: "p2-a"},
		{priority: 4096, value: "p4096-a"},
		{priority: 2, value: "p2-b"},
		{priority: 10000, value: "p10000-b"},
	} {
		if err := q.Push(push.priority, push.value); err != nil {
			t.Fatalf("Push(%d, %q) error = %v", push.priority, push.value, err)
		}
	}

	if got := q.ActivePageCount(); got != 3 {
		t.Fatalf("ActivePageCount() = %d, want 3", got)
	}

	want := []string{"p2-a", "p2-b", "p4096-a", "p10000-a", "p10000-b"}
	for i, wantValue := range want {
		got, ok := q.Pop()
		if !ok || got != wantValue {
			t.Fatalf("Pop() at %d = (%q, %v), want (%q, true)", i, got, ok, wantValue)
		}
	}

	if !q.IsEmpty() {
		t.Fatal("IsEmpty() = false, want true")
	}
}

func TestSparseQueueRejectsNegativePriority(t *testing.T) {
	t.Parallel()

	q := fastpq.NewSparse[int]()
	if err := q.Push(-1, 1); !errors.Is(err, fastpq.ErrPriorityOutOfRange) {
		t.Fatalf("Push(-1, 1) error = %v, want ErrPriorityOutOfRange", err)
	}
}

func mustNewQueue[T any](t *testing.T, priorities int) *fastpq.Queue[T] {
	t.Helper()

	q, err := fastpq.New[T](priorities)
	if err != nil {
		t.Fatalf("New(%d) error = %v", priorities, err)
	}

	return q
}

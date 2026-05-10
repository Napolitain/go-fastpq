// Package fastpq provides bucket-based priority queues.
//
// Queue is the default fixed-range live queue. Use it when Push and Pop can be
// interleaved and the number of priorities is known at construction time.
//
// BulkQueue is the fill-then-drain queue. Use it when all values are pushed
// before popping begins; it avoids live non-empty bitmap metadata and drains in
// O(items + priorities).
//
// SparseQueue is the sparse live queue. Use it when priorities are non-negative
// but the range may be very large and only a small number of 64-priority pages
// are active.
//
// All queues pop lower numeric priorities first and preserve FIFO ordering
// within each priority bucket. PriorityQueue is the shared interface for code
// that can accept any of the queue variants.
package fastpq

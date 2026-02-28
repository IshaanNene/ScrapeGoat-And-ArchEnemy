package engine

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/IshaanNene/ScrapeGoat/internal/types"
)

// Frontier is a thread-safe priority queue of crawl requests.
type Frontier struct {
	mu       sync.Mutex
	pq       priorityQueue
	cond     *sync.Cond
	closed   bool
	notEmpty chan struct{}
}

// NewFrontier creates a new Frontier.
func NewFrontier() *Frontier {
	f := &Frontier{
		pq:       make(priorityQueue, 0, 1024),
		notEmpty: make(chan struct{}, 1),
	}
	f.cond = sync.NewCond(&f.mu)
	heap.Init(&f.pq)
	return f
}

// Push adds a request to the frontier.
func (f *Frontier) Push(req *types.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return
	}

	heap.Push(&f.pq, &pqItem{request: req, priority: req.Priority})
	f.cond.Signal()
}

// Pop removes and returns the highest-priority request.
// Blocks until a request is available or the frontier is closed.
// Returns nil if the frontier is closed and empty.
func (f *Frontier) Pop(ctx context.Context) *types.Request {
	for {
		f.mu.Lock()
		if f.pq.Len() > 0 {
			item := heap.Pop(&f.pq).(*pqItem)
			f.mu.Unlock()
			return item.request
		}
		if f.closed {
			f.mu.Unlock()
			return nil
		}
		f.mu.Unlock()

		// Poll with context support — no goroutine leak
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(50 * time.Millisecond):
			// Re-check on next iteration
		}
	}
}

// TryPop attempts a non-blocking dequeue. Returns nil if empty.
func (f *Frontier) TryPop() *types.Request {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.pq.Len() == 0 {
		return nil
	}

	item := heap.Pop(&f.pq).(*pqItem)
	return item.request
}

// Len returns the number of requests in the frontier.
func (f *Frontier) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pq.Len()
}

// IsEmpty returns true if the frontier is empty.
func (f *Frontier) IsEmpty() bool {
	return f.Len() == 0
}

// Close closes the frontier, unblocking any waiting Pop calls.
func (f *Frontier) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	f.cond.Broadcast()
}

// IsClosed returns true if the frontier has been closed.
func (f *Frontier) IsClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

// Snapshot returns a copy of all queued requests without removing them.
// Safe for use during checkpointing while the crawl is running.
func (f *Frontier) Snapshot() []*types.Request {
	f.mu.Lock()
	defer f.mu.Unlock()

	requests := make([]*types.Request, f.pq.Len())
	for i, item := range f.pq {
		requests[i] = item.request
	}
	return requests
}

// Drain returns all remaining requests, removing them from the queue.
func (f *Frontier) Drain() []*types.Request {
	f.mu.Lock()
	defer f.mu.Unlock()

	requests := make([]*types.Request, 0, f.pq.Len())
	for f.pq.Len() > 0 {
		item := heap.Pop(&f.pq).(*pqItem)
		requests = append(requests, item.request)
	}
	return requests
}

// RestoreAll adds multiple requests back (for checkpoint restore).
func (f *Frontier) RestoreAll(reqs []*types.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, req := range reqs {
		heap.Push(&f.pq, &pqItem{request: req, priority: req.Priority})
	}
	f.cond.Broadcast()
}

// --- Priority Queue Implementation ---

type pqItem struct {
	request  *types.Request
	priority int
	index    int
}

type priorityQueue []*pqItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// Lower priority value = higher priority
	return pq[i].priority < pq[j].priority
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*pqItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // GC
	item.index = -1
	*pq = old[:n-1]
	return item
}

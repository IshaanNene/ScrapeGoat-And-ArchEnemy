package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IshaanNene/ScrapeGoat-And-ArchEnemy/internal/config"
	"github.com/IshaanNene/ScrapeGoat-And-ArchEnemy/internal/types"
)

// State represents the engine's current lifecycle state.
type State int32

const (
	StateIdle     State = 0
	StateRunning  State = 1
	StatePaused   State = 2
	StateStopping State = 3
	StateStopped  State = 4
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StatePaused:
		return "paused"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// Stats tracks crawl statistics.
type Stats struct {
	RequestsSent    atomic.Int64
	RequestsFailed  atomic.Int64
	ResponsesOK     atomic.Int64
	ResponsesError  atomic.Int64
	ItemsScraped    atomic.Int64
	ItemsDropped    atomic.Int64
	URLsEnqueued    atomic.Int64
	URLsFiltered    atomic.Int64
	BytesDownloaded atomic.Int64
	ActiveWorkers   atomic.Int32
	StartTime       time.Time
	mu              sync.RWMutex
	domainStats     map[string]*DomainStats
}

// DomainStats tracks per-domain statistics.
type DomainStats struct {
	Requests  int64
	Responses int64
	Errors    int64
	LastFetch time.Time
}

// Snapshot returns a copy of stats safe for reading.
func (s *Stats) Snapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"requests_sent":    s.RequestsSent.Load(),
		"requests_failed":  s.RequestsFailed.Load(),
		"responses_ok":     s.ResponsesOK.Load(),
		"responses_error":  s.ResponsesError.Load(),
		"items_scraped":    s.ItemsScraped.Load(),
		"items_dropped":    s.ItemsDropped.Load(),
		"urls_enqueued":    s.URLsEnqueued.Load(),
		"urls_filtered":    s.URLsFiltered.Load(),
		"bytes_downloaded": s.BytesDownloaded.Load(),
		"active_workers":   s.ActiveWorkers.Load(),
		"elapsed":          time.Since(s.StartTime).String(),
	}
}

// Fetcher is the interface for all fetcher implementations.
type Fetcher interface {
	Fetch(ctx context.Context, req *types.Request) (*types.Response, error)
	Close() error
}

// Parser is the interface for all parser implementations.
type Parser interface {
	Parse(resp *types.Response, rules []config.ParseRule) ([]*types.Item, []string, error)
}

// Pipeline is the interface for the item processing pipeline.
type Pipeline interface {
	Process(item *types.Item) (*types.Item, error)
}

// Storage is the interface for all storage backends.
type Storage interface {
	Store(items []*types.Item) error
	Close() error
}

// ResponseCallback is a function called when a response is received.
type ResponseCallback func(resp *types.Response) ([]*types.Item, []*types.Request, error)

// Engine is the core crawler orchestrator.
type Engine struct {
	cfg        *config.Config
	logger     *slog.Logger
	frontier   *Frontier
	dedup      *Deduplicator
	robots     *RobotsManager
	checkpoint *CheckpointManager
	scheduler  *Scheduler
	fetchers   map[string]Fetcher
	parser     Parser
	pipeline   Pipeline
	storage    Storage

	state      atomic.Int32
	stats      *Stats
	callbacks  map[string]ResponseCallback
	itemChan   chan *types.Item
	resultChan chan *types.Item
	errChan    chan error

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// New creates a new Engine with the given configuration.
func New(cfg *config.Config, logger *slog.Logger) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	e := &Engine{
		cfg:        cfg,
		logger:     logger,
		frontier:   NewFrontier(),
		dedup:      NewDeduplicator(1_000_000),
		robots:     NewRobotsManager(cfg.Engine.RespectRobotsTxt),
		checkpoint: NewCheckpointManager(cfg.Engine.CheckpointInterval),
		fetchers:   make(map[string]Fetcher),
		callbacks:  make(map[string]ResponseCallback),
		itemChan:   make(chan *types.Item, cfg.Engine.Concurrency*10),
		resultChan: make(chan *types.Item, cfg.Engine.Concurrency*10),
		errChan:    make(chan error, cfg.Engine.Concurrency*10),
		stats: &Stats{
			domainStats: make(map[string]*DomainStats),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	e.scheduler = NewScheduler(e)
	return e
}

// SetFetcher registers a fetcher for a given type.
func (e *Engine) SetFetcher(fetcherType string, f Fetcher) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fetchers[fetcherType] = f
}

// SetParser sets the parser implementation.
func (e *Engine) SetParser(p Parser) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.parser = p
}

// SetPipeline sets the pipeline implementation.
func (e *Engine) SetPipeline(p Pipeline) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.pipeline = p
}

// SetStorage sets the storage implementation.
func (e *Engine) SetStorage(s Storage) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.storage = s
}

// OnResponse registers a named callback for response processing.
func (e *Engine) OnResponse(name string, cb ResponseCallback) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.callbacks[name] = cb
}

// AddSeed adds a seed URL to the crawl frontier.
func (e *Engine) AddSeed(rawURL string) error {
	req, err := types.NewRequest(rawURL)
	if err != nil {
		return err
	}
	req.Priority = types.PriorityHighest
	req.Depth = 0
	return e.AddRequest(req)
}

// AddRequest adds a request to the crawl frontier.
func (e *Engine) AddRequest(req *types.Request) error {
	urlStr := req.URLString()

	// Check depth (seed pages are depth 0; discovered links are depth 1+)
	if req.Depth > e.cfg.Engine.MaxDepth {
		e.stats.URLsFiltered.Add(1)
		return types.ErrMaxDepth
	}

	// Check dedup
	if e.dedup.IsSeen(urlStr) {
		e.stats.URLsFiltered.Add(1)
		return types.ErrDuplicate
	}

	// Check robots.txt
	if e.cfg.Engine.RespectRobotsTxt && !e.robots.IsAllowed(urlStr) {
		e.stats.URLsFiltered.Add(1)
		return types.ErrBlocked
	}

	// Check domain filters
	if !e.isDomainAllowed(req.Domain()) {
		e.stats.URLsFiltered.Add(1)
		return fmt.Errorf("domain %q is not allowed", req.Domain())
	}

	e.dedup.MarkSeen(urlStr)
	e.frontier.Push(req)
	e.stats.URLsEnqueued.Add(1)
	return nil
}

// Start begins crawling.
func (e *Engine) Start() error {
	if !e.state.CompareAndSwap(int32(StateIdle), int32(StateRunning)) {
		return fmt.Errorf("engine is in state %s, cannot start", State(e.state.Load()))
	}

	e.logger.Info("engine starting",
		"concurrency", e.cfg.Engine.Concurrency,
		"max_depth", e.cfg.Engine.MaxDepth,
		"respect_robots", e.cfg.Engine.RespectRobotsTxt,
	)

	e.stats.StartTime = time.Now()

	// Start item pipeline processor
	e.wg.Add(1)
	go e.processItems()

	// Start result storage processor
	e.wg.Add(1)
	go e.storeResults()

	// Start checkpoint auto-save
	if e.cfg.Engine.CheckpointInterval > 0 {
		e.wg.Add(1)
		go e.autoCheckpoint()
	}

	// Start scheduler (worker pool)
	e.scheduler.Start(e.ctx)

	return nil
}

// Wait blocks until all work is done.
func (e *Engine) Wait() {
	e.scheduler.Wait()

	// Cancel context to stop checkpoint goroutine and other background tasks
	e.cancel()

	// Signal processors to stop
	close(e.itemChan)
	close(e.errChan)

	e.wg.Wait()
	e.state.Store(int32(StateStopped))

	// Close fetchers
	e.mu.RLock()
	for _, f := range e.fetchers {
		if err := f.Close(); err != nil {
			e.logger.Error("fetcher close error", "error", err)
		}
	}
	e.mu.RUnlock()

	e.logger.Info("engine stopped", "stats", e.stats.Snapshot())
}

// Stop gracefully stops the engine.
func (e *Engine) Stop() {
	if !e.state.CompareAndSwap(int32(StateRunning), int32(StateStopping)) {
		return
	}
	e.logger.Info("engine stopping...")
	// Close frontier first so all workers polling TryPop() see IsClosed() and exit
	e.frontier.Close()
	e.cancel()
}

// Pause pauses the engine.
func (e *Engine) Pause() {
	if e.state.CompareAndSwap(int32(StateRunning), int32(StatePaused)) {
		e.logger.Info("engine paused")
		e.scheduler.Pause()
	}
}

// Resume resumes a paused engine.
func (e *Engine) Resume() {
	if e.state.CompareAndSwap(int32(StatePaused), int32(StateRunning)) {
		e.logger.Info("engine resumed")
		e.scheduler.Resume()
	}
}

// Stats returns the current crawl statistics.
func (e *Engine) Stats() *Stats {
	return e.stats
}

// State returns the current engine state.
func (e *Engine) GetState() State {
	return State(e.state.Load())
}

// ResultsChan returns a channel for streaming scraped items.
func (e *Engine) ResultsChan() <-chan *types.Item {
	return e.resultChan
}

// isDomainAllowed checks domain allow/disallow lists.
func (e *Engine) isDomainAllowed(domain string) bool {
	// If allowed domains are set, domain must be in the list
	if len(e.cfg.Engine.AllowedDomains) > 0 {
		for _, d := range e.cfg.Engine.AllowedDomains {
			if d == domain {
				return true
			}
		}
		return false
	}

	// Check disallowed domains
	for _, d := range e.cfg.Engine.DisallowedDomains {
		if d == domain {
			return false
		}
	}
	return true
}

// processItems runs the pipeline on scraped items.
func (e *Engine) processItems() {
	defer e.wg.Done()
	for item := range e.itemChan {
		if e.pipeline != nil {
			processed, err := e.pipeline.Process(item)
			if err != nil {
				e.stats.ItemsDropped.Add(1)
				e.logger.Warn("pipeline dropped item", "url", item.URL, "error", err)
				continue
			}
			item = processed
		}
		e.stats.ItemsScraped.Add(1)
		e.resultChan <- item
	}
	close(e.resultChan)
}

// storeResults persists items from the result channel.
func (e *Engine) storeResults() {
	defer e.wg.Done()
	batch := make([]*types.Item, 0, e.cfg.Storage.BatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if e.storage != nil {
			if err := e.storage.Store(batch); err != nil {
				e.logger.Error("storage error", "error", err, "batch_size", len(batch))
			}
		}
		batch = batch[:0]
	}

	for item := range e.resultChan {
		batch = append(batch, item)
		if len(batch) >= e.cfg.Storage.BatchSize {
			flush()
		}
	}
	flush()

	if e.storage != nil {
		if err := e.storage.Close(); err != nil {
			e.logger.Error("storage close error", "error", err)
		}
	}
}

// autoCheckpoint periodically saves engine state.
func (e *Engine) autoCheckpoint() {
	defer e.wg.Done()
	ticker := time.NewTicker(e.cfg.Engine.CheckpointInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			// Save final checkpoint on shutdown
			if err := e.checkpoint.Save(e.frontier, e.dedup, e.stats); err != nil {
				e.logger.Error("final checkpoint save failed", "error", err)
			}
			return
		case <-ticker.C:
			if err := e.checkpoint.Save(e.frontier, e.dedup, e.stats); err != nil {
				e.logger.Error("checkpoint save failed", "error", err)
			} else {
				e.logger.Debug("checkpoint saved")
			}
		}
	}
}

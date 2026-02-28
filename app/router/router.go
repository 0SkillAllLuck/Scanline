package router

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/0skillallluck/scanline/internal/signals"
)

var logger = slog.With("module", "router")

// Router encapsulates navigation state, history, and signals.
type Router struct {
	history *History
	appCtx  any
	ctx     context.Context
	wg      sync.WaitGroup

	NavigationStarted   *signals.StatelessSignal[string]
	NavigationCompleted *signals.StatelessSignal[HistoryEntry]
	HistoryUpdated      *signals.StatelessSignal[*History]
}

// New creates a new Router instance.
func New() *Router {
	r := &Router{
		history: &History{
			RWMutex: sync.RWMutex{},
			Current: nil,
			Entries: []*HistoryEntry{},
		},
		ctx:                 context.Background(),
		NavigationStarted:   signals.NewStatelessSignal[string](),
		NavigationCompleted: signals.NewStatelessSignal[HistoryEntry](),
		HistoryUpdated:      signals.NewStatelessSignal[*History](),
	}
	return r
}

// SetContext stores the application context for route handler dispatch
// and the context.Context for cancelling in-flight navigations.
func (r *Router) SetContext(ctx any, c context.Context) {
	r.appCtx = ctx
	r.ctx = c
}

// Wait blocks until all in-flight navigations complete.
func (r *Router) Wait() {
	r.wg.Wait()
}

func (r *Router) Navigate(path string) {
	r.navigate(strings.TrimPrefix(path, "plex://"), false)
}

func (r *Router) NavigateClearing(path string) {
	r.history.Clear(r.HistoryUpdated)
	r.navigate(strings.TrimPrefix(path, "plex://"), true)
}

func (r *Router) navigate(path string, offRecord bool) {
	if r.history.IsCurrentlyOn(path) && !offRecord {
		logger.Debug("skipped navigation as we are already on the same page")
		return
	}

	logger.Debug("navigation started")
	r.NavigationStarted.Notify(path)

	handler := findHandler(path, r.appCtx)
	if handler == nil {
		logger.Info("no handler found", "path", path)
		handler = notFoundHandler
	}

	startTime := time.Now()
	ctx := r.ctx
	logger.Debug("executing route handler", "path", path, "started_at", startTime)
	r.wg.Add(1)
	go func(ctx context.Context, path string, handler Handler) {
		defer r.wg.Done()
		response, shouldCache := executeHandler(handler)
		if ctx.Err() != nil {
			logger.Debug("navigation cancelled", "path", path)
			return
		}
		logger.Info("navigation completed", "path", path, "duration_ms", time.Since(startTime).Milliseconds(), "should_cache", shouldCache)
		entry := &HistoryEntry{Path: path, PageTitle: response.PageTitle, ExpiresAt: response.ExpiresAt}
		if response.Toolbar != nil {
			entry.Toolbar = response.Toolbar.ToGTK()
		}
		if response.View != nil {
			entry.View = response.View.ToGTK()
		}
		if !shouldCache {
			entry.ExpiresAt = new(time.Now())
		}
		if !offRecord {
			r.history.Push(entry, r.HistoryUpdated)
		}
		r.NavigationCompleted.Notify(*entry)
	}(ctx, path, handler)
}

func (r *Router) Back() {
	if r.history.EntriesCount() == 0 {
		return
	}

	previous := r.history.Pop(r.HistoryUpdated)
	if previous == nil {
		return
	}

	if previous.View != nil {
		r.NavigationStarted.Notify(previous.Path)
		r.NavigationCompleted.Notify(*previous)
	} else {
		r.navigate(previous.Path, true)
	}
}

func (r *Router) Refresh() {
	path, ok := r.history.InvalidateCurrent()
	if !ok {
		return
	}

	r.navigate(path, true)
}

func (r *Router) Current() *HistoryEntry {
	return r.history.GetCurrent()
}

// --- Package-level default router and convenience functions ---

var defaultRouter = New()

// Default returns the package-level default Router instance.
func Default() *Router {
	return defaultRouter
}

// SetContext configures the default router's context.
func SetContext(ctx any, c context.Context) {
	defaultRouter.SetContext(ctx, c)
}

// Wait blocks until the default router's in-flight navigations complete.
func Wait() {
	defaultRouter.Wait()
}

// Navigate on the default router.
func Navigate(path string) {
	defaultRouter.Navigate(path)
}

// NavigateClearing on the default router.
func NavigateClearing(path string) {
	defaultRouter.NavigateClearing(path)
}

// Back on the default router.
func Back() {
	defaultRouter.Back()
}

// Refresh on the default router.
func Refresh() {
	defaultRouter.Refresh()
}

// Current on the default router.
func Current() *HistoryEntry {
	return defaultRouter.Current()
}

// Package-level signal accessors for the default router.
var (
	NavigationStarted   = defaultRouter.NavigationStarted
	NavigationCompleted = defaultRouter.NavigationCompleted
	HistoryUpdated      = defaultRouter.HistoryUpdated
)

package intelcmd

import (
	"context"
	"sync"
	"time"
)

const (
	DefaultShortcutParallelism = 3
	DefaultShortcutTimeout     = 120 * time.Second
)

// WithShortcutBudget wraps parent with an overall deadline for shortcut orchestration.
func WithShortcutBudget(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, DefaultShortcutTimeout)
}

// RunParallel runs tasks with at most limit concurrent goroutines. The first error is returned.
func RunParallel(limit int, tasks []func() error) error {
	if len(tasks) == 0 {
		return nil
	}
	if limit <= 0 || limit > len(tasks) {
		limit = len(tasks)
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for _, task := range tasks {
		task := task
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := task(); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

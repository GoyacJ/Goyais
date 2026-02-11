package workflow

import (
	"context"
	"log"
	"sync"
	"time"
)

type StepWorkerPool struct {
	repo        Repository
	workers     int
	idleBackoff time.Duration
	logger      *log.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewStepWorkerPool(repo Repository, workers int, idleBackoff time.Duration, logger *log.Logger) *StepWorkerPool {
	if workers <= 0 {
		workers = 1
	}
	if idleBackoff <= 0 {
		idleBackoff = 100 * time.Millisecond
	}
	if logger == nil {
		logger = log.Default()
	}
	return &StepWorkerPool{
		repo:        repo,
		workers:     workers,
		idleBackoff: idleBackoff,
		logger:      logger,
	}
}

func (p *StepWorkerPool) Start(parent context.Context) {
	if p == nil || p.repo == nil {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	p.cancel = cancel

	for i := 0; i < p.workers; i++ {
		workerID := "workflow-worker-" + string(rune('a'+i))
		p.wg.Add(1)
		go func(id string) {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				worked, err := p.repo.ProcessStepQueueOnce(ctx, id, time.Now().UTC())
				if err != nil {
					p.logger.Printf("workflow step worker %s: process queue: %v", id, err)
					select {
					case <-ctx.Done():
						return
					case <-time.After(p.idleBackoff):
						continue
					}
				}
				if worked {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(p.idleBackoff):
				}
			}
		}(workerID)
	}
}

func (p *StepWorkerPool) Stop() {
	if p == nil {
		return
	}
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

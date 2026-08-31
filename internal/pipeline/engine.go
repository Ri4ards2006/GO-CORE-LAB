// ═══════════════════════════════════════════════════════════════════════════
// Package pipeline implements the concurrent worker-pool execution engine.
// ═══════════════════════════════════════════════════
package pipeline

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/Ri4ards2006/go-core-lab/internal/hw"
	gonet "github.com/Ri4ards2006/go-core-lab/internal/net"
)

// EngineConfig configures the concurrency and buffering of the pipeline engine.
type EngineConfig struct {
	NumWorkers         int  // Number of parallel dissection workers
	QueueSize          int  // Ingestion buffer queue depth
	RingBufferSize     int  // In-memory ring buffer capacity
	DropOnBackpressure bool // Non-blocking ingestion if true; backpressure if false
}

// DefaultEngineConfig returns optimal defaults based on host CPU count.
func DefaultEngineConfig() EngineConfig {
	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	return EngineConfig{
		NumWorkers:         workers,
		QueueSize:          2048,
		RingBufferSize:     512,
		DropOnBackpressure: false,
	}
}

// PipelineEngine orchestrates the ingestion source, worker pool, and output sinks.
type PipelineEngine struct {
	Config     EngineConfig
	Source     Source
	Sinks      []Sink
	Stats      *StatsSink
	RingBuffer *RingBufferSink

	inChan chan IngestionEvent
	wg     sync.WaitGroup
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPipelineEngine constructs a new pipeline engine with default sinks.
func NewPipelineEngine(cfg EngineConfig, src Source) *PipelineEngine {
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = runtime.NumCPU()
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 1024
	}
	if cfg.RingBufferSize <= 0 {
		cfg.RingBufferSize = 256
	}

	stats := NewStatsSink()
	ring := NewRingBufferSink(cfg.RingBufferSize)

	pe := &PipelineEngine{
		Config:     cfg,
		Source:     src,
		Stats:      stats,
		RingBuffer: ring,
		Sinks:      []Sink{stats, ring},
		inChan:     make(chan IngestionEvent, cfg.QueueSize),
	}

	return pe
}

// AddSink registers an additional output sink.
func (pe *PipelineEngine) AddSink(sink Sink) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.Sinks = append(pe.Sinks, sink)
}

// Start spawns the worker pool and starts ingesting from the source.
func (pe *PipelineEngine) Start(parentCtx context.Context) error {
	pe.mu.Lock()
	pe.ctx, pe.cancel = context.WithCancel(parentCtx)
	pe.mu.Unlock()

	// 1. Start Source Ingestion
	srcChan, err := pe.Source.Start(pe.ctx)
	if err != nil {
		return fmt.Errorf("start source: %w", err)
	}

	// 2. Launch Worker Pool
	for i := 0; i < pe.Config.NumWorkers; i++ {
		pe.wg.Add(1)
		go pe.workerLoop(i)
	}

	// 3. Launch Ingestion Dispatcher
	go pe.ingestionLoop(srcChan)

	return nil
}

// ingestionLoop reads from the source channel and pushes to the internal worker channel.
func (pe *PipelineEngine) ingestionLoop(srcChan <-chan IngestionEvent) {
	defer close(pe.inChan)

	for {
		select {
		case <-pe.ctx.Done():
			return
		case event, ok := <-srcChan:
			if !ok {
				return
			}

			if pe.Config.DropOnBackpressure {
				select {
				case pe.inChan <- event:
				default:
					pe.Stats.RecordDrop()
				}
			} else {
				select {
				case pe.inChan <- event:
				case <-pe.ctx.Done():
					return
				}
			}
		}
	}
}

// workerLoop pulls raw events from inChan, runs protocol dissectors, and broadcasts to sinks.
func (pe *PipelineEngine) workerLoop(workerID int) {
	defer pe.wg.Done()

	for event := range pe.inChan {
		t0 := time.Now()
		pEvent := PipelineEvent{
			ID:        event.ID,
			Type:      event.Type,
			Timestamp: event.Timestamp,
			Raw:       event.Data,
			WorkerID:  workerID,
		}

		// Perform dissection according to event type
		switch event.Type {
		case EventNetwork:
			pkt, err := gonet.DissectWithTimestamp(event.Data, event.Timestamp)
			pEvent.Packet = pkt
			pEvent.Error = err

		case EventSerial:
			pEvent.Frame = &hw.Frame{
				Timestamp: event.Timestamp,
				Payload:   event.Data,
				Valid:     true,
			}

		case EventRaw:
			// Raw unparsed bytes
		}

		pEvent.Latency = time.Since(t0)

		// Broadcast to all registered sinks
		pe.broadcastToSinks(pEvent)
	}
}

// broadcastToSinks sends the processed event to all sinks.
func (pe *PipelineEngine) broadcastToSinks(event PipelineEvent) {
	pe.mu.Lock()
	sinks := pe.Sinks
	pe.mu.Unlock()

	for _, sink := range sinks {
		_ = sink.OnEvent(pe.ctx, event)
	}
}

// QueueDepth returns the number of unprocessed events currently in the worker channel.
func (pe *PipelineEngine) QueueDepth() int {
	return len(pe.inChan)
}

// Stop initiates graceful teardown, closes the source, waits for workers to finish, and flushes sinks.
func (pe *PipelineEngine) Stop() error {
	pe.mu.Lock()
	if pe.cancel != nil {
		pe.cancel()
	}
	pe.mu.Unlock()

	_ = pe.Source.Close()
	pe.wg.Wait()

	pe.mu.Lock()
	defer pe.mu.Unlock()

	var firstErr error
	for _, sink := range pe.Sinks {
		if err := sink.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}


package pipeline

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRingBufferBasic(t *testing.T) {
	rb := NewRingBufferSink(3)

	if rb.Capacity() != 3 {
		t.Fatalf("expected capacity 3, got %d", rb.Capacity())
	}
	if rb.Len() != 0 {
		t.Fatalf("expected len 0, got %d", rb.Len())
	}

	ctx := context.Background()

	// Push 1
	_ = rb.OnEvent(ctx, PipelineEvent{ID: 1})
	if rb.Len() != 1 {
		t.Errorf("expected len 1, got %d", rb.Len())
	}

	// Push 2, 3
	_ = rb.OnEvent(ctx, PipelineEvent{ID: 2})
	_ = rb.OnEvent(ctx, PipelineEvent{ID: 3})
	if rb.Len() != 3 {
		t.Errorf("expected len 3, got %d", rb.Len())
	}

	last3 := rb.Last(3)
	if len(last3) != 3 || last3[0].ID != 1 || last3[1].ID != 2 || last3[2].ID != 3 {
		t.Errorf("unexpected last 3 items: %v", last3)
	}

	// Push 4 (should wrap around and overwrite 1)
	_ = rb.OnEvent(ctx, PipelineEvent{ID: 4})
	if rb.Len() != 3 {
		t.Errorf("expected len to stay at 3, got %d", rb.Len())
	}

	lastAfterWrap := rb.Last(3)
	if lastAfterWrap[0].ID != 2 || lastAfterWrap[1].ID != 3 || lastAfterWrap[2].ID != 4 {
		t.Errorf("unexpected items after wrap-around: %v", lastAfterWrap)
	}

	// Request more than count
	last5 := rb.Last(5)
	if len(last5) != 3 {
		t.Errorf("expected 3 items when requesting 5, got %d", len(last5))
	}
}

func TestRingBufferConcurrent(t *testing.T) {
	rb := NewRingBufferSink(100)
	ctx := context.Background()

	var wg sync.WaitGroup
	numWriters := 10
	itemsPerWriter := 500

	// Concurrent Writers
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < itemsPerWriter; i++ {
				_ = rb.OnEvent(ctx, PipelineEvent{
					ID:        uint64(writerID*itemsPerWriter + i),
					Timestamp: time.Now(),
				})
			}
		}(w)
	}

	// Concurrent Readers
	stopReaders := make(chan struct{})
	var readerWg sync.WaitGroup
	for r := 0; r < 4; r++ {
		readerWg.Add(1)
		go func() {
			defer readerWg.Done()
			for {
				select {
				case <-stopReaders:
					return
				default:
					_ = rb.Last(10)
					_ = rb.Len()
				}
			}
		}()
	}

	wg.Wait()
	close(stopReaders)
	readerWg.Wait()

	if rb.Len() != 100 {
		t.Errorf("expected full buffer (100 items), got %d", rb.Len())
	}
}

func TestEventSummary(t *testing.T) {
	errEvent := PipelineEvent{
		Timestamp: time.Now(),
		Error:     fmt.Errorf("checksum mismatch"),
	}
	if !containsStr(errEvent.Summary(), "checksum mismatch") {
		t.Errorf("expected error in summary, got %q", errEvent.Summary())
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && bytesContains(s, substr))
}

func bytesContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}


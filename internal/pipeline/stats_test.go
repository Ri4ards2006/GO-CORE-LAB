package pipeline

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	gonet "github.com/Ri4ards2006/go-core-lab/internal/net"
)

func TestStatsSinkAtomics(t *testing.T) {
	stats := NewStatsSink()
	ctx := context.Background()

	var wg sync.WaitGroup
	workers := 8
	eventsPerWorker := 1000

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < eventsPerWorker; i++ {
				ev := PipelineEvent{
					ID:        uint64(i),
					Type:      EventNetwork,
					Timestamp: time.Now(),
					Latency:   time.Duration(10+i%50) * time.Microsecond,
					Raw:       make([]byte, 64),
					Packet: &gonet.Packet{
						IPv4: &gonet.IPv4Header{
							SrcIP: net.ParseIP("192.168.1.1"),
							DstIP: net.ParseIP("192.168.1.2"),
						},
						TCP: &gonet.TCPHeader{
							SrcPort: 80,
							DstPort: 12345,
						},
					},
				}
				_ = stats.OnEvent(ctx, ev)
			}
		}(w)
	}

	wg.Wait()

	snap := stats.Snapshot()
	expectedTotal := uint64(workers * eventsPerWorker)

	if snap.TotalProcessed != expectedTotal {
		t.Errorf("expected %d processed events, got %d", expectedTotal, snap.TotalProcessed)
	}

	if snap.IPv4Count != expectedTotal {
		t.Errorf("expected %d IPv4 packets, got %d", expectedTotal, snap.IPv4Count)
	}

	if snap.TCPCount != expectedTotal {
		t.Errorf("expected %d TCP packets, got %d", expectedTotal, snap.TCPCount)
	}

	if snap.TotalBytes != expectedTotal*64 {
		t.Errorf("expected %d total bytes, got %d", expectedTotal*64, snap.TotalBytes)
	}

	if snap.MinLatency == 0 || snap.MaxLatency == 0 || snap.AvgLatency == 0 {
		t.Errorf("expected valid latency metrics, got min=%v, max=%v, avg=%v",
			snap.MinLatency, snap.MaxLatency, snap.AvgLatency)
	}
}

func TestStatsDrops(t *testing.T) {
	stats := NewStatsSink()
	stats.RecordDrop()
	stats.RecordDrop()

	snap := stats.Snapshot()
	if snap.DroppedEvents != 2 {
		t.Errorf("expected 2 dropped events, got %d", snap.DroppedEvents)
	}
}


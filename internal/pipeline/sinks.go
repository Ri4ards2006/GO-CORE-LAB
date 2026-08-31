// ═══════════════════════════════════════════════════════════════════════════
// Package pipeline implements high-throughput thread-safe sinks:
// PcapSink, RingBufferSink, and atomic StatsSink.
// ═══════════════════════════════════════════════════
package pipeline

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Ri4ards2006/go-core-lab/pkg/export"
)

// ═══════════════════════════════════════════════════
// 1. PCAP SINK
// ═══════════════════════════════════════════════════

// PcapSink writes network events to a .pcap file in real time.
type PcapSink struct {
	mu     sync.Mutex
	file   *os.File
	writer *export.PcapWriter
}

// NewPcapSink initializes a thread-safe PCAP recording sink.
func NewPcapSink(path string) (*PcapSink, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create pcap sink file %q: %w", path, err)
	}

	pw, err := export.NewPcapWriter(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("init pcap writer: %w", err)
	}

	return &PcapSink{
		file:   f,
		writer: pw,
	}, nil
}

// OnEvent records network packets into the PCAP stream.
func (ps *PcapSink) OnEvent(ctx context.Context, event PipelineEvent) error {
	if event.Type != EventNetwork || len(event.Raw) == 0 {
		return nil
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	return ps.writer.WritePacket(event.Raw, event.Timestamp)
}

// Close closes the underlying file handle.
func (ps *PcapSink) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.file != nil {
		err := ps.file.Close()
		ps.file = nil
		return err
	}
	return nil
}

// ═══════════════════════════════════════════════════
// 2. RING BUFFER SINK
// ═══════════════════════════════════════════════════

// RingBufferSink stores the last N processed events in a thread-safe circular buffer.
type RingBufferSink struct {
	mu       sync.RWMutex
	capacity int
	buffer   []PipelineEvent
	head     int // Next write position
	count    int // Total items stored (up to capacity)
}

// NewRingBufferSink creates a ring buffer with a fixed capacity.
func NewRingBufferSink(capacity int) *RingBufferSink {
	if capacity <= 0 {
		capacity = 256
	}
	return &RingBufferSink{
		capacity: capacity,
		buffer:   make([]PipelineEvent, capacity),
	}
}

// OnEvent stores the event, overwriting the oldest event if full.
func (rb *RingBufferSink) OnEvent(ctx context.Context, event PipelineEvent) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.buffer[rb.head] = event
	rb.head = (rb.head + 1) % rb.capacity
	if rb.count < rb.capacity {
		rb.count++
	}

	return nil
}

// Last returns the most recent N events in chronological order.
func (rb *RingBufferSink) Last(n int) []PipelineEvent {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if n <= 0 || rb.count == 0 {
		return nil
	}
	if n > rb.count {
		n = rb.count
	}

	result := make([]PipelineEvent, n)
	start := (rb.head - n + rb.capacity) % rb.capacity

	for i := 0; i < n; i++ {
		idx := (start + i) % rb.capacity
		result[i] = rb.buffer[idx]
	}

	return result
}

// Len returns the current number of stored events.
func (rb *RingBufferSink) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Capacity returns the maximum capacity of the ring buffer.
func (rb *RingBufferSink) Capacity() int {
	return rb.capacity
}

// Close implements Sink.
func (rb *RingBufferSink) Close() error {
	return nil
}

// ═══════════════════════════════════════════════════
// 3. STATS SINK (Lock-Free Atomics)
// ═══════════════════════════════════════════════════

// StatsSink aggregates real-time performance and protocol metrics using lock-free atomics.
type StatsSink struct {
	startTime      time.Time
	lastTime       time.Time
	lastProcessed  uint64
	lastBytes      uint64

	TotalProcessed uint64
	TotalBytes     uint64
	DroppedEvents  uint64
	ErrorEvents    uint64

	// Protocol counters
	IPv4Count   uint64
	IPv6Count   uint64
	ARPCount    uint64
	TCPCount    uint64
	UDPCount    uint64
	ICMPCount   uint64
	SerialCount uint64
	OtherCount  uint64

	// Latency tracking in nanoseconds
	totalLatencyNs uint64
	minLatencyNs   uint64
	maxLatencyNs   uint64
	countLatency   uint64
	mu             sync.Mutex // Protects snapshot calculation
}

// NewStatsSink initializes atomic metrics.
func NewStatsSink() *StatsSink {
	now := time.Now()
	return &StatsSink{
		startTime: now,
		lastTime:  now,
	}
}

// OnEvent increments atomic counters based on the event's protocol.
func (s *StatsSink) OnEvent(ctx context.Context, event PipelineEvent) error {
	atomic.AddUint64(&s.TotalProcessed, 1)
	atomic.AddUint64(&s.TotalBytes, uint64(len(event.Raw)))

	if event.Error != nil {
		atomic.AddUint64(&s.ErrorEvents, 1)
	}

	// Update Latency
	latNs := uint64(event.Latency.Nanoseconds())
	if latNs > 0 {
		atomic.AddUint64(&s.totalLatencyNs, latNs)
		atomic.AddUint64(&s.countLatency, 1)

		// Atomic Min / Max
		for {
			curMin := atomic.LoadUint64(&s.minLatencyNs)
			if curMin != 0 && curMin <= latNs {
				break
			}
			if atomic.CompareAndSwapUint64(&s.minLatencyNs, curMin, latNs) {
				break
			}
		}

		for {
			curMax := atomic.LoadUint64(&s.maxLatencyNs)
			if curMax >= latNs {
				break
			}
			if atomic.CompareAndSwapUint64(&s.maxLatencyNs, curMax, latNs) {
				break
			}
		}
	}

	// Protocol Categorization
	if event.Type == EventSerial {
		atomic.AddUint64(&s.SerialCount, 1)
		return nil
	}

	if event.Packet != nil {
		if event.Packet.IPv4 != nil {
			atomic.AddUint64(&s.IPv4Count, 1)
		} else if event.Packet.IPv6 != nil {
			atomic.AddUint64(&s.IPv6Count, 1)
		} else if event.Packet.ARP != nil {
			atomic.AddUint64(&s.ARPCount, 1)
		}

		if event.Packet.TCP != nil {
			atomic.AddUint64(&s.TCPCount, 1)
		} else if event.Packet.UDP != nil {
			atomic.AddUint64(&s.UDPCount, 1)
		} else if event.Packet.ICMP != nil {
			atomic.AddUint64(&s.ICMPCount, 1)
		} else {
			atomic.AddUint64(&s.OtherCount, 1)
		}
	} else {
		atomic.AddUint64(&s.OtherCount, 1)
	}

	return nil
}

// RecordDrop increments the dropped events counter.
func (s *StatsSink) RecordDrop() {
	atomic.AddUint64(&s.DroppedEvents, 1)
}

// StatsSnapshot represents an instantaneous view of pipeline throughput and totals.
type StatsSnapshot struct {
	Uptime         time.Duration
	TotalProcessed uint64
	TotalBytes     uint64
	DroppedEvents  uint64
	ErrorEvents    uint64

	PacketsPerSec float64
	BytesPerSec   float64

	IPv4Count   uint64
	IPv6Count   uint64
	ARPCount    uint64
	TCPCount    uint64
	UDPCount    uint64
	ICMPCount   uint64
	SerialCount uint64
	OtherCount  uint64

	AvgLatency time.Duration
	MinLatency time.Duration
	MaxLatency time.Duration
}

// Snapshot computes rates and returns a copy of current metrics.
func (s *StatsSink) Snapshot() StatsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	uptime := now.Sub(s.startTime)
	deltaT := now.Sub(s.lastTime).Seconds()

	processed := atomic.LoadUint64(&s.TotalProcessed)
	bytes := atomic.LoadUint64(&s.TotalBytes)

	var pps, bps float64
	if deltaT > 0 {
		pps = float64(processed-s.lastProcessed) / deltaT
		bps = float64(bytes-s.lastBytes) / deltaT
	}

	s.lastTime = now
	s.lastProcessed = processed
	s.lastBytes = bytes

	totLat := atomic.LoadUint64(&s.totalLatencyNs)
	cntLat := atomic.LoadUint64(&s.countLatency)
	minLat := atomic.LoadUint64(&s.minLatencyNs)
	maxLat := atomic.LoadUint64(&s.maxLatencyNs)

	var avgLat time.Duration
	if cntLat > 0 {
		avgLat = time.Duration(totLat / cntLat)
	}

	return StatsSnapshot{
		Uptime:         uptime,
		TotalProcessed: processed,
		TotalBytes:     bytes,
		DroppedEvents:  atomic.LoadUint64(&s.DroppedEvents),
		ErrorEvents:    atomic.LoadUint64(&s.ErrorEvents),
		PacketsPerSec:  pps,
		BytesPerSec:    bps,
		IPv4Count:      atomic.LoadUint64(&s.IPv4Count),
		IPv6Count:      atomic.LoadUint64(&s.IPv6Count),
		ARPCount:       atomic.LoadUint64(&s.ARPCount),
		TCPCount:       atomic.LoadUint64(&s.TCPCount),
		UDPCount:       atomic.LoadUint64(&s.UDPCount),
		ICMPCount:      atomic.LoadUint64(&s.ICMPCount),
		SerialCount:    atomic.LoadUint64(&s.SerialCount),
		OtherCount:     atomic.LoadUint64(&s.OtherCount),
		AvgLatency:     avgLat,
		MinLatency:     time.Duration(minLat),
		MaxLatency:     time.Duration(maxLat),
	}
}

// Close implements Sink.
func (s *StatsSink) Close() error {
	return nil
}

// Ensure interface compliance
var (
	_ Sink = (*PcapSink)(nil)
	_ Sink = (*RingBufferSink)(nil)
	_ Sink = (*StatsSink)(nil)
)

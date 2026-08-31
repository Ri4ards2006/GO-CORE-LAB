// ═══════════════════════════════════════════════════════════════════════════
// Package pipeline implements source adapters for packet capture, PCAP replay,
// and hardware serial telemetry.
// ═══════════════════════════════════════════════════
package pipeline

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/Ri4ards2006/go-core-lab/internal/hw"
	"github.com/Ri4ards2006/go-core-lab/pkg/export"
)

// PcapReplaySource streams packets from a .pcap file into the pipeline.
type PcapReplaySource struct {
	Path     string
	Throttle bool
	file     *os.File
	counter  uint64
}

// NewPcapReplaySource creates a PCAP file replay source.
func NewPcapReplaySource(path string, throttle bool) *PcapReplaySource {
	return &PcapReplaySource{
		Path:     path,
		Throttle: throttle,
	}
}

// Start begins replaying packets into an IngestionEvent channel.
func (p *PcapReplaySource) Start(ctx context.Context) (<-chan IngestionEvent, error) {
	reader, f, err := export.OpenPcap(p.Path)
	if err != nil {
		return nil, fmt.Errorf("open pcap source %q: %w", p.Path, err)
	}
	p.file = f

	outChan := make(chan IngestionEvent, 128)

	go func() {
		defer close(outChan)
		defer f.Close()

		var lastPktTime time.Time

		for {
			select {
			case <-ctx.Done():
				return
			default:
				data, ts, err := reader.NextPacket()
				if err != nil {
					// Reached end of PCAP
					return
				}

				if p.Throttle && !lastPktTime.IsZero() {
					delta := ts.Sub(lastPktTime)
					if delta > 0 && delta < 500*time.Millisecond {
						time.Sleep(delta)
					}
				}
				lastPktTime = ts

				id := atomic.AddUint64(&p.counter, 1)
				event := IngestionEvent{
					ID:        id,
					Type:      EventNetwork,
					Timestamp: ts,
					Data:      data,
					Source:    p.Path,
				}

				select {
				case outChan <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outChan, nil
}

// Close closes the underlying PCAP file.
func (p *PcapReplaySource) Close() error {
	if p.file != nil {
		err := p.file.Close()
		p.file = nil
		return err
	}
	return nil
}

// SerialSource streams frames from a physical or simulated UART serial device.
type SerialSource struct {
	Config  hw.FrameConfig
	monitor *hw.SerialMonitor
	counter uint64
}

// NewSerialSource creates a new UART serial hardware source.
func NewSerialSource(cfg hw.FrameConfig) *SerialSource {
	return &SerialSource{
		Config:  cfg,
		monitor: hw.NewSerialMonitor(cfg),
	}
}

// Start begins reading serial frames and feeding them into the pipeline.
func (s *SerialSource) Start(ctx context.Context) (<-chan IngestionEvent, error) {
	frameChan, err := s.monitor.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("start serial source: %w", err)
	}

	outChan := make(chan IngestionEvent, 64)

	go func() {
		defer close(outChan)

		for frame := range frameChan {
			id := atomic.AddUint64(&s.counter, 1)
			event := IngestionEvent{
				ID:        id,
				Type:      EventSerial,
				Timestamp: frame.Timestamp,
				Data:      frame.Payload,
				Source:    s.Config.Device,
				Metadata: map[string]string{
					"seq": fmt.Sprintf("%d", frame.SeqNum),
				},
			}

			select {
			case outChan <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return outChan, nil
}

// Close terminates the serial stream.
func (s *SerialSource) Close() error {
	return s.monitor.Close()
}

// SliceSource is an in-memory source providing a predefined list of byte slices (useful for testing).
type SliceSource struct {
	Payloads  [][]byte
	EventType EventType
	counter   uint64
}

// NewSliceSource creates a test source from a slice of byte slices.
func NewSliceSource(payloads [][]byte, eventType EventType) *SliceSource {
	return &SliceSource{
		Payloads:  payloads,
		EventType: eventType,
	}
}

// Start streams the static slices into the channel.
func (s *SliceSource) Start(ctx context.Context) (<-chan IngestionEvent, error) {
	outChan := make(chan IngestionEvent, len(s.Payloads)+1)

	go func() {
		defer close(outChan)

		for _, data := range s.Payloads {
			id := atomic.AddUint64(&s.counter, 1)
			event := IngestionEvent{
				ID:        id,
				Type:      s.EventType,
				Timestamp: time.Now(),
				Data:      data,
				Source:    "slice-source",
			}

			select {
			case outChan <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return outChan, nil
}

// Close implements Source.
func (s *SliceSource) Close() error {
	return nil
}

// Ensure interface compliance
var (
	_ Source = (*PcapReplaySource)(nil)
	_ Source = (*SerialSource)(nil)
	_ Source = (*SliceSource)(nil)
)

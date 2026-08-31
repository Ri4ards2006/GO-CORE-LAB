// ═══════════════════════════════════════════════════════════════════════════
// Package pipeline implements the real-time concurrent worker pool and
// streaming event pipeline for GO-CORE-LAB.
// ═══════════════════════════════════════════════════
package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/Ri4ards2006/go-core-lab/internal/hw"
	gonet "github.com/Ri4ards2006/go-core-lab/internal/net"
)

// EventType categorizes the raw payload source.
type EventType int

const (
	EventNetwork EventType = iota // Raw Ethernet / IP packet
	EventSerial                   // UART / Serial hardware telemetry frame
	EventRaw                      // Generic raw byte stream
)

// String returns the name of the EventType.
func (e EventType) String() string {
	switch e {
	case EventNetwork:
		return "NETWORK"
	case EventSerial:
		return "SERIAL"
	case EventRaw:
		return "RAW"
	default:
		return "UNKNOWN"
	}
}

// IngestionEvent represents an unprocessed unit of data entering the pipeline.
type IngestionEvent struct {
	ID        uint64
	Type      EventType
	Timestamp time.Time
	Data      []byte
	Source    string
	Metadata  map[string]string
}

// PipelineEvent represents an event after worker dissection and enrichment.
type PipelineEvent struct {
	ID         uint64
	Type       EventType
	Timestamp  time.Time
	Latency    time.Duration
	Raw        []byte
	Packet     *gonet.Packet
	Frame      *hw.Frame
	Error      error
	WorkerID   int
	Attributes map[string]string
}

// Summary returns a single-line summary of the processed event.
func (pe PipelineEvent) Summary() string {
	if pe.Error != nil {
		return fmt.Sprintf("[%s] Error: %v", pe.Timestamp.Format("15:04:05.000000"), pe.Error)
	}

	if pe.Packet != nil {
		return pe.Packet.Summary()
	}

	if pe.Frame != nil {
		return pe.Frame.String()
	}

	return fmt.Sprintf("[%s] Raw Event #%d: len=%d bytes (from %s)",
		pe.Timestamp.Format("15:04:05.000000"), pe.ID, len(pe.Raw), pe.Type)
}

// Source produces a stream of IngestionEvents from a data provider.
type Source interface {
	Start(ctx context.Context) (<-chan IngestionEvent, error)
	Close() error
}

// Sink consumes processed PipelineEvents (e.g. storage, metrics, live UI).
type Sink interface {
	OnEvent(ctx context.Context, event PipelineEvent) error
	Close() error
}


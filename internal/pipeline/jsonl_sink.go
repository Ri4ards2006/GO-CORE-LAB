// ═══════════════════════════════════════════════════════════════════════════
// Package pipeline implements the streaming JsonlSink for real-time telemetry.
// ═══════════════════════════════════════════════════
package pipeline

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	gonet "github.com/Ri4ards2006/go-core-lab/internal/net"
	"github.com/Ri4ards2006/go-core-lab/pkg/export"
)

// JsonlSink writes streaming NDJSON records to an output file.
type JsonlSink struct {
	mu     sync.Mutex
	file   *os.File
	writer *export.JsonlWriter
}

// NewJsonlSink initializes a thread-safe JSONL streaming sink.
func NewJsonlSink(path string) (*JsonlSink, error) {
	writer, f, err := export.OpenJsonlFile(path)
	if err != nil {
		return nil, err
	}

	return &JsonlSink{
		file:   f,
		writer: writer,
	}, nil
}

// OnEvent serializes a PipelineEvent and flushes it to the buffered NDJSON stream.
func (js *JsonlSink) OnEvent(ctx context.Context, event PipelineEvent) error {
	rec := export.JSONEventRecord{
		ID:        event.ID,
		Type:      event.Type.String(),
		Timestamp: event.Timestamp.Format(time.RFC3339Nano),
		LatencyNs: event.Latency.Nanoseconds(),
		WorkerID:  event.WorkerID,
	}

	if event.Error != nil {
		rec.Error = event.Error.Error()
	}

	if event.Packet != nil {
		pRec := &export.JSONPacketRecord{
			PayloadLen: len(event.Packet.Payload),
		}
		if len(event.Packet.Payload) > 0 && len(event.Packet.Payload) <= 128 {
			pRec.PayloadHex = hex.EncodeToString(event.Packet.Payload)
		}

		if event.Packet.Ethernet != nil {
			pRec.SrcMAC = event.Packet.Ethernet.SrcMAC.String()
			pRec.DstMAC = event.Packet.Ethernet.DstMAC.String()
			pRec.EtherType = fmt.Sprintf("0x%04x", event.Packet.Ethernet.EtherType)
		}

		if event.Packet.IPv4 != nil {
			pRec.Protocol = gonet.IPProtocolName(event.Packet.IPv4.Protocol)
			pRec.SrcIP = event.Packet.IPv4.SrcIP.String()
			pRec.DstIP = event.Packet.IPv4.DstIP.String()
			pRec.TTL = event.Packet.IPv4.TTL
		} else if event.Packet.IPv6 != nil {
			pRec.Protocol = gonet.IPProtocolName(event.Packet.IPv6.NextHeader)
			pRec.SrcIP = event.Packet.IPv6.SrcIP.String()
			pRec.DstIP = event.Packet.IPv6.DstIP.String()
			pRec.TTL = event.Packet.IPv6.HopLimit
		} else if event.Packet.ARP != nil {
			pRec.Protocol = "ARP"
			pRec.SrcIP = event.Packet.ARP.SenderIP.String()
			pRec.DstIP = event.Packet.ARP.TargetIP.String()
		}

		if event.Packet.TCP != nil {
			pRec.SrcPort = event.Packet.TCP.SrcPort
			pRec.DstPort = event.Packet.TCP.DstPort
			pRec.TCPFlags = event.Packet.TCP.FlagsString()
		} else if event.Packet.UDP != nil {
			pRec.SrcPort = event.Packet.UDP.SrcPort
			pRec.DstPort = event.Packet.UDP.DstPort
		}

		rec.Packet = pRec
	}

	if event.Frame != nil {
		rec.Frame = &export.JSONFrameRecord{
			SeqNum:  uint64(event.Frame.SeqNum),
			Payload: string(event.Frame.Payload),
		}
	}

	return js.writer.WriteRecord(rec)
}

// Close flushes the buffer and closes the underlying file handle.
func (js *JsonlSink) Close() error {
	js.mu.Lock()
	defer js.mu.Unlock()

	var flushErr error
	if js.writer != nil {
		flushErr = js.writer.Flush()
	}

	var fileErr error
	if js.file != nil {
		fileErr = js.file.Close()
		js.file = nil
	}

	if flushErr != nil {
		return flushErr
	}
	return fileErr
}

// Ensure interface compliance
var _ Sink = (*JsonlSink)(nil)


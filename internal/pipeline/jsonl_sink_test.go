package pipeline

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ri4ards2006/go-core-lab/internal/hw"
	gonet "github.com/Ri4ards2006/go-core-lab/internal/net"
	"github.com/Ri4ards2006/go-core-lab/pkg/export"
)

func TestJsonlSink(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "events.jsonl")

	sink, err := NewJsonlSink(outPath)
	if err != nil {
		t.Fatalf("unexpected error creating JSONL sink: %v", err)
	}

	ctx := context.Background()

	// 1. Network Event
	netEvent := PipelineEvent{
		ID:        1,
		Type:      EventNetwork,
		Timestamp: time.Now(),
		Latency:   15 * time.Microsecond,
		WorkerID:  0,
		Packet: &gonet.Packet{
			IPv4: &gonet.IPv4Header{
				SrcIP:    net.ParseIP("192.168.1.10"),
				DstIP:    net.ParseIP("1.1.1.1"),
				Protocol: gonet.IPProtoTCP,
				TTL:      64,
			},
			TCP: &gonet.TCPHeader{
				SrcPort: 54321,
				DstPort: 443,
				Flags:   gonet.TCPFlagSYN,
			},
			Payload: []byte("GET / HTTP/1.1\r\n"),
		},
	}

	if err := sink.OnEvent(ctx, netEvent); err != nil {
		t.Fatalf("error writing net event: %v", err)
	}

	// 2. Serial Event
	serialEvent := PipelineEvent{
		ID:        2,
		Type:      EventSerial,
		Timestamp: time.Now(),
		Latency:   8 * time.Microsecond,
		WorkerID:  1,
		Frame: &hw.Frame{
			SeqNum:  42,
			Payload: []byte("STATUS: OK\n"),
		},
	}
	if err := sink.OnEvent(ctx, serialEvent); err != nil {
		t.Fatalf("error writing serial event: %v", err)
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("error closing JSONL sink: %v", err)
	}

	// Verify Output File
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("error opening written JSONL file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 JSONL lines, got %d", len(lines))
	}

	var rec1 export.JSONEventRecord
	if err := json.Unmarshal([]byte(lines[0]), &rec1); err != nil {
		t.Fatalf("failed to unmarshal line 1: %v", err)
	}
	if rec1.ID != 1 || rec1.Packet.SrcIP != "192.168.1.10" || rec1.Packet.DstPort != 443 {
		t.Errorf("unexpected record 1 fields: %+v", rec1)
	}
}


package pipeline

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	gonet "github.com/Ri4ards2006/go-core-lab/internal/net"
	"github.com/Ri4ards2006/go-core-lab/pkg/export"
)

func makeSyntheticPacket(seq uint32) []byte {
	buf := new(bytes.Buffer)
	// Ethernet
	buf.Write([]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55})
	buf.Write([]byte{0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB})
	binary.Write(buf, binary.BigEndian, gonet.EtherTypeIPv4)

	// IPv4
	buf.WriteByte(0x45)
	buf.WriteByte(0x00)
	binary.Write(buf, binary.BigEndian, uint16(40))
	binary.Write(buf, binary.BigEndian, uint16(seq))
	binary.Write(buf, binary.BigEndian, uint16(0))
	buf.WriteByte(64)
	buf.WriteByte(gonet.IPProtoTCP)
	binary.Write(buf, binary.BigEndian, uint16(0))
	buf.Write(net.ParseIP("10.0.0.1").To4())
	buf.Write(net.ParseIP("10.0.0.2").To4())

	// TCP
	binary.Write(buf, binary.BigEndian, uint16(5000))
	binary.Write(buf, binary.BigEndian, uint16(80))
	binary.Write(buf, binary.BigEndian, seq)
	binary.Write(buf, binary.BigEndian, uint32(0))
	buf.WriteByte(0x50)
	buf.WriteByte(gonet.TCPFlagSYN)
	binary.Write(buf, binary.BigEndian, uint16(65535))
	binary.Write(buf, binary.BigEndian, uint16(0))
	binary.Write(buf, binary.BigEndian, uint16(0))

	return buf.Bytes()
}

func TestPipelineEngineEndToEnd(t *testing.T) {
	totalPackets := 2000
	payloads := make([][]byte, totalPackets)
	for i := 0; i < totalPackets; i++ {
		payloads[i] = makeSyntheticPacket(uint32(i))
	}

	src := NewSliceSource(payloads, EventNetwork)
	cfg := EngineConfig{
		NumWorkers:     4,
		QueueSize:      512,
		RingBufferSize: 128,
	}

	engine := NewPipelineEngine(cfg, src)

	// Create temporary PCAP sink
	tmpPcap := filepath.Join(t.TempDir(), "pipeline_test.pcap")
	pcapSink, err := NewPcapSink(tmpPcap)
	if err != nil {
		t.Fatalf("unexpected error creating pcap sink: %v", err)
	}
	engine.AddSink(pcapSink)

	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("failed to start pipeline engine: %v", err)
	}

	// Wait until all packets have been processed
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snap := engine.Stats.Snapshot()
		if snap.TotalProcessed == uint64(totalPackets) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := engine.Stop(); err != nil {
		t.Fatalf("error stopping pipeline: %v", err)
	}

	snap := engine.Stats.Snapshot()
	if snap.TotalProcessed != uint64(totalPackets) {
		t.Errorf("expected %d processed packets, got %d", totalPackets, snap.TotalProcessed)
	}
	if snap.DroppedEvents != 0 {
		t.Errorf("expected 0 dropped events, got %d", snap.DroppedEvents)
	}
	if snap.ErrorEvents != 0 {
		t.Errorf("expected 0 error events, got %d", snap.ErrorEvents)
	}
	if snap.TCPCount != uint64(totalPackets) {
		t.Errorf("expected %d TCP packets, got %d", totalPackets, snap.TCPCount)
	}

	// Verify PCAP Sink output
	f, err := os.Open(tmpPcap)
	if err != nil {
		t.Fatalf("could not open generated pcap sink file: %v", err)
	}
	defer f.Close()

	reader, err := export.NewPcapReader(f)
	if err != nil {
		t.Fatalf("invalid pcap header: %v", err)
	}

	pcapCount := 0
	for {
		_, _, err := reader.NextPacket()
		if err != nil {
			break
		}
		pcapCount++
	}

	if pcapCount != totalPackets {
		t.Errorf("expected %d packets in PCAP sink, got %d", totalPackets, pcapCount)
	}
}

func TestPipelineSerialIngestion(t *testing.T) {
	frames := [][]byte{
		[]byte("STATUS: OK"),
		[]byte("VOLTAGE: 3.3V"),
		[]byte("CURRENT: 120mA"),
	}

	src := NewSliceSource(frames, EventSerial)
	cfg := EngineConfig{
		NumWorkers:     2,
		QueueSize:      64,
		RingBufferSize: 64,
	}

	engine := NewPipelineEngine(cfg, src)
	ctx := context.Background()

	if err := engine.Start(ctx); err != nil {
		t.Fatalf("failed to start pipeline: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	_ = engine.Stop()

	snap := engine.Stats.Snapshot()
	if snap.SerialCount != 3 {
		t.Errorf("expected 3 serial frames processed, got %d", snap.SerialCount)
	}
}


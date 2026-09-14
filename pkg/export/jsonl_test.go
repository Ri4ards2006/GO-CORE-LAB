package export

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestJsonlWriterBasic(t *testing.T) {
	buf := new(bytes.Buffer)
	writer := NewJsonlWriter(buf)

	rec1 := JSONEventRecord{
		ID:        1,
		Type:      "NETWORK",
		Timestamp: "2026-09-14T11:00:00Z",
		LatencyNs: 12500,
		WorkerID:  0,
		Packet: &JSONPacketRecord{
			Protocol:   "TCP",
			SrcIP:      "192.168.1.50",
			DstIP:      "192.168.1.1",
			SrcPort:    54321,
			DstPort:    443,
			TCPFlags:   "SYN",
			PayloadLen: 0,
		},
	}

	rec2 := JSONEventRecord{
		ID:        2,
		Type:      "SERIAL",
		Timestamp: "2026-09-14T11:00:01Z",
		LatencyNs: 8000,
		WorkerID:  1,
		Frame: &JSONFrameRecord{
			SeqNum:  100,
			Payload: "VOLTAGE: 3.3V",
		},
	}

	if err := writer.WriteRecord(rec1); err != nil {
		t.Fatalf("unexpected write error 1: %v", err)
	}
	if err := writer.WriteRecord(rec2); err != nil {
		t.Fatalf("unexpected write error 2: %v", err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	scanner := bufio.NewScanner(buf)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d", len(lines))
	}

	var parsed1 JSONEventRecord
	if err := json.Unmarshal([]byte(lines[0]), &parsed1); err != nil {
		t.Fatalf("failed to unmarshal line 1: %v", err)
	}
	if parsed1.ID != 1 || parsed1.Packet.SrcIP != "192.168.1.50" || parsed1.Packet.DstPort != 443 {
		t.Errorf("unexpected record 1 fields: %+v", parsed1)
	}
}

func TestJsonlWriterFileAndConcurrent(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "concurrent.jsonl")

	writer, f, err := OpenJsonlFile(outPath)
	if err != nil {
		t.Fatalf("failed to create jsonl file: %v", err)
	}
	defer f.Close()

	var wg sync.WaitGroup
	workers := 8
	itemsPerWorker := 100

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < itemsPerWorker; i++ {
				rec := JSONEventRecord{
					ID:        uint64(workerID*itemsPerWorker + i),
					Type:      "NETWORK",
					Timestamp: "2026-09-14T11:00:00Z",
					WorkerID:  workerID,
				}
				_ = writer.WriteRecord(rec)
			}
		}(w)
	}

	wg.Wait()
	_ = writer.Flush()

	// Verify line count
	readFile, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("open read file: %v", err)
	}
	defer readFile.Close()

	scanner := bufio.NewScanner(readFile)
	count := 0
	for scanner.Scan() {
		count++
	}

	if count != workers*itemsPerWorker {
		t.Errorf("expected %d lines, got %d", workers*itemsPerWorker, count)
	}
}


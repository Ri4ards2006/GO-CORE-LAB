// ═══════════════════════════════════════════════════════════════════════════
// Package export provides streaming JSONL (NDJSON) writers for
// network packet captures and hardware telemetry.
// ═══════════════════════════════════════════════════
package export

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// JSONPacketRecord represents a structured network packet entry in JSONL format.
type JSONPacketRecord struct {
	EtherType  string `json:"ether_type,omitempty"`
	SrcMAC     string `json:"src_mac,omitempty"`
	DstMAC     string `json:"dst_mac,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	SrcIP      string `json:"src_ip,omitempty"`
	DstIP      string `json:"dst_ip,omitempty"`
	TTL        uint8  `json:"ttl,omitempty"`
	SrcPort    uint16 `json:"src_port,omitempty"`
	DstPort    uint16 `json:"dst_port,omitempty"`
	TCPFlags   string `json:"tcp_flags,omitempty"`
	PayloadLen int    `json:"payload_len"`
	PayloadHex string `json:"payload_hex,omitempty"`
}

// JSONFrameRecord represents a structured serial frame in JSONL format.
type JSONFrameRecord struct {
	SeqNum  uint64 `json:"seq_num,omitempty"`
	Payload string `json:"payload"`
}

// JSONEventRecord is the root object emitted per line in the .jsonl stream.
type JSONEventRecord struct {
	ID        uint64            `json:"id"`
	Type      string            `json:"type"`
	Timestamp string            `json:"timestamp"`
	LatencyNs int64             `json:"latency_ns"`
	WorkerID  int               `json:"worker_id"`
	Error     string            `json:"error,omitempty"`
	Packet    *JSONPacketRecord `json:"packet,omitempty"`
	Frame     *JSONFrameRecord  `json:"frame,omitempty"`
}

// JsonlWriter serializes structured records into an io.Writer as newline-delimited JSON.
type JsonlWriter struct {
	mu     sync.Mutex
	writer *bufio.Writer
}

// NewJsonlWriter initializes a buffered JSONL writer.
func NewJsonlWriter(w io.Writer) *JsonlWriter {
	return &JsonlWriter{
		writer: bufio.NewWriterSize(w, 64*1024),
	}
}

// WriteRecord serializes a single JSONEventRecord followed by a newline.
func (jw *JsonlWriter) WriteRecord(rec JSONEventRecord) error {
	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal jsonl: %w", err)
	}

	jw.mu.Lock()
	defer jw.mu.Unlock()

	if _, err := jw.writer.Write(data); err != nil {
		return err
	}
	return jw.writer.WriteByte('\n')
}

// Flush flushes buffered data to the underlying io.Writer.
func (jw *JsonlWriter) Flush() error {
	jw.mu.Lock()
	defer jw.mu.Unlock()
	return jw.writer.Flush()
}

// OpenJsonlFile creates or truncates a file and returns a JsonlWriter and the os.File.
func OpenJsonlFile(path string) (*JsonlWriter, *os.File, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("create jsonl file %q: %w", path, err)
	}
	return NewJsonlWriter(f), f, nil
}


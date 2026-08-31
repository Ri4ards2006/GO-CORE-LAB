package hw

import (
	"bytes"
	"io"
	"testing"
)

func TestFrameReaderNewline(t *testing.T) {
	stream := "SENSOR_VAL: 24.5C\nSTATUS: OK\r\nBATTERY: 98%\n"
	reader := bytes.NewReader([]byte(stream))

	cfg := FrameConfig{
		Mode:         DelimiterNewline,
		MaxFrameSize: 1024,
	}

	fr := NewFrameReader(reader, cfg)

	// Frame 1
	f1, err := fr.ReadNextFrame()
	if err != nil {
		t.Fatalf("unexpected error reading frame 1: %v", err)
	}
	if string(f1.Payload) != "SENSOR_VAL: 24.5C" {
		t.Errorf("expected payload 'SENSOR_VAL: 24.5C', got %q", string(f1.Payload))
	}

	// Frame 2 (\r\n stripped)
	f2, err := fr.ReadNextFrame()
	if err != nil {
		t.Fatalf("unexpected error reading frame 2: %v", err)
	}
	if string(f2.Payload) != "STATUS: OK" {
		t.Errorf("expected payload 'STATUS: OK', got %q", string(f2.Payload))
	}

	// Frame 3
	f3, err := fr.ReadNextFrame()
	if err != nil {
		t.Fatalf("unexpected error reading frame 3: %v", err)
	}
	if string(f3.Payload) != "BATTERY: 98%" {
		t.Errorf("expected payload 'BATTERY: 98%%', got %q", string(f3.Payload))
	}

	// EOF
	_, err = fr.ReadNextFrame()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestFrameReaderSyncBytes(t *testing.T) {
	// Stream with noise preceding SOF 0xAA, followed by payload and EOF 0x55
	stream := []byte{
		0xFF, 0x00, 0x12, // Noise
		0xAA, 'T', 'E', 'L', 'E', 'M', '1', 0x55, // Frame 1
		0x00, 0x00, // Noise
		0xAA, 'T', 'E', 'L', 'E', 'M', '2', 0x55, // Frame 2
	}
	reader := bytes.NewReader(stream)

	cfg := FrameConfig{
		Mode:         DelimiterSyncByte,
		SOF:          0xAA,
		EOF:          0x55,
		MaxFrameSize: 1024,
	}

	fr := NewFrameReader(reader, cfg)

	f1, err := fr.ReadNextFrame()
	if err != nil {
		t.Fatalf("unexpected error reading frame 1: %v", err)
	}
	if string(f1.Payload) != "TELEM1" {
		t.Errorf("expected payload 'TELEM1', got %q", string(f1.Payload))
	}

	f2, err := fr.ReadNextFrame()
	if err != nil {
		t.Fatalf("unexpected error reading frame 2: %v", err)
	}
	if string(f2.Payload) != "TELEM2" {
		t.Errorf("expected payload 'TELEM2', got %q", string(f2.Payload))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Package hw provides hardware serial, UART, and bus telemetry frame decoders.
// ═══════════════════════════════════════════════════
package hw

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

// DelimiterMode specifies how hardware frame boundaries are recognized.
type DelimiterMode int

const (
	DelimiterNewline DelimiterMode = iota // Line-delimited (\n or \r\n)
	DelimiterSyncByte                     // Start-of-frame (SOF) and End-of-frame (EOF) bytes
	DelimiterLengthPrefixed               // 2-byte little-endian length prefix
)

// FrameConfig defines frame extraction parameters.
type FrameConfig struct {
	Mode         DelimiterMode
	SOF          byte   // Start-of-frame byte (e.g. 0xAA)
	EOF          byte   // End-of-frame byte (e.g. 0x55)
	MaxFrameSize int    // Maximum allowed frame size to prevent buffer bloat
	BaudRate     int    // UART Baud Rate (e.g., 115200, 9600)
	Device       string // Serial device path (e.g., "/dev/ttyUSB0")
}

// DefaultConfig returns standard 115200-baud newline-delimited serial config.
func DefaultConfig() FrameConfig {
	return FrameConfig{
		Mode:         DelimiterNewline,
		SOF:          0xAA,
		EOF:          0x55,
		MaxFrameSize: 4096,
		BaudRate:     115200,
	}
}

// Frame represents a discrete hardware telemetry frame received over a bus.
type Frame struct {
	Timestamp time.Time
	SeqNum    uint32
	Payload   []byte
	Raw       []byte
	Valid     bool
}

// String returns a single-line summary of the received frame.
func (f Frame) String() string {
	ts := f.Timestamp.Format("15:04:05.000000")
	return fmt.Sprintf("[%s] Frame #%d: len=%d bytes | %s",
		ts, f.SeqNum, len(f.Payload), string(bytes.TrimSpace(f.Payload)))
}

// FrameReader extracts discrete frames from an arbitrary byte stream.
type FrameReader struct {
	reader io.Reader
	cfg    FrameConfig
	buf    []byte
	seq    uint32
}

// NewFrameReader initializes a stream scanner with the given frame configuration.
func NewFrameReader(r io.Reader, cfg FrameConfig) *FrameReader {
	if cfg.MaxFrameSize <= 0 {
		cfg.MaxFrameSize = 4096
	}
	return &FrameReader{
		reader: r,
		cfg:    cfg,
		buf:    make([]byte, 0, cfg.MaxFrameSize*2),
	}
}

// ReadNextFrame scans the stream and returns the next aligned Frame.
func (fr *FrameReader) ReadNextFrame() (Frame, error) {
	temp := make([]byte, 512)

	for {
		// Attempt to extract a frame from the existing buffer
		if frame, ok, err := fr.tryExtractFrame(); ok || err != nil {
			return frame, err
		}

		// Read more bytes from the underlying stream
		n, err := fr.reader.Read(temp)
		if n > 0 {
			fr.buf = append(fr.buf, temp[:n]...)

			// Guard against runaway buffer overflow
			if len(fr.buf) > fr.cfg.MaxFrameSize*4 {
				// Drop oldest half of buffer
				fr.buf = fr.buf[len(fr.buf)/2:]
			}
		}
		if err != nil {
			// If EOF reached with leftover data, attempt final flush
			if err == io.EOF && len(fr.buf) > 0 {
				if frame, ok, _ := fr.tryExtractFrame(); ok {
					return frame, nil
				}
			}
			return Frame{}, err
		}
	}
}

func (fr *FrameReader) tryExtractFrame() (Frame, bool, error) {
	switch fr.cfg.Mode {
	case DelimiterNewline:
		idx := bytes.IndexByte(fr.buf, '\n')
		if idx >= 0 {
			raw := fr.buf[:idx+1]
			payload := bytes.TrimRight(fr.buf[:idx], "\r")
			fr.buf = fr.buf[idx+1:]
			fr.seq++
			return Frame{
				Timestamp: time.Now(),
				SeqNum:    fr.seq,
				Payload:   payload,
				Raw:       raw,
				Valid:     true,
			}, true, nil
		}

	case DelimiterSyncByte:
		sofIdx := bytes.IndexByte(fr.buf, fr.cfg.SOF)
		if sofIdx >= 0 {
			// Discard any garbage preceding the SOF
			fr.buf = fr.buf[sofIdx:]
			eofIdx := bytes.IndexByte(fr.buf[1:], fr.cfg.EOF)
			if eofIdx >= 0 {
				totalLen := eofIdx + 2 // 1 byte for SOF + eofIdx + 1 for EOF
				raw := fr.buf[:totalLen]
				payload := fr.buf[1 : totalLen-1] // Exclude SOF and EOF
				fr.buf = fr.buf[totalLen:]
				fr.seq++
				return Frame{
					Timestamp: time.Now(),
					SeqNum:    fr.seq,
					Payload:   payload,
					Raw:       raw,
					Valid:     true,
				}, true, nil
			}
		} else {
			// No SOF found in buffer; discard all
			fr.buf = fr.buf[:0]
		}
	}

	return Frame{}, false, nil
}

// BusMonitor defines an asynchronous stream monitor for hardware devices.
type BusMonitor interface {
	Start(ctx context.Context) (<-chan Frame, error)
	Close() error
}

// SerialMonitor streams frames from a physical or simulated serial device.
type SerialMonitor struct {
	Config FrameConfig
	file   *os.File
}

// NewSerialMonitor creates a new serial bus monitor.
func NewSerialMonitor(cfg FrameConfig) *SerialMonitor {
	return &SerialMonitor{Config: cfg}
}

// Start opens the serial device and streams frames to a channel.
func (sm *SerialMonitor) Start(ctx context.Context) (<-chan Frame, error) {
	f, err := os.Open(sm.Config.Device)
	if err != nil {
		return nil, fmt.Errorf("open serial device %q: %w", sm.Config.Device, err)
	}
	sm.file = f

	reader := NewFrameReader(f, sm.Config)
	outChan := make(chan Frame, 64)

	go func() {
		defer close(outChan)
		defer f.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				frame, err := reader.ReadNextFrame()
				if err != nil {
					return
				}
				select {
				case outChan <- frame:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outChan, nil
}

// Close terminates the monitor handle.
func (sm *SerialMonitor) Close() error {
	if sm.file != nil {
		err := sm.file.Close()
		sm.file = nil
		return err
	}
	return nil
}

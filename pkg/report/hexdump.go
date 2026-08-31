// ═══════════════════════════════════════════════════════════════════════════
// Package report provides a canonical 16-byte hex dump engine with ANSI
// syntax highlighting and modular tabular report formatting.
// ═══════════════════════════════════════════════════
package report

import (
	"fmt"
	"strings"

	"github.com/Ri4ards2006/go-core-lab/internal/mmap"
)

// ANSI Color Escape Sequences
const (
	ColorReset   = "\033[0m"
	ColorDim     = "\033[90m" // Dark gray for 0x00 null bytes
	ColorASCII   = "\033[32m" // Green for printable ASCII characters
	ColorControl = "\033[33m" // Yellow for non-null control characters (< 0x20)
	ColorHigh    = "\033[35m" // Magenta for high bytes (>= 0x80)
	ColorOffset  = "\033[36m" // Cyan for address offsets
)

// HexDumpOptions configures the formatting and colorization of the hex dump.
type HexDumpOptions struct {
	Offset   int64 // Starting byte offset to display
	Length   int   // Number of bytes to display (0 for all remaining)
	Colorize bool  // Enable ANSI color classification
	UpperHex bool  // Format hex digits in uppercase
}

// DefaultHexDumpOptions returns standard colorized options.
func DefaultHexDumpOptions() HexDumpOptions {
	return HexDumpOptions{
		Offset:   0,
		Length:   0,
		Colorize: true,
		UpperHex: false,
	}
}

// HexDump formats a byte slice into a 16-byte canonical hex grid with an ASCII sidebar.
func HexDump(data []byte, opts HexDumpOptions) string {
	if len(data) == 0 {
		return "(empty buffer)\n"
	}

	start := int(opts.Offset)
	if start < 0 {
		start = 0
	}
	if start >= len(data) {
		return fmt.Sprintf("Offset 0x%x is beyond data size (0x%x)\n", opts.Offset, len(data))
	}

	end := len(data)
	if opts.Length > 0 && start+opts.Length < end {
		end = start + opts.Length
	}

	slice := data[start:end]
	var out strings.Builder

	for i := 0; i < len(slice); i += 16 {
		chunkEnd := i + 16
		if chunkEnd > len(slice) {
			chunkEnd = len(slice)
		}
		chunk := slice[i:chunkEnd]
		currentOffset := int64(start + i)

		// 1. Offset Column (8-digit hex)
		if opts.Colorize {
			out.WriteString(fmt.Sprintf("%s%08x%s  ", ColorOffset, currentOffset, ColorReset))
		} else {
			out.WriteString(fmt.Sprintf("%08x  ", currentOffset))
		}

		// 2. Hex Bytes Column (16 bytes split into two 8-byte groups)
		for j := 0; j < 16; j++ {
			if j == 8 {
				out.WriteString(" ")
			}
			if j < len(chunk) {
				b := chunk[j]
				hexFmt := "%02x "
				if opts.UpperHex {
					hexFmt = "%02X "
				}

				if opts.Colorize {
					out.WriteString(colorForByte(b) + fmt.Sprintf(hexFmt, b) + ColorReset)
				} else {
					out.WriteString(fmt.Sprintf(hexFmt, b))
				}
			} else {
				out.WriteString("   ")
			}
		}

		out.WriteString(" |")

		// 3. ASCII Sidebar Column
		for _, b := range chunk {
			if b >= 32 && b <= 126 {
				if opts.Colorize {
					out.WriteString(ColorASCII + string(b) + ColorReset)
				} else {
					out.WriteByte(b)
				}
			} else {
				if opts.Colorize {
					out.WriteString(ColorDim + "." + ColorReset)
				} else {
					out.WriteByte('.')
				}
			}
		}

		out.WriteString("|\n")
	}

	return out.String()
}

// HexDumpFile reads or memory-maps a file and formats its canonical hex dump.
func HexDumpFile(path string, opts HexDumpOptions) (string, error) {
	mf, err := mmap.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for hexdump: %w", err)
	}
	defer mf.Close()

	return HexDump(mf.Bytes(), opts), nil
}

func colorForByte(b byte) string {
	if b == 0x00 {
		return ColorDim
	}
	if b >= 0x20 && b <= 0x7E {
		return ColorASCII
	}
	if b < 0x20 {
		return ColorControl
	}
	return ColorHigh
}

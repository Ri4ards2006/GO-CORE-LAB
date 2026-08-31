package report

import (
	"strings"
	"testing"
)

func TestHexDumpFormatting(t *testing.T) {
	data := []byte{
		0x7F, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		'T', 'E', 'S', 'T',
	}

	opts := HexDumpOptions{
		Offset:   0,
		Length:   0,
		Colorize: false,
		UpperHex: false,
	}

	out := HexDump(data, opts)

	if !strings.Contains(out, "00000000  7f 45 4c 46 02 01 01 00  00 00 00 00 00 00 00 00  |.ELF............|") {
		t.Errorf("unexpected line 1 in hexdump: %s", out)
	}

	if !strings.Contains(out, "00000010  54 45 53 54") {
		t.Errorf("unexpected line 2 in hexdump: %s", out)
	}
}

func TestTableFormatting(t *testing.T) {
	tbl := NewTable("Name", "Type", "Size")
	tbl.AddRow(".text", "PROGBITS", 1024)
	tbl.AddRow(".data", "PROGBITS", 256)

	rendered := tbl.Render()
	if !strings.Contains(rendered, ".text") || !strings.Contains(rendered, "1024") {
		t.Errorf("unexpected table output: %s", rendered)
	}
}

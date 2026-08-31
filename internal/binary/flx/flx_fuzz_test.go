package flx

import (
	"testing"
)

func FuzzParseFLX(f *testing.F) {
	// Seed with valid .flx container buffer
	f.Add(BuildTestFLXBuffer())
	f.Add([]byte{0x46, 0x4C, 0x58, 0x01})
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Fuzzing .flx container, constant pool, and disassembler against panics
		_, _ = ParseFLXBytes(data, "fuzz.flx")
	})
}

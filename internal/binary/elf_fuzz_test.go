package binary

import (
	"testing"
)

func FuzzParseELF(f *testing.F) {
	// Seed corpus with valid synthetic ELF headers
	f.Add(buildSyntheticELF64())
	f.Add([]byte{0x7f, 'E', 'L', 'F', 0x01, 0x01, 0x01, 0x00})
	f.Add([]byte{0x7f, 'E', 'L', 'F', 0x02, 0x02, 0x01, 0x00})
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Fuzzing parser resilience against unexpected panics / bounds crashes
		_, _ = ParseELFBytes(data, "fuzz.elf")
	})
}

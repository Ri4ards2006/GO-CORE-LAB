package binary

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkParseELF_Standard(b *testing.B) {
	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "bench_sample.elf")
	data := buildSyntheticELF64()
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		b.Fatalf("failed to write test file: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		elfFile, err := ParseELF(filePath)
		if err != nil || elfFile == nil {
			b.Fatalf("unexpected parse error: %v", err)
		}
	}
}

func BenchmarkParseELF_Mmap(b *testing.B) {
	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "bench_sample_mmap.elf")
	data := buildSyntheticELF64()
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		b.Fatalf("failed to write test file: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		elfFile, mf, err := ParseELFMmap(filePath)
		if err != nil || elfFile == nil {
			b.Fatalf("unexpected mmap parse error: %v", err)
		}
		_ = mf.Close()
	}
}

func BenchmarkParseELF_BytesZeroCopy(b *testing.B) {
	data := buildSyntheticELF64()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		elfFile, err := ParseELFBytes(data, "bench.elf")
		if err != nil || elfFile == nil {
			b.Fatalf("unexpected bytes parse error: %v", err)
		}
	}
}

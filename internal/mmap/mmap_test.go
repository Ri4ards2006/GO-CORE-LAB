package mmap

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMmapLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.bin")
	sampleData := []byte("HELLO_MMAP_WORLD_1234567890")

	if err := os.WriteFile(filePath, sampleData, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	mf, err := Open(filePath)
	if err != nil {
		t.Fatalf("failed to mmap file: %v", err)
	}
	defer mf.Close()

	if mf.Len() != len(sampleData) {
		t.Errorf("expected len %d, got %d", len(sampleData), mf.Len())
	}

	if !bytes.Equal(mf.Bytes(), sampleData) {
		t.Errorf("expected data %q, got %q", string(sampleData), string(mf.Bytes()))
	}

	// Test Sub-slice
	sub, err := mf.Slice(6, 4)
	if err != nil {
		t.Fatalf("unexpected slice error: %v", err)
	}
	if string(sub) != "MMAP" {
		t.Errorf("expected subslice 'MMAP', got %q", string(sub))
	}

	// Test Out of bounds
	_, err = mf.Slice(10, 100)
	if err == nil {
		t.Fatal("expected out of bounds error, got nil")
	}
}

func TestMmapZeroByteFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "empty.bin")
	if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write empty file: %v", err)
	}

	mf, err := Open(filePath)
	if err != nil {
		t.Fatalf("failed to open 0-byte file: %v", err)
	}
	defer mf.Close()

	if mf.Len() != 0 {
		t.Errorf("expected len 0, got %d", mf.Len())
	}
}


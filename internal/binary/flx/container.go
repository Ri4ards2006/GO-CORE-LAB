// ═══════════════════════════════════════════════════════════════════════════
// Package flx implements the container parser for flux-lang (.flx) binary files.
// ═══════════════════════════════════════════════════
package flx

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Magic bytes identifying a .flx binary: 'F' 'L' 'X' '\x01'
var Magic = [4]byte{0x46, 0x4c, 0x58, 0x01}

// Header represents the fixed 32-byte header of a .flx bytecode container.
type Header struct {
	Magic              [4]byte // 0x00..0x03: Magic identifier
	Version            uint16  // 0x04..0x05: Format version (Major.Minor)
	Flags              uint16  // 0x06..0x07: Compiler/runtime flags
	EntryOffset        uint32  // 0x08..0x0B: Bytecode entry offset
	ConstantPoolOffset uint32  // 0x0C..0x0F: Offset to constant pool
	ConstantPoolCount  uint32  // 0x10..0x13: Number of pool entries
	BytecodeOffset     uint32  // 0x14..0x17: Offset to raw bytecode stream
	BytecodeSize       uint32  // 0x18..0x1B: Bytecode size in bytes
	MetadataOffset     uint32  // 0x1C..0x1F: Offset to metadata key-value table
	MetadataCount      uint32  // 0x20..0x23: Number of metadata entries
}

// File represents a fully parsed .flx binary artifact.
type File struct {
	Header       Header
	Metadata     map[string]string
	Pool         ConstantPool
	Bytecode     []byte
	Instructions []Instruction
	Path         string
}

// ParseFile opens a file from disk and parses the .flx container.
func ParseFile(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open .flx file: %w", err)
	}
	defer f.Close()

	return ParseReader(f, path)
}

// ParseReader parses a .flx binary from an io.ReadSeeker.
func ParseReader(r io.ReadSeeker, path string) (*File, error) {
	flxFile := &File{
		Metadata: make(map[string]string),
		Path:     path,
	}

	// 1. Read and validate 36-byte Header
	var hdr Header
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	if hdr.Magic != Magic {
		return nil, fmt.Errorf("invalid .flx magic: expected % x, got % x", Magic, hdr.Magic)
	}

	flxFile.Header = hdr

	// 2. Parse Metadata Table (if present)
	if hdr.MetadataOffset > 0 && hdr.MetadataCount > 0 {
		if _, err := r.Seek(int64(hdr.MetadataOffset), io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek to metadata (offset 0x%x): %w", hdr.MetadataOffset, err)
		}

		for i := uint32(0); i < hdr.MetadataCount; i++ {
			var keyLen uint16
			if err := binary.Read(r, binary.LittleEndian, &keyLen); err != nil {
				return nil, fmt.Errorf("read metadata key length #%d: %w", i, err)
			}
			keyBytes := make([]byte, keyLen)
			if _, err := io.ReadFull(r, keyBytes); err != nil {
				return nil, fmt.Errorf("read metadata key #%d: %w", i, err)
			}

			var valLen uint16
			if err := binary.Read(r, binary.LittleEndian, &valLen); err != nil {
				return nil, fmt.Errorf("read metadata value length #%d: %w", i, err)
			}
			valBytes := make([]byte, valLen)
			if _, err := io.ReadFull(r, valBytes); err != nil {
				return nil, fmt.Errorf("read metadata value #%d: %w", i, err)
			}

			flxFile.Metadata[string(keyBytes)] = string(valBytes)
		}
	}

	// 3. Parse Constant Pool
	if hdr.ConstantPoolOffset > 0 && hdr.ConstantPoolCount > 0 {
		if _, err := r.Seek(int64(hdr.ConstantPoolOffset), io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek to constant pool (offset 0x%x): %w", hdr.ConstantPoolOffset, err)
		}

		pool, err := DecodePool(r, hdr.ConstantPoolCount)
		if err != nil {
			return nil, fmt.Errorf("decode constant pool: %w", err)
		}
		flxFile.Pool = pool
	}

	// 4. Read Raw Bytecode Stream
	if hdr.BytecodeOffset > 0 && hdr.BytecodeSize > 0 {
		if _, err := r.Seek(int64(hdr.BytecodeOffset), io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek to bytecode (offset 0x%x): %w", hdr.BytecodeOffset, err)
		}

		bytecode := make([]byte, hdr.BytecodeSize)
		if _, err := io.ReadFull(r, bytecode); err != nil {
			return nil, fmt.Errorf("read bytecode: %w", err)
		}
		flxFile.Bytecode = bytecode

		// 5. Disassemble Bytecode Stream
		instructions, err := Disassemble(bytecode, &flxFile.Pool)
		if err != nil {
			return nil, fmt.Errorf("disassemble bytecode: %w", err)
		}
		flxFile.Instructions = instructions
	}

	return flxFile, nil
}

// PrintHeader outputs formatted container header information.
func (f *File) PrintHeader() {
	major := f.Header.Version >> 8
	minor := f.Header.Version & 0xFF

	fmt.Printf("File:            %s\n", f.Path)
	fmt.Printf("Format:          flux-lang Bytecode Container (.flx)\n")
	fmt.Printf("Magic:           % x (%s)\n", f.Header.Magic, string(f.Header.Magic[:3]))
	fmt.Printf("Version:         v%d.%d (0x%04x)\n", major, minor, f.Header.Version)
	fmt.Printf("Flags:           0x%04x\n", f.Header.Flags)
	fmt.Printf("Entry Offset:    0x%04x\n", f.Header.EntryOffset)
	fmt.Printf("Constant Pool:   Offset 0x%04x (Count: %d)\n", f.Header.ConstantPoolOffset, f.Header.ConstantPoolCount)
	fmt.Printf("Bytecode Stream: Offset 0x%04x (Size: %d bytes)\n", f.Header.BytecodeOffset, f.Header.BytecodeSize)
	fmt.Printf("Metadata Table:  Offset 0x%04x (Count: %d)\n", f.Header.MetadataOffset, f.Header.MetadataCount)

	f.PrintMetadata()
}

// PrintMetadata outputs the key-value pairs stored in the metadata table.
func (f *File) PrintMetadata() {
	if len(f.Metadata) == 0 {
		return
	}
	fmt.Printf("\n[Metadata Table]\n")
	for k, v := range f.Metadata {
		fmt.Printf("  %-18s : %s\n", k, v)
	}
}

// PrintSummary outputs a complete inspection report (header, pool, disasm).
func (f *File) PrintSummary() {
	f.PrintHeader()
	f.Pool.Print()
	PrintDisassembly(f.Instructions)
}

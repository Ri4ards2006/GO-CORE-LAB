package binary

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestParseSegments64(t *testing.T) {
	buf := new(bytes.Buffer)

	// ELF64 Header (64 bytes)
	buf.Write([]byte{0x7f, 'E', 'L', 'F', Class64, DataLE, 1, 0})
	buf.Write(make([]byte, 8))                                     // pad
	binary.Write(buf, binary.LittleEndian, uint16(2))             // e_type (ET_EXEC)
	binary.Write(buf, binary.LittleEndian, ArchX86_64)            // e_machine
	binary.Write(buf, binary.LittleEndian, uint32(1))             // e_version
	binary.Write(buf, binary.LittleEndian, uint64(0x401000))      // e_entry
	binary.Write(buf, binary.LittleEndian, uint64(64))            // e_phoff (at offset 64)
	binary.Write(buf, binary.LittleEndian, uint64(0))             // e_shoff
	binary.Write(buf, binary.LittleEndian, uint32(0))             // e_flags
	binary.Write(buf, binary.LittleEndian, uint16(64))            // e_ehsize
	binary.Write(buf, binary.LittleEndian, uint16(56))            // e_phentsize (Elf64_Phdr size is 56)
	binary.Write(buf, binary.LittleEndian, uint16(3))             // e_phnum (3 segments)
	binary.Write(buf, binary.LittleEndian, uint16(64))            // e_shentsize
	binary.Write(buf, binary.LittleEndian, uint16(0))             // e_shnum
	binary.Write(buf, binary.LittleEndian, uint16(0))             // e_shstrndx

	// Program Header Table at offset 64 (3 * 56 = 168 bytes)
	// Elf64_Phdr layout: p_type(4), p_flags(4), p_offset(8), p_vaddr(8), p_paddr(8), p_filesz(8), p_memsz(8), p_align(8)

	// Phdr 0: PT_PHDR (PF_R)
	binary.Write(buf, binary.LittleEndian, PT_PHDR)
	binary.Write(buf, binary.LittleEndian, PF_R)
	binary.Write(buf, binary.LittleEndian, uint64(64))
	binary.Write(buf, binary.LittleEndian, uint64(0x400040))
	binary.Write(buf, binary.LittleEndian, uint64(0x400040))
	binary.Write(buf, binary.LittleEndian, uint64(168))
	binary.Write(buf, binary.LittleEndian, uint64(168))
	binary.Write(buf, binary.LittleEndian, uint64(8))

	// Phdr 1: PT_LOAD (PF_R | PF_X)
	binary.Write(buf, binary.LittleEndian, PT_LOAD)
	binary.Write(buf, binary.LittleEndian, PF_R|PF_X)
	binary.Write(buf, binary.LittleEndian, uint64(0))
	binary.Write(buf, binary.LittleEndian, uint64(0x400000))
	binary.Write(buf, binary.LittleEndian, uint64(0x400000))
	binary.Write(buf, binary.LittleEndian, uint64(0x1000))
	binary.Write(buf, binary.LittleEndian, uint64(0x1000))
	binary.Write(buf, binary.LittleEndian, uint64(0x1000))

	// Phdr 2: PT_GNU_STACK (PF_R | PF_W)
	binary.Write(buf, binary.LittleEndian, PT_GNU_STACK)
	binary.Write(buf, binary.LittleEndian, PF_R|PF_W)
	binary.Write(buf, binary.LittleEndian, uint64(0))
	binary.Write(buf, binary.LittleEndian, uint64(0))
	binary.Write(buf, binary.LittleEndian, uint64(0))
	binary.Write(buf, binary.LittleEndian, uint64(0))
	binary.Write(buf, binary.LittleEndian, uint64(0))
	binary.Write(buf, binary.LittleEndian, uint64(16))

	elf, err := ParseELFBytes(buf.Bytes(), "test_seg64.elf")
	if err != nil {
		t.Fatalf("unexpected error parsing 64-bit segments: %v", err)
	}

	if len(elf.Segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(elf.Segments))
	}

	// Phdr 0: PHDR
	p0 := elf.Segments[0]
	if p0.TypeString() != "PHDR" {
		t.Errorf("expected segment 0 type 'PHDR', got %q", p0.TypeString())
	}
	if p0.FlagsString() != "R  " {
		t.Errorf("expected segment 0 flags 'R  ', got %q", p0.FlagsString())
	}

	// Phdr 1: LOAD
	p1 := elf.Segments[1]
	if p1.TypeString() != "LOAD" {
		t.Errorf("expected segment 1 type 'LOAD', got %q", p1.TypeString())
	}
	if p1.FlagsString() != "R E" {
		t.Errorf("expected segment 1 flags 'R E', got %q", p1.FlagsString())
	}
	if p1.VAddr != 0x400000 {
		t.Errorf("expected segment 1 vaddr 0x400000, got 0x%x", p1.VAddr)
	}

	// Phdr 2: GNU_STACK
	p2 := elf.Segments[2]
	if p2.TypeString() != "GNU_STACK" {
		t.Errorf("expected segment 2 type 'GNU_STACK', got %q", p2.TypeString())
	}
	if p2.FlagsString() != "RW " {
		t.Errorf("expected segment 2 flags 'RW ', got %q", p2.FlagsString())
	}
}

func TestParseSegments32(t *testing.T) {
	buf := new(bytes.Buffer)

	// ELF32 Header (52 bytes)
	buf.Write([]byte{0x7f, 'E', 'L', 'F', Class32, DataLE, 1, 0})
	buf.Write(make([]byte, 8))                         // pad
	binary.Write(buf, binary.LittleEndian, uint16(2)) // e_type
	binary.Write(buf, binary.LittleEndian, ArchX86)   // e_machine
	binary.Write(buf, binary.LittleEndian, uint32(1)) // e_version
	binary.Write(buf, binary.LittleEndian, uint32(0x8048000)) // e_entry
	binary.Write(buf, binary.LittleEndian, uint32(52)) // e_phoff (at offset 52)
	binary.Write(buf, binary.LittleEndian, uint32(0))  // e_shoff
	binary.Write(buf, binary.LittleEndian, uint32(0))  // e_flags
	binary.Write(buf, binary.LittleEndian, uint16(52)) // e_ehsize
	binary.Write(buf, binary.LittleEndian, uint16(32)) // e_phentsize (Elf32_Phdr size is 32)
	binary.Write(buf, binary.LittleEndian, uint16(2))  // e_phnum (2 segments)
	binary.Write(buf, binary.LittleEndian, uint16(40)) // e_shentsize
	binary.Write(buf, binary.LittleEndian, uint16(0))  // e_shnum
	binary.Write(buf, binary.LittleEndian, uint16(0))  // e_shstrndx

	// Program Header Table at offset 52 (2 * 32 = 64 bytes)
	// Elf32_Phdr layout: p_type(4), p_offset(4), p_vaddr(4), p_paddr(4), p_filesz(4), p_memsz(4), p_flags(4), p_align(4)

	// Phdr 0: PT_LOAD (PF_R | PF_X)
	binary.Write(buf, binary.LittleEndian, PT_LOAD)
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(0x8048000))
	binary.Write(buf, binary.LittleEndian, uint32(0x8048000))
	binary.Write(buf, binary.LittleEndian, uint32(0x500))
	binary.Write(buf, binary.LittleEndian, uint32(0x500))
	binary.Write(buf, binary.LittleEndian, PF_R|PF_X)
	binary.Write(buf, binary.LittleEndian, uint32(0x1000))

	// Phdr 1: PT_DYNAMIC (PF_R | PF_W)
	binary.Write(buf, binary.LittleEndian, PT_DYNAMIC)
	binary.Write(buf, binary.LittleEndian, uint32(0x500))
	binary.Write(buf, binary.LittleEndian, uint32(0x8049500))
	binary.Write(buf, binary.LittleEndian, uint32(0x8049500))
	binary.Write(buf, binary.LittleEndian, uint32(0x100))
	binary.Write(buf, binary.LittleEndian, uint32(0x100))
	binary.Write(buf, binary.LittleEndian, PF_R|PF_W)
	binary.Write(buf, binary.LittleEndian, uint32(4))

	elf, err := ParseELFBytes(buf.Bytes(), "test_seg32.elf")
	if err != nil {
		t.Fatalf("unexpected error parsing 32-bit segments: %v", err)
	}

	if len(elf.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(elf.Segments))
	}

	p0 := elf.Segments[0]
	if p0.TypeString() != "LOAD" {
		t.Errorf("expected segment 0 type 'LOAD', got %q", p0.TypeString())
	}
	if p0.FlagsString() != "R E" {
		t.Errorf("expected segment 0 flags 'R E', got %q", p0.FlagsString())
	}

	p1 := elf.Segments[1]
	if p1.TypeString() != "DYNAMIC" {
		t.Errorf("expected segment 1 type 'DYNAMIC', got %q", p1.TypeString())
	}
	if p1.FlagsString() != "RW " {
		t.Errorf("expected segment 1 flags 'RW ', got %q", p1.FlagsString())
	}
}


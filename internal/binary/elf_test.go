package binary

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// buildSyntheticELF creates an in-memory 64-bit Little Endian ELF buffer for testing.
func buildSyntheticELF64() []byte {
	buf := new(bytes.Buffer)

	// ELF Header (64 bytes)
	// 0..3: Magic
	buf.Write([]byte{0x7f, 'E', 'L', 'F'})
	// 4: Class 64-bit
	buf.WriteByte(Class64)
	// 5: Data Little Endian
	buf.WriteByte(DataLE)
	// 6: Version
	buf.WriteByte(1)
	// 7: OSABI
	buf.WriteByte(0)
	// 8..15: Padding
	buf.Write(make([]byte, 8))
	// 16..17: e_type (2 = ET_EXEC)
	binary.Write(buf, binary.LittleEndian, uint16(2))
	// 18..19: e_machine (0x3E = x86_64)
	binary.Write(buf, binary.LittleEndian, ArchX86_64)
	// 20..23: e_version
	binary.Write(buf, binary.LittleEndian, uint32(1))
	// 24..31: e_entry
	binary.Write(buf, binary.LittleEndian, uint64(0x401000))
	// 32..39: e_phoff
	binary.Write(buf, binary.LittleEndian, uint64(0))
	// 40..47: e_shoff (will place section headers at offset 128)
	binary.Write(buf, binary.LittleEndian, uint64(128))
	// 48..51: e_flags
	binary.Write(buf, binary.LittleEndian, uint32(0))
	// 52..53: e_ehsize
	binary.Write(buf, binary.LittleEndian, uint16(64))
	// 54..55: e_phentsize
	binary.Write(buf, binary.LittleEndian, uint16(0))
	// 56..57: e_phnum
	binary.Write(buf, binary.LittleEndian, uint16(0))
	// 58..59: e_shentsize
	binary.Write(buf, binary.LittleEndian, uint16(64))
	// 60..61: e_shnum (3 sections: [0]=NULL, [1]=.text, [2]=.shstrtab)
	binary.Write(buf, binary.LittleEndian, uint16(3))
	// 62..63: e_shstrndx (index 2)
	binary.Write(buf, binary.LittleEndian, uint16(2))

	// Padding to offset 64 (start of payload data)
	// String Table at offset 64: "\x00.text\x00.shstrtab\x00"
	// index 0: ""
	// index 1: ".text"
	// index 7: ".shstrtab"
	shstrtabData := []byte("\x00.text\x00.shstrtab\x00")
	buf.Write(shstrtabData)

	// Pad to offset 128 (Section Header Table offset)
	if buf.Len() < 128 {
		buf.Write(make([]byte, 128-buf.Len()))
	}

	// Section 0: NULL Section
	buf.Write(make([]byte, 64))

	// Section 1: .text (offset 64 in file header table)
	// sh_name: 1 (".text")
	binary.Write(buf, binary.LittleEndian, uint32(1))
	// sh_type: SHT_PROGBITS (1)
	binary.Write(buf, binary.LittleEndian, SHT_PROGBITS)
	// sh_flags: SHF_ALLOC | SHF_EXECINSTR (0x6)
	binary.Write(buf, binary.LittleEndian, SHF_ALLOC|SHF_EXECINSTR)
	// sh_addr: 0x401000
	binary.Write(buf, binary.LittleEndian, uint64(0x401000))
	// sh_offset: 64
	binary.Write(buf, binary.LittleEndian, uint64(64))
	// sh_size: 16
	binary.Write(buf, binary.LittleEndian, uint64(16))
	// sh_link, sh_info
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(0))
	// sh_addralign, sh_entsize
	binary.Write(buf, binary.LittleEndian, uint64(16))
	binary.Write(buf, binary.LittleEndian, uint64(0))

	// Section 2: .shstrtab
	// sh_name: 7 (".shstrtab")
	binary.Write(buf, binary.LittleEndian, uint32(7))
	// sh_type: SHT_STRTAB (3)
	binary.Write(buf, binary.LittleEndian, SHT_STRTAB)
	// sh_flags: 0
	binary.Write(buf, binary.LittleEndian, uint64(0))
	// sh_addr: 0
	binary.Write(buf, binary.LittleEndian, uint64(0))
	// sh_offset: 64
	binary.Write(buf, binary.LittleEndian, uint64(64))
	// sh_size: len(shstrtabData)
	binary.Write(buf, binary.LittleEndian, uint64(len(shstrtabData)))
	// sh_link, sh_info
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(0))
	// sh_addralign, sh_entsize
	binary.Write(buf, binary.LittleEndian, uint64(1))
	binary.Write(buf, binary.LittleEndian, uint64(0))

	return buf.Bytes()
}

func TestParseELFReader64(t *testing.T) {
	raw := buildSyntheticELF64()
	reader := bytes.NewReader(raw)

	elf, err := ParseELFReader(reader, "synthetic_64.elf")
	if err != nil {
		t.Fatalf("unexpected error parsing ELF: %v", err)
	}

	if elf.Header.Class != Class64 {
		t.Errorf("expected Class64 (%d), got %d", Class64, elf.Header.Class)
	}

	if elf.Header.Data != DataLE {
		t.Errorf("expected DataLE (%d), got %d", DataLE, elf.Header.Data)
	}

	if elf.Header.Machine != ArchX86_64 {
		t.Errorf("expected Machine 0x%x, got 0x%x", ArchX86_64, elf.Header.Machine)
	}

	if elf.Header.Entry != 0x401000 {
		t.Errorf("expected entry 0x401000, got 0x%x", elf.Header.Entry)
	}

	if len(elf.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(elf.Sections))
	}

	if elf.Sections[1].Name != ".text" {
		t.Errorf("expected section 1 name '.text', got %q", elf.Sections[1].Name)
	}

	if elf.Sections[1].FlagsString() != "AX" {
		t.Errorf("expected section 1 flags 'AX', got %q", elf.Sections[1].FlagsString())
	}

	if elf.Sections[1].TypeString() != "PROGBITS" {
		t.Errorf("expected section 1 type 'PROGBITS', got %q", elf.Sections[1].TypeString())
	}

	if elf.Sections[2].Name != ".shstrtab" {
		t.Errorf("expected section 2 name '.shstrtab', got %q", elf.Sections[2].Name)
	}

	if elf.Sections[2].TypeString() != "STRTAB" {
		t.Errorf("expected section 2 type 'STRTAB', got %q", elf.Sections[2].TypeString())
	}
}

func TestInvalidMagic(t *testing.T) {
	raw := []byte{0x00, 0x01, 0x02, 0x03}
	reader := bytes.NewReader(raw)

	_, err := ParseELFReader(reader, "bad.elf")
	if err == nil {
		t.Fatal("expected error for invalid magic, got nil")
	}
}

func TestArchitectureNames(t *testing.T) {
	tests := []struct {
		id       uint16
		expected string
	}{
		{ArchX86, "x86"},
		{ArchARM, "ARM"},
		{ArchX86_64, "x86_64"},
		{ArchAArch64, "AArch64"},
		{ArchRISCV, "RISC-V"},
	}

	for _, tt := range tests {
		got := ArchitectureName(tt.id)
		if got != tt.expected {
			t.Errorf("ArchitectureName(0x%x) = %q; want %q", tt.id, got, tt.expected)
		}
	}
}


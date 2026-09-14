package binary

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// buildSyntheticELFWithSymbols64 builds a 64-bit ELF buffer with .shstrtab, .text, .strtab, and .symtab.
func buildSyntheticELFWithSymbols64() []byte {
	buf := new(bytes.Buffer)

	// ELF Header (64 bytes)
	buf.Write([]byte{0x7f, 'E', 'L', 'F', Class64, DataLE, 1, 0})
	buf.Write(make([]byte, 8))                                     // pad
	binary.Write(buf, binary.LittleEndian, uint16(2))             // e_type (ET_EXEC)
	binary.Write(buf, binary.LittleEndian, ArchX86_64)            // e_machine
	binary.Write(buf, binary.LittleEndian, uint32(1))             // e_version
	binary.Write(buf, binary.LittleEndian, uint64(0x401000))      // e_entry
	binary.Write(buf, binary.LittleEndian, uint64(0))             // e_phoff
	binary.Write(buf, binary.LittleEndian, uint64(256))           // e_shoff (section headers at offset 256)
	binary.Write(buf, binary.LittleEndian, uint32(0))             // e_flags
	binary.Write(buf, binary.LittleEndian, uint16(64))            // e_ehsize
	binary.Write(buf, binary.LittleEndian, uint16(0))             // e_phentsize
	binary.Write(buf, binary.LittleEndian, uint16(0))             // e_phnum
	binary.Write(buf, binary.LittleEndian, uint16(64))            // e_shentsize
	binary.Write(buf, binary.LittleEndian, uint16(5))             // e_shnum (NULL, .text, .shstrtab, .strtab, .symtab)
	binary.Write(buf, binary.LittleEndian, uint16(2))             // e_shstrndx (index 2)

	// Section 2: .shstrtab at offset 64
	shstrtabData := []byte("\x00.text\x00.shstrtab\x00.strtab\x00.symtab\x00")
	// Offsets in shstrtab:
	// 0: ""
	// 1: ".text"
	// 7: ".shstrtab"
	// 17: ".strtab"
	// 25: ".symtab"
	buf.Write(shstrtabData)

	// Section 3: .strtab at offset 120
	for buf.Len() < 120 {
		buf.WriteByte(0)
	}
	strtabData := []byte("\x00main\x00counter\x00printf\x00")
	// Offsets in strtab:
	// 0: ""
	// 1: "main"
	// 6: "counter"
	// 14: "printf"
	buf.Write(strtabData)

	// Section 4: .symtab at offset 160 (4 symbols of 24 bytes = 96 bytes)
	for buf.Len() < 160 {
		buf.WriteByte(0)
	}

	// Sym 0: NULL symbol (24 bytes)
	buf.Write(make([]byte, 24))

	// Sym 1: main (FUNC, GLOBAL, .text)
	// st_name (1 = "main")
	binary.Write(buf, binary.LittleEndian, uint32(1))
	// st_info (STB_GLOBAL << 4 | STT_FUNC = 0x12)
	buf.WriteByte(uint8(STB_GLOBAL<<4 | STB_GLOBAL&0x0 | 0x02)) // 0x12
	// st_other (STV_DEFAULT = 0)
	buf.WriteByte(0)
	// st_shndx (1 = .text)
	binary.Write(buf, binary.LittleEndian, uint16(1))
	// st_value (0x401000)
	binary.Write(buf, binary.LittleEndian, uint64(0x401000))
	// st_size (45 bytes)
	binary.Write(buf, binary.LittleEndian, uint64(45))

	// Sym 2: counter (OBJECT, LOCAL, .text / data)
	// st_name (6 = "counter")
	binary.Write(buf, binary.LittleEndian, uint32(6))
	// st_info (STB_LOCAL << 4 | STT_OBJECT = 0x01)
	buf.WriteByte(uint8(STB_LOCAL<<4 | 0x01))
	// st_other (STV_HIDDEN = 2)
	buf.WriteByte(uint8(STV_HIDDEN))
	// st_shndx (1 = .text)
	binary.Write(buf, binary.LittleEndian, uint16(1))
	// st_value (0x401050)
	binary.Write(buf, binary.LittleEndian, uint64(0x401050))
	// st_size (4 bytes)
	binary.Write(buf, binary.LittleEndian, uint64(4))

	// Sym 3: printf (FUNC, GLOBAL, UNDEF)
	// st_name (14 = "printf")
	binary.Write(buf, binary.LittleEndian, uint32(14))
	// st_info (STB_GLOBAL << 4 | STT_FUNC = 0x12)
	buf.WriteByte(uint8(STB_GLOBAL<<4 | 0x02))
	// st_other (STV_DEFAULT = 0)
	buf.WriteByte(0)
	// st_shndx (SHN_UNDEF = 0)
	binary.Write(buf, binary.LittleEndian, SHN_UNDEF)
	// st_value (0)
	binary.Write(buf, binary.LittleEndian, uint64(0))
	// st_size (0)
	binary.Write(buf, binary.LittleEndian, uint64(0))

	// Pad to Section Header Table offset 256
	for buf.Len() < 256 {
		buf.WriteByte(0)
	}

	// Section 0: NULL Section (64 bytes)
	buf.Write(make([]byte, 64))

	// Section 1: .text (64 bytes)
	binary.Write(buf, binary.LittleEndian, uint32(1))              // sh_name: ".text"
	binary.Write(buf, binary.LittleEndian, SHT_PROGBITS)           // sh_type
	binary.Write(buf, binary.LittleEndian, SHF_ALLOC|SHF_EXECINSTR) // sh_flags
	binary.Write(buf, binary.LittleEndian, uint64(0x401000))       // sh_addr
	binary.Write(buf, binary.LittleEndian, uint64(64))             // sh_offset
	binary.Write(buf, binary.LittleEndian, uint64(100))            // sh_size
	binary.Write(buf, binary.LittleEndian, uint32(0))              // sh_link
	binary.Write(buf, binary.LittleEndian, uint32(0))              // sh_info
	binary.Write(buf, binary.LittleEndian, uint64(16))             // sh_addralign
	binary.Write(buf, binary.LittleEndian, uint64(0))              // sh_entsize

	// Section 2: .shstrtab (64 bytes)
	binary.Write(buf, binary.LittleEndian, uint32(7))              // sh_name: ".shstrtab"
	binary.Write(buf, binary.LittleEndian, SHT_STRTAB)             // sh_type
	binary.Write(buf, binary.LittleEndian, uint64(0))              // sh_flags
	binary.Write(buf, binary.LittleEndian, uint64(0))              // sh_addr
	binary.Write(buf, binary.LittleEndian, uint64(64))             // sh_offset
	binary.Write(buf, binary.LittleEndian, uint64(len(shstrtabData))) // sh_size
	binary.Write(buf, binary.LittleEndian, uint32(0))              // sh_link
	binary.Write(buf, binary.LittleEndian, uint32(0))              // sh_info
	binary.Write(buf, binary.LittleEndian, uint64(1))              // sh_addralign
	binary.Write(buf, binary.LittleEndian, uint64(0))              // sh_entsize

	// Section 3: .strtab (64 bytes)
	binary.Write(buf, binary.LittleEndian, uint32(17))             // sh_name: ".strtab"
	binary.Write(buf, binary.LittleEndian, SHT_STRTAB)             // sh_type
	binary.Write(buf, binary.LittleEndian, uint64(0))              // sh_flags
	binary.Write(buf, binary.LittleEndian, uint64(0))              // sh_addr
	binary.Write(buf, binary.LittleEndian, uint64(120))            // sh_offset
	binary.Write(buf, binary.LittleEndian, uint64(len(strtabData))) // sh_size
	binary.Write(buf, binary.LittleEndian, uint32(0))              // sh_link
	binary.Write(buf, binary.LittleEndian, uint32(0))              // sh_info
	binary.Write(buf, binary.LittleEndian, uint64(1))              // sh_addralign
	binary.Write(buf, binary.LittleEndian, uint64(0))              // sh_entsize

	// Section 4: .symtab (64 bytes)
	binary.Write(buf, binary.LittleEndian, uint32(25))             // sh_name: ".symtab"
	binary.Write(buf, binary.LittleEndian, SHT_SYMTAB)             // sh_type
	binary.Write(buf, binary.LittleEndian, uint64(0))              // sh_flags
	binary.Write(buf, binary.LittleEndian, uint64(0))              // sh_addr
	binary.Write(buf, binary.LittleEndian, uint64(160))            // sh_offset
	binary.Write(buf, binary.LittleEndian, uint64(4*24))           // sh_size (4 symbols)
	binary.Write(buf, binary.LittleEndian, uint32(3))              // sh_link (points to section 3 .strtab)
	binary.Write(buf, binary.LittleEndian, uint32(2))              // sh_info
	binary.Write(buf, binary.LittleEndian, uint64(8))              // sh_addralign
	binary.Write(buf, binary.LittleEndian, uint64(24))             // sh_entsize (Elf64_Sym size)

	return buf.Bytes()
}

func TestSymbolTableExtraction64(t *testing.T) {
	raw := buildSyntheticELFWithSymbols64()
	elf, err := ParseELFBytes(raw, "test_sym64.elf")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(elf.Symbols) != 4 {
		t.Fatalf("expected 4 symbols, got %d", len(elf.Symbols))
	}

	// Sym 1: main
	s1 := elf.Symbols[1]
	if s1.Name != "main" {
		t.Errorf("expected symbol 1 name 'main', got %q", s1.Name)
	}
	if s1.Value != 0x401000 {
		t.Errorf("expected symbol 1 value 0x401000, got 0x%x", s1.Value)
	}
	if s1.Size != 45 {
		t.Errorf("expected symbol 1 size 45, got %d", s1.Size)
	}
	if s1.BindingString() != "GLOBAL" {
		t.Errorf("expected symbol 1 binding 'GLOBAL', got %q", s1.BindingString())
	}
	if s1.TypeString() != "FUNC" {
		t.Errorf("expected symbol 1 type 'FUNC', got %q", s1.TypeString())
	}
	if s1.Section != ".text" {
		t.Errorf("expected symbol 1 section '.text', got %q", s1.Section)
	}

	// Sym 2: counter
	s2 := elf.Symbols[2]
	if s2.Name != "counter" {
		t.Errorf("expected symbol 2 name 'counter', got %q", s2.Name)
	}
	if s2.BindingString() != "LOCAL" {
		t.Errorf("expected symbol 2 binding 'LOCAL', got %q", s2.BindingString())
	}
	if s2.TypeString() != "OBJECT" {
		t.Errorf("expected symbol 2 type 'OBJECT', got %q", s2.TypeString())
	}
	if s2.VisibilityString() != "HIDDEN" {
		t.Errorf("expected symbol 2 visibility 'HIDDEN', got %q", s2.VisibilityString())
	}

	// Sym 3: printf
	s3 := elf.Symbols[3]
	if s3.Name != "printf" {
		t.Errorf("expected symbol 3 name 'printf', got %q", s3.Name)
	}
	if s3.Section != "UNDEF" {
		t.Errorf("expected symbol 3 section 'UNDEF', got %q", s3.Section)
	}
}

func TestSymbol32Extraction(t *testing.T) {
	buf := new(bytes.Buffer)

	// ELF32 Header (52 bytes)
	buf.Write([]byte{0x7f, 'E', 'L', 'F', Class32, DataLE, 1, 0})
	buf.Write(make([]byte, 8))                         // pad
	binary.Write(buf, binary.LittleEndian, uint16(2)) // e_type
	binary.Write(buf, binary.LittleEndian, ArchX86)   // e_machine
	binary.Write(buf, binary.LittleEndian, uint32(1)) // e_version
	binary.Write(buf, binary.LittleEndian, uint32(0x8048000)) // e_entry
	binary.Write(buf, binary.LittleEndian, uint32(0)) // e_phoff
	binary.Write(buf, binary.LittleEndian, uint32(180)) // e_shoff
	binary.Write(buf, binary.LittleEndian, uint32(0)) // e_flags
	binary.Write(buf, binary.LittleEndian, uint16(52)) // e_ehsize
	binary.Write(buf, binary.LittleEndian, uint16(0)) // e_phentsize
	binary.Write(buf, binary.LittleEndian, uint16(0)) // e_phnum
	binary.Write(buf, binary.LittleEndian, uint16(40)) // e_shentsize (Elf32_Shdr is 40 bytes)
	binary.Write(buf, binary.LittleEndian, uint16(4)) // e_shnum
	binary.Write(buf, binary.LittleEndian, uint16(1)) // e_shstrndx (index 1)

	// String table at offset 60
	for buf.Len() < 60 {
		buf.WriteByte(0)
	}
	shstrtabData := []byte("\x00.shstrtab\x00.strtab\x00.symtab\x00")
	buf.Write(shstrtabData)

	// Symbol string table at offset 100
	for buf.Len() < 100 {
		buf.WriteByte(0)
	}
	strtabData := []byte("\x00init_core\x00")
	buf.Write(strtabData)

	// Symbol table at offset 120 (2 symbols of 16 bytes = 32 bytes)
	for buf.Len() < 120 {
		buf.WriteByte(0)
	}
	// Sym 0: NULL (16 bytes)
	buf.Write(make([]byte, 16))

	// Sym 1: init_core (Elf32_Sym: st_name(4), st_value(4), st_size(4), st_info(1), st_other(1), st_shndx(2))
	binary.Write(buf, binary.LittleEndian, uint32(1))         // st_name ("init_core")
	binary.Write(buf, binary.LittleEndian, uint32(0x8048050)) // st_value
	binary.Write(buf, binary.LittleEndian, uint32(32))        // st_size
	buf.WriteByte(uint8(STB_WEAK<<4 | 0x02))                  // st_info (WEAK, FUNC)
	buf.WriteByte(uint8(STV_DEFAULT))                         // st_other
	binary.Write(buf, binary.LittleEndian, SHN_ABS)           // st_shndx (ABS)

	// Section Header Table at offset 180 (4 * 40 = 160 bytes)
	for buf.Len() < 180 {
		buf.WriteByte(0)
	}

	// Sec 0: NULL (40 bytes)
	buf.Write(make([]byte, 40))

	// Sec 1: .shstrtab (40 bytes)
	binary.Write(buf, binary.LittleEndian, uint32(1))
	binary.Write(buf, binary.LittleEndian, SHT_STRTAB)
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(60))
	binary.Write(buf, binary.LittleEndian, uint32(len(shstrtabData)))
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(1))
	binary.Write(buf, binary.LittleEndian, uint32(0))

	// Sec 2: .strtab (40 bytes)
	binary.Write(buf, binary.LittleEndian, uint32(11))
	binary.Write(buf, binary.LittleEndian, SHT_STRTAB)
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(100))
	binary.Write(buf, binary.LittleEndian, uint32(len(strtabData)))
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(1))
	binary.Write(buf, binary.LittleEndian, uint32(0))

	// Sec 3: .symtab (40 bytes)
	binary.Write(buf, binary.LittleEndian, uint32(19))
	binary.Write(buf, binary.LittleEndian, SHT_SYMTAB)
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(0))
	binary.Write(buf, binary.LittleEndian, uint32(120))
	binary.Write(buf, binary.LittleEndian, uint32(32)) // 2 * 16
	binary.Write(buf, binary.LittleEndian, uint32(2))  // link to sec 2 (.strtab)
	binary.Write(buf, binary.LittleEndian, uint32(1))
	binary.Write(buf, binary.LittleEndian, uint32(4))
	binary.Write(buf, binary.LittleEndian, uint32(16))

	elf, err := ParseELFBytes(buf.Bytes(), "test_sym32.elf")
	if err != nil {
		t.Fatalf("unexpected parse error 32-bit ELF symbols: %v", err)
	}

	if len(elf.Symbols) != 2 {
		t.Fatalf("expected 2 symbols in 32-bit ELF, got %d", len(elf.Symbols))
	}

	s1 := elf.Symbols[1]
	if s1.Name != "init_core" {
		t.Errorf("expected symbol name 'init_core', got %q", s1.Name)
	}
	if s1.BindingString() != "WEAK" {
		t.Errorf("expected binding 'WEAK', got %q", s1.BindingString())
	}
	if s1.Section != "ABS" {
		t.Errorf("expected section 'ABS', got %q", s1.Section)
	}
}


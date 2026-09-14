// ═══════════════════════════════════════════════════════════════════════════
// Package binary provides symbol table extraction (.symtab / .dynsym)
// and symbol binding/type decoding for ELF binaries.
// ═══════════════════════════════════════════════════
package binary

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Symbol Binding (STB_*)
type SymbolBinding uint8

const (
	STB_LOCAL  SymbolBinding = 0  // Local symbol
	STB_GLOBAL SymbolBinding = 1  // Global symbol
	STB_WEAK   SymbolBinding = 2  // Weak symbol
	STB_NUM    SymbolBinding = 3  // Number of defined types
	STB_LOOS   SymbolBinding = 10 // Start of OS-specific
	STB_GNU_UNIQUE SymbolBinding = 10 // Unique symbol (GNU extension)
	STB_HIOS   SymbolBinding = 12 // End of OS-specific
	STB_LOPROC SymbolBinding = 13 // Start of processor-specific
	STB_HIPROC SymbolBinding = 15 // End of processor-specific
)

// Symbol Types (STT_*)
type SymbolType uint8

const (
	STT_NOTYPE  SymbolType = 0  // Symbol type is unspecified
	STT_OBJECT  SymbolType = 1  // Symbol is a data object (variable, array)
	STT_FUNC    SymbolType = 2  // Symbol is a code object (function)
	STT_SECTION SymbolType = 3  // Symbol associated with a section
	STT_FILE    SymbolType = 4  // Symbol gives a source file name
	STT_COMMON  SymbolType = 5  // Symbol is an uninitialized common block
	STT_TLS     SymbolType = 6  // Symbol is thread-local data object
	STT_NUM     SymbolType = 7  // Number of defined types
	STT_LOOS    SymbolType = 10 // Start of OS-specific
	STT_GNU_IFUNC SymbolType = 10 // Indirect code object (GNU extension)
	STT_HIOS    SymbolType = 12 // End of OS-specific
	STT_LOPROC  SymbolType = 13 // Start of processor-specific
	STT_HIPROC  SymbolType = 15 // End of processor-specific
)

// Symbol Visibility (STV_*)
type SymbolVisibility uint8

const (
	STV_DEFAULT   SymbolVisibility = 0 // Default symbol visibility rules
	STV_INTERNAL  SymbolVisibility = 1 // Processor-specific hidden class
	STV_HIDDEN    SymbolVisibility = 2 // Symbol unavailable in other modules
	STV_PROTECTED SymbolVisibility = 3 // Not preemptible, not exported
)

// Special Section Indices (SHN_*)
const (
	SHN_UNDEF     uint16 = 0      // Undefined section reference
	SHN_LORESERVE uint16 = 0xff00 // Start of reserved indices
	SHN_LOPROC    uint16 = 0xff00 // Start of processor-specific
	SHN_HIPROC    uint16 = 0xff1f // End of processor-specific
	SHN_LOOS      uint16 = 0xff20 // Start of OS-specific
	SHN_HIOS      uint16 = 0xff3f // End of OS-specific
	SHN_ABS       uint16 = 0xfff1 // Absolute value not affected by relocation
	SHN_COMMON    uint16 = 0xfff2 // Common block
	SHN_XINDEX    uint16 = 0xffff // Extended index in SHT_SYMTAB_SHNDX
	SHN_HIRESERVE uint16 = 0xffff // End of reserved indices
)

// Symbol models an unpacked ELF symbol entry.
type Symbol struct {
	Index      int              // Index within the symbol table
	Name       string           // Demangled / resolved symbol identifier
	NameOff    uint32           // Offset in associated string table (st_name)
	Value      uint64           // Value / address of the symbol (st_value)
	Size       uint64           // Size of the symbol in bytes (st_size)
	Info       uint8            // Type and binding attributes (st_info)
	Other      uint8            // Visibility attributes (st_other)
	Shndx      uint16           // Section header index (st_shndx)
	Section    string           // Name of the containing section or special index
	Binding    SymbolBinding    // Decoded binding (GLOBAL, LOCAL, WEAK)
	Type       SymbolType       // Decoded type (FUNC, OBJECT, FILE, etc.)
	Visibility SymbolVisibility // Decoded visibility (DEFAULT, HIDDEN, etc.)
}

// BindingString returns the human-readable string representation of the symbol binding.
func (s Symbol) BindingString() string {
	switch s.Binding {
	case STB_LOCAL:
		return "LOCAL"
	case STB_GLOBAL:
		return "GLOBAL"
	case STB_WEAK:
		return "WEAK"
	case STB_GNU_UNIQUE:
		return "UNIQUE"
	default:
		return fmt.Sprintf("OS/PROC(%d)", s.Binding)
	}
}

// TypeString returns the human-readable string representation of the symbol type.
func (s Symbol) TypeString() string {
	switch s.Type {
	case STT_NOTYPE:
		return "NOTYPE"
	case STT_OBJECT:
		return "OBJECT"
	case STT_FUNC:
		return "FUNC"
	case STT_SECTION:
		return "SECTION"
	case STT_FILE:
		return "FILE"
	case STT_COMMON:
		return "COMMON"
	case STT_TLS:
		return "TLS"
	case STT_GNU_IFUNC:
		return "IFUNC"
	default:
		return fmt.Sprintf("0x%x", s.Type)
	}
}

// VisibilityString returns the visibility description.
func (s Symbol) VisibilityString() string {
	switch s.Visibility {
	case STV_DEFAULT:
		return "DEFAULT"
	case STV_INTERNAL:
		return "INTERNAL"
	case STV_HIDDEN:
		return "HIDDEN"
	case STV_PROTECTED:
		return "PROTECTED"
	default:
		return fmt.Sprintf("%d", s.Visibility)
	}
}

// parseSymbolTable extracts symbols from a symbol table section (SHT_SYMTAB or SHT_DYNSYM).
func parseSymbolTable(r io.ReadSeeker, symSec Section, sections []Section, class uint8, bo binary.ByteOrder) ([]Symbol, error) {
	if symSec.Size == 0 || symSec.Offset == 0 {
		return nil, nil
	}

	// 1. Resolve Associated String Table (.strtab or .dynstr via symSec.Link)
	var strTable []byte
	if int(symSec.Link) < len(sections) && symSec.Link != 0 {
		strSec := sections[symSec.Link]
		if strSec.Type == SHT_STRTAB && strSec.Size > 0 {
			strTable = make([]byte, strSec.Size)
			if _, err := r.Seek(int64(strSec.Offset), io.SeekStart); err == nil {
				_, _ = io.ReadFull(r, strTable)
			}
		}
	}

	// 2. Seek to Symbol Table
	if _, err := r.Seek(int64(symSec.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to symbol table offset 0x%x: %w", symSec.Offset, err)
	}

	var symbols []Symbol

	if class == Class64 {
		const symSize64 = 24 // Elf64_Sym is 24 bytes
		numSyms := int(symSec.Size / symSize64)
		buf := make([]byte, numSyms*symSize64)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, fmt.Errorf("read 64-bit symbol table: %w", err)
		}

		symbols = make([]Symbol, numSyms)
		for i := 0; i < numSyms; i++ {
			off := i * symSize64
			nameOff := bo.Uint32(buf[off : off+4])
			info := buf[off+4]
			other := buf[off+5]
			shndx := bo.Uint16(buf[off+6 : off+8])
			val := bo.Uint64(buf[off+8 : off+16])
			sz := bo.Uint64(buf[off+16 : off+24])

			sym := Symbol{
				Index:      i,
				NameOff:    nameOff,
				Value:      val,
				Size:       sz,
				Info:       info,
				Other:      other,
				Shndx:      shndx,
				Binding:    SymbolBinding(info >> 4),
				Type:       SymbolType(info & 0x0f),
				Visibility: SymbolVisibility(other & 0x03),
			}

			if len(strTable) > 0 {
				sym.Name = extractNullString(strTable, nameOff)
			}

			sym.Section = resolveSectionIndex(shndx, sections)
			symbols[i] = sym
		}
	} else {
		const symSize32 = 16 // Elf32_Sym is 16 bytes
		numSyms := int(symSec.Size / symSize32)
		buf := make([]byte, numSyms*symSize32)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, fmt.Errorf("read 32-bit symbol table: %w", err)
		}

		symbols = make([]Symbol, numSyms)
		for i := 0; i < numSyms; i++ {
			off := i * symSize32
			nameOff := bo.Uint32(buf[off : off+4])
			val := uint64(bo.Uint32(buf[off+4 : off+8]))
			sz := uint64(bo.Uint32(buf[off+8 : off+12]))
			info := buf[off+12]
			other := buf[off+13]
			shndx := bo.Uint16(buf[off+14 : off+16])

			sym := Symbol{
				Index:      i,
				NameOff:    nameOff,
				Value:      val,
				Size:       sz,
				Info:       info,
				Other:      other,
				Shndx:      shndx,
				Binding:    SymbolBinding(info >> 4),
				Type:       SymbolType(info & 0x0f),
				Visibility: SymbolVisibility(other & 0x03),
			}

			if len(strTable) > 0 {
				sym.Name = extractNullString(strTable, nameOff)
			}

			sym.Section = resolveSectionIndex(shndx, sections)
			symbols[i] = sym
		}
	}

	return symbols, nil
}

// resolveSectionIndex converts a section header index into a descriptive string.
func resolveSectionIndex(shndx uint16, sections []Section) string {
	switch shndx {
	case SHN_UNDEF:
		return "UNDEF"
	case SHN_ABS:
		return "ABS"
	case SHN_COMMON:
		return "COMMON"
	default:
		if int(shndx) < len(sections) {
			secName := sections[shndx].Name
			if secName != "" {
				return secName
			}
			return fmt.Sprintf("[%d]", shndx)
		}
		return fmt.Sprintf("[%d]", shndx)
	}
}

// PrintSymbols formats and outputs static symbols to standard output.
func (e *ELFFile) PrintSymbols() {
	if len(e.Symbols) == 0 {
		fmt.Println("\nNo static symbol table (.symtab) present in binary.")
		return
	}

	fmt.Printf("\n[Symbol Table (.symtab) (%d entries)]\n", len(e.Symbols))
	if e.Header.Class == Class64 {
		fmt.Printf("  [Nr] %-18s %-8s %-8s %-8s %-8s %-10s %s\n",
			"Value", "Size", "Type", "Bind", "Vis", "Section", "Name")
		for _, s := range e.Symbols {
			fmt.Printf("  [%4d] %016x %-8d %-8s %-8s %-8s %-10s %s\n",
				s.Index, s.Value, s.Size, s.TypeString(), s.BindingString(), s.VisibilityString(), s.Section, s.Name)
		}
	} else {
		fmt.Printf("  [Nr] %-10s %-8s %-8s %-8s %-8s %-10s %s\n",
			"Value", "Size", "Type", "Bind", "Vis", "Section", "Name")
		for _, s := range e.Symbols {
			fmt.Printf("  [%4d] %08x %-8d %-8s %-8s %-8s %-10s %s\n",
				s.Index, s.Value, s.Size, s.TypeString(), s.BindingString(), s.VisibilityString(), s.Section, s.Name)
		}
	}
}

// PrintDynSymbols formats and outputs dynamic symbols to standard output.
func (e *ELFFile) PrintDynSymbols() {
	if len(e.DynSymbols) == 0 {
		fmt.Println("\nNo dynamic symbol table (.dynsym) present in binary.")
		return
	}

	fmt.Printf("\n[Dynamic Symbol Table (.dynsym) (%d entries)]\n", len(e.DynSymbols))
	if e.Header.Class == Class64 {
		fmt.Printf("  [Nr] %-18s %-8s %-8s %-8s %-8s %-10s %s\n",
			"Value", "Size", "Type", "Bind", "Vis", "Section", "Name")
		for _, s := range e.DynSymbols {
			fmt.Printf("  [%4d] %016x %-8d %-8s %-8s %-8s %-10s %s\n",
				s.Index, s.Value, s.Size, s.TypeString(), s.BindingString(), s.VisibilityString(), s.Section, s.Name)
		}
	} else {
		fmt.Printf("  [Nr] %-10s %-8s %-8s %-8s %-8s %-10s %s\n",
			"Value", "Size", "Type", "Bind", "Vis", "Section", "Name")
		for _, s := range e.DynSymbols {
			fmt.Printf("  [%4d] %08x %-8d %-8s %-8s %-8s %-10s %s\n",
				s.Index, s.Value, s.Size, s.TypeString(), s.BindingString(), s.VisibilityString(), s.Section, s.Name)
		}
	}
}


// ═══════════════════════════════════════════════════════════════════════════
// Package flx implements the container parser, constant pool deserializer,
// and scalable table-driven disassembler for flux-lang (.flx) bytecode.
// ═══════════════════════════════════════════════════
package flx

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Constant Pool Tag Identifiers
const (
	TagNull   uint8 = 0x00 // Null / Void constant
	TagInt    uint8 = 0x01 // 64-bit signed integer
	TagFloat  uint8 = 0x02 // 64-bit floating point
	TagString uint8 = 0x03 // Length-prefixed UTF-8 string
	TagSymbol uint8 = 0x04 // Function / Variable / Identifier symbol
	TagBool   uint8 = 0x05 // Boolean (1 byte: 0 or 1)
)

// TagName maps tag bytes to human-readable strings.
func TagName(tag uint8) string {
	switch tag {
	case TagNull:
		return "NULL"
	case TagInt:
		return "INT"
	case TagFloat:
		return "FLOAT"
	case TagString:
		return "STRING"
	case TagSymbol:
		return "SYMBOL"
	case TagBool:
		return "BOOL"
	default:
		return fmt.Sprintf("UNKNOWN(0x%02x)", tag)
	}
}

// Constant represents an entry in the constant pool.
type Constant struct {
	Index int
	Tag   uint8
	Value any
}

// String returns a representation of the constant suitable for debug output.
func (c Constant) String() string {
	switch c.Tag {
	case TagNull:
		return "null"
	case TagInt:
		return fmt.Sprintf("%d", c.Value)
	case TagFloat:
		return fmt.Sprintf("%f", c.Value)
	case TagString:
		return fmt.Sprintf("%q", c.Value)
	case TagSymbol:
		return fmt.Sprintf("`%v`", c.Value)
	case TagBool:
		return fmt.Sprintf("%t", c.Value)
	default:
		return fmt.Sprintf("%v", c.Value)
	}
}

// ConstantPool stores literal constants referenced by bytecode instructions.
type ConstantPool struct {
	Entries []Constant
}

// Get returns the constant at the given index, or an error if out of bounds.
func (cp *ConstantPool) Get(idx int) (Constant, error) {
	if idx < 0 || idx >= len(cp.Entries) {
		return Constant{}, fmt.Errorf("constant pool index %d out of bounds (len %d)", idx, len(cp.Entries))
	}
	return cp.Entries[idx], nil
}

// Format returns a formatted annotation string for disassembly display.
func (cp *ConstantPool) Format(idx int) string {
	c, err := cp.Get(idx)
	if err != nil {
		return fmt.Sprintf("<invalid pool idx %d>", idx)
	}
	switch c.Tag {
	case TagString:
		return fmt.Sprintf("str: %s", c.String())
	case TagSymbol:
		return fmt.Sprintf("sym: %v", c.Value)
	case TagInt:
		return fmt.Sprintf("int: %d", c.Value)
	case TagFloat:
		return fmt.Sprintf("float: %f", c.Value)
	case TagBool:
		return fmt.Sprintf("bool: %t", c.Value)
	case TagNull:
		return "null"
	default:
		return fmt.Sprintf("val: %s", c.String())
	}
}

// Print displays a formatted table of all constant pool entries.
func (cp *ConstantPool) Print() {
	if len(cp.Entries) == 0 {
		fmt.Println("\nConstant Pool: (empty)")
		return
	}

	fmt.Printf("\n[Constant Pool (%d entries)]\n", len(cp.Entries))
	fmt.Printf("  [Idx] %-8s %s\n", "Type", "Value")
	fmt.Printf("  --------------------------------------------------\n")
	for _, entry := range cp.Entries {
		fmt.Printf("  [%3d] %-8s %s\n", entry.Index, TagName(entry.Tag), entry.String())
	}
}

// DecodePool deserializes the constant pool from an io.Reader.
func DecodePool(r io.Reader, count uint32) (ConstantPool, error) {
	pool := ConstantPool{
		Entries: make([]Constant, count),
	}

	for i := uint32(0); i < count; i++ {
		var tag uint8
		if err := binary.Read(r, binary.LittleEndian, &tag); err != nil {
			return pool, fmt.Errorf("read tag for constant #%d: %w", i, err)
		}

		var val any
		switch tag {
		case TagNull:
			val = nil

		case TagInt:
			var intVal int64
			if err := binary.Read(r, binary.LittleEndian, &intVal); err != nil {
				return pool, fmt.Errorf("read int constant #%d: %w", i, err)
			}
			val = intVal

		case TagFloat:
			var bits uint64
			if err := binary.Read(r, binary.LittleEndian, &bits); err != nil {
				return pool, fmt.Errorf("read float constant #%d: %w", i, err)
			}
			val = math.Float64frombits(bits)

		case TagString, TagSymbol:
			var strLen uint32
			if err := binary.Read(r, binary.LittleEndian, &strLen); err != nil {
				return pool, fmt.Errorf("read string length for constant #%d: %w", i, err)
			}
			strBytes := make([]byte, strLen)
			if _, err := io.ReadFull(r, strBytes); err != nil {
				return pool, fmt.Errorf("read string bytes for constant #%d: %w", i, err)
			}
			val = string(strBytes)

		case TagBool:
			var b uint8
			if err := binary.Read(r, binary.LittleEndian, &b); err != nil {
				return pool, fmt.Errorf("read bool constant #%d: %w", i, err)
			}
			val = b != 0

		default:
			return pool, fmt.Errorf("unsupported constant tag 0x%02x at index %d", tag, i)
		}

		pool.Entries[i] = Constant{
			Index: int(i),
			Tag:   tag,
			Value: val,
		}
	}

	return pool, nil
}


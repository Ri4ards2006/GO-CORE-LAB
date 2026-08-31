// ═══════════════════════════════════════════════════════════════════════════
// Package flx implements the table-driven bytecode disassembler for flux-lang.
// ═══════════════════════════════════════════════════
package flx

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Opcode represents a single-byte flux-lang instruction.
type Opcode uint8

// Standard flux-lang opcode definitions
const (
	// Stack & Memory (0x00 - 0x0F)
	OpNOP        Opcode = 0x00
	OpLoadConst  Opcode = 0x01
	OpLoadVar    Opcode = 0x02
	OpStoreVar   Opcode = 0x03
	OpPop        Opcode = 0x04
	OpDup        Opcode = 0x05
	OpSwap       Opcode = 0x06

	// Arithmetic & Bitwise (0x10 - 0x1F)
	OpAdd        Opcode = 0x10
	OpSub        Opcode = 0x11
	OpMul        Opcode = 0x12
	OpDiv        Opcode = 0x13
	OpMod        Opcode = 0x14
	OpNeg        Opcode = 0x15
	OpBitAnd     Opcode = 0x16
	OpBitOr      Opcode = 0x17
	OpBitXor     Opcode = 0x18
	OpBitNot     Opcode = 0x19

	// Comparisons (0x20 - 0x2F)
	OpCmpEq      Opcode = 0x20
	OpCmpNe      Opcode = 0x21
	OpCmpLt      Opcode = 0x22
	OpCmpLe      Opcode = 0x23
	OpCmpGt      Opcode = 0x24
	OpCmpGe      Opcode = 0x25

	// Control Flow (0x30 - 0x3F)
	OpJump        Opcode = 0x30
	OpJumpIfTrue  Opcode = 0x31
	OpJumpIfFalse Opcode = 0x32
	OpLoop        Opcode = 0x33

	// Subroutines & Lifecycle (0x40 - 0xFF)
	OpCall       Opcode = 0x40
	OpRet        Opcode = 0x41
	OpPrint      Opcode = 0x42
	OpAssert     Opcode = 0x43
	OpDebug      Opcode = 0xFE
	OpHalt       Opcode = 0xFF
)

// OperandKind describes the binary encoding format of instruction operands.
type OperandKind int

const (
	OperandNone        OperandKind = iota // 0 operand bytes
	OperandPoolIdx16                      // 2-byte uint16 constant pool index
	OperandU8                             // 1-byte uint8 value (e.g., argc, reg)
	OperandU16                            // 2-byte uint16 value
	OperandRelJumpI16                     // 2-byte int16 relative jump offset
)

// InstructionDef defines metadata for an opcode in the instruction table.
type InstructionDef struct {
	Opcode      Opcode
	Mnemonic    string
	OperandKind OperandKind
	Description string
}

// OpcodeTable defines the scalable instruction decoding lookup table.
var OpcodeTable = map[Opcode]InstructionDef{
	// Stack & Memory
	OpNOP:        {Opcode: OpNOP, Mnemonic: "NOP", OperandKind: OperandNone, Description: "No operation"},
	OpLoadConst:  {Opcode: OpLoadConst, Mnemonic: "LOAD_CONST", OperandKind: OperandPoolIdx16, Description: "Push constant from pool onto stack"},
	OpLoadVar:    {Opcode: OpLoadVar, Mnemonic: "LOAD_VAR", OperandKind: OperandPoolIdx16, Description: "Load variable by symbol name"},
	OpStoreVar:   {Opcode: OpStoreVar, Mnemonic: "STORE_VAR", OperandKind: OperandPoolIdx16, Description: "Store TOS into symbol name"},
	OpPop:        {Opcode: OpPop, Mnemonic: "POP", OperandKind: OperandNone, Description: "Discard top of stack"},
	OpDup:        {Opcode: OpDup, Mnemonic: "DUP", OperandKind: OperandNone, Description: "Duplicate top of stack"},
	OpSwap:       {Opcode: OpSwap, Mnemonic: "SWAP", OperandKind: OperandNone, Description: "Swap top two elements on stack"},

	// Arithmetic & Bitwise
	OpAdd:        {Opcode: OpAdd, Mnemonic: "ADD", OperandKind: OperandNone, Description: "Binary addition: TOS-1 + TOS"},
	OpSub:        {Opcode: OpSub, Mnemonic: "SUB", OperandKind: OperandNone, Description: "Binary subtraction: TOS-1 - TOS"},
	OpMul:        {Opcode: OpMul, Mnemonic: "MUL", OperandKind: OperandNone, Description: "Binary multiplication: TOS-1 * TOS"},
	OpDiv:        {Opcode: OpDiv, Mnemonic: "DIV", OperandKind: OperandNone, Description: "Binary division: TOS-1 / TOS"},
	OpMod:        {Opcode: OpMod, Mnemonic: "MOD", OperandKind: OperandNone, Description: "Modulo: TOS-1 % TOS"},
	OpNeg:        {Opcode: OpNeg, Mnemonic: "NEG", OperandKind: OperandNone, Description: "Unary negation: -TOS"},
	OpBitAnd:     {Opcode: OpBitAnd, Mnemonic: "BIT_AND", OperandKind: OperandNone, Description: "Bitwise AND"},
	OpBitOr:      {Opcode: OpBitOr, Mnemonic: "BIT_OR", OperandKind: OperandNone, Description: "Bitwise OR"},
	OpBitXor:     {Opcode: OpBitXor, Mnemonic: "BIT_XOR", OperandKind: OperandNone, Description: "Bitwise XOR"},
	OpBitNot:     {Opcode: OpBitNot, Mnemonic: "BIT_NOT", OperandKind: OperandNone, Description: "Bitwise NOT"},

	// Comparisons
	OpCmpEq:      {Opcode: OpCmpEq, Mnemonic: "CMP_EQ", OperandKind: OperandNone, Description: "Compare equality: TOS-1 == TOS"},
	OpCmpNe:      {Opcode: OpCmpNe, Mnemonic: "CMP_NE", OperandKind: OperandNone, Description: "Compare inequality: TOS-1 != TOS"},
	OpCmpLt:      {Opcode: OpCmpLt, Mnemonic: "CMP_LT", OperandKind: OperandNone, Description: "Compare less than: TOS-1 < TOS"},
	OpCmpLe:      {Opcode: OpCmpLe, Mnemonic: "CMP_LE", OperandKind: OperandNone, Description: "Compare less or equal: TOS-1 <= TOS"},
	OpCmpGt:      {Opcode: OpCmpGt, Mnemonic: "CMP_GT", OperandKind: OperandNone, Description: "Compare greater than: TOS-1 > TOS"},
	OpCmpGe:      {Opcode: OpCmpGe, Mnemonic: "CMP_GE", OperandKind: OperandNone, Description: "Compare greater or equal: TOS-1 >= TOS"},

	// Control Flow
	OpJump:        {Opcode: OpJump, Mnemonic: "JUMP", OperandKind: OperandRelJumpI16, Description: "Unconditional relative jump"},
	OpJumpIfTrue:  {Opcode: OpJumpIfTrue, Mnemonic: "JUMP_IF_TRUE", OperandKind: OperandRelJumpI16, Description: "Jump if TOS is truthy"},
	OpJumpIfFalse: {Opcode: OpJumpIfFalse, Mnemonic: "JUMP_IF_FALSE", OperandKind: OperandRelJumpI16, Description: "Jump if TOS is falsy"},
	OpLoop:        {Opcode: OpLoop, Mnemonic: "LOOP", OperandKind: OperandRelJumpI16, Description: "Backward relative jump for loops"},

	// Subroutines & Lifecycle
	OpCall:       {Opcode: OpCall, Mnemonic: "CALL", OperandKind: OperandU8, Description: "Call function with N arguments"},
	OpRet:        {Opcode: OpRet, Mnemonic: "RET", OperandKind: OperandNone, Description: "Return from function"},
	OpPrint:      {Opcode: OpPrint, Mnemonic: "PRINT", OperandKind: OperandNone, Description: "Debug print TOS to stdout"},
	OpAssert:     {Opcode: OpAssert, Mnemonic: "ASSERT", OperandKind: OperandNone, Description: "Assert TOS is truthy"},
	OpDebug:      {Opcode: OpDebug, Mnemonic: "DEBUG", OperandKind: OperandNone, Description: "Trigger debugger breakpoint"},
	OpHalt:       {Opcode: OpHalt, Mnemonic: "HALT", OperandKind: OperandNone, Description: "Halt execution"},
}

// Instruction represents a disassembled bytecode instruction.
type Instruction struct {
	Offset     int      // Byte offset in stream
	Opcode     Opcode   // Opcode identifier
	Mnemonic   string   // Human-readable mnemonic
	RawBytes   []byte   // Raw instruction byte sequence
	OperandRaw any      // Parsed numerical operand value
	Annotation string   // Disassembly annotation (resolved constant / jump target)
}

// Disassemble parses a raw bytecode slice into annotated instructions using the constant pool.
func Disassemble(bytecode []byte, pool *ConstantPool) ([]Instruction, error) {
	var instructions []Instruction
	offset := 0

	for offset < len(bytecode) {
		op := Opcode(bytecode[offset])
		def, ok := OpcodeTable[op]
		if !ok {
			// Unknown opcode: fallback to raw byte representation
			instructions = append(instructions, Instruction{
				Offset:     offset,
				Opcode:     op,
				Mnemonic:   fmt.Sprintf("DB 0x%02X", op),
				RawBytes:   []byte{uint8(op)},
				Annotation: "unknown opcode",
			})
			offset++
			continue
		}

		inst := Instruction{
			Offset:   offset,
			Opcode:   op,
			Mnemonic: def.Mnemonic,
		}

		switch def.OperandKind {
		case OperandNone:
			inst.RawBytes = []byte{uint8(op)}
			offset++

		case OperandPoolIdx16:
			if offset+2 >= len(bytecode) {
				return instructions, fmt.Errorf("truncated operand at offset 0x%04x for %s", offset, def.Mnemonic)
			}
			idx := binary.LittleEndian.Uint16(bytecode[offset+1 : offset+3])
			inst.RawBytes = bytecode[offset : offset+3]
			inst.OperandRaw = idx
			if pool != nil {
				inst.Annotation = pool.Format(int(idx))
			} else {
				inst.Annotation = fmt.Sprintf("idx: %d", idx)
			}
			offset += 3

		case OperandU8:
			if offset+1 >= len(bytecode) {
				return instructions, fmt.Errorf("truncated operand at offset 0x%04x for %s", offset, def.Mnemonic)
			}
			val := bytecode[offset+1]
			inst.RawBytes = bytecode[offset : offset+2]
			inst.OperandRaw = val
			inst.Annotation = fmt.Sprintf("argc: %d", val)
			offset += 2

		case OperandU16:
			if offset+2 >= len(bytecode) {
				return instructions, fmt.Errorf("truncated operand at offset 0x%04x for %s", offset, def.Mnemonic)
			}
			val := binary.LittleEndian.Uint16(bytecode[offset+1 : offset+3])
			inst.RawBytes = bytecode[offset : offset+3]
			inst.OperandRaw = val
			inst.Annotation = fmt.Sprintf("val: 0x%04x", val)
			offset += 3

		case OperandRelJumpI16:
			if offset+2 >= len(bytecode) {
				return instructions, fmt.Errorf("truncated operand at offset 0x%04x for %s", offset, def.Mnemonic)
			}
			rel := int16(binary.LittleEndian.Uint16(bytecode[offset+1 : offset+3]))
			inst.RawBytes = bytecode[offset : offset+3]
			inst.OperandRaw = rel
			target := offset + 3 + int(rel)
			if rel >= 0 {
				inst.Annotation = fmt.Sprintf("+%d -> [0x%04x]", rel, target)
			} else {
				inst.Annotation = fmt.Sprintf("%d -> [0x%04x]", rel, target)
			}
			offset += 3
		}

		instructions = append(instructions, inst)
	}

	return instructions, nil
}

// PrintDisassembly formats and prints a list of disassembled instructions.
func PrintDisassembly(instructions []Instruction) {
	if len(instructions) == 0 {
		fmt.Println("\nDisassembly: (empty bytecode stream)")
		return
	}

	fmt.Printf("\n[Bytecode Disassembly (%d instructions)]\n", len(instructions))
	fmt.Printf("  %-8s %-12s %-16s %s\n", "Offset", "Bytes", "Instruction", "Annotation")
	fmt.Printf("  ----------------------------------------------------------------------\n")

	for _, inst := range instructions {
		var hexBytes []string
		for _, b := range inst.RawBytes {
			hexBytes = append(hexBytes, fmt.Sprintf("%02x", b))
		}
		byteStr := strings.Join(hexBytes, " ")

		operandStr := ""
		if inst.OperandRaw != nil {
			operandStr = fmt.Sprintf("%v", inst.OperandRaw)
		}

		instStr := inst.Mnemonic
		if operandStr != "" {
			instStr = fmt.Sprintf("%-12s %s", inst.Mnemonic, operandStr)
		}

		annotationStr := ""
		if inst.Annotation != "" {
			annotationStr = fmt.Sprintf("; %s", inst.Annotation)
		}

		fmt.Printf("  0x%04x:  %-12s %-20s %s\n", inst.Offset, byteStr, instStr, annotationStr)
	}
}

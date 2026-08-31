package flx

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// BuildTestFLXBuffer constructs a complete in-memory .flx binary.
func BuildTestFLXBuffer() []byte {
	buf := new(bytes.Buffer)

	// 1. Metadata bytes
	metaBuf := new(bytes.Buffer)
	metadata := []struct{ k, v string }{
		{"Author", "Richard"},
		{"CompilerVersion", "fluxc-0.2.0"},
		{"Target", "flux-vm"},
	}
	for _, m := range metadata {
		binary.Write(metaBuf, binary.LittleEndian, uint16(len(m.k)))
		metaBuf.WriteString(m.k)
		binary.Write(metaBuf, binary.LittleEndian, uint16(len(m.v)))
		metaBuf.WriteString(m.v)
	}

	// 2. Constant Pool bytes
	poolBuf := new(bytes.Buffer)
	// #0: String "Hello, flux-lang!"
	poolBuf.WriteByte(TagString)
	str0 := "Hello, flux-lang!"
	binary.Write(poolBuf, binary.LittleEndian, uint32(len(str0)))
	poolBuf.WriteString(str0)

	// #1: Symbol "count"
	poolBuf.WriteByte(TagSymbol)
	sym1 := "count"
	binary.Write(poolBuf, binary.LittleEndian, uint32(len(sym1)))
	poolBuf.WriteString(sym1)

	// #2: Int 42
	poolBuf.WriteByte(TagInt)
	binary.Write(poolBuf, binary.LittleEndian, int64(42))

	// #3: Float 3.14159
	poolBuf.WriteByte(TagFloat)
	binary.Write(poolBuf, binary.LittleEndian, float64(3.14159))

	// #4: Bool true
	poolBuf.WriteByte(TagBool)
	poolBuf.WriteByte(1)

	// 3. Bytecode bytes
	codeBuf := new(bytes.Buffer)
	// 0000: LOAD_CONST 0 ("Hello, flux-lang!")
	codeBuf.WriteByte(byte(OpLoadConst))
	binary.Write(codeBuf, binary.LittleEndian, uint16(0))
	// 0003: PRINT
	codeBuf.WriteByte(byte(OpPrint))
	// 0004: LOAD_CONST 2 (42)
	codeBuf.WriteByte(byte(OpLoadConst))
	binary.Write(codeBuf, binary.LittleEndian, uint16(2))
	// 0007: STORE_VAR 1 (`count`)
	codeBuf.WriteByte(byte(OpStoreVar))
	binary.Write(codeBuf, binary.LittleEndian, uint16(1))
	// 000A: LOAD_VAR 1 (`count`)
	codeBuf.WriteByte(byte(OpLoadVar))
	binary.Write(codeBuf, binary.LittleEndian, uint16(1))
	// 000D: LOAD_CONST 2 (42)
	codeBuf.WriteByte(byte(OpLoadConst))
	binary.Write(codeBuf, binary.LittleEndian, uint16(2))
	// 0010: ADD
	codeBuf.WriteByte(byte(OpAdd))
	// 0011: JUMP_IF_FALSE +3 -> [0017]
	codeBuf.WriteByte(byte(OpJumpIfFalse))
	binary.Write(codeBuf, binary.LittleEndian, int16(3))
	// 0014: PRINT
	codeBuf.WriteByte(byte(OpPrint))
	// 0015: HALT
	codeBuf.WriteByte(byte(OpHalt))

	// Header size = 36 bytes
	const hdrSize = 36
	metaOffset := uint32(hdrSize)
	poolOffset := metaOffset + uint32(metaBuf.Len())
	codeOffset := poolOffset + uint32(poolBuf.Len())

	hdr := Header{
		Magic:              Magic,
		Version:            0x0100, // v1.0
		Flags:              0x0001, // DEBUG_SYMBOLS
		EntryOffset:        0,
		ConstantPoolOffset: poolOffset,
		ConstantPoolCount:  5,
		BytecodeOffset:     codeOffset,
		BytecodeSize:       uint32(codeBuf.Len()),
		MetadataOffset:     metaOffset,
		MetadataCount:      uint32(len(metadata)),
	}

	binary.Write(buf, binary.LittleEndian, &hdr)
	buf.Write(metaBuf.Bytes())
	buf.Write(poolBuf.Bytes())
	buf.Write(codeBuf.Bytes())

	return buf.Bytes()
}

func TestParseFLXContainer(t *testing.T) {
	raw := BuildTestFLXBuffer()
	reader := bytes.NewReader(raw)

	flxFile, err := ParseReader(reader, "test.flx")
	if err != nil {
		t.Fatalf("unexpected error parsing .flx: %v", err)
	}

	// Verify Header
	if flxFile.Header.Magic != Magic {
		t.Errorf("expected magic %v, got %v", Magic, flxFile.Header.Magic)
	}
	if flxFile.Header.Version != 0x0100 {
		t.Errorf("expected version 0x0100, got 0x%04x", flxFile.Header.Version)
	}
	if flxFile.Header.ConstantPoolCount != 5 {
		t.Errorf("expected 5 pool entries, got %d", flxFile.Header.ConstantPoolCount)
	}

	// Verify Metadata
	if flxFile.Metadata["Author"] != "Richard" {
		t.Errorf("expected Author 'Richard', got %q", flxFile.Metadata["Author"])
	}
	if flxFile.Metadata["CompilerVersion"] != "fluxc-0.2.0" {
		t.Errorf("expected CompilerVersion 'fluxc-0.2.0', got %q", flxFile.Metadata["CompilerVersion"])
	}

	// Verify Pool
	if len(flxFile.Pool.Entries) != 5 {
		t.Fatalf("expected 5 decoded pool constants, got %d", len(flxFile.Pool.Entries))
	}
	c0, _ := flxFile.Pool.Get(0)
	if c0.Tag != TagString || c0.Value != "Hello, flux-lang!" {
		t.Errorf("expected pool #0 to be string 'Hello, flux-lang!', got %v", c0)
	}
	c1, _ := flxFile.Pool.Get(1)
	if c1.Tag != TagSymbol || c1.Value != "count" {
		t.Errorf("expected pool #1 to be symbol 'count', got %v", c1)
	}
	c2, _ := flxFile.Pool.Get(2)
	if c2.Tag != TagInt || c2.Value != int64(42) {
		t.Errorf("expected pool #2 to be int 42, got %v", c2)
	}

	// Verify Disassembly
	if len(flxFile.Instructions) != 10 {
		t.Fatalf("expected 10 disassembled instructions, got %d", len(flxFile.Instructions))
	}

	inst0 := flxFile.Instructions[0]
	if inst0.Mnemonic != "LOAD_CONST" {
		t.Errorf("expected inst 0 to be LOAD_CONST, got %s", inst0.Mnemonic)
	}
	if inst0.Annotation != `str: "Hello, flux-lang!"` {
		t.Errorf("expected annotation %q, got %q", `str: "Hello, flux-lang!"`, inst0.Annotation)
	}

	instJump := flxFile.Instructions[7]
	if instJump.Mnemonic != "JUMP_IF_FALSE" {
		t.Errorf("expected inst 7 to be JUMP_IF_FALSE, got %s", instJump.Mnemonic)
	}
	if instJump.Annotation != "+3 -> [0x0017]" {
		t.Errorf("expected jump target +3 -> [0x0017], got %q", instJump.Annotation)
	}

	lastInst := flxFile.Instructions[9]
	if lastInst.Mnemonic != "HALT" {
		t.Errorf("expected last inst to be HALT, got %s", lastInst.Mnemonic)
	}
}

func TestInvalidFLXMagic(t *testing.T) {
	raw := []byte{0x00, 0x01, 0x02, 0x03}
	reader := bytes.NewReader(raw)

	_, err := ParseReader(reader, "bad.flx")
	if err == nil {
		t.Fatal("expected error on invalid magic, got nil")
	}
}

func TestConstantPoolOutOfBounds(t *testing.T) {
	pool := ConstantPool{
		Entries: []Constant{
			{Index: 0, Tag: TagInt, Value: int64(100)},
		},
	}

	_, err := pool.Get(5)
	if err == nil {
		t.Fatal("expected error on out-of-bounds index, got nil")
	}

	formatted := pool.Format(5)
	if formatted != "<invalid pool idx 5>" {
		t.Errorf("expected formatted invalid index string, got %q", formatted)
	}
}

func TestGenerateSampleFLXFixture(t *testing.T) {
	// Create testdata fixture directory and sample.flx
	dir := filepath.Join("..", "..", "..", "testdata")
	_ = os.MkdirAll(dir, 0755)

	flxBytes := BuildTestFLXBuffer()
	filePath := filepath.Join(dir, "sample.flx")
	if err := os.WriteFile(filePath, flxBytes, 0644); err != nil {
		t.Logf("warning: could not write testdata/sample.flx fixture: %v", err)
	}
}

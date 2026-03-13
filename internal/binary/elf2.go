// ═══════════════════════════════════════════════════
// PACKAGE — wie namespace in C++
// Alles in diesem File gehört zu "binary"
// import "...../internal/binary" → dann binary.ParseELF()
// ═══════════════════════════════════════════════════
package binary

import (
    "encoding/binary" // Stdlib: bytes ↔ integers konvertieren (Little/Big Endian)
    "fmt"             // Stdlib: print, format strings
    "os"              // Stdlib: Dateisystem, Files öffnen
)

// ═══════════════════════════════════════════════════
// var — globale Variable, existiert solange das Programm läuft
// [4]byte — Array mit GENAU 4 bytes, Größe ist fix (kein slice!)
// {0x7f, 'E', 'L', 'F'} — Array literal, direkt befüllt
// 'E' == 0x45 — single quotes = ein byte/rune, double quotes = string
// ═══════════════════════════════════════════════════
var elfMagic = [4]byte{0x7f, 'E', 'L', 'F'}

// ═══════════════════════════════════════════════════
// const — compile-time konstant, kann nie geändert werden
// = #define in C aber typsicher
// Gruppierung mit () — mehrere consts auf einmal
// ═══════════════════════════════════════════════════
const (
    Class32  = 1 // ELF spec: byte[4] == 1 → 32bit binary
    Class64  = 2 // ELF spec: byte[4] == 2 → 64bit binary
    DataLE   = 1 // ELF spec: byte[5] == 1 → Little Endian
    DataBE   = 2 // ELF spec: byte[5] == 2 → Big Endian
)

// ═══════════════════════════════════════════════════
// map[KeyTyp]ValueTyp — wie unordered_map<K,V> in C++
// uint16 = 2-byte unsigned int (Machine field im ELF Header)
// string = Go string
// { key: value, } — map literal
// ═══════════════════════════════════════════════════
var machineNames = map[uint16]string{
    0x03: "x86",
    0x3E: "x86_64",
    0x28: "ARM",
    0xB7: "ARM64",
    0xF3: "RISC-V",
}

// ═══════════════════════════════════════════════════
// type XYZ struct — neue Datenstruktur definieren
// = struct in C, aber kann Methoden haben
// uint8  = 1 byte (0-255)
// uint16 = 2 bytes (0-65535)
// uint64 = 8 bytes (0-18 Trillionen)
// Felder mit Großbuchstaben = public (exportiert)
// Felder mit kleinbuchstaben = private (nur in diesem package)
// ═══════════════════════════════════════════════════
type ELFHeader struct {
    Class    uint8  // byte[4]:  1=32bit,  2=64bit
    Data     uint8  // byte[5]:  1=LE,     2=BE
    Machine  uint16 // byte[18]: CPU typ   z.b. 0x3E=x86_64
    Entry    uint64 // byte[24]: Adresse wo main() im Speicher liegt
    ShOff    uint64 // byte[40]: Offset im File wo Section-Liste beginnt
    ShNum    uint16 // byte[60]: Wie viele Sections gibt es
    ShStrIdx uint16 // byte[62]: Welche Section hat die Namen (.shstrtab)
}

type Section struct {
    Name   string // z.b. ".text", ".data", ".rodata"
    Type   uint32 // 1=PROGBITS, 3=STRTAB, etc
    Offset uint64 // wo im File fängt diese Section an
    Size   uint64 // wie groß ist sie in bytes
}

// ═══════════════════════════════════════════════════
// []Section — SLICE von Sections (dynamisches Array, wie vector<Section>)
// [] ohne Zahl = slice (variable Länge)
// [4] mit Zahl = array (fixe Länge)
// ═══════════════════════════════════════════════════
type ELFFile struct {
    Header   ELFHeader  // eingebetteter Struct — kein Pointer, direkt drin
    Sections []Section  // slice, anfangs leer, wächst mit append()
    Path     string     // Dateipfad als String
}

// ═══════════════════════════════════════════════════
// func Name(params) (returnTypen)
// *ELFFile — Pointer auf ELFFile (wie ELFFile* in C)
// error    — Go's eingebauter Error Interface typ
// zwei Rückgabewerte: (Ergebnis, Fehler) — Go-Philosophie
// bei Erfolg: return elf, nil   (nil = kein Fehler)
// bei Fehler: return nil, err   (nil = kein Ergebnis)
// ═══════════════════════════════════════════════════
func ParseELF(path string) (*ELFFile, error) {

    // os.Open gibt (file, error) zurück
    // f = file handle, err = nil wenn ok
    f, err := os.Open(path)
    if err != nil {
        // %w — wrap error, behält original Fehlermeldung + fügt context hinzu
        return nil, fmt.Errorf("open: %w", err)
    }
    // defer = "mach das wenn die Funktion endet" — egal ob error oder nicht
    // verhindert resource leaks, = RAII in C++
    defer f.Close()

    // make([]byte, 64) — slice mit 64 bytes allokieren, alle auf 0
    // = malloc(64) aber mit GC, kein free() nötig
    raw := make([]byte, 64)

    // f.Read füllt raw mit bytes aus dem file
    // _ bedeutet "den ersten Rückgabewert interessiert mich nicht"
    // (Read gibt (anzahl_gelesener_bytes, error) zurück)
    if _, err := f.Read(raw); err != nil {
        return nil, fmt.Errorf("read header: %w", err)
    }

    // raw[0] — index in slice/array, wie in C
    // != — ungleich
    // || — logisches OR, kennst du aus C
    if raw[0] != elfMagic[0] || raw[1] != elfMagic[1] ||
        raw[2] != elfMagic[2] || raw[3] != elfMagic[3] {
        return nil, fmt.Errorf("not an ELF file")
    }

    // & — Adresse von, gibt Pointer zurück (wie in C)
    // &ELFFile{Path: path} — erstelle ELFFile, setze Path Feld, gib Pointer zurück
    elf := &ELFFile{Path: path}

    // Struct Felder via . zugreifen — wie in C
    // raw[4] = byte an Position 4 im slice
    elf.Header.Class = raw[4]
    elf.Header.Data  = raw[5]

    // binary.ByteOrder ist ein INTERFACE — definiert Uint16(), Uint32(), Uint64()
    // var bo binary.ByteOrder — Variable vom Interface-Typ, noch nil
    var bo binary.ByteOrder
    if elf.Header.Data == DataLE {
        bo = binary.LittleEndian // implementiert ByteOrder Interface
    } else {
        bo = binary.BigEndian    // implementiert ByteOrder Interface auch
    }

    // raw[18:20] — SLICE SYNTAX: bytes von index 18 bis 19 (20 exklusiv)
    // = &raw[18] mit length 2 in C
    // bo.Uint16() liest 2 bytes und gibt uint16 zurück
    elf.Header.Machine = bo.Uint16(raw[18:20])

    if elf.Header.Class == Class64 {
        // 64bit ELF: Entry ist 8 bytes groß
        elf.Header.Entry    = bo.Uint64(raw[24:32]) // bytes 24-31
        elf.Header.ShOff    = bo.Uint64(raw[40:48]) // bytes 40-47
        elf.Header.ShNum    = bo.Uint16(raw[60:62])
        elf.Header.ShStrIdx = bo.Uint16(raw[62:64])
    } else {
        // 32bit ELF: Entry ist nur 4 bytes groß
        // uint64(...) — expliziter type cast, wie (uint64) in C
        elf.Header.Entry    = uint64(bo.Uint32(raw[24:28]))
        elf.Header.ShOff    = uint64(bo.Uint32(raw[32:36]))
        elf.Header.ShNum    = bo.Uint16(raw[48:50])
        elf.Header.ShStrIdx = bo.Uint16(raw[50:52])
    }

    return elf, nil // Erfolg: Pointer zurück, kein Fehler
}

// ═══════════════════════════════════════════════════
// (e *ELFFile) — METHODE auf ELFFile
// e = receiver, wie "this" in C++/Kotlin
// * = Pointer receiver — arbeitet auf Original, nicht Kopie
// ohne * wäre es eine Kopie (teuer + Änderungen gehen verloren)
// ═══════════════════════════════════════════════════
func (e *ELFFile) Print() {

    // map lookup: value, ok := map[key]
    // ok = true wenn key existiert, false wenn nicht
    // kennst du aus Kotlin: map["key"] ?: "default"
    arch, ok := machineNames[e.Header.Machine]
    if !ok {
        // %x — hex lowercase,  %X — hex uppercase
        // Sprintf = wie Printf aber gibt string zurück statt zu printen
        arch = fmt.Sprintf("0x%x", e.Header.Machine)
    }

    // inline map literal — direkt als Variable
    // map[uint8]string — key=uint8, value=string
    bits   := map[uint8]string{Class32: "32-bit", Class64: "64-bit"}
    endian := map[uint8]string{DataLE: "Little Endian", DataBE: "Big Endian"}

    // Printf Format-Zeichen:
    // %s = string
    // %d = integer dezimal
    // %x = integer hex lowercase (z.b. 0x6aa0)
    // \n = newline
    fmt.Printf("File:     %s\n", e.Path)
    fmt.Printf("Arch:     %s (%s)\n", arch, bits[e.Header.Class])
    fmt.Printf("Endian:   %s\n", endian[e.Header.Data])
    fmt.Printf("Entry:    0x%x\n", e.Header.Entry)
    fmt.Printf("Sections: %d\n", e.Header.ShNum)
}
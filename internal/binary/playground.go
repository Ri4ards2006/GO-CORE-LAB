package binary 

import "fmt"

func add (a, b int) int {

	return a +b 
}

func divide (a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil	
}


// Maps	  wie unordered map in C++ oder dict in Python, Schlüssel-Wert-Paare

func learnMaps() {

	archNames := map[uint16]string{
		0x3E: "x86-64",
		0x28: "ARM",
		0xB7: "AArch64",
	}

	arch := uint16(0x3E) // Beispielarchitektur
	name, ok := archNames[arch] // ok = true, wenn Schlüssel existiert, sonst false
	if ok {
		fmt.Printf("Architecture 0x%X: %s\n", arch, name)
	} else {
		fmt.Printf("Unknown architecture: 0x%X\n", arch)
	}
}

func learnSlices() {
	// Slices sind dynamische Arrays, die eine Länge und eine Kapazität haben.
	// Sie sind flexibler als Arrays, da sie zur Laufzeit wachsen können.
	
	// Ein Slice von Strings erstellen
	bytes := []byte{0x7f, 0x45, 0x4c, 0x46} // Beispiel-Byte-Slice (ELF-Magic-Bytes)

fmt.Printf("Magic Bytes: %v\n", bytes)

bytes = append(bytes, 0x01, 0x02) // Slice erweitern

fmt.Printf("Length: %d, Capacity: %d\n", len(bytes), cap(bytes)) // Länge und Kapazität anzeigen

}
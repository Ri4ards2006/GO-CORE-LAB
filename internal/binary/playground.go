package binary

import "fmt"

func add(a, b int) int {
	return a + b
}

func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

// Maps in Go: key-value storage similar to std::unordered_map in C++
func learnMaps() {
	archNames := map[uint16]string{
		0x3E: "x86-64",
		0x28: "ARM",
		0xB7: "AArch64",
	}

	arch := uint16(0x3E)
	name, ok := archNames[arch]
	if ok {
		fmt.Printf("Architecture 0x%X: %s\n", arch, name)
	} else {
		fmt.Printf("Unknown architecture: 0x%X\n", arch)
	}
}

func learnSlices() {
	// Slices are dynamic windows backed by arrays.
	bytes := []byte{0x7f, 0x45, 0x4c, 0x46} // ELF Magic Bytes
	fmt.Printf("Magic Bytes: %v\n", bytes)

	bytes = append(bytes, 0x01, 0x02)
	fmt.Printf("Length: %d, Capacity: %d\n", len(bytes), cap(bytes))
}

// Playground runs sample educational functions.
func Playground() {
	fmt.Println("Playground for learning Go features")

	result := add(10, 32)
	fmt.Println("Addition result:", result)

	val, err := divide(10.0, 3.0)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Division result:", val)
	}

	fmt.Println("Maps example:")
	learnMaps()

	fmt.Println("\nSlices example:")
	learnSlices()
}

// PlaygroundSection models a sample section for playground demonstrations
type PlaygroundSection struct {
	Name   string
	Offset uint64
	Size   uint64
}

// Print formats section details
func (s PlaygroundSection) Print() {
	fmt.Printf("Section: %s, Offset: 0x%X, Size: %d bytes\n", s.Name, s.Offset, s.Size)
}

// Printable interface defines types that can format their state
type Printable interface {
	Print()
}

// PrintAll iterates over any slice of types implementing Printable
func PrintAll(items []Printable) {
	for _, item := range items {
		item.Print()
	}
}

func learnStructs() {
	s1 := PlaygroundSection{Name: ".text", Offset: 0x1000, Size: 512}
	s2 := PlaygroundSection{Name: ".data", Offset: 0x2000, Size: 256}

	s1.Print()
	s2.Print()

	sections := []Printable{s1, s2}
	for i, sec := range sections {
		fmt.Printf("Section %d:\n", i)
		sec.Print()
	}
}

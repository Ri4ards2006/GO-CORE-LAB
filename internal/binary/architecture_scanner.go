package binary

import "fmt"

// Architecture identification constants matching ELF e_machine values
const (
	ArchX86     = uint16(0x03)
	ArchARM     = uint16(0x28)
	ArchX86_64  = uint16(0x3E)
	ArchAArch64 = uint16(0xB7)
	ArchRISCV   = uint16(0xF3)
)

// ArchitectureName returns the human-readable string representation of an ELF machine ID.
func ArchitectureName(machine uint16) string {
	switch machine {
	case ArchX86:
		return "x86"
	case ArchARM:
		return "ARM"
	case ArchX86_64:
		return "x86_64"
	case ArchAArch64:
		return "AArch64"
	case ArchRISCV:
		return "RISC-V"
	default:
		return fmt.Sprintf("Unknown (0x%x)", machine)
	}
}

// LearnControlFlow demonstrates Go control flow structures.
func LearnControlFlow() {
	x := 42

	if x > 10 {
		fmt.Println("x is greater than 10:", x)
	} else if x == 10 {
		fmt.Println("x is equal to 10:", x)
	} else {
		fmt.Println("x is less than 10:", x)
	}

	// For loop with initialization, condition, and post statement
	for i := 0; i < 5; i++ {
		fmt.Printf("i: %d\n", i)
	}

	// For loop acting as a while loop
	m := 0
	for m < 5 {
		fmt.Printf("m: %d\n", m)
		m++
	}

	arch := ArchX86_64
	switch arch {
	case ArchX86_64:
		fmt.Println("x86-64 architecture")
	case ArchARM:
		fmt.Println("ARM architecture")
	case ArchAArch64:
		fmt.Println("AArch64 architecture")
	default:
		fmt.Printf("Unknown architecture: 0x%X\n", arch)
	}
}

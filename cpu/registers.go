package cpu

// Registers is a snapshot of 6502 architectural register state, read
// directly from the simulator's storage nodes.
type Registers struct {
	A, X, Y, S, P uint8
	PC            uint16
}

// Status flag bits within the P register.
const (
	FlagC uint8 = 1 << 0 // Carry
	FlagZ uint8 = 1 << 1 // Zero
	FlagI uint8 = 1 << 2 // Interrupt disable
	FlagD uint8 = 1 << 3 // Decimal mode
	FlagB uint8 = 1 << 4 // Break (only set when pushed by BRK/PHP)
	FlagU uint8 = 1 << 5 // Conventionally always 1 when pushed
	FlagV uint8 = 1 << 6 // Overflow
	FlagN uint8 = 1 << 7 // Negative
)

// Storage-node IDs from Visual6502's nodenames.js.
// See https://github.com/trebonian/visual6502/blob/master/nodenames.js.
var (
	nodesA   = [8]int{737, 1234, 978, 162, 727, 858, 1136, 1653}
	nodesX   = [8]int{1216, 98, 1, 1648, 85, 589, 448, 777}
	nodesY   = [8]int{64, 1148, 573, 305, 989, 615, 115, 843}
	nodesS   = [8]int{1403, 183, 81, 1532, 1702, 1098, 1212, 1435}
	nodesPCL = [8]int{1139, 1022, 655, 1359, 900, 622, 377, 1611}
	nodesPCH = [8]int{1670, 292, 502, 584, 948, 49, 1551, 205}
	// P storage nodes: bits 0..3,6,7 are real status bits; bits 4 and
	// 5 don't exist in hardware. We mask them off.
	nodesP = [8]int{32, 627, 1553, 348, -1, -1, 1625, 69}
)

// Registers returns the current architectural state.
func (c *CPU) Registers() Registers {
	return Registers{
		A:  c.readByteFromNodes(nodesA),
		X:  c.readByteFromNodes(nodesX),
		Y:  c.readByteFromNodes(nodesY),
		S:  c.readByteFromNodes(nodesS),
		P:  c.readByteFromNodes(nodesP),
		PC: uint16(c.readByteFromNodes(nodesPCL)) | uint16(c.readByteFromNodes(nodesPCH))<<8,
	}
}

func (c *CPU) readByteFromNodes(ids [8]int) uint8 {
	var v uint8
	for bit, id := range ids {
		if id < 0 || id >= NodeDefCount {
			continue
		}
		n := c.nodes[id]
		if n != nil && n.State {
			v |= 1 << bit
		}
	}
	return v
}

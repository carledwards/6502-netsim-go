package cpu

// Constants for special node IDs
const (
	NodeGND  = 558  // vss
	NodePWR  = 657  // vcc
	NodeCLK0 = 1171 // clock 0
	NodeRDY  = 89   // ready
	NodeSO   = 1672 // stack overflow
	NodeNMI  = 1297 // non maskable interrupt
	NodeIRQ  = 103  // interrupt
	NodeRES  = 159  // reset
	NodeRW   = 1156 // read/write

	NodeDefCount = 1725
)

// ReadFromBus is a callback function type for reading from the bus
type ReadFromBus func(address int) uint8

// WriteToBus is a callback function type for writing to the bus
type WriteToBus func(address int, data uint8)

// Transistor represents a transistor in the CPU
type Transistor struct {
	ID         int
	GateNodeID int
	C1NodeID   int
	C2NodeID   int
	On         bool
}

// Node represents a node in the CPU
type Node struct {
	ID              int
	State           bool
	PullUp          bool
	PullDown        int
	GateTransistors []*Transistor
	C1C2Transistors []*Transistor
	InNodeGroup     bool
}

// DataLine represents data lines D0-D7
type DataLine uint8

// AddressLine represents address lines A0-A15
type AddressLine uint16

// Constants for data line node IDs
var DataLineVals = map[int]int{
	0: 1005, // D0
	1: 82,   // D1
	2: 945,  // D2
	3: 650,  // D3
	4: 1393, // D4
	5: 175,  // D5
	6: 1591, // D6
	7: 1349, // D7
}

// Constants for address line node IDs
var AddressLineVals = map[int]int{
	0:  268,  // A0
	1:  451,  // A1
	2:  1340, // A2
	3:  211,  // A3
	4:  435,  // A4
	5:  736,  // A5
	6:  887,  // A6
	7:  1493, // A7
	8:  230,  // A8
	9:  148,  // A9
	10: 1443, // A10
	11: 399,  // A11
	12: 1237, // A12
	13: 349,  // A13
	14: 672,  // A14
	15: 195,  // A15
}

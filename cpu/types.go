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

// ReadFromBus is a callback the CPU invokes during the bus-read phase
// of a clock cycle. The address is whatever the address-bus nodes
// resolve to.
type ReadFromBus func(address uint16) uint8

// WriteToBus is a callback the CPU invokes during the bus-write phase
// of a clock cycle.
type WriteToBus func(address uint16, data uint8)

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
	// InRecalcList tracks membership in the current or next recalc
	// slice — the per-node flag replaces the old map-as-set approach
	// so we get O(1) dedup with no allocations and deterministic
	// (insertion-order) iteration without sorting each round.
	InRecalcList bool
}

// DataLine represents data lines D0-D7
type DataLine uint8

// AddressLine represents address lines A0-A15
type AddressLine uint16

// DataLineVals maps data line bit (0-7) to its segment node ID.
// Indexed by bit so iteration order is deterministic; transistor
// simulation correctness depends on stable iteration order.
var DataLineVals = [8]int{
	1005, // D0
	82,   // D1
	945,  // D2
	650,  // D3
	1393, // D4
	175,  // D5
	1591, // D6
	1349, // D7
}

// AddressLineVals maps address line bit (0-15) to its segment node ID.
var AddressLineVals = [16]int{
	268,  // A0
	451,  // A1
	1340, // A2
	211,  // A3
	435,  // A4
	736,  // A5
	887,  // A6
	1493, // A7
	230,  // A8
	148,  // A9
	1443, // A10
	399,  // A11
	1237, // A12
	349,  // A13
	672,  // A14
	195,  // A15
}

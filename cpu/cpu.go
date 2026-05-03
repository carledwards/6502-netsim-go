// Package cpu is a transistor-level simulator of the MOS 6502, derived
// from the Visual6502 project.
//
// The simulator drives a real bus through ReadFromBus / WriteToBus
// callbacks supplied by the caller. Transistor and segment definitions
// are embedded into the binary, so consumers don't need to ship the
// data files.
package cpu

import (
	"bufio"
	"bytes"
	_ "embed"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// connectTransistors still uses sort.Ints for deterministic build
// of GateTransistors/C1C2Transistors lists. recalcNodeList no longer
// needs sort — slice insertion order is the natural deterministic
// order now.

//go:embed data/transdefs.txt
var transDefs []byte

//go:embed data/segdefs.txt
var segDefs []byte

// CPU represents the 6502 processor.
type CPU struct {
	transistors     map[int]*Transistor
	nodes           [NodeDefCount]*Node
	gndNode         *Node
	pwrNode         *Node
	recalcNodeGroup []*Node
	// recalcCurr / recalcNext are reused across recalc rounds so the
	// hot path makes zero allocations. Membership is tracked by the
	// per-Node InRecalcList flag for O(1) dedup.
	recalcCurr  []*Node
	recalcNext  []*Node
	readFromBus ReadFromBus
	writeToBus  WriteToBus
	addressBits uint16
	dataBits    uint8
}

// New creates a CPU wired to the supplied bus callbacks. Transistor
// and segment definitions are loaded from the embedded data files.
func New(readFromBus ReadFromBus, writeToBus WriteToBus) (*CPU, error) {
	cpu := &CPU{
		transistors:     make(map[int]*Transistor),
		recalcNodeGroup: make([]*Node, 0),
		readFromBus:     readFromBus,
		writeToBus:      writeToBus,
	}

	if err := cpu.setupTransistors(bytes.NewReader(transDefs)); err != nil {
		return nil, err
	}
	if err := cpu.setupNodes(bytes.NewReader(segDefs)); err != nil {
		return nil, err
	}
	cpu.connectTransistors()
	return cpu, nil
}

func (c *CPU) setupTransistors(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == ',' || unicode.IsSpace(r)
		})
		if len(parts) < 4 {
			continue
		}

		id, _ := strconv.Atoi(parts[0])
		gate, _ := strconv.Atoi(parts[1])
		c1, _ := strconv.Atoi(parts[2])
		c2, _ := strconv.Atoi(parts[3])

		// Handle special cases for GND and PWR connections
		if c1 == NodeGND {
			c1, c2 = c2, NodeGND
		}
		if c1 == NodePWR {
			c1, c2 = c2, NodePWR
		}

		trans := &Transistor{
			ID:         id,
			GateNodeID: gate,
			C1NodeID:   c1,
			C2NodeID:   c2,
		}
		c.transistors[trans.ID] = trans
	}
	return scanner.Err()
}

func (c *CPU) setupNodes(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == ',' || unicode.IsSpace(r)
		})
		if len(parts) < 2 {
			continue
		}

		id, _ := strconv.Atoi(parts[0])
		pullup, _ := strconv.Atoi(parts[1])

		if c.nodes[id] == nil {
			c.nodes[id] = &Node{
				ID:              id,
				PullUp:          pullup == 1,
				GateTransistors: make([]*Transistor, 0),
				C1C2Transistors: make([]*Transistor, 0),
				PullDown:        -1,
			}
		}
	}
	return scanner.Err()
}

func (c *CPU) connectTransistors() {
	// Iterate transistors in stable ID order so that GateTransistors
	// and C1C2Transistors slices are built consistently across runs.
	// Map iteration is randomized in Go, and any order-dependence
	// downstream (transistor toggle propagation) would be flaky.
	ids := make([]int, 0, len(c.transistors))
	for id := range c.transistors {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		trans := c.transistors[id]
		if c.nodes[trans.GateNodeID] == nil {
			c.nodes[trans.GateNodeID] = &Node{
				ID:              trans.GateNodeID,
				GateTransistors: make([]*Transistor, 0),
				C1C2Transistors: make([]*Transistor, 0),
				PullDown:        -1,
			}
		}
		if c.nodes[trans.C1NodeID] == nil {
			c.nodes[trans.C1NodeID] = &Node{
				ID:              trans.C1NodeID,
				GateTransistors: make([]*Transistor, 0),
				C1C2Transistors: make([]*Transistor, 0),
				PullDown:        -1,
			}
		}
		if c.nodes[trans.C2NodeID] == nil {
			c.nodes[trans.C2NodeID] = &Node{
				ID:              trans.C2NodeID,
				GateTransistors: make([]*Transistor, 0),
				C1C2Transistors: make([]*Transistor, 0),
				PullDown:        -1,
			}
		}
		c.nodes[trans.GateNodeID].GateTransistors = append(c.nodes[trans.GateNodeID].GateTransistors, trans)
		c.nodes[trans.C1NodeID].C1C2Transistors = append(c.nodes[trans.C1NodeID].C1C2Transistors, trans)
		c.nodes[trans.C2NodeID].C1C2Transistors = append(c.nodes[trans.C2NodeID].C1C2Transistors, trans)
	}
}

func (c *CPU) getNodeValue() bool {
	if c.gndNode.InNodeGroup {
		return false
	}
	if c.pwrNode.InNodeGroup {
		return true
	}

	for _, node := range c.recalcNodeGroup {
		if node.PullUp {
			return true
		}
		if node.PullDown == 1 {
			return false
		}
		if node.State {
			return true
		}
	}
	return false
}

func (c *CPU) addSubNodesToGroup(node *Node) {
	if node.InNodeGroup {
		return
	}

	c.recalcNodeGroup = append(c.recalcNodeGroup, node)
	node.InNodeGroup = true

	if node.ID == NodeGND || node.ID == NodePWR {
		return
	}

	for _, trans := range node.C1C2Transistors {
		if !trans.On {
			continue
		}
		targetNodeID := trans.C1NodeID
		if targetNodeID == node.ID {
			targetNodeID = trans.C2NodeID
		}
		c.addSubNodesToGroup(c.nodes[targetNodeID])
	}
}

func (c *CPU) recalcNode(node *Node, recalc *[]*Node) {
	if node.ID == NodeGND || node.ID == NodePWR {
		return
	}

	c.recalcNodeGroup = c.recalcNodeGroup[:0]
	c.addSubNodesToGroup(node)

	newState := c.getNodeValue()
	for _, n := range c.recalcNodeGroup {
		n.InNodeGroup = false
		if n.State == newState {
			continue
		}
		n.State = newState
		for _, trans := range n.GateTransistors {
			if newState {
				c.turnTransistorOn(trans, recalc)
			} else {
				c.turnTransistorOff(trans, recalc)
			}
		}
	}
}

func (c *CPU) turnTransistorOn(trans *Transistor, recalc *[]*Node) {
	if trans.On {
		return
	}
	trans.On = true
	if trans.C1NodeID != NodeGND && trans.C1NodeID != NodePWR {
		n := c.nodes[trans.C1NodeID]
		if !n.InRecalcList {
			n.InRecalcList = true
			*recalc = append(*recalc, n)
		}
	}
}

func (c *CPU) turnTransistorOff(trans *Transistor, recalc *[]*Node) {
	if !trans.On {
		return
	}
	trans.On = false
	if trans.C1NodeID != NodeGND && trans.C1NodeID != NodePWR {
		n := c.nodes[trans.C1NodeID]
		if !n.InRecalcList {
			n.InRecalcList = true
			*recalc = append(*recalc, n)
		}
	}
	if trans.C2NodeID != NodeGND && trans.C2NodeID != NodePWR {
		n := c.nodes[trans.C2NodeID]
		if !n.InRecalcList {
			n.InRecalcList = true
			*recalc = append(*recalc, n)
		}
	}
}

func (c *CPU) recalcNodeList(initial []*Node) {
	// Move the caller's seed into our reused current slice, deduping
	// via the per-Node flag.
	c.recalcCurr = c.recalcCurr[:0]
	for _, n := range initial {
		if !n.InRecalcList {
			n.InRecalcList = true
			c.recalcCurr = append(c.recalcCurr, n)
		}
	}

	for len(c.recalcCurr) > 0 {
		c.recalcNext = c.recalcNext[:0]
		// Each node gets its flag cleared before recalcNode runs, so
		// it can be re-added to the next round if a transistor toggle
		// inside recalcNode would push it back in.
		for _, node := range c.recalcCurr {
			node.InRecalcList = false
			c.recalcNode(node, &c.recalcNext)
		}
		c.recalcCurr, c.recalcNext = c.recalcNext, c.recalcCurr
	}
}

func (c *CPU) setLow(node *Node) {
	node.PullUp = false
	node.PullDown = 1
	arr := [1]*Node{node}
	c.recalcNodeList(arr[:])
}

func (c *CPU) setHigh(node *Node) {
	node.PullUp = true
	node.PullDown = 0
	arr := [1]*Node{node}
	c.recalcNodeList(arr[:])
}

// Reset performs the standard 6502 power-on reset sequence.
func (c *CPU) Reset() {
	for i := 0; i < NodeDefCount; i++ {
		if c.nodes[i] != nil {
			c.nodes[i].State = false
			c.nodes[i].InNodeGroup = false
		}
	}

	c.gndNode = c.nodes[NodeGND]
	c.gndNode.State = false

	c.pwrNode = c.nodes[NodePWR]
	c.pwrNode.State = true

	for _, trans := range c.transistors {
		trans.On = false
	}

	clk0 := c.nodes[NodeCLK0]
	c.setLow(c.nodes[NodeRES])
	c.setLow(clk0)
	c.setHigh(c.nodes[NodeRDY])
	c.setLow(c.nodes[NodeSO])
	c.setHigh(c.nodes[NodeIRQ])
	c.setHigh(c.nodes[NodeNMI])

	allNodes := make([]*Node, 0, NodeDefCount)
	for i := 0; i < NodeDefCount; i++ {
		if c.nodes[i] != nil {
			allNodes = append(allNodes, c.nodes[i])
		}
	}
	c.recalcNodeList(allNodes)

	for i := 0; i < 8; i++ {
		c.setHigh(clk0)
		c.setLow(clk0)
	}

	c.setHigh(c.nodes[NodeRES])

	for i := 0; i < 6; i++ {
		c.setHigh(clk0)
		c.setLow(clk0)
	}
}

func (c *CPU) readAddressBus() uint16 {
	var address uint16
	for bit, nodeID := range AddressLineVals {
		if c.nodes[nodeID].State {
			address |= 1 << bit
		}
	}
	c.addressBits = address
	return address
}

func (c *CPU) readDataBus() uint8 {
	var data uint8
	for bit, nodeID := range DataLineVals {
		if c.nodes[nodeID].State {
			data |= 1 << bit
		}
	}
	c.dataBits = data
	return data
}

func (c *CPU) handleBusRead() {
	if c.nodes[NodeRW].State {
		address := c.readAddressBus()
		data := c.readFromBus(address)

		var list [8]*Node
		for bit, nodeID := range DataLineVals {
			node := c.nodes[nodeID]
			list[bit] = node
			if data&(1<<bit) != 0 {
				node.PullDown = 0
				node.PullUp = true
			} else {
				node.PullDown = 1
				node.PullUp = false
			}
		}
		c.recalcNodeList(list[:])
	}
}

func (c *CPU) handleBusWrite() {
	if !c.nodes[NodeRW].State {
		address := c.readAddressBus()
		data := c.readDataBus()
		c.writeToBus(address, data)
	}
}

// AddressBus reads the 16 address-line nodes and returns the live
// value on the address bus. Pure read — no side effects on cache.
func (c *CPU) AddressBus() uint16 {
	var address uint16
	for bit, nodeID := range AddressLineVals {
		if c.nodes[nodeID].State {
			address |= 1 << bit
		}
	}
	return address
}

// DataBus reads the 8 data-line nodes.
func (c *CPU) DataBus() uint8 {
	var data uint8
	for bit, nodeID := range DataLineVals {
		if c.nodes[nodeID].State {
			data |= 1 << bit
		}
	}
	return data
}

// IsReadCycle reports R/W pin state — true means the CPU is reading.
func (c *CPU) IsReadCycle() bool {
	n := c.nodes[NodeRW]
	return n != nil && n.State
}

// IRQ reports the IRQ pin state. Active low — true means inactive.
func (c *CPU) IRQ() bool {
	n := c.nodes[NodeIRQ]
	return n == nil || n.State
}

// NMI reports the NMI pin state. Active low — true means inactive.
func (c *CPU) NMI() bool {
	n := c.nodes[NodeNMI]
	return n == nil || n.State
}

// SYNC reports the SYNC pin state. Active high — true means the CPU
// is currently fetching an opcode (T1 cycle). External observers
// can use this to detect instruction boundaries without decoding
// the opcode stream themselves.
func (c *CPU) SYNC() bool {
	n := c.nodes[NodeSYNC]
	return n != nil && n.State
}

// HalfStep advances the simulation by half a clock cycle.
func (c *CPU) HalfStep() {
	clk0 := c.nodes[NodeCLK0]
	if clk0.State {
		c.setLow(clk0)
		c.handleBusRead()
	} else {
		c.setHigh(clk0)
		c.handleBusWrite()
	}
}

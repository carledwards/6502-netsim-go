package cpu

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// CPU represents the 6502 processor
type CPU struct {
	transistors     map[int]*Transistor
	nodes           [NodeDefCount]*Node
	gndNode         *Node
	pwrNode         *Node
	recalcNodeGroup []*Node
	readFromBus     ReadFromBus
	writeToBus      WriteToBus
	addressBits     uint16
	dataBits        uint8
}

// New creates a new CPU instance
func New(readFromBus ReadFromBus, writeToBus WriteToBus, transDefsPath, segDefsPath string) (*CPU, error) {
	cpu := &CPU{
		transistors:     make(map[int]*Transistor),
		recalcNodeGroup: make([]*Node, 0),
		readFromBus:     readFromBus,
		writeToBus:      writeToBus,
	}

	if err := cpu.setupTransistors(transDefsPath); err != nil {
		return nil, err
	}

	if err := cpu.setupNodes(segDefsPath); err != nil {
		return nil, err
	}

	cpu.connectTransistors()
	return cpu, nil
}

func (c *CPU) setupTransistors(path string) error {
	fmt.Printf("Opening transistor definitions from: %s\n", path)
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open transistor definitions file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
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

	fmt.Printf("Loaded %d transistors\n", len(c.transistors))
	return scanner.Err()
}

func (c *CPU) setupNodes(path string) error {
	fmt.Printf("Opening segment definitions from: %s\n", path)
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open segment definitions file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
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

	nodeCount := 0
	for i := 0; i < NodeDefCount; i++ {
		if c.nodes[i] != nil {
			nodeCount++
		}
	}
	fmt.Printf("Loaded %d nodes\n", nodeCount)
	return scanner.Err()
}

func (c *CPU) connectTransistors() {
	fmt.Printf("Connecting transistors to nodes...\n")
	for _, trans := range c.transistors {
		// Create nodes if they don't exist
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

		// Connect transistors to nodes
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

func (c *CPU) recalcNode(node *Node, recalcList map[int]*Node) {
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
				c.turnTransistorOn(trans, recalcList)
			} else {
				c.turnTransistorOff(trans, recalcList)
			}
		}
	}
}

func (c *CPU) turnTransistorOn(trans *Transistor, recalcList map[int]*Node) {
	if trans.On {
		return
	}
	trans.On = true
	if trans.C1NodeID != NodeGND && trans.C1NodeID != NodePWR {
		recalcList[trans.C1NodeID] = c.nodes[trans.C1NodeID]
	}
}

func (c *CPU) turnTransistorOff(trans *Transistor, recalcList map[int]*Node) {
	if !trans.On {
		return
	}
	trans.On = false
	if trans.C1NodeID != NodeGND && trans.C1NodeID != NodePWR {
		recalcList[trans.C1NodeID] = c.nodes[trans.C1NodeID]
	}
	if trans.C2NodeID != NodeGND && trans.C2NodeID != NodePWR {
		recalcList[trans.C2NodeID] = c.nodes[trans.C2NodeID]
	}
}

func (c *CPU) recalcNodeList(list map[int]*Node) {
	// Reuse maps to avoid allocations
	currentList := list
	nextList := make(map[int]*Node, len(list)) // Pre-allocate with initial capacity

	for len(currentList) > 0 {
		// Clear nextList without deallocating
		for k := range nextList {
			delete(nextList, k)
		}

		// Process current nodes
		for _, node := range currentList {
			c.recalcNode(node, nextList)
		}

		// Swap lists
		currentList, nextList = nextList, currentList
	}
}

func (c *CPU) setLow(node *Node) {
	node.PullUp = false
	node.PullDown = 1
	list := map[int]*Node{node.ID: node}
	c.recalcNodeList(list)
}

func (c *CPU) setHigh(node *Node) {
	node.PullUp = true
	node.PullDown = 0
	list := map[int]*Node{node.ID: node}
	c.recalcNodeList(list)
}

// Reset resets the CPU to its initial state
func (c *CPU) Reset() {
	fmt.Println("Starting CPU reset...")
	// Reset all nodes
	for i := 0; i < NodeDefCount; i++ {
		if c.nodes[i] != nil {
			c.nodes[i].State = false
			c.nodes[i].InNodeGroup = false
		}
	}

	c.gndNode = c.nodes[NodeGND]
	if c.gndNode == nil {
		fmt.Printf("Warning: GND node (ID: %d) not found\n", NodeGND)
	}
	c.gndNode.State = false

	c.pwrNode = c.nodes[NodePWR]
	if c.pwrNode == nil {
		fmt.Printf("Warning: PWR node (ID: %d) not found\n", NodePWR)
	}
	c.pwrNode.State = true

	// Reset all transistors
	for _, trans := range c.transistors {
		trans.On = false
	}

	clk0 := c.nodes[NodeCLK0]
	if clk0 == nil {
		fmt.Printf("Warning: CLK0 node (ID: %d) not found\n", NodeCLK0)
	}
	c.setLow(c.nodes[NodeRES])
	c.setLow(clk0)
	c.setHigh(c.nodes[NodeRDY])
	c.setLow(c.nodes[NodeSO])
	c.setHigh(c.nodes[NodeIRQ])
	c.setHigh(c.nodes[NodeNMI])

	// Initial recalc of all nodes
	allNodes := make(map[int]*Node)
	for i := 0; i < NodeDefCount; i++ {
		if c.nodes[i] != nil {
			allNodes[i] = c.nodes[i]
		}
	}
	c.recalcNodeList(allNodes)

	// Initial clock cycles
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
		data := c.readFromBus(int(address))

		// Update data bus nodes
		list := make(map[int]*Node)
		for bit, nodeID := range DataLineVals {
			node := c.nodes[nodeID]
			list[nodeID] = node
			if data&(1<<bit) != 0 {
				node.PullDown = 0
				node.PullUp = true
			} else {
				node.PullDown = 1
				node.PullUp = false
			}
		}
		c.recalcNodeList(list)
	}
}

func (c *CPU) handleBusWrite() {
	if !c.nodes[NodeRW].State {
		address := c.readAddressBus()
		data := c.readDataBus()
		c.writeToBus(int(address), data)
	}
}

// HalfStep performs half a clock cycle
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

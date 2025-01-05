package motherboard

import (
	"github.com/carledwards/6502-netsim-go/src/cpu"
	"github.com/carledwards/6502-netsim-go/src/memory"
)

const (
	RamSize     = 8 * 1024 // 8K
	RomSize     = 8 * 1024 // 8K
	RamBaseAddr = 0x0000
	RomBaseAddr = 0xE000
)

// Motherboard represents the main system board
type Motherboard struct {
	cpu *cpu.CPU
	ram *memory.Memory
	rom *memory.Memory
}

// New creates a new Motherboard instance
func New(transDefsPath, segDefsPath string) (*Motherboard, error) {
	m := &Motherboard{
		ram: memory.New(RamSize, false),
		rom: memory.New(RomSize, true),
	}

	var err error
	m.cpu, err = cpu.New(m.readFromBus, m.writeToBus, transDefsPath, segDefsPath)
	if err != nil {
		return nil, err
	}

	m.cpu.Reset()
	return m, nil
}

// readFromBus handles memory reads from the CPU
func (m *Motherboard) readFromBus(addr int) uint8 {
	if addr < RamBaseAddr+RamSize {
		return m.ram.Read(addr)
	} else if addr >= RomBaseAddr {
		return m.rom.Read(addr - RomBaseAddr)
	}
	return 0x00
}

// writeToBus handles memory writes from the CPU
func (m *Motherboard) writeToBus(addr int, val uint8) {
	if addr < RamBaseAddr+RamSize {
		m.ram.Write(addr, val)
	}
	// Writes to ROM are ignored
}

// ClockTick performs one clock cycle
func (m *Motherboard) ClockTick() {
	m.cpu.HalfStep()
}

// Run continuously runs the CPU
func (m *Motherboard) Run() {
	for {
		m.ClockTick()
	}
}

// GetROM returns the ROM memory for initialization
func (m *Motherboard) GetROM() *memory.Memory {
	return m.rom
}

// GetRAM returns the RAM memory for debugging
func (m *Motherboard) GetRAM() *memory.Memory {
	return m.ram
}

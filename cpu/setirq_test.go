package cpu

import "testing"

// SetIRQ drives the active-low IRQ netlist node. After Reset the pin is
// released (inactive, high → IRQ() true). Asserting pulls it low
// (IRQ() false); deasserting releases it again. This is the host hook
// a peripheral's IRQ output wires to.
func TestSetIRQDrivesThePin(t *testing.T) {
	mem := make([]uint8, 0x10000)
	mem[0xFFFC] = 0x00 // reset vector → $E000
	mem[0xFFFD] = 0xE0
	mem[0xE000] = 0xEA // NOP

	c, err := New(
		func(a uint16) uint8 { return mem[a] },
		func(a uint16, v uint8) { mem[a] = v },
	)
	if err != nil {
		t.Fatal(err)
	}
	c.Reset()

	if !c.IRQ() {
		t.Fatal("after Reset the IRQ pin must be inactive (high) → IRQ() true")
	}
	c.SetIRQ(true)
	if c.IRQ() {
		t.Fatal("SetIRQ(true) must pull IRQ active (low) → IRQ() false")
	}
	c.SetIRQ(false)
	if !c.IRQ() {
		t.Fatal("SetIRQ(false) must release IRQ inactive (high) → IRQ() true")
	}
}

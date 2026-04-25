package cpu

import "testing"

// Run a known program and check the architectural state matches what
// the instructions should produce. This validates both that the CPU
// is executing correctly and that the register accessors are reading
// from the right storage nodes.
func TestRegistersAfterProgram(t *testing.T) {
	rom := make([]uint8, 0x2000) // $E000-$FFFF

	// $E000: LDA #$42       ; A = $42
	// $E002: LDX #$11       ; X = $11
	// $E004: LDY #$22       ; Y = $22
	// $E006: STA $00        ; mem[0] = $42
	// $E008: STX $01        ; mem[1] = $11
	// $E00A: STY $02        ; mem[2] = $22
	// $E00C: JMP $E00C      ; spin
	prog := []uint8{
		0xA9, 0x42,
		0xA2, 0x11,
		0xA0, 0x22,
		0x8D, 0x00, 0x00,
		0x8E, 0x01, 0x00,
		0x8C, 0x02, 0x00,
		0x4C, 0x0C, 0xE0,
	}
	copy(rom, prog)
	rom[0x1FFC] = 0x00 // reset vector → $E000
	rom[0x1FFD] = 0xE0

	ram := make([]uint8, 0x100)
	read := func(addr uint16) uint8 {
		if addr >= 0xE000 {
			return rom[addr-0xE000]
		}
		if addr < 0x100 {
			return ram[addr]
		}
		return 0
	}
	write := func(addr uint16, val uint8) {
		if addr < 0x100 {
			ram[addr] = val
		}
	}

	c, err := New(read, write)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.Reset()

	// Run plenty of half-cycles. The full program is ~17 cycles ≈ 34
	// half-cycles; 200 leaves comfortable margin and lets the spin
	// loop go around a few times.
	for i := 0; i < 200; i++ {
		c.HalfStep()
	}

	regs := c.Registers()
	if regs.A != 0x42 {
		t.Errorf("A: got $%02X, want $42", regs.A)
	}
	if regs.X != 0x11 {
		t.Errorf("X: got $%02X, want $11", regs.X)
	}
	if regs.Y != 0x22 {
		t.Errorf("Y: got $%02X, want $22", regs.Y)
	}
	if ram[0] != 0x42 {
		t.Errorf("ram[0]: got $%02X, want $42 (STA didn't fire)", ram[0])
	}
	if ram[1] != 0x11 {
		t.Errorf("ram[1]: got $%02X, want $11 (STX didn't fire)", ram[1])
	}
	if ram[2] != 0x22 {
		t.Errorf("ram[2]: got $%02X, want $22 (STY didn't fire)", ram[2])
	}

	// PC should be parked somewhere in the spin loop. The loop is
	// STY $02 (3 bytes) + JMP $E00C (3 bytes), so PC lands anywhere
	// in $E00C..$E011 depending on which half-cycle we stopped at.
	if regs.PC < 0xE00C || regs.PC > 0xE011 {
		t.Errorf("PC: got $%04X, want in spin loop $E00C..$E011", regs.PC)
	}
}

// Reset should leave the CPU with reasonable state and the address
// bus at the reset vector contents.
func TestResetRegisters(t *testing.T) {
	rom := make([]uint8, 0x2000)
	rom[0x0000] = 0xEA // NOP
	rom[0x0001] = 0x4C // JMP $E000
	rom[0x0002] = 0x00
	rom[0x0003] = 0xE0
	rom[0x1FFC] = 0x00
	rom[0x1FFD] = 0xE0

	read := func(addr uint16) uint8 {
		if addr >= 0xE000 {
			return rom[addr-0xE000]
		}
		return 0
	}
	write := func(addr uint16, val uint8) {}

	c, err := New(read, write)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.Reset()

	// Run just a few cycles to settle past the reset sequence.
	for i := 0; i < 20; i++ {
		c.HalfStep()
	}

	regs := c.Registers()
	// Stack pointer is undefined per real hardware but should be
	// reading some value (i.e. accessor doesn't panic).
	_ = regs.S
	// PC should be in our ROM region.
	if regs.PC < 0xE000 {
		t.Errorf("PC after reset: got $%04X, want >= $E000", regs.PC)
	}
}

package cpu

import "testing"

// Smoke test: build the CPU, reset it, run a handful of half-cycles
// against a stub bus, confirm we observed the reset vector being
// fetched (canonical behavior is that CPU reads $FFFC then $FFFD
// shortly after reset).
func TestResetFetchesVector(t *testing.T) {
	rom := make([]uint8, 0x2000) // covers $E000-$FFFF
	rom[0x1FFC] = 0x00           // reset vector low → $E000
	rom[0x1FFD] = 0xE0
	// Place a NOP-ish loop at $E000 so the CPU has something to fetch
	// when it gets there.
	rom[0x0000] = 0xEA // NOP
	rom[0x0001] = 0x4C // JMP
	rom[0x0002] = 0x00 // $E000
	rom[0x0003] = 0xE0

	addrs := make(map[uint16]int)
	read := func(addr uint16) uint8 {
		addrs[addr]++
		if addr >= 0xE000 {
			return rom[addr-0xE000]
		}
		return 0x00
	}
	write := func(addr uint16, val uint8) {}

	c, err := New(read, write)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.Reset()

	for i := 0; i < 200; i++ {
		c.HalfStep()
	}

	if addrs[0xFFFC] == 0 {
		t.Errorf("expected CPU to fetch reset vector low at $FFFC")
	}
	if addrs[0xFFFD] == 0 {
		t.Errorf("expected CPU to fetch reset vector high at $FFFD")
	}
	if addrs[0xE000] == 0 {
		t.Errorf("expected CPU to fetch instruction at $E000 after reset")
	}
}

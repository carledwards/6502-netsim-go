// Benchmark drives the netsim CPU against a stub memory map and
// reports elapsed time. Useful as a smoke test and for capturing
// CPU profiles.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/carledwards/6502-netsim-go/cpu"
)

const (
	ramSize = 8 * 1024
	romBase = 0xE000
	romSize = 8 * 1024
)

// Tiny test program: load #$50 into A, store at $1000, decrement that
// location, jump back. Just keeps the CPU busy.
var appCode = []uint8{
	0xA9, 0x50, // lda #$50
	0x8D, 0x00, 0x10, // sta $1000
	0xCE, 0x00, 0x10, // dec $1000
	0x4C, 0x05, 0xE0, // jmp $E005
}

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	ticks := flag.Int("ticks", 10000, "number of half-clock ticks to run")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	ram := make([]uint8, ramSize)
	rom := make([]uint8, romSize)
	copy(rom, appCode)
	rom[0x1FFC] = 0x00 // reset vector low
	rom[0x1FFD] = 0xE0 // reset vector high → $E000

	read := func(addr uint16) uint8 {
		switch {
		case addr < ramSize:
			return ram[addr]
		case addr >= romBase:
			return rom[addr-romBase]
		}
		return 0
	}
	write := func(addr uint16, val uint8) {
		if addr < ramSize {
			ram[addr] = val
		}
	}

	c, err := cpu.New(read, write)
	if err != nil {
		log.Fatalf("cpu init: %v", err)
	}
	c.Reset()

	start := time.Now()
	for i := 0; i < *ticks; i++ {
		c.HalfStep()
	}
	elapsed := time.Since(start)
	fmt.Printf("ticks=%d elapsed=%dms (%.0f ticks/s)\n",
		*ticks, elapsed.Milliseconds(), float64(*ticks)/elapsed.Seconds())
}

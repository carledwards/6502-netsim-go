package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/carledwards/6502-netsim-go/src/motherboard"
)

var appCode = []uint8{
	0xA9, 0x50, // lda #$FF
	0x8D, 0x00, 0x10, // sta $1000
	0xCE, 0x00, 0x10, // dec $1000
	0x4C, 0x05, 0xE0, // jmp $E002
}

func main() {
	// Get paths relative to current directory
	transDefsPath := filepath.Join("data", "transdefs.txt")
	segDefsPath := filepath.Join("data", "segdefs.txt")

	fmt.Printf("Loading definition files:\n  trans: %s\n  segs: %s\n", transDefsPath, segDefsPath)

	// Initialize motherboard
	mb, err := motherboard.New(transDefsPath, segDefsPath)
	if err != nil {
		log.Fatalf("Failed to initialize motherboard: %v", err)
	}

	// Initialize ROM with test program
	rom := mb.GetROM()
	for i, v := range appCode {
		rom.Write(i, v)
	}

	// Set 6502 reset vectors for starting address of app: $E000
	rom.Write(0x1FFC, 0x00)
	rom.Write(0x1FFD, 0xE0)

	// Run CPU for a short time and measure performance
	start := time.Now()
	for i := 0; i < 10000; i++ {
		mb.ClockTick()
	}
	elapsed := time.Since(start)
	fmt.Printf("Elapsed time: %v\n", elapsed.Milliseconds())
}

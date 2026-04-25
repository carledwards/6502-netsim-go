# 6502-netsim-go

A Go transistor-level simulator of the MOS Technology 6502, derived
from the [Visual6502](https://github.com/trebonian/visual6502) project.
Drop it in your own circuit by supplying read / write callbacks for
the data bus.

## Project structure

```
.
├── cpu/
│   ├── cpu.go            # transistor sim core
│   ├── types.go          # node/transistor types, bus signal IDs
│   ├── cpu_test.go       # smoke test
│   └── data/             # transistor & segment definitions (embedded)
│       ├── segdefs.txt
│       └── transdefs.txt
└── cmd/
    └── benchmark/
        └── main.go       # tiny perf harness, supports -cpuprofile
```

The data files are embedded into the binary via `go:embed`, so
consumers don't need to ship them separately.

## Usage

```go
import "github.com/carledwards/6502-netsim-go/cpu"

read := func(addr uint16) uint8 {
    // return memory contents at addr
}
write := func(addr uint16, val uint8) {
    // store val at addr
}

c, err := cpu.New(read, write)
if err != nil { ... }
c.Reset()

for {
    c.HalfStep() // advance half a clock cycle
}
```

`HalfStep` is the lowest-granularity API. The bus callbacks are
invoked during the read and write phases of each cycle.

## Building and running

### Prerequisites

- Go 1.21+

### Build the benchmark

```bash
go build -o bin/benchmark ./cmd/benchmark
./bin/benchmark -ticks 10000
```

Output:

```
ticks=10000 elapsed=684ms (14620 ticks/s)
```

### Test

```bash
go test ./...
```

## Profiling

The benchmark accepts `-cpuprofile`:

```bash
go build -o bin/benchmark ./cmd/benchmark
./bin/benchmark -cpuprofile=cpu.prof -ticks 50000
go tool pprof cpu.prof
```

Common pprof commands:

- `top` — top CPU consumers
- `web` — browser visualization (requires graphviz)
- `list <funcname>` — source-level profiling
- `peek <regexp>` — match by regexp

## Attribution

Based on [Visual6502](https://github.com/trebonian/visual6502)
([www.visual6502.org](http://www.visual6502.org)) by Greg James, Brian
Silverman, and Barry Silverman. Original work is licensed under
[CC BY-NC-SA 3.0](http://creativecommons.org/licenses/by-nc-sa/3.0/).

The transistor and segment definition files in `cpu/data/` come from
that project and describe the internal structure of the 6502 die.

## License

MIT — see [LICENSE](LICENSE).

# 6502-netsim-go

A Go implementation of a 6502 processor simulator, derived from the Visual6502 project. This simulator provides a detailed emulation of the MOS Technology 6502 processor, including transistor-level simulation capabilities.

## Project Structure

```
.
├── src/          # Source code directory
│   ├── cpu/      # CPU implementation
│   ├── memory/   # Memory management
│   └── motherboard/ # Motherboard simulation
├── data/         # Simulation data files
│   ├── segdefs.txt    # Segment definitions from Visual6502
│   └── transdefs.txt  # Transistor definitions from Visual6502
```

## Building and Running

### Prerequisites

- Go 1.21 or later
- Git

### Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/carledwards/6502-netsim-go.git
   cd 6502-netsim-go
   ```

2. Build the project:
   ```bash
   go build ./src
   ```

3. Run the simulator:
   ```bash
   ./main
   ```

## Profiling

The simulator includes CPU profiling capabilities to help analyze performance. To use the profiler:

1. Run the simulator with profiling enabled:
   ```bash
   ./6502-netsim-go -cpuprofile=cpu.prof
   ```

2. Analyze the profile using Go's pprof tool:
   ```bash
   go tool pprof cpu.prof
   ```

   Common pprof commands:
   - `top`: Shows the top CPU consumers
   - `web`: Opens browser visualization (requires graphviz)
   - `list <funcname>`: Shows source-level profiling data
   - `peek <regexp>`: Shows functions matching regexp

You can generate multiple profiles with different names for comparison:
```bash
./6502-netsim-go -cpuprofile=before.prof
# Make changes
./6502-netsim-go -cpuprofile=after.prof
```

## Attribution

This project is based on the work from [Visual6502](https://github.com/trebonian/visual6502) (www.visual6502.org), originally created by Greg James, Brian Silverman, and Barry Silverman. The original work was licensed under [Creative Commons Attribution-NonCommercial-ShareAlike 3.0](http://creativecommons.org/licenses/by-nc-sa/3.0/).

The segment and transistor definition files used in this project are derived from the Visual6502 project and are essential components for the transistor-level simulation of the 6502 processor. These files contain the detailed mapping of the processor's internal structure and connections.

## Data Files

- `data/segdefs.txt`: Contains segment definitions that describe the various components and pathways within the 6502 processor
- `data/transdefs.txt`: Contains transistor definitions that specify the switching elements and their connections within the processor

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

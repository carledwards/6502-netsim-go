package memory

// Memory represents either RAM or ROM memory
type Memory struct {
	size       int
	isReadOnly bool
	memory     []uint8
}

// New creates a new Memory instance
func New(size int, isReadOnly bool) *Memory {
	return &Memory{
		size:       size,
		isReadOnly: isReadOnly,
		memory:     make([]uint8, size),
	}
}

// Write writes data to memory if not read-only
func (m *Memory) Write(address int, data uint8) {
	if !m.isReadOnly {
		m.memory[address] = data
	}
}

// Read reads data from memory
func (m *Memory) Read(address int) uint8 {
	return m.memory[address]
}

// Reset resets memory to zero if not read-only
func (m *Memory) Reset() {
	if !m.isReadOnly {
		for i := range m.memory {
			m.memory[i] = 0
		}
	}
}

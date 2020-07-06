package internals

import "sync"

// MapsHashValueOffset defines an interface to look up the
type MapsHashValueOffset interface {
	Add(reportFile string, hashValue Hash, offset uint64)
	Offset(reportFile string, hashValue Hash) uint64
}

// GetMapsHashValueOffset returns the MapsHashValueOffset
// corresponding to hashValueSize (in bytes) or nil
func GetMapsHashValueOffset(hashValueSize int) MapsHashValueOffset {
	switch hashValueSize {
	case 4:
		return InitMapsHashValueOffset32()
	case 8:
		return InitMapsHashValueOffset64()
	case 16:
		return InitMapsHashValueOffset128()
	case 20:
		return InitMapsHashValueOffset160()
	case 32:
		return InitMapsHashValueOffset256()
	case 64:
		return InitMapsHashValueOffset512()
	default:
		return nil
	}
}

// MapsHashValueOffset32 implements MapsHashValueOffset for hash values of 32 bits
type MapsHashValueOffset32 struct {
	mut  *sync.Mutex
	data map[string]map[[4]byte]uint64
}

// InitMapsHashValueOffset32 returns a newly-initialized instance
func InitMapsHashValueOffset32() *MapsHashValueOffset32 {
	inst := new(MapsHashValueOffset32)
	inst.data = make(map[string]map[[4]byte]uint64)
	inst.mut = new(sync.Mutex)
	return inst
}

// Add adds a hashvalue-offset association to the map.
func (m *MapsHashValueOffset32) Add(reportFile string, hashValue Hash, offset uint64) {
	var key [4]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	m.data[reportFile][key] = offset
}

// Offset returns the value associated with key hashValue
func (m *MapsHashValueOffset32) Offset(reportFile string, hashValue Hash) uint64 {
	var key [4]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	val, ok := m.data[reportFile][key]
	if !ok {
		return 0
	}
	return val
}

// MapsHashValueOffset64 implements MapsHashValueOffset for hash values of 64 bits
type MapsHashValueOffset64 struct {
	mut  *sync.Mutex
	data map[string]map[[8]byte]uint64
}

// InitMapsHashValueOffset64 returns a newly-initialized instance
func InitMapsHashValueOffset64() *MapsHashValueOffset64 {
	inst := new(MapsHashValueOffset64)
	inst.data = make(map[string]map[[8]byte]uint64)
	inst.mut = new(sync.Mutex)
	return inst
}

// Add adds a hashvalue-offset association to the map.
func (m *MapsHashValueOffset64) Add(reportFile string, hashValue Hash, offset uint64) {
	var key [8]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	m.data[reportFile][key] = offset
}

// Offset returns the value associated with key hashValue
func (m *MapsHashValueOffset64) Offset(reportFile string, hashValue Hash) uint64 {
	var key [8]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	val, ok := m.data[reportFile][key]
	if !ok {
		return 0
	}
	return val
}

// MapsHashValueOffset128 implements MapsHashValueOffset for hash values of 128 bits
type MapsHashValueOffset128 struct {
	mut  *sync.Mutex
	data map[string]map[[16]byte]uint64
}

// InitMapsHashValueOffset128 returns a newly-initialized instance
func InitMapsHashValueOffset128() *MapsHashValueOffset128 {
	inst := new(MapsHashValueOffset128)
	inst.data = make(map[string]map[[16]byte]uint64)
	inst.mut = new(sync.Mutex)
	return inst
}

// Add adds a hashvalue-offset association to the map.
func (m *MapsHashValueOffset128) Add(reportFile string, hashValue Hash, offset uint64) {
	var key [16]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	m.data[reportFile][key] = offset
}

// Offset returns the value associated with key hashValue
func (m *MapsHashValueOffset128) Offset(reportFile string, hashValue Hash) uint64 {
	var key [16]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	val, ok := m.data[reportFile][key]
	if !ok {
		return 0
	}
	return val
}

// MapsHashValueOffset160 implements MapsHashValueOffset for hash values of 160 bits
type MapsHashValueOffset160 struct {
	mut  *sync.Mutex
	data map[string]map[[20]byte]uint64
}

// InitMapsHashValueOffset160 returns a newly-initialized instance
func InitMapsHashValueOffset160() *MapsHashValueOffset160 {
	inst := new(MapsHashValueOffset160)
	inst.data = make(map[string]map[[20]byte]uint64)
	inst.mut = new(sync.Mutex)
	return inst
}

// Add adds a hashvalue-offset association to the map.
func (m *MapsHashValueOffset160) Add(reportFile string, hashValue Hash, offset uint64) {
	var key [20]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	m.data[reportFile][key] = offset
}

// Offset returns the value associated with key hashValue
func (m *MapsHashValueOffset160) Offset(reportFile string, hashValue Hash) uint64 {
	var key [20]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	val, ok := m.data[reportFile][key]
	if !ok {
		return 0
	}
	return val
}

// MapsHashValueOffset256 implements MapsHashValueOffset for hash values of 256 bits
type MapsHashValueOffset256 struct {
	mut  *sync.Mutex
	data map[string]map[[32]byte]uint64
}

// InitMapsHashValueOffset256 returns a newly-initialized instance
func InitMapsHashValueOffset256() *MapsHashValueOffset256 {
	inst := new(MapsHashValueOffset256)
	inst.data = make(map[string]map[[32]byte]uint64)
	inst.mut = new(sync.Mutex)
	return inst
}

// Add adds a hashvalue-offset association to the map.
func (m *MapsHashValueOffset256) Add(reportFile string, hashValue Hash, offset uint64) {
	var key [32]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	m.data[reportFile][key] = offset
}

// Offset returns the value associated with key hashValue
func (m *MapsHashValueOffset256) Offset(reportFile string, hashValue Hash) uint64 {
	var key [32]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	val, ok := m.data[reportFile][key]
	if !ok {
		return 0
	}
	return val
}

// MapsHashValueOffset512 implements MapsHashValueOffset for hash values of 512 bits
type MapsHashValueOffset512 struct {
	mut  *sync.Mutex
	data map[string]map[[64]byte]uint64
}

// InitMapsHashValueOffset512 returns a newly-initialized instance
func InitMapsHashValueOffset512() *MapsHashValueOffset512 {
	inst := new(MapsHashValueOffset512)
	inst.data = make(map[string]map[[64]byte]uint64)
	inst.mut = new(sync.Mutex)
	return inst
}

// Add adds a hashvalue-offset association to the map.
func (m *MapsHashValueOffset512) Add(reportFile string, hashValue Hash, offset uint64) {
	var key [64]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	m.data[reportFile][key] = offset
}

// Offset returns the value associated with key hashValue
func (m *MapsHashValueOffset512) Offset(reportFile string, hashValue Hash) uint64 {
	var key [64]byte
	copy(key[:], hashValue)
	m.mut.Lock()
	defer m.mut.Unlock()
	val, ok := m.data[reportFile][key]
	if !ok {
		return 0
	}
	return val
}

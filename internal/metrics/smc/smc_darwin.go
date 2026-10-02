//go:build darwin

// Package smc reads values from the Apple System Management Controller using
// IOKit via purego (no cgo). Reading SMC keys does not require root.
package smc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	kernelIndexSMC = 2
	cmdReadBytes   = 5
	cmdReadKeyInfo = 9
)

type keyDataVers struct {
	Major, Minor, Build, Reserved uint8
	Release                       uint16
}

type keyDataPLimit struct {
	Version, Length uint16
	CPU, GPU, Mem   uint32
}

type keyInfo struct {
	DataSize       uint32
	DataType       uint32
	DataAttributes uint8
}

// keyData mirrors SMCKeyData_t (80 bytes).
type keyData struct {
	Key     uint32
	Vers    keyDataVers
	PLimit  keyDataPLimit
	KeyInfo keyInfo
	Result  uint8
	Status  uint8
	Data8   uint8
	Data32  uint32
	Bytes   [32]byte
}

var (
	once    sync.Once
	initErr error

	ioServiceMatching           func(name string) uintptr
	ioServiceGetMatchingService func(mainPort uint32, matching uintptr) uint32
	ioServiceOpen               func(service uint32, owningTask uint32, typ uint32, conn *uint32) int32
	ioServiceClose              func(conn uint32) int32
	ioObjectRelease             func(obj uint32) int32
	ioConnectCallStructMethod   func(conn uint32, selector uint32, in unsafe.Pointer, inSize uintptr, out unsafe.Pointer, outSize *uintptr) int32
	taskSelf                    uint32
)

func load() {
	iokit, err := purego.Dlopen("/System/Library/Frameworks/IOKit.framework/IOKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		initErr = err
		return
	}
	sys, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		initErr = err
		return
	}
	purego.RegisterLibFunc(&ioServiceMatching, iokit, "IOServiceMatching")
	purego.RegisterLibFunc(&ioServiceGetMatchingService, iokit, "IOServiceGetMatchingService")
	purego.RegisterLibFunc(&ioServiceOpen, iokit, "IOServiceOpen")
	purego.RegisterLibFunc(&ioServiceClose, iokit, "IOServiceClose")
	purego.RegisterLibFunc(&ioObjectRelease, iokit, "IOObjectRelease")
	purego.RegisterLibFunc(&ioConnectCallStructMethod, iokit, "IOConnectCallStructMethod")
	sym, err := purego.Dlsym(sys, "mach_task_self_")
	if err != nil {
		initErr = err
		return
	}
	taskSelf = **(**uint32)(unsafe.Pointer(&sym))
}

// Conn is an open connection to the SMC. It is not safe for concurrent use.
type Conn struct {
	conn  uint32
	cache map[string]keyInfo
}

// Open opens a connection to AppleSMC.
func Open() (*Conn, error) {
	once.Do(load)
	if initErr != nil {
		return nil, initErr
	}
	svc := ioServiceGetMatchingService(0, ioServiceMatching("AppleSMC"))
	if svc == 0 {
		return nil, errors.New("AppleSMC service not found")
	}
	defer ioObjectRelease(svc)
	var c uint32
	if r := ioServiceOpen(svc, taskSelf, 0, &c); r != 0 {
		return nil, fmt.Errorf("IOServiceOpen failed: %#x", r)
	}
	return &Conn{conn: c, cache: map[string]keyInfo{}}, nil
}

// Close releases the connection.
func (c *Conn) Close() {
	if c != nil && c.conn != 0 {
		ioServiceClose(c.conn)
		c.conn = 0
	}
}

func fourCC(s string) uint32 {
	if len(s) != 4 {
		return 0
	}
	return binary.BigEndian.Uint32([]byte(s))
}

func (c *Conn) call(in *keyData) (keyData, error) {
	var out keyData
	outSize := unsafe.Sizeof(out)
	r := ioConnectCallStructMethod(c.conn, kernelIndexSMC, unsafe.Pointer(in), unsafe.Sizeof(*in), unsafe.Pointer(&out), &outSize)
	if r != 0 {
		return out, fmt.Errorf("SMC call failed: %#x", r)
	}
	if out.Result != 0 {
		return out, fmt.Errorf("SMC key error: %d", out.Result)
	}
	return out, nil
}

// Value is a decoded SMC value.
type Value struct {
	Type  string
	Bytes []byte
}

// Read reads a raw key.
func (c *Conn) Read(key string) (Value, error) {
	info, ok := c.cache[key]
	if !ok {
		in := keyData{Key: fourCC(key), Data8: cmdReadKeyInfo}
		out, err := c.call(&in)
		if err != nil {
			return Value{}, err
		}
		info = out.KeyInfo
		c.cache[key] = info
	}
	if info.DataSize == 0 || info.DataSize > 32 {
		return Value{}, errors.New("unsupported key size")
	}
	in := keyData{Key: fourCC(key), Data8: cmdReadBytes, KeyInfo: info}
	out, err := c.call(&in)
	if err != nil {
		return Value{}, err
	}
	var t [4]byte
	binary.BigEndian.PutUint32(t[:], info.DataType)
	return Value{Type: string(t[:]), Bytes: append([]byte(nil), out.Bytes[:info.DataSize]...)}, nil
}

// Float decodes common numeric SMC types into a float64.
func (v Value) Float() (float64, bool) {
	b := v.Bytes
	switch v.Type {
	case "flt ":
		if len(b) >= 4 {
			f := float64(math.Float32frombits(binary.LittleEndian.Uint32(b)))
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return 0, false
			}
			return f, true
		}
	case "fpe2":
		if len(b) >= 2 {
			return float64(binary.BigEndian.Uint16(b)) / 4, true
		}
	case "sp78":
		if len(b) >= 2 {
			return float64(int16(binary.BigEndian.Uint16(b))) / 256, true
		}
	case "ui8 ":
		if len(b) >= 1 {
			return float64(b[0]), true
		}
	case "ui16":
		if len(b) >= 2 {
			return float64(binary.BigEndian.Uint16(b)), true
		}
	case "ui32":
		if len(b) >= 4 {
			return float64(binary.BigEndian.Uint32(b)), true
		}
	}
	return 0, false
}

// ReadFloat reads a key and decodes it as a number.
func (c *Conn) ReadFloat(key string) (float64, error) {
	v, err := c.Read(key)
	if err != nil {
		return 0, err
	}
	f, ok := v.Float()
	if !ok {
		return 0, fmt.Errorf("unsupported SMC type %q", v.Type)
	}
	return f, nil
}

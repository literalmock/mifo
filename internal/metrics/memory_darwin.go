//go:build darwin

package metrics

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/shirou/gopsutil/v4/mem"
	"golang.org/x/sys/unix"
)

// vmStatistics64 mirrors struct vm_statistics64 from <mach/vm_statistics.h>.
type vmStatistics64 struct {
	FreeCount                          uint32
	ActiveCount                        uint32
	InactiveCount                      uint32
	WireCount                          uint32
	ZeroFillCount                      uint64
	Reactivations                      uint64
	Pageins                            uint64
	Pageouts                           uint64
	Faults                             uint64
	CowFaults                          uint64
	Lookups                            uint64
	Hits                               uint64
	Purges                             uint64
	PurgeableCount                     uint32
	SpeculativeCount                   uint32
	Decompressions                     uint64
	Compressions                       uint64
	Swapins                            uint64
	Swapouts                           uint64
	CompressorPageCount                uint32
	ThrottledCount                     uint32
	ExternalPageCount                  uint32
	InternalPageCount                  uint32
	TotalUncompressedPagesInCompressor uint64
}

const hostVMInfo64 = 4

var (
	machOnce          sync.Once
	machErr           error
	machHostSelf      func() uint32
	hostStatistics64  func(host uint32, flavor int32, info unsafe.Pointer, count *uint32) int32
	machPortDealloc   func(task uint32, name uint32) int32
	machTaskSelfValue uint32
)

func loadMach() {
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		machErr = err
		return
	}
	purego.RegisterLibFunc(&machHostSelf, lib, "mach_host_self")
	purego.RegisterLibFunc(&hostStatistics64, lib, "host_statistics64")
	purego.RegisterLibFunc(&machPortDealloc, lib, "mach_port_deallocate")
	sym, err := purego.Dlsym(lib, "mach_task_self_")
	if err != nil {
		machErr = err
		return
	}
	machTaskSelfValue = **(**uint32)(unsafe.Pointer(&sym))
}

func readVMStats() (vmStatistics64, error) {
	var vm vmStatistics64
	machOnce.Do(loadMach)
	if machErr != nil {
		return vm, machErr
	}
	host := machHostSelf()
	defer machPortDealloc(machTaskSelfValue, host)
	count := uint32(unsafe.Sizeof(vm) / 4)
	if r := hostStatistics64(host, hostVMInfo64, unsafe.Pointer(&vm), &count); r != 0 {
		return vm, fmt.Errorf("host_statistics64: kern_return %d", r)
	}
	return vm, nil
}

func pageSize() uint64 {
	if v, err := unix.SysctlUint32("vm.pagesize"); err == nil && v > 0 {
		return uint64(v)
	}
	if v, err := unix.SysctlUint64("hw.pagesize"); err == nil && v > 0 {
		return v
	}
	return 4096
}

// pressureFromKernel maps kern.memorystatus_vm_pressure_level.
func pressureFromKernel(level uint32) PressureLevel {
	switch level {
	case 1:
		return PressureNormal
	case 2:
		return PressureWarning
	case 4:
		return PressureCritical
	}
	return PressureUnknown
}

func readMemory() MemoryStats {
	var st MemoryStats
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil || total == 0 {
		st.Err = errors.New("hw.memsize unavailable")
		return st
	}
	vm, err := readVMStats()
	if err != nil {
		st.Err = err
		return st
	}
	computeMemory(&st, total, pageSize(), vm)
	if lvl, err := unix.SysctlUint32("kern.memorystatus_vm_pressure_level"); err == nil {
		st.Pressure = pressureFromKernel(lvl)
	}
	if pct, err := unix.SysctlUint32("kern.memorystatus_level"); err == nil && pct <= 100 {
		st.FreePercent = Some(float64(pct))
	}
	if sw, err := mem.SwapMemory(); err == nil {
		st.Swap = Some(SwapStats{Total: sw.Total, Used: sw.Used})
	}
	return st
}

// computeMemory derives Activity Monitor style categories from raw page
// counts. It is a pure function so it can be unit tested.
func computeMemory(st *MemoryStats, total, page uint64, vm vmStatistics64) {
	p := func(n uint32) uint64 { return uint64(n) * page }
	st.Total = total
	st.Wired = p(vm.WireCount)
	st.Compressed = p(vm.CompressorPageCount)
	st.Active = p(vm.ActiveCount)
	st.Inactive = p(vm.InactiveCount)
	st.Free = p(vm.FreeCount)
	st.Purgeable = p(vm.PurgeableCount)
	st.Speculative = p(vm.SpeculativeCount)
	internal := p(vm.InternalPageCount)
	if internal > st.Purgeable {
		st.App = internal - st.Purgeable
	}
	st.Cached = p(vm.ExternalPageCount) + st.Purgeable
	st.Used = st.App + st.Wired + st.Compressed
	if st.Used > total {
		st.Used = total
	}
	st.Available = total - st.Used
	st.UsedPercent = 100 * float64(st.Used) / float64(total)
	st.Valid = true
}

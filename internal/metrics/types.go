// Package metrics collects live system metrics on macOS.
//
// Every value that may be unavailable on a particular Mac is represented with
// an explicit validity flag (see Opt). Providers never substitute zeros for
// data they could not read.
package metrics

import (
	"context"
	"time"
)

// Opt is an optional value. OK is false when the value is unavailable.
type Opt[T any] struct {
	V  T
	OK bool
}

// Some wraps a value as available.
func Some[T any](v T) Opt[T] { return Opt[T]{V: v, OK: true} }

// Source is anything that can produce snapshots. The UI depends only on this
// interface so tests can use deterministic mock sources.
type Source interface {
	// Host returns static information about the machine.
	Host(ctx context.Context) HostInfo
	// Collect gathers a full snapshot. It must be safe to call from a
	// goroutine and must not panic when an individual provider fails.
	Collect(ctx context.Context) Snapshot
}

// HostInfo is static machine information gathered once at startup.
type HostInfo struct {
	ModelName     string // "MacBook Air"
	ModelID       string // "Mac14,2"
	Chip          string // "Apple M2"
	OSName        string // "macOS Tahoe"
	OSVersion     string // "26.6.2"
	Hostname      string
	Arch          string
	BootTime      time.Time
	PhysicalCores int
	LogicalCores  int
	PCores        int
	ECores        int
	MemoryTotal   uint64
	AppleSilicon  bool
}

// Snapshot is a complete, immutable sample of the system.
type Snapshot struct {
	Time      time.Time
	Took      time.Duration
	CPU       CPUStats
	Memory    MemoryStats
	Disk      DiskStats
	Network   NetworkStats
	Battery   BatteryStats
	Sensors   SensorStats
	Processes ProcessStats
}

// CPUStats contains CPU utilisation percentages (0-100).
type CPUStats struct {
	Valid   bool
	Err     error
	Total   float64
	User    float64
	System  float64
	Idle    float64
	Nice    float64
	PerCore []float64
	// CoreKinds labels each logical core "P" (performance), "E"
	// (efficiency) or "" when unknown.
	CoreKinds []string
	MaxFreq   Opt[float64] // MHz
	Load      Opt[[3]float64]
}

// PressureLevel is the macOS memory pressure state.
type PressureLevel int

const (
	PressureUnknown PressureLevel = iota
	PressureNormal
	PressureWarning
	PressureCritical
)

func (p PressureLevel) String() string {
	switch p {
	case PressureNormal:
		return "Normal"
	case PressureWarning:
		return "Warning"
	case PressureCritical:
		return "Critical"
	}
	return "Unknown"
}

// MemoryStats follows Activity Monitor's model: Used = App + Wired +
// Compressed, and file cache is reported separately as reclaimable.
type MemoryStats struct {
	Valid       bool
	Err         error
	Total       uint64
	Used        uint64
	App         uint64
	Wired       uint64
	Compressed  uint64
	Cached      uint64
	Free        uint64
	Available   uint64
	Active      uint64
	Inactive    uint64
	Purgeable   uint64
	Speculative uint64
	UsedPercent float64
	Pressure    PressureLevel
	// FreePercent is the kernel's memorystatus_level (percentage of memory
	// considered available before pressure rises).
	FreePercent Opt[float64]
	Swap        Opt[SwapStats]
}

// SwapStats describes swap usage.
type SwapStats struct {
	Total uint64
	Used  uint64
}

// Volume is a mounted filesystem.
type Volume struct {
	Name        string
	Device      string
	Mount       string
	FSType      string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
	ReadOnly    bool
	Internal    Opt[bool]
	SolidState  Opt[bool]
	Removable   Opt[bool]
	SMART       string // "Verified", "Failing", "" when unknown
	Protocol    string // "Apple Fabric", "USB", ...
}

// DiskStats holds volumes and aggregated physical disk throughput.
type DiskStats struct {
	Valid     bool
	Err       error
	Volumes   []Volume
	IOValid   bool
	IOErr     error
	ReadBps   float64
	WriteBps  float64
	ReadIOPS  float64
	WriteIOPS float64
	ReadTotal uint64
	WriteTot  uint64
}

// Interface is a network interface.
type Interface struct {
	Name    string
	Kind    string // "Wi-Fi", "Ethernet", "Loopback", "VPN / Tunnel", ...
	Up      bool
	Running bool
	Primary bool
	MAC     string
	MTU     int
	Addrs   []string // CIDR notation
	RxBps   float64
	TxBps   float64
	RxTotal uint64
	TxTotal uint64
	RxErrs  uint64
	TxErrs  uint64
}

// NetworkStats aggregates network interfaces.
type NetworkStats struct {
	Valid      bool
	Err        error
	Interfaces []Interface
	Primary    string
	Connected  bool
	RxBps      float64
	TxBps      float64
	RxTotal    uint64
	TxTotal    uint64
}

// BatteryStats describes the internal battery, if present.
type BatteryStats struct {
	Valid        bool
	Err          error
	Present      bool
	Percent      float64
	Charging     bool
	ACPower      bool
	FullyCharged bool
	CycleCount   Opt[int]
	DesignCycles Opt[int]
	Health       Opt[float64] // max capacity / design capacity, percent
	MaxCapacity  Opt[int]     // mAh
	DesignCap    Opt[int]     // mAh
	Voltage      Opt[float64] // volts
	Amperage     Opt[float64] // mA, negative while discharging
	Power        Opt[float64] // watts flowing in/out of battery
	Temperature  Opt[float64] // celsius
	TimeToEmpty  Opt[time.Duration]
	TimeToFull   Opt[time.Duration]
	AdapterWatts Opt[float64]
	SystemLoad   Opt[float64] // watts, from power telemetry
	Condition    string
}

// ThermalPressure mirrors the OS thermal pressure notification states.
type ThermalPressure int

const (
	ThermalUnknown ThermalPressure = iota
	ThermalNominal
	ThermalModerate
	ThermalHeavy
	ThermalTrapping
	ThermalSleeping
)

func (t ThermalPressure) String() string {
	switch t {
	case ThermalNominal:
		return "Nominal"
	case ThermalModerate:
		return "Moderate"
	case ThermalHeavy:
		return "Heavy"
	case ThermalTrapping:
		return "Trapping"
	case ThermalSleeping:
		return "Sleeping"
	}
	return "Unknown"
}

// Temperature is a single sensor reading.
type Temperature struct {
	Name    string
	Group   string // "SoC", "GPU", "Battery", "Storage", "Other"
	Celsius float64
}

// Fan is a fan speed reading.
type Fan struct {
	Index int
	RPM   float64
	Min   Opt[float64]
	Max   Opt[float64]
}

// PowerReading is a power rail measurement in watts.
type PowerReading struct {
	Name  string
	Watts float64
}

// FanState explains whether fan data exists.
type FanState int

const (
	FansUnavailable FanState = iota // could not read
	FansNone                        // SMC reports no fans (e.g. MacBook Air)
	FansOK
)

// ProviderStatus records the health of one sensor backend.
type ProviderStatus struct {
	Name string
	OK   bool
	Err  error
}

// SensorStats aggregates hardware sensor providers.
type SensorStats struct {
	Temps     []Temperature
	CPUTemp   Opt[float64] // average SoC/CPU die temperature
	CPUMax    Opt[float64]
	GPUTemp   Opt[float64]
	Fans      []Fan
	FanState  FanState
	Thermal   ThermalPressure
	Power     []PowerReading
	Providers []ProviderStatus
}

// ProcKey uniquely identifies a process across PID reuse.
type ProcKey struct {
	PID   int
	Start int64 // unix seconds
}

// Process is one running process.
type Process struct {
	PID        int
	PPID       int
	UID        int
	User       string
	Name       string
	Path       string
	State      string // raw ps state, e.g. "Ss"
	RSS        uint64
	VSZ        uint64
	CPUPercent float64
	CPUValid   bool // false on the first sample of a process
	CPUTime    time.Duration
	Threads    Opt[int]
	Start      time.Time
}

// Key returns the PID-reuse-safe identity of the process.
func (p Process) Key() ProcKey { return ProcKey{PID: p.PID, Start: p.Start.Unix()} }

// Status returns a human readable process status.
func (p Process) Status() string {
	if p.State == "" {
		return "Unknown"
	}
	switch p.State[0] {
	case 'R':
		return "Running"
	case 'S':
		return "Sleeping"
	case 'I':
		return "Idle"
	case 'T':
		return "Stopped"
	case 'U':
		return "Waiting"
	case 'Z':
		return "Zombie"
	}
	return p.State
}

// ProcessStats is the process table.
type ProcessStats struct {
	Valid   bool
	Err     error
	List    []Process
	Threads int
}

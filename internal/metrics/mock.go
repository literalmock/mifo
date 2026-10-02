package metrics

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MockSource is a deterministic Source for tests. It is never used by the
// real application.
type MockSource struct {
	mu       sync.Mutex
	HostInfo HostInfo
	// Next produces the n-th snapshot (starting at 0).
	Next  func(n int) Snapshot
	calls int
}

// Host implements Source.
func (m *MockSource) Host(context.Context) HostInfo { return m.HostInfo }

// Collect implements Source.
func (m *MockSource) Collect(context.Context) Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := m.calls
	m.calls++
	if m.Next == nil {
		return SampleSnapshot(time.Now(), n)
	}
	return m.Next(n)
}

// Calls returns how many snapshots were collected.
func (m *MockSource) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// SampleHost returns plausible static host information for tests.
func SampleHost() HostInfo {
	return HostInfo{
		ModelName: "MacBook Pro", ModelID: "Mac15,3", Chip: "Apple M3", OSName: "macOS Sequoia",
		OSVersion: "15.1", Hostname: "test-mac", Arch: "arm64",
		BootTime:      time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
		PhysicalCores: 8, LogicalCores: 8, PCores: 4, ECores: 4, MemoryTotal: 16 << 30, AppleSilicon: true,
	}
}

// SampleSnapshot returns a fully populated, deterministic snapshot.
func SampleSnapshot(now time.Time, n int) Snapshot {
	gb := uint64(1 << 30)
	procs := []Process{
		{PID: 1, PPID: 0, UID: 0, User: "root", Name: "launchd", Path: "/sbin/launchd", State: "Ss", RSS: 20 << 20, CPUPercent: 0.1, CPUValid: true, Threads: Some(4), Start: now.Add(-48 * time.Hour)},
		{PID: 420, PPID: 1, UID: 501, User: "alice", Name: "Safari", Path: "/Applications/Safari.app/Contents/MacOS/Safari", State: "S", RSS: 900 << 20, CPUPercent: 12.5, CPUValid: true, Threads: Some(30), Start: now.Add(-2 * time.Hour)},
		{PID: 777, PPID: 1, UID: 501, User: "alice", Name: "Xcode", Path: "/Applications/Xcode.app/Contents/MacOS/Xcode", State: "R", RSS: 2 * gb, CPUPercent: 85, CPUValid: true, Threads: Some(60), Start: now.Add(-time.Hour)},
		{PID: 1500, PPID: 1, UID: 0, User: "root", Name: "kernel_task", Path: "kernel_task", State: "R", RSS: 1 << 30, CPUPercent: 4, CPUValid: true, Start: now.Add(-48 * time.Hour)},
		{PID: 2001, PPID: 420, UID: 501, User: "alice", Name: "Safari Web Content", Path: "/System/Library/Frameworks/WebKit.framework/XPCServices/com.apple.WebKit.WebContent.xpc", State: "S", RSS: 300 << 20, CPUPercent: 3, CPUValid: true, Threads: Some(12), Start: now.Add(-time.Hour)},
	}
	return Snapshot{
		Time: now,
		Took: 40 * time.Millisecond,
		CPU: CPUStats{
			Valid: true, Total: 42 + float64(n%5), User: 30, System: 12, Idle: 58,
			PerCore:   []float64{80, 60, 40, 20, 10, 5, 50, 70},
			CoreKinds: []string{"E", "E", "E", "E", "P", "P", "P", "P"},
			MaxFreq:   Some(4050.0), Load: Some([3]float64{2.5, 2.1, 1.9}),
		},
		Memory: MemoryStats{
			Valid: true, Total: 16 * gb, Used: 10 * gb, App: 6 * gb, Wired: 2 * gb, Compressed: 2 * gb,
			Cached: 4 * gb, Free: 1 * gb, Available: 6 * gb, Active: 5 * gb, Inactive: 4 * gb,
			UsedPercent: 62.5, Pressure: PressureNormal, FreePercent: Some(70.0),
			Swap: Some(SwapStats{Total: 2 * gb, Used: gb / 2}),
		},
		Disk: DiskStats{
			Valid: true,
			Volumes: []Volume{
				{Name: "Macintosh HD", Device: "/dev/disk3s1s1", Mount: "/", FSType: "apfs", Total: 500e9, Used: 300e9, Free: 200e9, UsedPercent: 60, ReadOnly: true, Internal: Some(true), SolidState: Some(true), SMART: "Verified"},
				{Name: "Data", Device: "/dev/disk3s5", Mount: "/System/Volumes/Data", FSType: "apfs", Total: 500e9, Used: 300e9, Free: 200e9, UsedPercent: 60, Internal: Some(true), SolidState: Some(true), SMART: "Verified"},
				{Name: "Backup", Device: "/dev/disk5s1", Mount: "/Volumes/Backup", FSType: "apfs", Total: 2e12, Used: 1.9e12, Free: 1e11, UsedPercent: 95, Internal: Some(false), Protocol: "USB"},
			},
			IOValid: true, ReadBps: 12e6, WriteBps: 3e6, ReadIOPS: 120, WriteIOPS: 40, ReadTotal: 500e9, WriteTot: 300e9,
		},
		Network: NetworkStats{
			Valid: true, Primary: "en0", Connected: true, RxBps: 2.5e6, TxBps: 300e3, RxTotal: 40e9, TxTotal: 4e9,
			Interfaces: []Interface{
				{Name: "en0", Kind: "Wi-Fi", Up: true, Running: true, Primary: true, MAC: "aa:bb:cc:dd:ee:ff", MTU: 1500, Addrs: []string{"192.168.1.20/24", "fe80::1/64"}, RxBps: 2.5e6, TxBps: 300e3, RxTotal: 40e9, TxTotal: 4e9},
				{Name: "utun3", Kind: "VPN / Tunnel", Up: true, Running: true, Addrs: []string{"10.8.0.2/32"}, RxTotal: 1e6, TxTotal: 1e6},
				{Name: "lo0", Kind: "Loopback", Up: true, Running: true, Addrs: []string{"127.0.0.1/8"}, RxTotal: 1e6, TxTotal: 1e6},
			},
		},
		Battery: BatteryStats{
			Valid: true, Present: true, Percent: 76, Charging: false, ACPower: false,
			CycleCount: Some(120), DesignCycles: Some(1000), Health: Some(92.0), MaxCapacity: Some(4200), DesignCap: Some(4563),
			Voltage: Some(12.3), Amperage: Some(-800.0), Power: Some(-9.8), Temperature: Some(31.0),
			TimeToEmpty: Some(5*time.Hour + 20*time.Minute), SystemLoad: Some(9.5), Condition: "Normal",
		},
		Sensors: SensorStats{
			Temps: []Temperature{
				{Name: "PMU tdie1", Group: "SoC", Celsius: 52}, {Name: "PMU tdie2", Group: "SoC", Celsius: 55},
				{Name: "NAND CH0 temp", Group: "Storage", Celsius: 38}, {Name: "gas gauge battery", Group: "Battery", Celsius: 31},
			},
			CPUTemp: Some(53.5), CPUMax: Some(55.0),
			Fans:     []Fan{{Index: 0, RPM: 1800, Min: Some(1200.0), Max: Some(5000.0)}},
			FanState: FansOK, Thermal: ThermalNominal,
			Power:     []PowerReading{{Name: "System total", Watts: 12.4}},
			Providers: []ProviderStatus{{Name: "Temperature sensors", OK: true}, {Name: "SMC (fans, power)", OK: true}, {Name: "Thermal pressure", OK: true}},
		},
		Processes: ProcessStats{Valid: true, List: procs, Threads: 106},
	}
}

// EmptySnapshot returns a snapshot where every provider failed.
func EmptySnapshot(now time.Time) Snapshot {
	err := errors.New("not permitted")
	return Snapshot{
		Time:      now,
		CPU:       CPUStats{Err: err},
		Memory:    MemoryStats{Err: err},
		Disk:      DiskStats{Err: err, IOErr: err},
		Network:   NetworkStats{Err: err},
		Battery:   BatteryStats{Err: err},
		Sensors:   SensorStats{FanState: FansUnavailable, Providers: []ProviderStatus{{Name: "Temperature sensors", Err: err}}},
		Processes: ProcessStats{Err: err},
	}
}

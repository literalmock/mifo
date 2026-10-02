package metrics

import (
	"math"
	"testing"
	"time"
)

const psSample = `    1     0     0 root             Ss    17504 435245664  42:17.30 Thu Sep 24 00:05:20 2026     /sbin/launchd
  420     1   501 alice            S    921600 435394096   1:02:03.50 Wed Sep  3 15:43:46 2026     /Applications/Google Chrome.app/Contents/MacOS/Google Chrome
  999     1   501 alice            R+       10       20   1-02:00:00 Wed Sep 30 15:43:46 2026     (zombie thing)
garbage line
`

func TestParsePS(t *testing.T) {
	list := parsePS([]byte(psSample), time.UTC)
	if len(list) != 3 {
		t.Fatalf("got %d processes", len(list))
	}
	p := list[1]
	if p.PID != 420 || p.PPID != 1 || p.UID != 501 || p.User != "alice" {
		t.Errorf("ids wrong: %+v", p)
	}
	if p.Name != "Google Chrome" || p.Path != "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" {
		t.Errorf("name/path wrong: %q %q", p.Name, p.Path)
	}
	if p.RSS != 921600*1024 {
		t.Errorf("rss = %d", p.RSS)
	}
	if p.CPUTime != time.Hour+2*time.Minute+3500*time.Millisecond {
		t.Errorf("cputime = %v", p.CPUTime)
	}
	if !p.Start.Equal(time.Date(2026, 9, 3, 15, 43, 46, 0, time.UTC)) {
		t.Errorf("start = %v", p.Start)
	}
	if list[2].CPUTime != 26*time.Hour {
		t.Errorf("day-format cputime = %v", list[2].CPUTime)
	}
	if list[2].Status() != "Running" || list[0].Status() != "Sleeping" {
		t.Errorf("status mapping wrong")
	}
}

func TestParseCPUTimeInvalid(t *testing.T) {
	if _, ok := parseCPUTime("abc"); ok {
		t.Error("expected failure")
	}
}

func TestParseThreadCounts(t *testing.T) {
	out := `USER               PID   TT   %CPU STAT PRI     STIME     UTIME COMMAND
root                 1   ??    0.0 S    31T   0:00.28   0:00.05 /sbin/launchd
                     1         0.0 S    37T   0:00.00   0:00.00
                     1         0.2 S    37T   0:00.00   0:00.00
alice               94   ??    0.0 S    31T   0:00.00   0:00.00 /usr/libexec/logd
`
	m := parseThreadCounts([]byte(out))
	if m[1] != 3 || m[94] != 1 {
		t.Errorf("counts = %v", m)
	}
}

func TestComputeProcessCPUHandlesPIDReuse(t *testing.T) {
	t0 := time.Unix(1000, 0)
	start := time.Unix(500, 0)
	first := []Process{{PID: 10, Start: start, CPUTime: 10 * time.Second}}
	prev := computeProcessCPU(first, nil, t0)
	if first[0].CPUValid {
		t.Error("first sample must not have a CPU value")
	}
	// Same process 2s later used 1s of CPU: 50%.
	second := []Process{{PID: 10, Start: start, CPUTime: 11 * time.Second}}
	prev = computeProcessCPU(second, prev, t0.Add(2*time.Second))
	if !second[0].CPUValid || math.Abs(second[0].CPUPercent-50) > 1e-9 {
		t.Errorf("cpu = %v valid=%v", second[0].CPUPercent, second[0].CPUValid)
	}
	// PID reused by a new process (different start): no inherited delta.
	reused := []Process{{PID: 10, Start: time.Unix(1001, 0), CPUTime: 2 * time.Second}}
	computeProcessCPU(reused, prev, t0.Add(4*time.Second))
	if reused[0].CPUValid {
		t.Error("reused PID must not inherit CPU history")
	}
}

func TestComputeMemory(t *testing.T) {
	var st MemoryStats
	vm := vmStatistics64{
		FreeCount: 100, ActiveCount: 300, InactiveCount: 200, WireCount: 150,
		PurgeableCount: 10, CompressorPageCount: 50, ExternalPageCount: 90, InternalPageCount: 400,
	}
	computeMemory(&st, 1000*16384, 16384, vm)
	pg := uint64(16384)
	if st.App != 390*pg || st.Wired != 150*pg || st.Compressed != 50*pg {
		t.Errorf("categories wrong: %+v", st)
	}
	if st.Cached != 100*pg {
		t.Errorf("cached = %d", st.Cached/pg)
	}
	if st.Used != 590*pg || st.Available != 410*pg {
		t.Errorf("used=%d avail=%d", st.Used/pg, st.Available/pg)
	}
	if math.Abs(st.UsedPercent-59) > 1e-9 || !st.Valid {
		t.Errorf("pct=%v", st.UsedPercent)
	}
}

func TestPressureFromKernel(t *testing.T) {
	if pressureFromKernel(1) != PressureNormal || pressureFromKernel(2) != PressureWarning ||
		pressureFromKernel(4) != PressureCritical || pressureFromKernel(9) != PressureUnknown {
		t.Error("mapping wrong")
	}
}

const batteryPlist = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><array><dict>
<key>BatteryInstalled</key><true/>
<key>CurrentCapacity</key><integer>83</integer>
<key>MaxCapacity</key><integer>100</integer>
<key>DesignCapacity</key><integer>4563</integer>
<key>AppleRawMaxCapacity</key><integer>3856</integer>
<key>CycleCount</key><integer>269</integer>
<key>DesignCycleCount9C</key><integer>1000</integer>
<key>IsCharging</key><false/>
<key>ExternalConnected</key><false/>
<key>FullyCharged</key><false/>
<key>Voltage</key><integer>12000</integer>
<key>InstantAmperage</key><integer>18446744073709549819</integer>
<key>Temperature</key><integer>3062</integer>
<key>AvgTimeToEmpty</key><integer>493</integer>
<key>AvgTimeToFull</key><integer>65535</integer>
<key>PowerTelemetryData</key><dict><key>SystemLoad</key><integer>15988</integer></dict>
</dict></array></plist>`

func TestParseBatteryLaptop(t *testing.T) {
	b := parseBattery([]byte(batteryPlist))
	if !b.Valid || !b.Present || b.Err != nil {
		t.Fatalf("battery = %+v", b)
	}
	if b.Percent != 83 || b.Charging || b.ACPower {
		t.Errorf("state wrong: %+v", b)
	}
	if !b.Health.OK || math.Abs(b.Health.V-84.5) > 0.1 {
		t.Errorf("health = %+v", b.Health)
	}
	if !b.Amperage.OK || b.Amperage.V != -1797 {
		t.Errorf("amperage = %+v (two's complement not decoded)", b.Amperage)
	}
	if !b.Power.OK || math.Abs(b.Power.V+21.564) > 0.01 {
		t.Errorf("power = %+v", b.Power)
	}
	if !b.TimeToEmpty.OK || b.TimeToEmpty.V != 493*time.Minute {
		t.Errorf("tte = %+v", b.TimeToEmpty)
	}
	if b.TimeToFull.OK {
		t.Error("65535 must be treated as no estimate")
	}
	if !b.SystemLoad.OK || math.Abs(b.SystemLoad.V-15.988) > 1e-9 {
		t.Errorf("system load = %+v", b.SystemLoad)
	}
	if !b.Temperature.OK || math.Abs(b.Temperature.V-30.62) > 1e-9 {
		t.Errorf("temperature = %+v", b.Temperature)
	}
}

func TestParseBatteryDesktop(t *testing.T) {
	b := parseBattery(nil)
	if !b.Valid || b.Present {
		t.Errorf("desktop = %+v", b)
	}
	b = parseBattery([]byte(`<plist><array></array></plist>`))
	if !b.Valid || b.Present {
		t.Errorf("empty array = %+v", b)
	}
}

func TestParseBatteryMissingCapacity(t *testing.T) {
	b := parseBattery([]byte(`<plist><array><dict><key>BatteryInstalled</key><true/></dict></array></plist>`))
	if b.Valid || b.Err == nil {
		t.Errorf("expected error, got %+v", b)
	}
}

func TestParsePing(t *testing.T) {
	out := []byte(`PING 1.1.1.1 (1.1.1.1): 56 data bytes
--- 1.1.1.1 ping statistics ---
5 packets transmitted, 4 packets received, 20.0% packet loss
round-trip min/avg/max/stddev = 10.123/14.500/20.001/3.2 ms`)
	var r DiagResult
	parsePing(out, &r)
	if r.Sent != 5 || r.Received != 4 || !r.Loss.OK || r.Loss.V != 20 {
		t.Errorf("stats = %+v", r)
	}
	if !r.AvgMs.OK || r.AvgMs.V != 14.5 || r.MinMs.V != 10.123 || r.MaxMs.V != 20.001 {
		t.Errorf("rtt = %+v", r)
	}
}

func TestParseNetTopUsesDeltaBlock(t *testing.T) {
	out := []byte(`,bytes_in,bytes_out,
launchd.1,0,0,
Safari.420,99999,99999,
,bytes_in,bytes_out,
Safari.420,2000,500,
mDNSResponder.250,10,0,
idle.5,0,0,
`)
	e := parseNetTop(out, 1)
	if len(e) != 2 || e[0].Name != "Safari" || e[0].PID != 420 || e[0].RxBps != 2000 {
		t.Errorf("entries = %+v", e)
	}
}

func TestParseHardwarePorts(t *testing.T) {
	out := []byte("Hardware Port: Wi-Fi\nDevice: en0\nEthernet Address: aa\n\nHardware Port: Thunderbolt Bridge\nDevice: bridge0\n")
	m := parseHardwarePorts(out)
	if m["en0"] != "Wi-Fi" || m["bridge0"] != "Thunderbolt Bridge" {
		t.Errorf("ports = %v", m)
	}
	if interfaceKind("utun2", m) != "VPN / Tunnel" || interfaceKind("lo0", m) != "Loopback" {
		t.Error("kind fallback wrong")
	}
}

func TestRingWrapAndSince(t *testing.T) {
	r := NewRing(3)
	base := time.Unix(0, 0)
	for i := 1; i <= 5; i++ {
		r.Push(base.Add(time.Duration(i)*time.Second), float64(i))
	}
	if r.Len() != 3 {
		t.Fatalf("len = %d", r.Len())
	}
	got := r.Since(base, nil)
	if len(got) != 3 || got[0] != 3 || got[2] != 5 {
		t.Errorf("since all = %v", got)
	}
	got = r.Since(base.Add(4*time.Second), nil)
	if len(got) != 1 || got[0] != 5 {
		t.Errorf("since 4s = %v", got)
	}
	if last, ok := r.Last(); !ok || last.V != 5 {
		t.Errorf("last = %v", last)
	}
}

func TestHistorySkipsInvalidMetrics(t *testing.T) {
	h := NewHistory(10)
	h.Record(EmptySnapshot(time.Now()))
	if h.CPU.Len() != 0 || h.Memory.Len() != 0 || h.NetRx.Len() != 0 || h.CPUTemp.Len() != 0 {
		t.Error("invalid data must not be recorded as zeros")
	}
	h.Record(SampleSnapshot(time.Now(), 0))
	if h.CPU.Len() != 1 || len(h.Cores) != 8 || h.Power.Len() != 1 || h.Battery.Len() != 1 {
		t.Errorf("valid data not recorded")
	}
	if _, ok := h.Iface["en0"]; !ok {
		t.Error("per-interface history missing")
	}
}

func TestSensorClassificationAndSummary(t *testing.T) {
	if classifyTemperature("PMU tdie3") != "SoC" || classifyTemperature("gas gauge battery") != "Battery" ||
		classifyTemperature("NAND CH0 temp") != "Storage" || classifyTemperature("TG0P") != "GPU" {
		t.Error("classification wrong")
	}
	if plausibleTemp(-2) || plausibleTemp(0) || plausibleTemp(200) || !plausibleTemp(45) {
		t.Error("plausibility wrong")
	}
	st := SensorStats{Temps: []Temperature{{Name: "a", Group: "SoC", Celsius: 40}, {Name: "b", Group: "SoC", Celsius: 50}, {Name: "c", Group: "Battery", Celsius: 30}}}
	summarizeTemps(&st)
	if !st.CPUTemp.OK || st.CPUTemp.V != 45 || st.CPUMax.V != 50 || st.GPUTemp.OK {
		t.Errorf("summary = %+v", st)
	}
	var empty SensorStats
	summarizeTemps(&empty)
	if empty.CPUTemp.OK {
		t.Error("no sensors must not yield a temperature")
	}
}

func TestThermalFromState(t *testing.T) {
	if thermalFromState(0) != ThermalNominal || thermalFromState(2) != ThermalHeavy || thermalFromState(42) != ThermalUnknown {
		t.Error("mapping wrong")
	}
}

func TestRate(t *testing.T) {
	if rate(150, 100, 2) != 25 || rate(50, 100, 1) != 0 || rate(1, 0, 0) != 0 {
		t.Error("rate wrong")
	}
}

func TestCutFields(t *testing.T) {
	f, rest := cutFields("  a  b c   rest of  line ", 3)
	if len(f) != 3 || f[2] != "c" || rest != "rest of  line" {
		t.Errorf("got %v %q", f, rest)
	}
}

package alerts

import (
	"testing"
	"time"

	"github.com/literalmock/mifo/internal/config"
	"github.com/literalmock/mifo/internal/metrics"
)

func snap(now time.Time) metrics.Snapshot { return metrics.SampleSnapshot(now, 0) }

func TestNoAlertsFromMissingData(t *testing.T) {
	e := NewEngine()
	now := time.Now()
	cfg := config.Default()
	if got := e.Evaluate(metrics.EmptySnapshot(now), cfg, now); len(got) != 0 {
		t.Fatalf("alerts from empty snapshot: %+v", got)
	}
}

func TestStaleSnapshotIgnored(t *testing.T) {
	e := NewEngine()
	now := time.Now()
	s := snap(now.Add(-time.Minute))
	s.Battery.Percent = 3
	if got := e.Evaluate(s, config.Default(), now); len(got) != 0 {
		t.Fatalf("stale data raised alerts: %+v", got)
	}
}

func TestDiskAlertForExternalVolume(t *testing.T) {
	e := NewEngine()
	now := time.Now()
	got := e.Evaluate(snap(now), config.Default(), now)
	if len(got) != 1 || got[0].Key != "disk:/Volumes/Backup" {
		t.Fatalf("expected one disk alert, got %+v", got)
	}
}

func TestCPUSustainedAndResolve(t *testing.T) {
	e := NewEngine()
	cfg := config.Default()
	cfg.Alerts.CPUPercent = 80
	cfg.Alerts.CPUSustainSecs = 10
	now := time.Now()
	s := snap(now)
	s.Disk.Volumes = nil
	s.CPU.Total = 95
	if got := e.Evaluate(s, cfg, now); len(got) != 0 {
		t.Fatal("alert fired before sustain period")
	}
	now = now.Add(11 * time.Second)
	s.Time = now
	got := e.Evaluate(s, cfg, now)
	if len(got) != 1 || got[0].Key != "cpu" {
		t.Fatalf("expected cpu alert, got %+v", got)
	}
	if len(e.Active()) != 1 {
		t.Fatal("alert not active")
	}
	now = now.Add(2 * time.Second)
	s.Time, s.CPU.Total = now, 20
	e.Evaluate(s, cfg, now)
	if len(e.Active()) != 0 {
		t.Fatal("alert not resolved")
	}
	if h := e.History(); len(h) != 1 || h[0].Active() {
		t.Errorf("history = %+v", h)
	}
}

func TestCooldownPreventsSpam(t *testing.T) {
	e := NewEngine()
	cfg := config.Default()
	cfg.Alerts.CooldownSecs = 60
	now := time.Now()
	s := snap(now)
	s.Disk.Volumes = nil
	s.Battery.Percent = 10
	if len(e.Evaluate(s, cfg, now)) != 1 {
		t.Fatal("battery alert expected")
	}
	// Resolve, then re-trigger within the cooldown.
	now = now.Add(5 * time.Second)
	s.Time, s.Battery.ACPower = now, true
	e.Evaluate(s, cfg, now)
	now = now.Add(5 * time.Second)
	s.Time, s.Battery.ACPower = now, false
	if got := e.Evaluate(s, cfg, now); len(got) != 0 {
		t.Fatalf("alert re-fired during cooldown: %+v", got)
	}
	now = now.Add(61 * time.Second)
	s.Time = now
	if got := e.Evaluate(s, cfg, now); len(got) != 1 {
		t.Fatalf("alert should fire after cooldown, got %+v", got)
	}
}

func TestMissingDataDoesNotResolve(t *testing.T) {
	e := NewEngine()
	cfg := config.Default()
	now := time.Now()
	s := snap(now)
	s.Disk.Volumes = nil
	s.Memory.Pressure = metrics.PressureWarning
	e.Evaluate(s, cfg, now)
	if len(e.Active()) != 1 {
		t.Fatal("memory alert expected")
	}
	empty := metrics.EmptySnapshot(now.Add(time.Second))
	e.Evaluate(empty, cfg, now.Add(time.Second))
	if len(e.Active()) != 1 {
		t.Fatal("missing data must not resolve an alert")
	}
}

func TestTemperatureOnlyWithReliableSensor(t *testing.T) {
	e := NewEngine()
	cfg := config.Default()
	now := time.Now()
	s := snap(now)
	s.Disk.Volumes = nil
	s.Sensors.CPUMax = metrics.Opt[float64]{}
	if got := e.Evaluate(s, cfg, now); len(got) != 0 {
		t.Fatalf("temperature alert without sensor: %+v", got)
	}
	s.Sensors.CPUMax = metrics.Some(101.0)
	if got := e.Evaluate(s, cfg, now); len(got) != 1 || got[0].Key != "temp" {
		t.Fatalf("expected temp alert, got %+v", got)
	}
}

func TestDismissAndEscalate(t *testing.T) {
	e := NewEngine()
	cfg := config.Default()
	now := time.Now()
	s := snap(now)
	s.Disk.Volumes = nil
	s.Memory.Pressure = metrics.PressureWarning
	e.Evaluate(s, cfg, now)
	a := e.Active()[0]
	e.Dismiss(a.ID)
	if len(e.Active()) != 0 {
		t.Fatal("dismissed alert still shown")
	}
	s.Memory.Pressure = metrics.PressureCritical
	e.Evaluate(s, cfg, now.Add(time.Second))
	act := e.Active()
	if len(act) != 1 || act[0].Severity != Critical {
		t.Fatalf("escalation should un-dismiss: %+v", act)
	}
}

func TestAlertsDisabled(t *testing.T) {
	e := NewEngine()
	cfg := config.Default()
	cfg.AlertsEnabled = false
	now := time.Now()
	if got := e.Evaluate(snap(now), cfg, now); len(got) != 0 {
		t.Fatal("alerts fired while disabled")
	}
}

func TestHealth(t *testing.T) {
	cfg := config.Default()
	s := snap(time.Now())
	s.Disk.Volumes = s.Disk.Volumes[:2]
	if sev, _ := Health(s, cfg); sev != OK {
		t.Errorf("sev = %v", sev)
	}
	s.Sensors.Thermal = metrics.ThermalHeavy
	if sev, reason := Health(s, cfg); sev != Critical || reason == "" {
		t.Errorf("sev = %v %q", sev, reason)
	}
}

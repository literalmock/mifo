// Package alerts evaluates snapshots against user thresholds.
//
// Alerts are only ever raised from valid, fresh data: a metric that could not
// be read can never trigger (or clear) an alert.
package alerts

import (
	"fmt"
	"sort"
	"time"

	"github.com/literalmock/mifo/internal/config"
	"github.com/literalmock/mifo/internal/format"
	"github.com/literalmock/mifo/internal/metrics"
)

// Severity of an alert or of the overall system health.
type Severity int

const (
	OK Severity = iota
	Warning
	Critical
)

func (s Severity) String() string {
	switch s {
	case Warning:
		return "Warning"
	case Critical:
		return "Critical"
	}
	return "Healthy"
}

// Alert is one raised condition.
type Alert struct {
	ID        int
	Key       string
	Severity  Severity
	Title     string
	Detail    string
	At        time.Time
	Resolved  time.Time
	Dismissed bool
}

// Active reports whether the condition is still ongoing.
func (a Alert) Active() bool { return a.Resolved.IsZero() }

// MaxHistory bounds the alert history.
const MaxHistory = 200

// Engine tracks alert state across snapshots.
type Engine struct {
	active    map[string]*Alert
	history   []Alert
	lastFired map[string]time.Time
	cpuSince  time.Time
	nextID    int
}

// NewEngine creates an empty engine.
func NewEngine() *Engine {
	return &Engine{active: map[string]*Alert{}, lastFired: map[string]time.Time{}}
}

type condition struct {
	key      string
	severity Severity
	title    string
	detail   string
}

// conditions returns every threshold currently exceeded. The second result
// lists keys that have valid data, so only those can be resolved.
func (e *Engine) conditions(s metrics.Snapshot, cfg config.Config, now time.Time) ([]condition, map[string]bool) {
	t := cfg.Alerts
	var out []condition
	known := map[string]bool{}

	if s.CPU.Valid {
		known["cpu"] = true
		if s.CPU.Total >= t.CPUPercent {
			if e.cpuSince.IsZero() {
				e.cpuSince = now
			}
			if now.Sub(e.cpuSince) >= time.Duration(t.CPUSustainSecs)*time.Second {
				out = append(out, condition{"cpu", Warning, "High CPU usage",
					fmt.Sprintf("CPU at %s for over %ds (threshold %.0f%%)", format.Percent(s.CPU.Total), t.CPUSustainSecs, t.CPUPercent)})
			}
		} else {
			e.cpuSince = time.Time{}
		}
	}
	if s.Memory.Valid && s.Memory.Pressure != metrics.PressureUnknown {
		known["memory"] = true
		min := metrics.PressureWarning
		if t.MemoryPressure == "critical" {
			min = metrics.PressureCritical
		}
		if s.Memory.Pressure >= min {
			sev := Warning
			if s.Memory.Pressure == metrics.PressureCritical {
				sev = Critical
			}
			out = append(out, condition{"memory", sev, "Memory pressure " + s.Memory.Pressure.String(),
				fmt.Sprintf("%s of %s in use; swap may slow the system", format.Bytes(s.Memory.Used), format.Bytes(s.Memory.Total))})
		}
	}
	if s.Disk.Valid {
		for _, v := range s.Disk.Volumes {
			if v.ReadOnly || v.Total == 0 {
				continue
			}
			key := "disk:" + v.Mount
			known[key] = true
			freePct := 100 * float64(v.Free) / float64(v.Total)
			if freePct < t.DiskFreePercent {
				sev := Warning
				if freePct < t.DiskFreePercent/2 {
					sev = Critical
				}
				out = append(out, condition{key, sev, "Low storage on " + v.Name,
					fmt.Sprintf("%s free (%.1f%%) on %s", format.BytesSI(v.Free), freePct, v.Mount)})
			}
		}
	}
	if s.Sensors.CPUMax.OK {
		known["temp"] = true
		if s.Sensors.CPUMax.V >= t.TempCelsius {
			out = append(out, condition{"temp", Warning, "High SoC temperature",
				fmt.Sprintf("Hottest die sensor at %.1f°C (threshold %.0f°C)", s.Sensors.CPUMax.V, t.TempCelsius)})
		}
	}
	if s.Battery.Valid && s.Battery.Present {
		known["battery"] = true
		if !s.Battery.ACPower && s.Battery.Percent <= t.BatteryPercent {
			sev := Warning
			if s.Battery.Percent <= 5 {
				sev = Critical
			}
			out = append(out, condition{"battery", sev, "Low battery",
				fmt.Sprintf("%.0f%% remaining on battery power", s.Battery.Percent)})
		}
	}
	return out, known
}

// Evaluate updates alert state from a snapshot and returns alerts that were
// newly raised (for native notifications). stale snapshots are ignored.
func (e *Engine) Evaluate(s metrics.Snapshot, cfg config.Config, now time.Time) []Alert {
	if !cfg.AlertsEnabled || s.Time.IsZero() || now.Sub(s.Time) > 3*cfg.Interval()+5*time.Second {
		return nil
	}
	conds, known := e.conditions(s, cfg, now)
	firing := map[string]bool{}
	var raised []Alert
	cooldown := time.Duration(cfg.Alerts.CooldownSecs) * time.Second
	for _, c := range conds {
		firing[c.key] = true
		if a, ok := e.active[c.key]; ok {
			// Escalate in place; keep dismissal unless severity rises.
			if c.severity > a.Severity {
				a.Severity = c.severity
				a.Dismissed = false
			}
			a.Detail = c.detail
			e.syncHistory(a)
			continue
		}
		if last, ok := e.lastFired[c.key]; ok && now.Sub(last) < cooldown {
			continue
		}
		e.nextID++
		a := &Alert{ID: e.nextID, Key: c.key, Severity: c.severity, Title: c.title, Detail: c.detail, At: now}
		e.active[c.key] = a
		e.lastFired[c.key] = now
		e.history = append(e.history, *a)
		if len(e.history) > MaxHistory {
			e.history = e.history[len(e.history)-MaxHistory:]
		}
		raised = append(raised, *a)
	}
	for key, a := range e.active {
		if !firing[key] && known[key] {
			a.Resolved = now
			e.syncHistory(a)
			delete(e.active, key)
		}
	}
	return raised
}

func (e *Engine) syncHistory(a *Alert) {
	for i := len(e.history) - 1; i >= 0; i-- {
		if e.history[i].ID == a.ID {
			e.history[i] = *a
			return
		}
	}
}

// Active returns undismissed ongoing alerts, most severe first.
func (e *Engine) Active() []Alert {
	var out []Alert
	for _, a := range e.active {
		if !a.Dismissed {
			out = append(out, *a)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity > out[j].Severity
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// History returns all alerts, newest first.
func (e *Engine) History() []Alert {
	out := make([]Alert, len(e.history))
	for i, a := range e.history {
		out[len(e.history)-1-i] = a
	}
	return out
}

// Dismiss hides an active alert until it resolves or escalates.
func (e *Engine) Dismiss(id int) {
	for _, a := range e.active {
		if a.ID == id {
			a.Dismissed = true
			e.syncHistory(a)
		}
	}
}

// DismissAll hides all active alerts.
func (e *Engine) DismissAll() {
	for _, a := range e.active {
		a.Dismissed = true
		e.syncHistory(a)
	}
}

// Health summarises current system state independently of dismissals.
func Health(s metrics.Snapshot, cfg config.Config) (Severity, string) {
	sev, reason := OK, "All systems normal"
	raise := func(v Severity, why string) {
		if v > sev {
			sev, reason = v, why
		}
	}
	switch s.Memory.Pressure {
	case metrics.PressureWarning:
		raise(Warning, "Memory pressure elevated")
	case metrics.PressureCritical:
		raise(Critical, "Memory pressure critical")
	}
	switch s.Sensors.Thermal {
	case metrics.ThermalModerate:
		raise(Warning, "Thermal pressure moderate")
	case metrics.ThermalHeavy, metrics.ThermalTrapping, metrics.ThermalSleeping:
		raise(Critical, "Thermal pressure "+s.Sensors.Thermal.String())
	}
	if s.Disk.Valid {
		for _, v := range s.Disk.Volumes {
			if v.ReadOnly || v.Total == 0 {
				continue
			}
			if 100*float64(v.Free)/float64(v.Total) < cfg.Alerts.DiskFreePercent {
				raise(Warning, "Low storage on "+v.Name)
			}
		}
	}
	if s.Battery.Valid && s.Battery.Present && !s.Battery.ACPower {
		if s.Battery.Percent <= 5 {
			raise(Critical, "Battery nearly empty")
		} else if s.Battery.Percent <= cfg.Alerts.BatteryPercent {
			raise(Warning, "Battery low")
		}
	}
	if s.CPU.Valid && s.CPU.Total >= cfg.Alerts.CPUPercent {
		raise(Warning, "CPU saturated")
	}
	return sev, reason
}

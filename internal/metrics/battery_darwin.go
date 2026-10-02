//go:build darwin

package metrics

import (
	"context"
	"errors"
	"time"
)

func readBattery(ctx context.Context) BatteryStats {
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := command(cctx, "/usr/sbin/ioreg", "-rn", "AppleSmartBattery", "-a").Output()
	if err != nil {
		return BatteryStats{Err: err}
	}
	return parseBattery(out)
}

// parseBattery interprets `ioreg -rn AppleSmartBattery -a` output. An empty
// result means the Mac has no battery (desktop Macs).
func parseBattery(out []byte) BatteryStats {
	var st BatteryStats
	if len(out) == 0 {
		st.Valid = true // valid answer: no battery
		return st
	}
	v, err := parsePlist(out)
	if err != nil {
		st.Err = err
		return st
	}
	arr, _ := v.([]any)
	if len(arr) == 0 {
		st.Valid = true
		return st
	}
	m, _ := arr[0].(map[string]any)
	d := plistDict(m)
	if installed, ok := d.bool("BatteryInstalled"); ok && !installed {
		st.Valid = true
		return st
	}
	cur, ok := d.int("CurrentCapacity")
	maxPct, ok2 := d.int("MaxCapacity")
	if !ok || !ok2 || maxPct <= 0 {
		st.Err = errors.New("battery capacity not reported")
		return st
	}
	st.Present = true
	st.Valid = true
	// On Apple Silicon CurrentCapacity/MaxCapacity are percentages; on
	// Intel they are mAh. The ratio is correct in both cases.
	st.Percent = clampPct(100 * float64(cur) / float64(maxPct))
	st.Charging, _ = d.bool("IsCharging")
	st.ACPower, _ = d.bool("ExternalConnected")
	st.FullyCharged, _ = d.bool("FullyCharged")
	if c, ok := d.int("CycleCount"); ok {
		st.CycleCount = Some(int(c))
	}
	if c, ok := d.int("DesignCycleCount9C"); ok && c > 0 {
		st.DesignCycles = Some(int(c))
	}
	design, okD := d.int("DesignCapacity")
	rawMax, okM := d.int("AppleRawMaxCapacity")
	if !okM {
		rawMax, okM = d.int("NominalChargeCapacity")
	}
	if okD && design > 0 {
		st.DesignCap = Some(int(design))
	}
	if okM && rawMax > 0 {
		st.MaxCapacity = Some(int(rawMax))
	}
	if okD && okM && design > 0 && rawMax > 0 {
		st.Health = Some(min(100*float64(rawMax)/float64(design), 100))
	}
	if mv, ok := d.int("Voltage"); ok && mv > 0 {
		st.Voltage = Some(float64(mv) / 1000)
	}
	amp, okA := d.int("InstantAmperage")
	if !okA {
		amp, okA = d.int("Amperage")
	}
	if okA {
		st.Amperage = Some(float64(amp))
		if st.Voltage.OK {
			st.Power = Some(st.Voltage.V * float64(amp) / 1000)
		}
	}
	if t, ok := d.int("Temperature"); ok && t > 0 {
		st.Temperature = Some(float64(t) / 100)
	}
	// 65535 means "still calculating"; only report reliable estimates.
	if !st.ACPower {
		if m, ok := d.int("AvgTimeToEmpty"); ok && m > 0 && m < 65535 {
			st.TimeToEmpty = Some(time.Duration(m) * time.Minute)
		}
	} else if st.Charging {
		if m, ok := d.int("AvgTimeToFull"); ok && m > 0 && m < 65535 {
			st.TimeToFull = Some(time.Duration(m) * time.Minute)
		}
	}
	if ad := d.dict("AdapterDetails"); ad != nil {
		if w, ok := ad.int("Watts"); ok && w > 0 {
			st.AdapterWatts = Some(float64(w))
		}
	}
	if pt := d.dict("PowerTelemetryData"); pt != nil {
		if mw, ok := pt.int("SystemLoad"); ok && mw > 0 {
			st.SystemLoad = Some(float64(mw) / 1000)
		}
	}
	if pf, ok := d.int("PermanentFailureStatus"); ok && pf != 0 {
		st.Condition = "Service recommended"
	} else if st.Health.OK && st.Health.V < 80 {
		st.Condition = "Service recommended"
	} else {
		st.Condition = "Normal"
	}
	return st
}

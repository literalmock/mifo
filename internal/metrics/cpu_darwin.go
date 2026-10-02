//go:build darwin

package metrics

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
)

type cpuSampler struct {
	prev      []cpu.TimesStat
	kinds     []string
	kindsDone bool
	maxFreq   Opt[float64]
	freqDone  bool
}

func busyIdle(t cpu.TimesStat) (busy, idle float64) {
	busy = t.User + t.System + t.Nice + t.Irq + t.Softirq + t.Steal
	return busy, t.Idle + t.Iowait
}

var cpuBlockRe = regexp.MustCompile(`"logical-cpu-id" = (\d+)|"cluster-type" = <"([A-Z])">`)

// readCoreKinds reads each logical CPU's cluster type (P/E) from the IOKit
// device tree. This works without privileges on Apple Silicon.
func readCoreKinds(ctx context.Context, n int) []string {
	kinds := make([]string, n)
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := command(cctx, "/usr/sbin/ioreg", "-p", "IODeviceTree", "-l", "-w0", "-c", "IOPlatformDevice").Output()
	if err != nil {
		return kinds
	}
	for _, block := range strings.Split(string(out), "+-o cpu")[1:] {
		id, kind := -1, ""
		for _, m := range cpuBlockRe.FindAllStringSubmatch(block, -1) {
			if m[1] != "" && id < 0 {
				id, _ = strconv.Atoi(m[1])
			}
			if m[2] != "" && kind == "" {
				kind = m[2]
			}
		}
		if id >= 0 && id < n {
			kinds[id] = kind
		}
	}
	return kinds
}

func (c *cpuSampler) sample(ctx context.Context) CPUStats {
	var st CPUStats
	cur, err := cpu.TimesWithContext(ctx, true)
	if err != nil || len(cur) == 0 {
		st.Err = err
		return st
	}
	if !c.kindsDone {
		c.kinds = readCoreKinds(ctx, len(cur))
		c.kindsDone = true
	}
	if !c.freqDone {
		c.freqDone = true
		if info, err := cpu.InfoWithContext(ctx); err == nil && len(info) > 0 && info[0].Mhz > 0 {
			c.maxFreq = Some(info[0].Mhz)
		}
	}
	st.CoreKinds = c.kinds
	st.MaxFreq = c.maxFreq
	if l, err := load.AvgWithContext(ctx); err == nil {
		st.Load = Some([3]float64{l.Load1, l.Load5, l.Load15})
	}
	prev := c.prev
	c.prev = cur
	if len(prev) != len(cur) {
		return st // need two samples for utilisation
	}
	var tot cpu.TimesStat
	st.PerCore = make([]float64, len(cur))
	for i := range cur {
		d := diffTimes(cur[i], prev[i])
		b, id := busyIdle(d)
		if b+id > 0 {
			st.PerCore[i] = clampPct(100 * b / (b + id))
		}
		tot.User += d.User
		tot.System += d.System
		tot.Nice += d.Nice
		tot.Idle += d.Idle
		tot.Iowait += d.Iowait
		tot.Irq += d.Irq
		tot.Softirq += d.Softirq
		tot.Steal += d.Steal
	}
	b, id := busyIdle(tot)
	all := b + id
	if all <= 0 {
		return st
	}
	st.Total = clampPct(100 * b / all)
	st.User = clampPct(100 * tot.User / all)
	st.System = clampPct(100 * tot.System / all)
	st.Nice = clampPct(100 * tot.Nice / all)
	st.Idle = clampPct(100 * id / all)
	st.Valid = true
	return st
}

func diffTimes(a, b cpu.TimesStat) cpu.TimesStat {
	pos := func(x float64) float64 { return max(x, 0) }
	return cpu.TimesStat{
		User: pos(a.User - b.User), System: pos(a.System - b.System), Nice: pos(a.Nice - b.Nice),
		Idle: pos(a.Idle - b.Idle), Iowait: pos(a.Iowait - b.Iowait), Irq: pos(a.Irq - b.Irq),
		Softirq: pos(a.Softirq - b.Softirq), Steal: pos(a.Steal - b.Steal),
	}
}

func clampPct(v float64) float64 { return min(max(v, 0), 100) }

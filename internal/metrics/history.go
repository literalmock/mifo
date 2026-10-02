package metrics

import "time"

// Sample is a timestamped value.
type Sample struct {
	T time.Time
	V float64
}

// Ring is a fixed-capacity ring buffer of samples. It never grows, so memory
// use is bounded regardless of how long MIFO runs.
type Ring struct {
	buf  []Sample
	head int // index of next write
	n    int
}

// NewRing creates a ring with the given capacity.
func NewRing(capacity int) *Ring {
	if capacity < 1 {
		capacity = 1
	}
	return &Ring{buf: make([]Sample, capacity)}
}

// Push appends a sample, overwriting the oldest when full.
func (r *Ring) Push(t time.Time, v float64) {
	r.buf[r.head] = Sample{T: t, V: v}
	r.head = (r.head + 1) % len(r.buf)
	if r.n < len(r.buf) {
		r.n++
	}
}

// Len returns the number of stored samples.
func (r *Ring) Len() int { return r.n }

// Cap returns the capacity.
func (r *Ring) Cap() int { return len(r.buf) }

// Last returns the most recent sample.
func (r *Ring) Last() (Sample, bool) {
	if r.n == 0 {
		return Sample{}, false
	}
	return r.buf[(r.head-1+len(r.buf))%len(r.buf)], true
}

// at returns the i-th oldest sample.
func (r *Ring) at(i int) Sample {
	start := (r.head - r.n + len(r.buf)) % len(r.buf)
	return r.buf[(start+i)%len(r.buf)]
}

// Since appends to dst the values of samples newer than cutoff, oldest first,
// and returns the extended slice. Passing a reused dst avoids allocations.
func (r *Ring) Since(cutoff time.Time, dst []float64) []float64 {
	// Binary search for the first sample after cutoff.
	lo, hi := 0, r.n
	for lo < hi {
		mid := (lo + hi) / 2
		if r.at(mid).T.After(cutoff) {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	for i := lo; i < r.n; i++ {
		dst = append(dst, r.at(i).V)
	}
	return dst
}

// History stores bounded time series for the charts.
type History struct {
	CPU     *Ring
	Memory  *Ring
	Swap    *Ring
	NetRx   *Ring
	NetTx   *Ring
	DiskR   *Ring
	DiskW   *Ring
	CPUTemp *Ring
	Power   *Ring
	Battery *Ring
	Cores   []*Ring
	// Iface holds per-interface [rx, tx] throughput rings.
	Iface map[string][2]*Ring
	capac int
}

// HistoryCapacity holds one hour of samples at the fastest (1s) interval.
const HistoryCapacity = 3600

// NewHistory allocates all rings.
func NewHistory(capacity int) *History {
	return &History{
		CPU: NewRing(capacity), Memory: NewRing(capacity), Swap: NewRing(capacity),
		NetRx: NewRing(capacity), NetTx: NewRing(capacity),
		DiskR: NewRing(capacity), DiskW: NewRing(capacity),
		CPUTemp: NewRing(capacity), Power: NewRing(capacity), Battery: NewRing(capacity),
		Iface: map[string][2]*Ring{},
		capac: capacity,
	}
}

// Record pushes the valid parts of a snapshot. Invalid metrics are skipped so
// gaps never appear as fake zeros.
func (h *History) Record(s Snapshot) {
	t := s.Time
	if s.CPU.Valid {
		h.CPU.Push(t, s.CPU.Total)
		if len(h.Cores) != len(s.CPU.PerCore) {
			h.Cores = make([]*Ring, len(s.CPU.PerCore))
			for i := range h.Cores {
				h.Cores[i] = NewRing(min(h.capac, 300))
			}
		}
		for i, v := range s.CPU.PerCore {
			h.Cores[i].Push(t, v)
		}
	}
	if s.Memory.Valid {
		h.Memory.Push(t, s.Memory.UsedPercent)
		if s.Memory.Swap.OK {
			h.Swap.Push(t, float64(s.Memory.Swap.V.Used))
		}
	}
	if s.Network.Valid {
		h.NetRx.Push(t, s.Network.RxBps)
		h.NetTx.Push(t, s.Network.TxBps)
		for _, ifc := range s.Network.Interfaces {
			r, ok := h.Iface[ifc.Name]
			if !ok {
				if len(h.Iface) >= 64 {
					continue
				}
				r = [2]*Ring{NewRing(h.capac), NewRing(h.capac)}
				h.Iface[ifc.Name] = r
			}
			r[0].Push(t, ifc.RxBps)
			r[1].Push(t, ifc.TxBps)
		}
	}
	if s.Disk.IOValid {
		h.DiskR.Push(t, s.Disk.ReadBps)
		h.DiskW.Push(t, s.Disk.WriteBps)
	}
	if s.Sensors.CPUTemp.OK {
		h.CPUTemp.Push(t, s.Sensors.CPUTemp.V)
	}
	if s.Battery.Valid && s.Battery.Present {
		h.Battery.Push(t, s.Battery.Percent)
	}
	for _, p := range s.Sensors.Power {
		if p.Name == "System total" {
			h.Power.Push(t, p.Watts)
		}
	}
}

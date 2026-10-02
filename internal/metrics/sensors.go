package metrics

import (
	"context"
	"sort"
	"strings"
)

// SensorProvider is a hardware sensor backend. New backends (for example a
// privileged powermetrics helper) can be added by implementing this
// interface and registering it with the collector.
type SensorProvider interface {
	Name() string
	// Read fills the parts of out that this provider supports. It returns
	// an error when the backend is unavailable on this Mac.
	Read(ctx context.Context, out *SensorStats) error
}

// classifyTemperature assigns a group to a raw sensor name.
func classifyTemperature(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "gpu"), strings.HasPrefix(name, "TG"):
		return "GPU"
	case strings.Contains(n, "battery"), strings.HasPrefix(name, "TB"):
		return "Battery"
	case strings.Contains(n, "nand"), strings.Contains(n, "ssd"), strings.HasPrefix(name, "TH"):
		return "Storage"
	case strings.Contains(n, "tdie"), strings.Contains(n, "cpu"), strings.Contains(n, "soc"),
		strings.HasPrefix(name, "TC"), strings.HasPrefix(name, "Tp"), strings.HasPrefix(name, "Te"):
		return "SoC"
	}
	return "Other"
}

// plausibleTemp filters out disconnected or uncalibrated sensors, which
// report negative or absurd values. We never show these as readings.
func plausibleTemp(c float64) bool { return c > 5 && c < 130 }

// summarizeTemps computes the aggregate SoC/GPU temperatures.
func summarizeTemps(st *SensorStats) {
	sort.SliceStable(st.Temps, func(i, j int) bool {
		if st.Temps[i].Group != st.Temps[j].Group {
			return groupRank(st.Temps[i].Group) < groupRank(st.Temps[j].Group)
		}
		return st.Temps[i].Name < st.Temps[j].Name
	})
	var sum, mx, gsum float64
	var n, gn int
	for _, t := range st.Temps {
		switch t.Group {
		case "SoC":
			sum += t.Celsius
			mx = max(mx, t.Celsius)
			n++
		case "GPU":
			gsum += t.Celsius
			gn++
		}
	}
	if n > 0 {
		st.CPUTemp = Some(sum / float64(n))
		st.CPUMax = Some(mx)
	}
	if gn > 0 {
		st.GPUTemp = Some(gsum / float64(gn))
	}
}

func groupRank(g string) int {
	switch g {
	case "SoC":
		return 0
	case "GPU":
		return 1
	case "Storage":
		return 2
	case "Battery":
		return 3
	}
	return 4
}

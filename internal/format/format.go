// Package format renders numbers, sizes and durations for display.
package format

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Bytes formats a size using binary multiples, labelled the way macOS
// labels memory (1 GB = 1024³ bytes).
func Bytes(b uint64) string {
	return scaled(float64(b), 1024, "B")
}

// BytesSI formats a size using decimal multiples, as Finder does for storage.
func BytesSI(b uint64) string {
	return scaled(float64(b), 1000, "B")
}

// Rate formats a throughput in bytes per second (decimal, like Activity
// Monitor).
func Rate(bps float64) string {
	if bps < 0 || math.IsNaN(bps) {
		bps = 0
	}
	return scaled(bps, 1000, "B") + "/s"
}

func scaled(v, base float64, unit string) string {
	units := []string{"", "K", "M", "G", "T", "P"}
	i := 0
	for v >= base && i < len(units)-1 {
		v /= base
		i++
	}
	switch {
	case i == 0:
		return fmt.Sprintf("%.0f %s", v, unit)
	case v >= 100:
		return fmt.Sprintf("%.0f %s%s", v, units[i], unit)
	case v >= 10:
		return fmt.Sprintf("%.1f %s%s", v, units[i], unit)
	}
	return fmt.Sprintf("%.2f %s%s", v, units[i], unit)
}

// Percent formats a percentage with sensible precision.
func Percent(p float64) string {
	if p >= 99.95 {
		return "100%"
	}
	if p < 10 {
		return fmt.Sprintf("%.1f%%", p)
	}
	return fmt.Sprintf("%.0f%%", p)
}

// Uptime formats a long duration compactly: "9d 4h", "3h 12m", "45m".
func Uptime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// Duration formats a remaining-time estimate: "2h 05m" or "42m".
func Duration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// CPUTime formats accumulated CPU time like Activity Monitor: "1:02:03.45".
func CPUTime(d time.Duration) string {
	total := d.Seconds()
	h := int(total) / 3600
	m := (int(total) % 3600) / 60
	s := math.Mod(total, 60)
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%05.2f", h, m, s)
	}
	return fmt.Sprintf("%d:%05.2f", m, s)
}

// Freq formats a frequency in MHz.
func Freq(mhz float64) string {
	if mhz >= 1000 {
		return fmt.Sprintf("%.2f GHz", mhz/1000)
	}
	return fmt.Sprintf("%.0f MHz", mhz)
}

// Temp formats a temperature in Celsius.
func Temp(c float64) string { return fmt.Sprintf("%.1f°C", c) }

// TempASCII formats a temperature without the degree sign.
func TempASCII(c float64) string { return fmt.Sprintf("%.1f C", c) }

// Watts formats power.
func Watts(w float64) string {
	if math.Abs(w) < 10 {
		return fmt.Sprintf("%.2f W", w)
	}
	return fmt.Sprintf("%.1f W", w)
}

// Truncate shortens s to at most n display cells, adding an ellipsis.
func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

// TruncateLeft keeps the end of s (useful for paths).
func TruncateLeft(s string, n int) string {
	r := []rune(s)
	if len(r) <= n || n <= 1 {
		return s
	}
	return "…" + string(r[len(r)-n+1:])
}

// PadRight pads s with spaces to width n (by rune count).
func PadRight(s string, n int) string {
	l := len([]rune(s))
	if l >= n {
		return s
	}
	return s + strings.Repeat(" ", n-l)
}

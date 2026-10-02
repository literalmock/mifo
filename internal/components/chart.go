// Package components contains reusable rendering primitives for MIFO.
package components

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/styles"
)

// Resample maps values onto exactly n points. Longer inputs are bucketed
// (keeping each bucket's maximum so short spikes stay visible); shorter
// inputs are right-aligned with NaN padding so charts fill from the right.
// dst is reused when it has enough capacity.
func Resample(values []float64, n int, dst []float64) []float64 {
	if n <= 0 {
		return dst[:0]
	}
	if cap(dst) < n {
		dst = make([]float64, n)
	}
	dst = dst[:n]
	if len(values) <= n {
		pad := n - len(values)
		for i := 0; i < pad; i++ {
			dst[i] = math.NaN()
		}
		copy(dst[pad:], values)
		return dst
	}
	for i := 0; i < n; i++ {
		lo := i * len(values) / n
		hi := (i + 1) * len(values) / n
		if hi <= lo {
			hi = lo + 1
		}
		m := values[lo]
		for _, v := range values[lo+1 : hi] {
			m = max(m, v)
		}
		dst[i] = m
	}
	return dst
}

func minOf(v []float64) float64 {
	m := math.Inf(1)
	for _, x := range v {
		if !math.IsNaN(x) && x < m {
			m = x
		}
	}
	if math.IsInf(m, 1) {
		return 0
	}
	return m
}

// MaxOf returns the largest finite value, or 0.
func MaxOf(vs ...[]float64) float64 {
	m := 0.0
	for _, v := range vs {
		for _, x := range v {
			if !math.IsNaN(x) && x > m {
				m = x
			}
		}
	}
	return m
}

// Sparkline renders a one-line chart of the given width scaled from zero to
// maxV (or the data maximum when maxV <= 0).
func Sparkline(t styles.Theme, values []float64, width int, maxV float64, c color.Color) string {
	return SparklineRange(t, values, width, 0, maxV, c)
}

// SparklineRange renders a one-line chart scaled between lo and hi. When
// hi <= lo the range is taken from the data, which suits slowly varying
// values such as temperatures.
func SparklineRange(t styles.Theme, values []float64, width int, lo, hi float64, c color.Color) string {
	if width <= 0 {
		return ""
	}
	pts := Resample(values, width, nil)
	if hi <= lo {
		hi = MaxOf(pts)
		if lo > 0 || hi <= 0 {
			lo = minOf(pts)
		}
	}
	span := hi - lo
	levels := t.Sym.Spark
	var sb strings.Builder
	for _, v := range pts {
		if math.IsNaN(v) || span <= 0 {
			if !math.IsNaN(v) && span <= 0 && hi > 0 {
				sb.WriteString(levels[len(levels)/2])
				continue
			}
			sb.WriteString(" ")
			continue
		}
		idx := int(math.Round((v - lo) / span * float64(len(levels)-1)))
		idx = min(max(idx, 0), len(levels)-1)
		if v > 0 && idx == 0 {
			idx = 1
		}
		sb.WriteString(levels[idx])
	}
	return lipgloss.NewStyle().Foreground(c).Render(sb.String())
}

var brailleBits = [2][4]rune{{0x01, 0x02, 0x04, 0x40}, {0x08, 0x10, 0x20, 0x80}}

// areaGrid computes per-cell braille runes for an area chart that grows
// upward (up=true) or downward from the top edge.
func areaGrid(values []float64, width, height int, maxV float64, up bool) [][]rune {
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
	}
	pts := Resample(values, width*2, nil)
	dots := height * 4
	for x, v := range pts {
		if math.IsNaN(v) || maxV <= 0 {
			continue
		}
		level := int(math.Round(v / maxV * float64(dots)))
		if v > 0 && level == 0 {
			level = 1 // show non-zero activity as at least one dot
		}
		level = min(level, dots)
		col, sub := x/2, x%2
		for d := 0; d < level; d++ {
			var row, dot int
			if up {
				row = height - 1 - d/4
				dot = 3 - d%4
			} else {
				row = d / 4
				dot = d % 4
			}
			grid[row][col] |= brailleBits[sub][dot]
		}
	}
	return grid
}

func renderGrid(t styles.Theme, grid [][]rune, c color.Color) []string {
	st := lipgloss.NewStyle().Foreground(c)
	lines := make([]string, len(grid))
	for i, row := range grid {
		var sb strings.Builder
		for _, r := range row {
			switch {
			case r == 0:
				sb.WriteByte(' ')
			case t.Sym.Unicode:
				sb.WriteRune(0x2800 + r)
			default:
				sb.WriteByte(asciiDensity(r))
			}
		}
		lines[i] = st.Render(sb.String())
	}
	return lines
}

func asciiDensity(r rune) byte {
	n := 0
	for b := r; b != 0; b &= b - 1 {
		n++
	}
	switch {
	case n >= 7:
		return '#'
	case n >= 5:
		return '+'
	case n >= 3:
		return ':'
	}
	return '.'
}

// AreaChart renders a braille area chart with a right-hand scale.
// label formats axis values (e.g. percent or bytes/s).
func AreaChart(t styles.Theme, values []float64, width, height int, maxV float64, c color.Color, label func(float64) string) string {
	if width < 4 || height < 1 {
		return ""
	}
	top, bot := label(maxV), label(0)
	axisW := max(lipgloss.Width(top), lipgloss.Width(bot)) + 1
	plotW := width - axisW
	if plotW < 2 {
		plotW, axisW = width, 0
	}
	grid := areaGrid(values, plotW, height, maxV, true)
	lines := renderGrid(t, grid, c)
	for i := range lines {
		if axisW == 0 {
			continue
		}
		var a string
		switch i {
		case 0:
			a = top
		case height - 1:
			a = bot
		}
		lines[i] += t.Faint.Render(" " + padLeft(a, axisW-1))
	}
	return strings.Join(lines, "\n")
}

// MirrorChart renders two series around a centre line: the first grows up
// (e.g. download), the second grows down (e.g. upload). Both share a scale.
func MirrorChart(t styles.Theme, upVals, downVals []float64, width, height int, cUp, cDown color.Color, label func(float64) string) string {
	if width < 4 || height < 2 {
		return ""
	}
	maxV := niceMax(MaxOf(upVals, downVals))
	topH := height / 2
	botH := height - topH
	ax := label(maxV)
	axisW := lipgloss.Width(ax) + 1
	plotW := width - axisW
	if plotW < 2 {
		plotW, axisW = width, 0
	}
	upLines := renderGrid(t, areaGrid(upVals, plotW, topH, maxV, true), cUp)
	downLines := renderGrid(t, areaGrid(downVals, plotW, botH, maxV, false), cDown)
	lines := append(upLines, downLines...)
	if axisW > 0 {
		for i := range lines {
			a := ""
			if i == 0 || i == len(lines)-1 {
				a = ax
			}
			lines[i] += t.Faint.Render(" " + padLeft(a, axisW-1))
		}
	}
	return strings.Join(lines, "\n")
}

// niceMax rounds a maximum up to a 1/2/5 × 10^n step so axes read cleanly.
func niceMax(v float64) float64 {
	if v <= 0 {
		return 1
	}
	exp := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range []float64{1, 2, 5, 10} {
		if v <= m*exp {
			return m * exp
		}
	}
	return 10 * exp
}

// NiceMax is exported for callers that want a consistent scale.
func NiceMax(v float64) float64 { return niceMax(v) }

func padLeft(s string, w int) string {
	if n := lipgloss.Width(s); n < w {
		return strings.Repeat(" ", w-n) + s
	}
	return s
}

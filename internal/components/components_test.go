package components

import (
	"math"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/styles"
)

var theme = styles.New(styles.DarkPalette, styles.UnicodeSymbols)

func TestResample(t *testing.T) {
	got := Resample([]float64{1, 2}, 4, nil)
	if !math.IsNaN(got[0]) || !math.IsNaN(got[1]) || got[2] != 1 || got[3] != 2 {
		t.Errorf("pad = %v", got)
	}
	got = Resample([]float64{1, 5, 2, 2, 9, 1}, 3, nil)
	if got[0] != 5 || got[1] != 2 || got[2] != 9 {
		t.Errorf("bucket max = %v", got)
	}
	if len(Resample([]float64{1}, 0, nil)) != 0 {
		t.Error("zero width")
	}
}

func TestPanelExactSize(t *testing.T) {
	for _, sz := range [][2]int{{20, 5}, {60, 12}, {7, 3}} {
		p := Panel(theme, "A very long panel title that will not fit", "info", "line one\nline two that is much longer than the panel width\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13", sz[0], sz[1], true)
		lines := strings.Split(p, "\n")
		if len(lines) != sz[1] {
			t.Errorf("%v: height %d", sz, len(lines))
		}
		for i, l := range lines {
			if w := lipgloss.Width(l); w != sz[0] {
				t.Errorf("%v: line %d width %d", sz, i, w)
			}
		}
	}
}

func TestFit(t *testing.T) {
	out := Fit("abcdef\nxy", 4, 3)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 || lines[0] != "abcd" || lines[1] != "xy  " || lines[2] != "    " {
		t.Errorf("fit = %q", lines)
	}
}

func TestBarWidth(t *testing.T) {
	for _, pct := range []float64{-5, 0, 0.1, 50, 100, 150} {
		if w := lipgloss.Width(Bar(theme, pct, 20, theme.P.Accent)); w != 20 {
			t.Errorf("bar(%v) width %d", pct, w)
		}
	}
}

func TestChartsFillBox(t *testing.T) {
	vals := []float64{1, 5, 3, 8, 2, 9, 4}
	lbl := func(v float64) string { return "x" }
	for _, c := range []string{
		AreaChart(theme, vals, 30, 6, 10, theme.P.Accent, lbl),
		MirrorChart(theme, vals, vals, 30, 6, theme.P.Accent, theme.P.Secondary, lbl),
	} {
		lines := strings.Split(c, "\n")
		if len(lines) != 6 {
			t.Errorf("height %d", len(lines))
		}
		for _, l := range lines {
			if w := lipgloss.Width(l); w != 30 {
				t.Errorf("width %d", w)
			}
		}
	}
}

func TestASCIIChartHasNoBraille(t *testing.T) {
	at := styles.New(styles.DarkPalette, styles.ASCIISymbols)
	c := AreaChart(at, []float64{1, 2, 3, 4}, 20, 4, 4, at.P.Accent, func(float64) string { return "" })
	for _, r := range c {
		if r >= 0x2800 && r <= 0x28FF {
			t.Fatal("braille rune in ASCII mode")
		}
	}
}

func TestNiceMax(t *testing.T) {
	for in, want := range map[float64]float64{0: 1, 0.7: 1, 3: 5, 12: 20, 4500: 5000, 100: 100} {
		if got := niceMax(in); got != want {
			t.Errorf("niceMax(%v) = %v want %v", in, got, want)
		}
	}
}

func TestSplit(t *testing.T) {
	s := Split(10, 3, 1)
	if s[0]+s[1]+s[2]+2 != 10 {
		t.Errorf("split = %v", s)
	}
}

func TestBigText(t *testing.T) {
	r := BigText("42%", true)
	if lipgloss.Width(r[0]) != lipgloss.Width(r[2]) || r[0] == "" {
		t.Errorf("big text rows uneven: %q", r)
	}
}

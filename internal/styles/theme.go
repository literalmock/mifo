// Package styles defines MIFO's palette, typography and symbols.
package styles

import (
	"image/color"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

// Palette is a set of semantic colours.
type Palette struct {
	Name      string
	Dark      bool
	Bg        color.Color
	Surface   color.Color
	Border    color.Color
	BorderHi  color.Color
	Text      color.Color
	Muted     color.Color
	Faint     color.Color
	Accent    color.Color
	AccentBg  color.Color
	Secondary color.Color // cyan/blue series
	Tertiary  color.Color
	Warn      color.Color
	Crit      color.Color
	Track     color.Color
}

// DarkPalette is the default deep-charcoal palette.
var DarkPalette = Palette{
	Name:      "dark",
	Dark:      true,
	Bg:        lipgloss.Color("#141518"),
	Surface:   lipgloss.Color("#1b1c20"),
	Border:    lipgloss.Color("#2c2e34"),
	BorderHi:  lipgloss.Color("#3d7a62"),
	Text:      lipgloss.Color("#e4e5e9"),
	Muted:     lipgloss.Color("#8a8e98"),
	Faint:     lipgloss.Color("#555963"),
	Accent:    lipgloss.Color("#3ecf8e"),
	AccentBg:  lipgloss.Color("#1d3329"),
	Secondary: lipgloss.Color("#5ac8fa"),
	Tertiary:  lipgloss.Color("#a78bfa"),
	Warn:      lipgloss.Color("#f5b04c"),
	Crit:      lipgloss.Color("#ff6058"),
	Track:     lipgloss.Color("#2a2c31"),
}

// LightPalette is a soft light palette.
var LightPalette = Palette{
	Name:      "light",
	Dark:      false,
	Bg:        lipgloss.Color("#f6f6f7"),
	Surface:   lipgloss.Color("#ffffff"),
	Border:    lipgloss.Color("#d6d8dd"),
	BorderHi:  lipgloss.Color("#2fa676"),
	Text:      lipgloss.Color("#1d1d1f"),
	Muted:     lipgloss.Color("#6b6e76"),
	Faint:     lipgloss.Color("#a3a6ad"),
	Accent:    lipgloss.Color("#0e9f6e"),
	AccentBg:  lipgloss.Color("#dcf2e8"),
	Secondary: lipgloss.Color("#0a7fd1"),
	Tertiary:  lipgloss.Color("#7c5cd6"),
	Warn:      lipgloss.Color("#b86e00"),
	Crit:      lipgloss.Color("#d1242f"),
	Track:     lipgloss.Color("#e3e4e8"),
}

// Symbols holds glyphs with ASCII fallbacks.
type Symbols struct {
	Unicode  bool
	BarFull  string
	BarEmpty string
	Select   string
	Dot      string
	Down     string
	Up       string
	Degree   string
	Sep      string
	Check    string
	Cross    string
	Ellipsis string
	Spark    []string
	Logo     string
	Border   lipgloss.Border
}

// UnicodeSymbols are used on capable terminals.
var UnicodeSymbols = Symbols{
	Unicode: true, BarFull: "━", BarEmpty: "━", Select: "▍", Dot: "●",
	Down: "↓", Up: "↑", Degree: "°", Sep: "·", Check: "✓", Cross: "✕", Ellipsis: "…",
	Spark:  []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"},
	Logo:   "◆",
	Border: lipgloss.RoundedBorder(),
}

// ASCIISymbols are the fallback for limited terminals and fonts.
var ASCIISymbols = Symbols{
	Unicode: false, BarFull: "=", BarEmpty: "-", Select: ">", Dot: "*",
	Down: "v", Up: "^", Degree: " ", Sep: "|", Check: "+", Cross: "x", Ellipsis: "...",
	Spark:  []string{" ", ".", ".", ":", ":", "+", "+", "#", "#"},
	Logo:   "*",
	Border: lipgloss.NormalBorder(),
}

// DetectUnicode guesses whether the terminal can render Unicode glyphs.
func DetectUnicode() bool {
	if os.Getenv("TERM") == "linux" {
		return false
	}
	for _, v := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if s := os.Getenv(v); s != "" {
			s = strings.ToUpper(s)
			return strings.Contains(s, "UTF-8") || strings.Contains(s, "UTF8")
		}
	}
	// macOS terminals are UTF-8 by default even when LANG is unset.
	return true
}

// Theme bundles the palette with prebuilt styles.
type Theme struct {
	P   Palette
	Sym Symbols

	Base      lipgloss.Style
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Label     lipgloss.Style
	Value     lipgloss.Style
	BigValue  lipgloss.Style
	Muted     lipgloss.Style
	Faint     lipgloss.Style
	Accent    lipgloss.Style
	Second    lipgloss.Style
	Third     lipgloss.Style
	Warn      lipgloss.Style
	Crit      lipgloss.Style
	Panel     lipgloss.Style
	PanelHi   lipgloss.Style
	Key       lipgloss.Style
	Selected  lipgloss.Style
	TableHead lipgloss.Style
	Badge     lipgloss.Style
}

// New builds a theme.
func New(p Palette, sym Symbols) Theme {
	t := Theme{P: p, Sym: sym}
	t.Base = lipgloss.NewStyle().Foreground(p.Text)
	t.Title = lipgloss.NewStyle().Foreground(p.Text).Bold(true)
	t.Subtitle = lipgloss.NewStyle().Foreground(p.Muted)
	t.Label = lipgloss.NewStyle().Foreground(p.Muted)
	t.Value = lipgloss.NewStyle().Foreground(p.Text)
	t.BigValue = lipgloss.NewStyle().Foreground(p.Text).Bold(true)
	t.Muted = lipgloss.NewStyle().Foreground(p.Muted)
	t.Faint = lipgloss.NewStyle().Foreground(p.Faint)
	t.Accent = lipgloss.NewStyle().Foreground(p.Accent)
	t.Second = lipgloss.NewStyle().Foreground(p.Secondary)
	t.Third = lipgloss.NewStyle().Foreground(p.Tertiary)
	t.Warn = lipgloss.NewStyle().Foreground(p.Warn)
	t.Crit = lipgloss.NewStyle().Foreground(p.Crit)
	t.Panel = lipgloss.NewStyle().Border(sym.Border).BorderForeground(p.Border).Padding(0, 1)
	t.PanelHi = t.Panel.BorderForeground(p.BorderHi)
	t.Key = lipgloss.NewStyle().Foreground(p.Text).Bold(true)
	t.Selected = lipgloss.NewStyle().Background(p.AccentBg).Foreground(p.Text)
	t.TableHead = lipgloss.NewStyle().Foreground(p.Muted).Bold(true)
	t.Badge = lipgloss.NewStyle().Padding(0, 1)
	return t
}

// Level returns the style for a 0-100 load value using thresholds.
func (t Theme) Level(pct, warn, crit float64) lipgloss.Style {
	switch {
	case pct >= crit:
		return t.Crit
	case pct >= warn:
		return t.Warn
	}
	return t.Accent
}

// LevelColor returns the colour for a load value.
func (t Theme) LevelColor(pct, warn, crit float64) color.Color {
	switch {
	case pct >= crit:
		return t.P.Crit
	case pct >= warn:
		return t.P.Warn
	}
	return t.P.Accent
}

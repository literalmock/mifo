package pages

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/config"
)

type settingKind int

const (
	kindChoice settingKind = iota
	kindNumber
	kindAction
)

type setting struct {
	section string
	label   string
	help    string
	kind    settingKind
	// choice
	value func(config.Config) string
	cycle func(config.Config, int) config.Config
	// number
	num    func(config.Config) float64
	setNum func(config.Config, float64) config.Config
	lo, hi float64
	unit   string
	// action
	run func(*Settings, *Context) tea.Cmd
}

func cycleStr(opts []string, cur string, dir int) string {
	i := slices.Index(opts, cur)
	if i < 0 {
		return opts[0]
	}
	return opts[(i+dir+len(opts))%len(opts)]
}

func cycleInt(opts []int, cur, dir int) int {
	i := slices.Index(opts, cur)
	if i < 0 {
		return opts[0]
	}
	return opts[(i+dir+len(opts))%len(opts)]
}

func onOff(b bool) string {
	if b {
		return "On"
	}
	return "Off"
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func buildSettings() []setting {
	widgetLabels := map[string]string{
		"cpu": "CPU card", "memory": "Memory card", "disk": "Disk card", "network": "Network card",
		"battery": "Battery / power card", "thermal": "Thermal card", "charts": "History charts", "processes": "Top processes & alerts",
	}
	s := []setting{
		{section: "General", label: "Refresh interval", help: "How often metrics are sampled. Faster uses slightly more CPU.",
			value: func(c config.Config) string { return fmt.Sprintf("%ds", c.IntervalSecs) },
			cycle: func(c config.Config, d int) config.Config {
				c.IntervalSecs = cycleInt(config.Intervals, c.IntervalSecs, d)
				return c
			}},
		{section: "General", label: "Graph time window", help: "History shown in charts. Up to one hour is kept in memory.",
			value: func(c config.Config) string { return windowLabel(c.WindowSecs) },
			cycle: func(c config.Config, d int) config.Config {
				c.WindowSecs = cycleInt(config.Windows, c.WindowSecs, d)
				return c
			}},
		{section: "General", label: "Start page", help: "Page shown when MIFO launches.",
			value: func(c config.Config) string { return title(c.StartPage) },
			cycle: func(c config.Config, d int) config.Config {
				c.StartPage = cycleStr(config.Pages, c.StartPage, d)
				return c
			}},
		{section: "Appearance", label: "Theme", help: "System follows your terminal's background colour.",
			value: func(c config.Config) string { return title(c.Theme) },
			cycle: func(c config.Config, d int) config.Config { c.Theme = cycleStr(config.Themes, c.Theme, d); return c }},
		{section: "Appearance", label: "Symbols", help: "ASCII avoids Unicode glyphs for limited fonts or terminals.",
			value: func(c config.Config) string {
				if c.Symbols == config.SymbolsASCII {
					return "ASCII"
				}
				return title(c.Symbols)
			},
			cycle: func(c config.Config, d int) config.Config {
				c.Symbols = cycleStr(config.SymbolModes, c.Symbols, d)
				return c
			}},
		{section: "Navigation", label: "Sidebar", help: "Auto collapses the sidebar on narrow terminals.",
			value: func(c config.Config) string { return title(c.Sidebar) },
			cycle: func(c config.Config, d int) config.Config {
				c.Sidebar = cycleStr(config.SidebarMode, c.Sidebar, d)
				return c
			}},
		{section: "Navigation", label: "Vim keys (j / k)", help: "Use j and k to move in lists in addition to the arrow keys.",
			value: func(c config.Config) string { return onOff(c.VimKeys) },
			cycle: func(c config.Config, _ int) config.Config { c.VimKeys = !c.VimKeys; return c }},
	}
	for _, name := range config.WidgetNames {
		n := name
		s = append(s, setting{section: "Dashboard widgets", label: widgetLabels[n], help: "Show or hide this widget on the Overview.",
			value: func(c config.Config) string { return onOff(c.Widgets[n]) },
			cycle: func(c config.Config, _ int) config.Config {
				w := make(map[string]bool, len(c.Widgets))
				for k, v := range c.Widgets {
					w[k] = v
				}
				w[n] = !w[n]
				c.Widgets = w
				return c
			}})
	}
	s = append(s,
		setting{section: "Alerts", label: "Alerts", help: "Evaluate thresholds and show alerts. Alerts never fire from missing data.",
			value: func(c config.Config) string { return onOff(c.AlertsEnabled) },
			cycle: func(c config.Config, _ int) config.Config { c.AlertsEnabled = !c.AlertsEnabled; return c }},
		setting{section: "Alerts", label: "CPU above", kind: kindNumber, unit: "%", lo: 10, hi: 100, help: "Alert when total CPU stays above this level.",
			num:    func(c config.Config) float64 { return c.Alerts.CPUPercent },
			setNum: func(c config.Config, v float64) config.Config { c.Alerts.CPUPercent = v; return c }},
		setting{section: "Alerts", label: "CPU sustained for", kind: kindNumber, unit: "s", lo: 0, hi: 3600, help: "Seconds CPU must stay high before alerting.",
			num:    func(c config.Config) float64 { return float64(c.Alerts.CPUSustainSecs) },
			setNum: func(c config.Config, v float64) config.Config { c.Alerts.CPUSustainSecs = int(v); return c }},
		setting{section: "Alerts", label: "Memory pressure", help: "Pressure level that raises an alert.",
			value: func(c config.Config) string { return title(c.Alerts.MemoryPressure) },
			cycle: func(c config.Config, d int) config.Config {
				c.Alerts.MemoryPressure = cycleStr([]string{"warning", "critical"}, c.Alerts.MemoryPressure, d)
				return c
			}},
		setting{section: "Alerts", label: "Free storage below", kind: kindNumber, unit: "%", lo: 1, hi: 50, help: "Alert when a writable volume has less free space than this.",
			num:    func(c config.Config) float64 { return c.Alerts.DiskFreePercent },
			setNum: func(c config.Config, v float64) config.Config { c.Alerts.DiskFreePercent = v; return c }},
		setting{section: "Alerts", label: "Temperature above", kind: kindNumber, unit: "°C", lo: 40, hi: 120, help: "Only used when this Mac exposes reliable die sensors.",
			num:    func(c config.Config) float64 { return c.Alerts.TempCelsius },
			setNum: func(c config.Config, v float64) config.Config { c.Alerts.TempCelsius = v; return c }},
		setting{section: "Alerts", label: "Battery below", kind: kindNumber, unit: "%", lo: 1, hi: 50, help: "Alert when running on battery below this charge.",
			num:    func(c config.Config) float64 { return c.Alerts.BatteryPercent },
			setNum: func(c config.Config, v float64) config.Config { c.Alerts.BatteryPercent = v; return c }},
		setting{section: "Alerts", label: "Cooldown", kind: kindNumber, unit: "s", lo: 10, hi: 86400, help: "Minimum time before the same alert can fire again.",
			num:    func(c config.Config) float64 { return float64(c.Alerts.CooldownSecs) },
			setNum: func(c config.Config, v float64) config.Config { c.Alerts.CooldownSecs = int(v); return c }},
		setting{section: "Alerts", label: "Native notifications", help: "Post macOS notifications via osascript. macOS may ask to allow notifications for your terminal.",
			value: func(c config.Config) string { return onOff(c.NativeNotify) },
			cycle: func(c config.Config, _ int) config.Config { c.NativeNotify = !c.NativeNotify; return c }},
		setting{section: "Configuration", label: "Reset to defaults", kind: kindAction, help: "Restore every setting to its default. Press enter twice to confirm.",
			run: func(s *Settings, c *Context) tea.Cmd {
				if !s.armReset {
					s.armReset = true
					return func() tea.Msg { return StatusMsg{Text: "Press enter again to reset all settings"} }
				}
				s.armReset = false
				d := config.Default()
				return tea.Batch(
					func() tea.Msg { return ConfigChangedMsg{Cfg: d} },
					func() tea.Msg { return StatusMsg{Text: "Settings reset to defaults"} },
				)
			}},
	)
	return s
}

// Settings is the interactive preferences page.
type Settings struct {
	items    []setting
	cursor   int
	offset   int
	editing  bool
	input    textinput.Model
	editErr  string
	armReset bool
	styled   string
}

func NewSettings() *Settings {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 8
	return &Settings{items: buildSettings(), input: ti}
}

func (*Settings) ID() string        { return "settings" }
func (*Settings) Title() string     { return "Settings" }
func (s *Settings) Capturing() bool { return s.editing }

var (
	keyChange = key.NewBinding(key.WithKeys("left", "right", "h", "l", "space", "enter"), key.WithHelp("←/→/enter", "change"))
)

func (s *Settings) Help(*Context) []key.Binding {
	if s.editing {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
	}
	return []key.Binding{keyDown, keyChange}
}

func (s *Settings) HandleKey(c *Context, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	it := &s.items[s.cursor]
	if s.editing {
		switch msg.String() {
		case "esc":
			s.editing, s.editErr = false, ""
			s.input.Blur()
			return nil, true
		case "enter":
			v, err := strconv.ParseFloat(strings.TrimSpace(s.input.Value()), 64)
			if err != nil {
				s.editErr = "Enter a number"
				return nil, true
			}
			if v < it.lo || v > it.hi {
				s.editErr = fmt.Sprintf("Must be between %g and %g", it.lo, it.hi)
				return nil, true
			}
			s.editing, s.editErr = false, ""
			s.input.Blur()
			cfg := it.setNum(c.Cfg, v)
			return func() tea.Msg { return ConfigChangedMsg{Cfg: cfg} }, true
		}
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		return cmd, true
	}
	if msg.String() != "enter" {
		s.armReset = false
	}
	switch {
	case key.Matches(msg, keyDown):
		s.cursor = min(s.cursor+1, len(s.items)-1)
		return nil, true
	case key.Matches(msg, keyUp):
		s.cursor = max(s.cursor-1, 0)
		return nil, true
	case msg.String() == "home" || msg.String() == "g":
		s.cursor = 0
		return nil, true
	case msg.String() == "end" || msg.String() == "G":
		s.cursor = len(s.items) - 1
		return nil, true
	}
	dir := 0
	switch msg.String() {
	case "right", "l", "space", "enter":
		dir = 1
	case "left", "h":
		dir = -1
	default:
		return nil, false
	}
	switch it.kind {
	case kindChoice:
		cfg := it.cycle(c.Cfg, dir)
		return func() tea.Msg { return ConfigChangedMsg{Cfg: cfg} }, true
	case kindNumber:
		if msg.String() != "enter" && msg.String() != "space" {
			return nil, true
		}
		s.editing = true
		s.input.SetValue(strconv.FormatFloat(it.num(c.Cfg), 'f', -1, 64))
		s.input.CursorEnd()
		return s.input.Focus(), true
	case kindAction:
		if msg.String() == "enter" {
			return it.run(s, c), true
		}
	}
	return nil, true
}

func (s *Settings) Update(c *Context, msg tea.Msg) tea.Cmd {
	if s.editing {
		if _, ok := msg.(tea.KeyPressMsg); !ok {
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return cmd
		}
	}
	return nil
}

func (s *Settings) View(c *Context, w, h int, focused bool) string {
	t := c.Theme
	if s.styled != t.P.Name {
		s.styled = t.P.Name
		ts := textinput.DefaultStyles(t.P.Dark)
		ts.Focused.Text = t.Accent.Bold(true)
		s.input.SetStyles(ts)
	}
	helpH := 5
	listH := h - helpH
	iw, ih := components.PanelContentSize(w, listH)
	labelW := 26
	// Build lines with section headers; track the line of each item.
	type line struct {
		text string
		item int
	}
	var lines []line
	section := ""
	for i, it := range s.items {
		if it.section != section {
			if section != "" {
				lines = append(lines, line{"", -1})
			}
			section = it.section
			lines = append(lines, line{t.Accent.Bold(true).Render(strings.ToUpper(it.section)), -1})
		}
		var val string
		switch it.kind {
		case kindChoice:
			v := it.value(c.Cfg)
			val = t.Faint.Render("‹ ") + t.Value.Render(v) + t.Faint.Render(" ›")
			if !t.Sym.Unicode {
				val = t.Faint.Render("< ") + t.Value.Render(v) + t.Faint.Render(" >")
			}
			if v == "On" {
				val = t.Faint.Render("‹ ") + t.Accent.Render(v) + t.Faint.Render(" ›")
			}
		case kindNumber:
			if s.editing && i == s.cursor {
				s.input.SetWidth(8)
				val = t.Faint.Render("[") + s.input.View() + t.Faint.Render("] ") + t.Muted.Render(it.unit)
			} else {
				val = t.Value.Render(strconv.FormatFloat(it.num(c.Cfg), 'f', -1, 64)) + t.Muted.Render(" "+it.unit)
			}
		case kindAction:
			val = t.Muted.Render("enter")
			if s.armReset && i == s.cursor {
				val = t.Warn.Render("press enter to confirm")
			}
		}
		label := "  " + lipgloss.NewStyle().Width(labelW).Render(it.label)
		text := label + " " + val
		if i == s.cursor {
			if focused {
				text = t.Accent.Render(t.Sym.Select) + t.Selected.Render(components.Fit(" "+lipgloss.NewStyle().Width(labelW).Render(it.label), labelW+1, 1)) + " " + val
			} else {
				text = t.Faint.Render(t.Sym.Select) + " " + lipgloss.NewStyle().Width(labelW).Render(it.label) + " " + val
			}
		}
		lines = append(lines, line{text, i})
	}
	// Scroll to keep the cursor visible.
	cl := 0
	for i, l := range lines {
		if l.item == s.cursor {
			cl = i
		}
	}
	if cl < s.offset {
		s.offset = cl
	}
	if cl >= s.offset+ih {
		s.offset = cl - ih + 1
	}
	s.offset = min(s.offset, max(len(lines)-ih, 0))
	var out []string
	for i := s.offset; i < len(lines) && len(out) < ih; i++ {
		out = append(out, lines[i].text)
	}
	list := components.Panel(t, "Settings", t.Faint.Render(fmt.Sprintf("%d/%d", s.cursor+1, len(s.items))), strings.Join(out, "\n"), w, listH, focused)

	it := s.items[s.cursor]
	helpLines := []string{t.Value.Render(it.label), t.Muted.Render(it.help)}
	if s.editErr != "" {
		helpLines = append(helpLines, t.Crit.Render(s.editErr))
	} else {
		helpLines = append(helpLines, t.Faint.Render("Saved automatically to "+c.CfgPath))
	}
	_ = iw
	help := components.Panel(t, "About this setting", "", strings.Join(helpLines, "\n"), w, helpH, false)
	return components.Fit(list+"\n"+help, w, h)
}

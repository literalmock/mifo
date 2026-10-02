// Package config loads, validates and persists MIFO settings.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Theme options.
const (
	ThemeDark   = "dark"
	ThemeLight  = "light"
	ThemeSystem = "system"
)

// Symbol modes.
const (
	SymbolsAuto    = "auto"
	SymbolsUnicode = "unicode"
	SymbolsASCII   = "ascii"
)

// Sidebar modes.
const (
	SidebarAuto     = "auto"
	SidebarExpanded = "expanded"
	SidebarCompact  = "compact"
)

// Valid option sets, exposed for the settings UI.
var (
	Intervals   = []int{1, 2, 5, 10}
	Windows     = []int{60, 300, 900, 3600}
	Themes      = []string{ThemeDark, ThemeLight, ThemeSystem}
	SymbolModes = []string{SymbolsAuto, SymbolsUnicode, SymbolsASCII}
	SidebarMode = []string{SidebarAuto, SidebarExpanded, SidebarCompact}
	Pages       = []string{"overview", "cpu", "memory", "processes", "disk", "network", "sensors", "battery", "settings"}
	WidgetNames = []string{"cpu", "memory", "disk", "network", "battery", "thermal", "charts", "processes"}
)

// Thresholds configures alerting.
type Thresholds struct {
	CPUPercent      float64 `json:"cpu_percent"`
	CPUSustainSecs  int     `json:"cpu_sustain_seconds"`
	MemoryPressure  string  `json:"memory_pressure"` // "warning" or "critical"
	DiskFreePercent float64 `json:"disk_free_percent"`
	TempCelsius     float64 `json:"temperature_celsius"`
	BatteryPercent  float64 `json:"battery_percent"`
	CooldownSecs    int     `json:"cooldown_seconds"`
}

// Config is the persisted user configuration.
type Config struct {
	Version       int             `json:"version"`
	IntervalSecs  int             `json:"refresh_interval_seconds"`
	Theme         string          `json:"theme"`
	WindowSecs    int             `json:"graph_window_seconds"`
	Widgets       map[string]bool `json:"widgets"`
	Symbols       string          `json:"symbols"`
	Sidebar       string          `json:"sidebar"`
	StartPage     string          `json:"start_page"`
	VimKeys       bool            `json:"vim_keys"`
	Alerts        Thresholds      `json:"alerts"`
	AlertsEnabled bool            `json:"alerts_enabled"`
	NativeNotify  bool            `json:"native_notifications"`
}

// Default returns the default configuration.
func Default() Config {
	w := map[string]bool{}
	for _, n := range WidgetNames {
		w[n] = true
	}
	return Config{
		Version:      1,
		IntervalSecs: 2,
		Theme:        ThemeDark,
		WindowSecs:   300,
		Widgets:      w,
		Symbols:      SymbolsAuto,
		Sidebar:      SidebarAuto,
		StartPage:    "overview",
		VimKeys:      true,
		Alerts: Thresholds{
			CPUPercent: 90, CPUSustainSecs: 30, MemoryPressure: "warning",
			DiskFreePercent: 10, TempCelsius: 95, BatteryPercent: 15, CooldownSecs: 300,
		},
		AlertsEnabled: true,
		NativeNotify:  false,
	}
}

// Interval returns the refresh interval as a duration.
func (c Config) Interval() time.Duration { return time.Duration(c.IntervalSecs) * time.Second }

// Window returns the graph window as a duration.
func (c Config) Window() time.Duration { return time.Duration(c.WindowSecs) * time.Second }

// Validate repairs out-of-range values, returning a description of each
// repair. It never fails: a bad field falls back to its default.
func (c *Config) Validate() []string {
	d := Default()
	var fixes []string
	fix := func(field string) { fixes = append(fixes, field) }
	if !slices.Contains(Intervals, c.IntervalSecs) {
		c.IntervalSecs = d.IntervalSecs
		fix("refresh_interval_seconds")
	}
	if !slices.Contains(Themes, c.Theme) {
		c.Theme = d.Theme
		fix("theme")
	}
	if !slices.Contains(Windows, c.WindowSecs) {
		c.WindowSecs = d.WindowSecs
		fix("graph_window_seconds")
	}
	if !slices.Contains(SymbolModes, c.Symbols) {
		c.Symbols = d.Symbols
		fix("symbols")
	}
	if !slices.Contains(SidebarMode, c.Sidebar) {
		c.Sidebar = d.Sidebar
		fix("sidebar")
	}
	if !slices.Contains(Pages, c.StartPage) {
		c.StartPage = d.StartPage
		fix("start_page")
	}
	if c.Widgets == nil {
		c.Widgets = map[string]bool{}
	}
	for _, n := range WidgetNames {
		if _, ok := c.Widgets[n]; !ok {
			c.Widgets[n] = true
		}
	}
	for k := range c.Widgets {
		if !slices.Contains(WidgetNames, k) {
			delete(c.Widgets, k)
		}
	}
	a := &c.Alerts
	if a.CPUPercent < 10 || a.CPUPercent > 100 {
		a.CPUPercent = d.Alerts.CPUPercent
		fix("alerts.cpu_percent")
	}
	if a.CPUSustainSecs < 0 || a.CPUSustainSecs > 3600 {
		a.CPUSustainSecs = d.Alerts.CPUSustainSecs
		fix("alerts.cpu_sustain_seconds")
	}
	if a.MemoryPressure != "warning" && a.MemoryPressure != "critical" {
		a.MemoryPressure = d.Alerts.MemoryPressure
		fix("alerts.memory_pressure")
	}
	if a.DiskFreePercent < 1 || a.DiskFreePercent > 50 {
		a.DiskFreePercent = d.Alerts.DiskFreePercent
		fix("alerts.disk_free_percent")
	}
	if a.TempCelsius < 40 || a.TempCelsius > 120 {
		a.TempCelsius = d.Alerts.TempCelsius
		fix("alerts.temperature_celsius")
	}
	if a.BatteryPercent < 1 || a.BatteryPercent > 50 {
		a.BatteryPercent = d.Alerts.BatteryPercent
		fix("alerts.battery_percent")
	}
	if a.CooldownSecs < 10 || a.CooldownSecs > 86400 {
		a.CooldownSecs = d.Alerts.CooldownSecs
		fix("alerts.cooldown_seconds")
	}
	c.Version = 1
	return fixes
}

// DefaultPath returns ~/Library/Application Support/MIFO/config.json, or
// $MIFO_CONFIG when set.
func DefaultPath() (string, error) {
	if p := os.Getenv("MIFO_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "MIFO", "config.json"), nil
}

// LoadResult describes the outcome of loading a configuration file.
type LoadResult struct {
	Config Config
	// Warning is non-empty when the file was corrupted or repaired.
	Warning string
}

// Load reads the config at path. A missing file yields defaults; a corrupted
// file is moved aside to path+".corrupt" and defaults are used, so the app
// always starts.
func Load(path string) LoadResult {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return LoadResult{Config: Default()}
	}
	if err != nil {
		return LoadResult{Config: Default(), Warning: fmt.Sprintf("could not read config (%v); using defaults", err)}
	}
	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		backup := path + ".corrupt"
		_ = os.Rename(path, backup)
		return LoadResult{Config: Default(), Warning: fmt.Sprintf("config was corrupted and moved to %s; using defaults", filepath.Base(backup))}
	}
	if fixes := cfg.Validate(); len(fixes) > 0 {
		return LoadResult{Config: cfg, Warning: fmt.Sprintf("repaired invalid settings: %v", fixes)}
	}
	return LoadResult{Config: cfg}
}

// Save writes the config atomically (write to temp file, then rename).
func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

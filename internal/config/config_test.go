package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	c := Default()
	if fixes := c.Validate(); len(fixes) != 0 {
		t.Fatalf("default config needed repairs: %v", fixes)
	}
}

func TestValidateRepairsBadValues(t *testing.T) {
	c := Default()
	c.IntervalSecs = 3
	c.Theme = "neon"
	c.WindowSecs = 7
	c.Alerts.CPUPercent = 500
	c.Alerts.MemoryPressure = "sometimes"
	c.Widgets = map[string]bool{"bogus": true}
	fixes := c.Validate()
	if len(fixes) < 5 {
		t.Fatalf("expected repairs, got %v", fixes)
	}
	d := Default()
	if c.IntervalSecs != d.IntervalSecs || c.Theme != d.Theme || c.WindowSecs != d.WindowSecs {
		t.Errorf("fields not reset: %+v", c)
	}
	if _, ok := c.Widgets["bogus"]; ok {
		t.Error("unknown widget kept")
	}
	for _, n := range WidgetNames {
		if !c.Widgets[n] {
			t.Errorf("widget %s missing", n)
		}
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	r := Load(filepath.Join(t.TempDir(), "nope.json"))
	if r.Warning != "" {
		t.Errorf("unexpected warning %q", r.Warning)
	}
	if r.Config.IntervalSecs != Default().IntervalSecs {
		t.Error("not defaults")
	}
}

func TestLoadCorruptedMovesFileAside(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := Load(p)
	if !strings.Contains(r.Warning, "corrupted") {
		t.Errorf("warning = %q", r.Warning)
	}
	if _, err := os.Stat(p + ".corrupt"); err != nil {
		t.Errorf("backup not created: %v", err)
	}
	if r.Config.Theme != ThemeDark {
		t.Error("defaults not used")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.json")
	c := Default()
	c.IntervalSecs = 5
	c.Theme = ThemeLight
	c.Widgets["charts"] = false
	c.Alerts.BatteryPercent = 25
	if err := Save(p, c); err != nil {
		t.Fatal(err)
	}
	r := Load(p)
	if r.Warning != "" {
		t.Fatalf("warning %q", r.Warning)
	}
	got := r.Config
	if got.IntervalSecs != 5 || got.Theme != ThemeLight || got.Widgets["charts"] || got.Alerts.BatteryPercent != 25 {
		t.Errorf("round trip mismatch: %+v", got)
	}
}

func TestPartialFileKeepsDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(p, []byte(`{"theme":"light"}`), 0o644)
	r := Load(p)
	if r.Config.Theme != ThemeLight || r.Config.IntervalSecs != 2 || !r.Config.Widgets["cpu"] {
		t.Errorf("partial merge wrong: %+v", r.Config)
	}
}

func TestOutOfRangeFileIsRepairedWithWarning(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(p, []byte(`{"refresh_interval_seconds":0}`), 0o644)
	r := Load(p)
	if r.Config.IntervalSecs != 2 || !strings.Contains(r.Warning, "repaired") {
		t.Errorf("got %+v / %q", r.Config.IntervalSecs, r.Warning)
	}
}

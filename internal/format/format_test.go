package format

import (
	"testing"
	"time"
)

func TestBytes(t *testing.T) {
	cases := map[uint64]string{
		0:             "0 B",
		512:           "512 B",
		1536:          "1.50 KB",
		16 << 30:      "16.0 GB",
		200 << 20:     "200 MB",
		1<<40 + 1<<39: "1.50 TB",
	}
	for in, want := range cases {
		if got := Bytes(in); got != want {
			t.Errorf("Bytes(%d) = %q, want %q", in, got, want)
		}
	}
	if got := BytesSI(245_107_195_904); got != "245 GB" {
		t.Errorf("BytesSI = %q", got)
	}
}

func TestRate(t *testing.T) {
	if got := Rate(2_500_000); got != "2.50 MB/s" {
		t.Errorf("Rate = %q", got)
	}
	if got := Rate(-5); got != "0 B/s" {
		t.Errorf("negative rate = %q", got)
	}
}

func TestPercent(t *testing.T) {
	for in, want := range map[float64]string{0: "0.0%", 5.26: "5.3%", 42.4: "42%", 99.96: "100%"} {
		if got := Percent(in); got != want {
			t.Errorf("Percent(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestDurations(t *testing.T) {
	if got := Uptime(9*24*time.Hour + 3*time.Hour + 8*time.Minute); got != "9d 3h 8m" {
		t.Errorf("Uptime = %q", got)
	}
	if got := Uptime(45 * time.Minute); got != "45m" {
		t.Errorf("Uptime short = %q", got)
	}
	if got := Duration(2*time.Hour + 5*time.Minute); got != "2h 05m" {
		t.Errorf("Duration = %q", got)
	}
	if got := CPUTime(3723*time.Second + 450*time.Millisecond); got != "1:02:03.45" {
		t.Errorf("CPUTime = %q", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("Safari Web Content", 8); got != "Safari …" {
		t.Errorf("Truncate = %q", got)
	}
	if got := TruncateLeft("/Applications/Safari.app", 8); got != "…ari.app" {
		t.Errorf("TruncateLeft = %q", got)
	}
	if got := Truncate("abc", 0); got != "" {
		t.Errorf("Truncate zero = %q", got)
	}
}

func TestFreqAndWatts(t *testing.T) {
	if got := Freq(3504); got != "3.50 GHz" {
		t.Errorf("Freq = %q", got)
	}
	if got := Watts(8.928); got != "8.93 W" {
		t.Errorf("Watts = %q", got)
	}
}

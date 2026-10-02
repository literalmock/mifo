//go:build darwin

package metrics

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func sysctlString(name string) string {
	s, err := unix.Sysctl(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

func sysctlInt(name string) (int, bool) {
	v, err := unix.SysctlUint32(name)
	if err != nil {
		return 0, false
	}
	return int(v), true
}

// macOSName maps a major version to its marketing name.
func macOSName(version string) string {
	major := version
	if i := strings.IndexByte(version, '.'); i >= 0 {
		major = version[:i]
	}
	names := map[string]string{
		"11": "Big Sur", "12": "Monterey", "13": "Ventura", "14": "Sonoma",
		"15": "Sequoia", "26": "Tahoe",
	}
	if n, ok := names[major]; ok {
		return "macOS " + n
	}
	return "macOS"
}

// modelFamily guesses a product family from a model identifier such as
// "MacBookPro18,3". Newer identifiers ("Mac14,2") carry no family name.
func modelFamily(id string) string {
	prefixes := []struct{ p, n string }{
		{"MacBookAir", "MacBook Air"}, {"MacBookPro", "MacBook Pro"}, {"MacBook", "MacBook"},
		{"Macmini", "Mac mini"}, {"MacPro", "Mac Pro"}, {"iMacPro", "iMac Pro"}, {"iMac", "iMac"},
	}
	for _, x := range prefixes {
		if strings.HasPrefix(id, x.p) {
			return x.n
		}
	}
	return "Mac"
}

func readHost(ctx context.Context) HostInfo {
	h := HostInfo{
		ModelID:   sysctlString("hw.model"),
		Chip:      sysctlString("machdep.cpu.brand_string"),
		OSVersion: sysctlString("kern.osproductversion"),
		Arch:      runtime.GOARCH,
	}
	h.OSName = macOSName(h.OSVersion)
	h.ModelName = modelFamily(h.ModelID)
	h.AppleSilicon = strings.HasPrefix(h.Chip, "Apple")
	if mem, err := unix.SysctlUint64("hw.memsize"); err == nil {
		h.MemoryTotal = mem
	}
	h.PhysicalCores, _ = sysctlInt("hw.physicalcpu")
	h.LogicalCores, _ = sysctlInt("hw.logicalcpu")
	if n, ok := sysctlInt("hw.nperflevels"); ok && n > 1 {
		h.PCores, _ = sysctlInt("hw.perflevel0.logicalcpu")
		h.ECores, _ = sysctlInt("hw.perflevel1.logicalcpu")
	}
	if tv, err := unix.SysctlTimeval("kern.boottime"); err == nil {
		h.BootTime = time.Unix(tv.Sec, int64(tv.Usec)*1000)
	}
	h.Hostname, _ = os.Hostname()

	// system_profiler gives the friendly model name ("MacBook Air") on
	// Apple Silicon where the identifier does not. It is slow, so bound it.
	cctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	if out, err := command(cctx, "/usr/sbin/system_profiler", "SPHardwareDataType", "-json").Output(); err == nil {
		var doc struct {
			SP []struct {
				MachineName string `json:"machine_name"`
				ChipType    string `json:"chip_type"`
			} `json:"SPHardwareDataType"`
		}
		if json.Unmarshal(out, &doc) == nil && len(doc.SP) > 0 {
			if doc.SP[0].MachineName != "" {
				h.ModelName = doc.SP[0].MachineName
			}
			if ct := doc.SP[0].ChipType; ct != "" && ct != "Unknown" {
				h.Chip = ct
			}
		}
	}
	if out, err := command(cctx, "/usr/sbin/scutil", "--get", "ComputerName").Output(); err == nil {
		if n := strings.TrimSpace(string(out)); n != "" {
			h.Hostname = n
		}
	}
	return h
}

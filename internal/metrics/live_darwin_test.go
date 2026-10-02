//go:build darwin

package metrics

import (
	"context"
	"os"
	"testing"
)

func TestLiveCollectSmoke(t *testing.T) {
	if os.Getenv("MIFO_LIVE") == "" {
		t.Skip("set MIFO_LIVE=1 to run against real hardware")
	}
	s := NewSystem()
	ctx := context.Background()
	h := s.Host(ctx)
	t.Logf("host %+v", h)
	snap := s.Collect(ctx)
	snap = s.Collect(ctx)
	t.Logf("took %v", snap.Took)
	t.Logf("cpu valid=%v total=%.1f user=%.1f sys=%.1f cores=%v kinds=%v freq=%v load=%v err=%v", snap.CPU.Valid, snap.CPU.Total, snap.CPU.User, snap.CPU.System, len(snap.CPU.PerCore), snap.CPU.CoreKinds, snap.CPU.MaxFreq, snap.CPU.Load, snap.CPU.Err)
	m := snap.Memory
	t.Logf("mem valid=%v used=%d app=%d wired=%d comp=%d cached=%d free=%d pct=%.1f pressure=%v freepct=%v swap=%+v err=%v", m.Valid, m.Used>>20, m.App>>20, m.Wired>>20, m.Compressed>>20, m.Cached>>20, m.Free>>20, m.UsedPercent, m.Pressure, m.FreePercent, m.Swap, m.Err)
	for _, v := range snap.Disk.Volumes {
		t.Logf("vol %+v", v)
	}
	t.Logf("diskio valid=%v r=%.0f w=%.0f err=%v", snap.Disk.IOValid, snap.Disk.ReadBps, snap.Disk.WriteBps, snap.Disk.IOErr)
	for _, i := range snap.Network.Interfaces {
		t.Logf("if %s kind=%s up=%v prim=%v addrs=%v rx=%.0f tx=%.0f", i.Name, i.Kind, i.Up, i.Primary, i.Addrs, i.RxBps, i.TxBps)
	}
	t.Logf("battery %+v", snap.Battery)
	t.Logf("sensors cpu=%v max=%v gpu=%v fans=%v state=%v thermal=%v power=%v prov=%+v ntemps=%d", snap.Sensors.CPUTemp, snap.Sensors.CPUMax, snap.Sensors.GPUTemp, snap.Sensors.Fans, snap.Sensors.FanState, snap.Sensors.Thermal, snap.Sensors.Power, snap.Sensors.Providers, len(snap.Sensors.Temps))
	t.Logf("procs n=%d threads=%d err=%v", len(snap.Processes.List), snap.Processes.Threads, snap.Processes.Err)
	for _, p := range snap.Processes.List[:5] {
		t.Logf("  %+v", p)
	}
}

//go:build darwin

package metrics

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
)

type diskInfo struct {
	internal, solid, removable Opt[bool]
	smart, protocol, volName   string
}

type diskSampler struct {
	prevIO   map[string]disk.IOCountersStat
	prevTime time.Time
	info     map[string]diskInfo // keyed by device, refreshed when mounts change
}

// hiddenMount reports mounts that are implementation details of APFS or
// virtual filesystems and would only confuse the volume list.
func hiddenMount(p disk.PartitionStat) bool {
	switch p.Fstype {
	case "devfs", "autofs", "nullfs", "lifs", "fdesc":
		return true
	}
	if p.Mountpoint == "/System/Volumes/Data" {
		return false
	}
	if strings.HasPrefix(p.Mountpoint, "/System/Volumes/") || strings.HasPrefix(p.Mountpoint, "/private/var/vm") {
		return true
	}
	if strings.HasPrefix(p.Mountpoint, "/Library/Developer/CoreSimulator") {
		return true
	}
	return false
}

func readDiskInfo(ctx context.Context, dev string) diskInfo {
	var di diskInfo
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := command(cctx, "/usr/sbin/diskutil", "info", "-plist", dev).Output()
	if err != nil {
		return di
	}
	v, err := parsePlist(out)
	if err != nil {
		return di
	}
	d, _ := v.(map[string]any)
	pd := plistDict(d)
	if b, ok := pd.bool("Internal"); ok {
		di.internal = Some(b)
	}
	if b, ok := pd.bool("SolidState"); ok {
		di.solid = Some(b)
	}
	if b, ok := pd.bool("RemovableMediaOrExternalDevice"); ok {
		di.removable = Some(b)
	} else if b, ok := pd.bool("Removable"); ok {
		di.removable = Some(b)
	}
	di.smart = pd.str("SMARTStatus")
	di.protocol = pd.str("BusProtocol")
	di.volName = pd.str("VolumeName")
	return di
}

func (d *diskSampler) sample(ctx context.Context, now time.Time) DiskStats {
	var st DiskStats
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		st.Err = err
	}
	if d.info == nil {
		d.info = map[string]diskInfo{}
	}
	seen := map[string]bool{}
	for _, p := range parts {
		if hiddenMount(p) || seen[p.Mountpoint] {
			continue
		}
		seen[p.Mountpoint] = true
		u, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil || u.Total == 0 {
			continue
		}
		info, ok := d.info[p.Device]
		if !ok && strings.HasPrefix(p.Device, "/dev/") {
			info = readDiskInfo(ctx, p.Device)
			d.info[p.Device] = info
		}
		v := Volume{
			Device: p.Device, Mount: p.Mountpoint, FSType: p.Fstype,
			Total: u.Total, Used: u.Used, Free: u.Free, UsedPercent: u.UsedPercent,
			Internal: info.internal, SolidState: info.solid, Removable: info.removable,
			SMART: info.smart, Protocol: info.protocol,
		}
		for _, o := range p.Opts {
			if o == "ro" {
				v.ReadOnly = true
			}
		}
		v.Name = info.volName
		if v.Name == "" {
			switch p.Mountpoint {
			case "/":
				v.Name = "System"
			case "/System/Volumes/Data":
				v.Name = "Data"
			default:
				v.Name = filepath.Base(p.Mountpoint)
			}
		}
		if p.Mountpoint == "/System/Volumes/Data" && info.volName != "" {
			v.Name = info.volName
		}
		st.Volumes = append(st.Volumes, v)
	}
	st.Valid = len(st.Volumes) > 0

	io, err := disk.IOCountersWithContext(ctx)
	if err != nil {
		st.IOErr = err
		return st
	}
	var rb, wb, rc, wc uint64
	var prb, pwb, prc, pwc uint64
	for name, c := range io {
		rb += c.ReadBytes
		wb += c.WriteBytes
		rc += c.ReadCount
		wc += c.WriteCount
		if p, ok := d.prevIO[name]; ok {
			prb += p.ReadBytes
			pwb += p.WriteBytes
			prc += p.ReadCount
			pwc += p.WriteCount
		} else {
			prb += c.ReadBytes
			pwb += c.WriteBytes
			prc += c.ReadCount
			pwc += c.WriteCount
		}
	}
	st.ReadTotal, st.WriteTot = rb, wb
	if d.prevIO != nil && !d.prevTime.IsZero() {
		dt := now.Sub(d.prevTime).Seconds()
		if dt > 0 {
			st.ReadBps = rate(rb, prb, dt)
			st.WriteBps = rate(wb, pwb, dt)
			st.ReadIOPS = rate(rc, prc, dt)
			st.WriteIOPS = rate(wc, pwc, dt)
			st.IOValid = true
		}
	}
	d.prevIO = io
	d.prevTime = now
	return st
}

// rate returns (cur-prev)/dt, treating counter resets as zero.
func rate(cur, prev uint64, dt float64) float64 {
	if cur < prev || dt <= 0 {
		return 0
	}
	return float64(cur-prev) / dt
}

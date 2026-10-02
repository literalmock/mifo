//go:build darwin

package metrics

import (
	"bufio"
	"bytes"
	"context"
	"sort"
	"strings"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
)

type netSampler struct {
	prev        map[string]gnet.IOCountersStat
	prevTime    time.Time
	hwPorts     map[string]string // device -> hardware port name
	hwDone      bool
	primary     string
	primaryTime time.Time
}

// readHardwarePorts maps BSD names to friendly hardware port names
// ("en0" -> "Wi-Fi") using networksetup, which needs no privileges.
func readHardwarePorts(ctx context.Context) map[string]string {
	m := map[string]string{}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := command(cctx, "/usr/sbin/networksetup", "-listallhardwareports").Output()
	if err != nil {
		return m
	}
	return parseHardwarePorts(out)
}

func parseHardwarePorts(out []byte) map[string]string {
	m := map[string]string{}
	var port string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if v, ok := strings.CutPrefix(line, "Hardware Port: "); ok {
			port = strings.TrimSpace(v)
		} else if v, ok := strings.CutPrefix(line, "Device: "); ok && port != "" {
			m[strings.TrimSpace(v)] = port
			port = ""
		}
	}
	return m
}

func readPrimaryInterface(ctx context.Context) string {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	out, err := command(cctx, "/sbin/route", "-n", "get", "default").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "interface:"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// interfaceKind classifies an interface by name when networksetup does not
// know it.
func interfaceKind(name string, hw map[string]string) string {
	if k, ok := hw[name]; ok {
		return k
	}
	switch {
	case name == "lo0":
		return "Loopback"
	case strings.HasPrefix(name, "utun"), strings.HasPrefix(name, "ipsec"), strings.HasPrefix(name, "ppp"):
		return "VPN / Tunnel"
	case strings.HasPrefix(name, "awdl"):
		return "AirDrop (AWDL)"
	case strings.HasPrefix(name, "llw"):
		return "Low-latency WLAN"
	case strings.HasPrefix(name, "bridge"):
		return "Bridge"
	case strings.HasPrefix(name, "anpi"), strings.HasPrefix(name, "ap"):
		return "Internal"
	case strings.HasPrefix(name, "gif"), strings.HasPrefix(name, "stf"):
		return "Tunnel (inactive)"
	case strings.HasPrefix(name, "en"):
		return "Ethernet"
	}
	return "Other"
}

// noiseInterface hides interfaces that never carry user traffic.
func noiseInterface(name string) bool {
	for _, p := range []string{"gif", "stf", "anpi", "ap"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func (n *netSampler) sample(ctx context.Context, now time.Time) NetworkStats {
	var st NetworkStats
	if !n.hwDone {
		n.hwPorts = readHardwarePorts(ctx)
		n.hwDone = true
	}
	if now.Sub(n.primaryTime) > 5*time.Second {
		n.primary = readPrimaryInterface(ctx)
		n.primaryTime = now
	}
	counters, err := gnet.IOCountersWithContext(ctx, true)
	if err != nil {
		st.Err = err
		return st
	}
	ifaces, _ := gnet.InterfacesWithContext(ctx)
	byName := map[string]gnet.InterfaceStat{}
	for _, i := range ifaces {
		byName[i.Name] = i
	}
	cur := map[string]gnet.IOCountersStat{}
	dt := now.Sub(n.prevTime).Seconds()
	haveRates := n.prev != nil && dt > 0
	for _, c := range counters {
		cur[c.Name] = c
		if noiseInterface(c.Name) {
			continue
		}
		info := byName[c.Name]
		ifc := Interface{
			Name: c.Name, Kind: interfaceKind(c.Name, n.hwPorts),
			MAC: info.HardwareAddr, MTU: info.MTU,
			RxTotal: c.BytesRecv, TxTotal: c.BytesSent,
			RxErrs: c.Errin, TxErrs: c.Errout,
			Primary: c.Name == n.primary,
		}
		for _, f := range info.Flags {
			switch f {
			case "up":
				ifc.Up = true
			case "running":
				ifc.Running = true
			}
		}
		for _, a := range info.Addrs {
			ifc.Addrs = append(ifc.Addrs, a.Addr)
		}
		if p, ok := n.prev[c.Name]; ok && haveRates {
			ifc.RxBps = rate(c.BytesRecv, p.BytesRecv, dt)
			ifc.TxBps = rate(c.BytesSent, p.BytesSent, dt)
		}
		// Skip interfaces that are down and have never carried traffic.
		if !ifc.Up && ifc.RxTotal == 0 && ifc.TxTotal == 0 {
			continue
		}
		if ifc.Kind != "Loopback" {
			st.RxBps += ifc.RxBps
			st.TxBps += ifc.TxBps
			st.RxTotal += ifc.RxTotal
			st.TxTotal += ifc.TxTotal
		}
		st.Interfaces = append(st.Interfaces, ifc)
	}
	sortInterfaces(st.Interfaces)
	st.Primary = n.primary
	st.Connected = n.primary != ""
	st.Valid = haveRates
	n.prev = cur
	n.prevTime = now
	return st
}

// sortInterfaces orders primary first, then active, then by name.
func sortInterfaces(list []Interface) {
	rank := func(i Interface) int {
		switch {
		case i.Primary:
			return 0
		case i.Kind == "Loopback":
			return 3
		case i.Up && len(i.Addrs) > 0:
			return 1
		}
		return 2
	}
	sort.SliceStable(list, func(a, b int) bool {
		ra, rb := rank(list[a]), rank(list[b])
		if ra != rb {
			return ra < rb
		}
		return list[a].Name < list[b].Name
	})
}

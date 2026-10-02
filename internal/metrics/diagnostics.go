package metrics

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DiagResult is the outcome of an explicitly started connectivity test.
type DiagResult struct {
	Target     string
	Sent       int
	Received   int
	Loss       Opt[float64] // percent
	MinMs      Opt[float64]
	AvgMs      Opt[float64]
	MaxMs      Opt[float64]
	PingErr    error
	DNSHost    string
	DNSAddrs   []string
	DNSTime    Opt[time.Duration]
	DNSErr     error
	StartedAt  time.Time
	FinishedAt time.Time
}

var (
	pingStatsRe = regexp.MustCompile(`(\d+) packets transmitted, (\d+) packets received, ([\d.]+)% packet loss`)
	pingRTTRe   = regexp.MustCompile(`= ([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+) ms`)
)

// parsePing parses the summary of macOS ping(8).
func parsePing(out []byte, r *DiagResult) {
	if m := pingStatsRe.FindSubmatch(out); m != nil {
		r.Sent, _ = strconv.Atoi(string(m[1]))
		r.Received, _ = strconv.Atoi(string(m[2]))
		if l, err := strconv.ParseFloat(string(m[3]), 64); err == nil {
			r.Loss = Some(l)
		}
	}
	if m := pingRTTRe.FindSubmatch(out); m != nil {
		if v, err := strconv.ParseFloat(string(m[1]), 64); err == nil {
			r.MinMs = Some(v)
		}
		if v, err := strconv.ParseFloat(string(m[2]), 64); err == nil {
			r.AvgMs = Some(v)
		}
		if v, err := strconv.ParseFloat(string(m[3]), 64); err == nil {
			r.MaxMs = Some(v)
		}
	}
}

// RunDiagnostics performs a short ICMP ping (via the system ping binary, no
// privileges needed) and a DNS lookup. It is only run when the user asks.
func RunDiagnostics(ctx context.Context) DiagResult {
	r := DiagResult{Target: "1.1.1.1", DNSHost: "apple.com", StartedAt: time.Now()}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := command(cctx, "/sbin/ping", "-c", "5", "-i", "0.2", "-t", "6", r.Target).CombinedOutput()
	parsePing(out, &r)
	if err != nil && !r.Loss.OK {
		r.PingErr = err
	}
	start := time.Now()
	addrs, err := net.DefaultResolver.LookupHost(cctx, r.DNSHost)
	if err != nil {
		r.DNSErr = err
	} else {
		r.DNSTime = Some(time.Since(start))
		r.DNSAddrs = addrs
	}
	r.FinishedAt = time.Now()
	return r
}

// NetTopEntry is per-process network traffic over the sampling period.
type NetTopEntry struct {
	Name  string
	PID   int
	RxBps float64
	TxBps float64
}

// NetTopResult is a per-process network snapshot.
type NetTopResult struct {
	Entries []NetTopEntry
	Err     error
	At      time.Time
}

// parseNetTop parses `nettop -P -x -J bytes_in,bytes_out -L 2 -d` output,
// using only the last (delta) block.
func parseNetTop(out []byte, secs float64) []NetTopEntry {
	var blocks [][]NetTopEntry
	var cur []NetTopEntry
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, ",bytes_in") || strings.HasPrefix(line, "time,") {
			if cur != nil {
				blocks = append(blocks, cur)
			}
			cur = []NetTopEntry{}
			continue
		}
		f := strings.Split(line, ",")
		if len(f) < 3 {
			continue
		}
		name := f[0]
		i := strings.LastIndexByte(name, '.')
		if i < 0 {
			continue
		}
		pid, err := strconv.Atoi(name[i+1:])
		if err != nil {
			continue
		}
		in, _ := strconv.ParseFloat(f[1], 64)
		outB, _ := strconv.ParseFloat(f[2], 64)
		cur = append(cur, NetTopEntry{Name: name[:i], PID: pid, RxBps: in / secs, TxBps: outB / secs})
	}
	if cur != nil {
		blocks = append(blocks, cur)
	}
	if len(blocks) == 0 {
		return nil
	}
	last := blocks[len(blocks)-1]
	var active []NetTopEntry
	for _, e := range last {
		if e.RxBps > 0 || e.TxBps > 0 {
			active = append(active, e)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		return active[i].RxBps+active[i].TxBps > active[j].RxBps+active[j].TxBps
	})
	return active
}

// RunNetTop samples per-process network throughput over one second using
// nettop(1). Attribution is best-effort: traffic from system daemons on
// behalf of apps is credited to the daemon.
func RunNetTop(ctx context.Context) NetTopResult {
	r := NetTopResult{At: time.Now()}
	cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	out, err := command(cctx, "/usr/bin/nettop", "-P", "-x", "-d", "-L", "2", "-s", "1", "-J", "bytes_in,bytes_out").Output()
	if err != nil {
		r.Err = err
		if errors.Is(cctx.Err(), context.DeadlineExceeded) {
			r.Err = errors.New("nettop timed out")
		}
		return r
	}
	r.Entries = parseNetTop(out, 1)
	return r
}

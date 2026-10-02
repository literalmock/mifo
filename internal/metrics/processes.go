package metrics

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// psFormat is the column list passed to ps(1). comm must be last because it
// may contain spaces; lstart is a fixed five-token date.
const psFormat = "pid=,ppid=,uid=,user=,state=,rss=,vsz=,time=,lstart=,comm="

// cutFields splits off the first n whitespace-separated fields and returns
// them with the untouched remainder of the line.
func cutFields(line string, n int) ([]string, string) {
	fields := make([]string, 0, n)
	rest := line
	for len(fields) < n {
		rest = strings.TrimLeft(rest, " \t")
		if rest == "" {
			break
		}
		i := strings.IndexAny(rest, " \t")
		if i < 0 {
			fields = append(fields, rest)
			rest = ""
			break
		}
		fields = append(fields, rest[:i])
		rest = rest[i:]
	}
	return fields, strings.TrimSpace(rest)
}

// parseCPUTime parses ps TIME values: "MM:SS.ss", "HH:MM:SS.ss" or
// "D-HH:MM:SS".
func parseCPUTime(s string) (time.Duration, bool) {
	var days float64
	if i := strings.IndexByte(s, '-'); i >= 0 {
		d, err := strconv.ParseFloat(s[:i], 64)
		if err != nil {
			return 0, false
		}
		days = d
		s = s[i+1:]
	}
	parts := strings.Split(s, ":")
	var secs float64
	for _, p := range parts {
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return 0, false
		}
		secs = secs*60 + v
	}
	secs += days * 86400
	return time.Duration(secs * float64(time.Second)), true
}

// parsePS parses `ps -axww -o psFormat` output.
func parsePS(out []byte, loc *time.Location) []Process {
	var list []Process
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		f, comm := cutFields(sc.Text(), 13)
		if len(f) < 13 {
			continue
		}
		pid, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		p := Process{PID: pid, User: f[3], State: f[4], Path: comm}
		p.PPID, _ = strconv.Atoi(f[1])
		p.UID, _ = strconv.Atoi(f[2])
		if kb, err := strconv.ParseUint(f[5], 10, 64); err == nil {
			p.RSS = kb * 1024
		}
		if kb, err := strconv.ParseUint(f[6], 10, 64); err == nil {
			p.VSZ = kb * 1024
		}
		p.CPUTime, _ = parseCPUTime(f[7])
		if t, err := time.ParseInLocation("Mon Jan 2 15:04:05 2006", strings.Join(f[8:13], " "), loc); err == nil {
			p.Start = t
		}
		p.Name = filepath.Base(comm)
		if p.Name == "" || p.Name == "." {
			p.Name = comm
		}
		list = append(list, p)
	}
	return list
}

// parseThreadCounts counts per-PID threads from `ps -axM` output, where the
// first line of each process has USER then PID, and continuation lines start
// with whitespace followed by the PID.
func parseThreadCounts(out []byte) map[int]int {
	m := map[int]int{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	first := true
	for sc.Scan() {
		line := sc.Text()
		if first {
			first = false
			if strings.HasPrefix(line, "USER") {
				continue
			}
		}
		if line == "" {
			continue
		}
		f, _ := cutFields(line, 2)
		var field string
		if line[0] == ' ' || line[0] == '\t' {
			if len(f) < 1 {
				continue
			}
			field = f[0]
		} else {
			if len(f) < 2 {
				continue
			}
			field = f[1]
		}
		if pid, err := strconv.Atoi(field); err == nil {
			m[pid]++
		}
	}
	return m
}

type procPrev struct {
	cpu time.Duration
	at  time.Time
}

// computeProcessCPU fills CPUPercent from the delta of CPU time since the
// previous sample of the same process (identified by PID and start time, so a
// reused PID never inherits another process's history). It returns the new
// state map.
func computeProcessCPU(list []Process, prev map[ProcKey]procPrev, now time.Time) map[ProcKey]procPrev {
	next := make(map[ProcKey]procPrev, len(list))
	for i := range list {
		p := &list[i]
		k := p.Key()
		if pp, ok := prev[k]; ok {
			dt := now.Sub(pp.at).Seconds()
			if dt > 0 && p.CPUTime >= pp.cpu {
				p.CPUPercent = 100 * (p.CPUTime - pp.cpu).Seconds() / dt
				p.CPUValid = true
			}
		}
		next[k] = procPrev{cpu: p.CPUTime, at: now}
	}
	return next
}

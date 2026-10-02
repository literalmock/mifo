//go:build darwin

package metrics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
)

type procSampler struct {
	prev map[ProcKey]procPrev
	// Thread counts come from a second, comparatively expensive ps call,
	// so they are refreshed at most every threadTTL.
	threads   map[int]int
	threadsAt time.Time
}

const threadTTL = 5 * time.Second

func (s *procSampler) sample(ctx context.Context, now time.Time) ProcessStats {
	var st ProcessStats
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// ps is setuid root on macOS, so it reports CPU time and memory for
	// every process, including root-owned ones that unprivileged APIs hide.
	out, err := command(cctx, "/bin/ps", "-axww", "-o", psFormat).Output()
	if err != nil {
		st.Err = err
		return st
	}
	st.List = parsePS(out, time.Local)
	if s.threads == nil || now.Sub(s.threadsAt) >= threadTTL {
		if tout, err := command(cctx, "/bin/ps", "-axM").Output(); err == nil {
			s.threads, s.threadsAt = parseThreadCounts(tout), now
		}
	}
	if counts := s.threads; counts != nil {
		for i := range st.List {
			if n, ok := counts[st.List[i].PID]; ok && n > 0 {
				st.List[i].Threads = Some(n)
				st.Threads += n
			}
		}
	}
	s.prev = computeProcessCPU(st.List, s.prev, now)
	st.Valid = len(st.List) > 0
	return st
}

// ProcessDetail is extra information fetched on demand for one process.
type ProcessDetail struct {
	Key     ProcKey
	Args    string
	Err     error
	Fetched time.Time
}

// FetchProcessDetail retrieves the full command line for a process, verifying
// it is still the same process (PID and start time).
func FetchProcessDetail(ctx context.Context, key ProcKey) ProcessDetail {
	d := ProcessDetail{Key: key, Fetched: time.Now()}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	start, err := processStart(cctx, key.PID)
	if err != nil {
		d.Err = err
		return d
	}
	if start.Unix() != key.Start {
		d.Err = errors.New("process has exited (PID was reused)")
		return d
	}
	out, err := command(cctx, "/bin/ps", "-ww", "-o", "args=", "-p", fmt.Sprint(key.PID)).Output()
	if err != nil {
		d.Err = err
		return d
	}
	d.Args = strings.TrimSpace(string(out))
	return d
}

func processStart(ctx context.Context, pid int) (time.Time, error) {
	out, err := command(ctx, "/bin/ps", "-o", "lstart=", "-p", fmt.Sprint(pid)).Output()
	if err != nil {
		return time.Time{}, errors.New("process no longer exists")
	}
	f := strings.Fields(string(out))
	if len(f) != 5 {
		return time.Time{}, errors.New("process no longer exists")
	}
	return time.ParseInLocation("Mon Jan 2 15:04:05 2006", strings.Join(f, " "), time.Local)
}

// Signal sends sig to a process after verifying its identity, guarding
// against PID reuse. It refuses to signal PID 0/1 and MIFO itself.
func Signal(ctx context.Context, key ProcKey, sig syscall.Signal, self int) error {
	if key.PID <= 1 {
		return errors.New("refusing to signal the kernel or launchd")
	}
	if key.PID == self {
		return errors.New("refusing to signal MIFO itself; press q to quit")
	}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	start, err := processStart(cctx, key.PID)
	if err != nil {
		return err
	}
	if start.Unix() != key.Start {
		return errors.New("process already exited and its PID was reused; nothing was sent")
	}
	if err := syscall.Kill(key.PID, sig); err != nil {
		if errors.Is(err, syscall.EPERM) {
			return ErrPermission
		}
		if errors.Is(err, syscall.ESRCH) {
			return errors.New("process already exited")
		}
		return err
	}
	return nil
}

// ErrPermission is returned when macOS refuses to deliver a signal.
var ErrPermission = errors.New("permission denied: this process belongs to another user or is protected by macOS (System Integrity Protection). MIFO does not escalate privileges")

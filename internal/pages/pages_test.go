package pages

import (
	"testing"
	"time"

	"github.com/literalmock/mifo/internal/config"
	"github.com/literalmock/mifo/internal/metrics"
)

func sampleProcs() []metrics.Process {
	return metrics.SampleSnapshot(time.Now(), 0).Processes.List
}

func TestSortProcesses(t *testing.T) {
	list := sampleProcs()
	SortProcesses(list, SortCPU, true)
	if list[0].Name != "Xcode" {
		t.Errorf("cpu desc first = %s", list[0].Name)
	}
	SortProcesses(list, SortMemory, true)
	if list[0].Name != "Xcode" || list[1].Name != "kernel_task" {
		t.Errorf("mem desc = %s, %s", list[0].Name, list[1].Name)
	}
	SortProcesses(list, SortPID, false)
	if list[0].PID != 1 || list[len(list)-1].PID != 2001 {
		t.Errorf("pid asc wrong")
	}
	SortProcesses(list, SortName, false)
	if list[0].Name != "kernel_task" {
		t.Errorf("name asc first = %s", list[0].Name)
	}
	SortProcesses(list, SortThreads, true)
	if list[0].Name != "Xcode" {
		t.Errorf("threads desc first = %s", list[0].Name)
	}
}

func TestFilterProcesses(t *testing.T) {
	list := sampleProcs()
	if got := FilterProcesses(list, "safari", -1); len(got) != 2 {
		t.Errorf("name filter = %d", len(got))
	}
	if got := FilterProcesses(list, "WEBKIT", -1); len(got) != 1 {
		t.Errorf("path filter (case-insensitive) = %d", len(got))
	}
	if got := FilterProcesses(list, "root", -1); len(got) != 2 {
		t.Errorf("user filter = %d", len(got))
	}
	if got := FilterProcesses(list, "77", -1); len(got) != 1 || got[0].PID != 777 {
		t.Errorf("pid prefix filter = %+v", got)
	}
	if got := FilterProcesses(list, "", 501); len(got) != 3 {
		t.Errorf("uid filter = %d", len(got))
	}
	if got := FilterProcesses(list, "nothing-matches", -1); len(got) != 0 {
		t.Errorf("expected none")
	}
}

func TestTopProcesses(t *testing.T) {
	list := sampleProcs()
	top := TopProcesses(list, 2, false)
	if len(top) != 2 || top[0].Name != "Xcode" || top[1].Name != "Safari" {
		t.Errorf("top = %+v", top)
	}
	// Must not reorder the caller's slice.
	if list[0].PID != 1 {
		t.Error("TopProcesses mutated input")
	}
}

func TestStepInterval(t *testing.T) {
	c := config.Default()
	c.IntervalSecs = 2
	if stepInterval(c, -1).IntervalSecs != 1 || stepInterval(c, 1).IntervalSecs != 5 {
		t.Error("step wrong")
	}
	c.IntervalSecs = 1
	if stepInterval(c, -1).IntervalSecs != 1 {
		t.Error("should clamp at fastest")
	}
}

func TestInterestingInterfaces(t *testing.T) {
	s := metrics.SampleSnapshot(time.Now(), 0)
	var names []string
	for _, i := range s.Network.Interfaces {
		if interesting(i) {
			names = append(names, i.Name)
		}
	}
	if len(names) != 2 || names[0] != "en0" || names[1] != "utun3" {
		t.Errorf("interesting = %v", names)
	}
}

package app

import (
	"testing"
	"time"

	"github.com/literalmock/mifo/internal/metrics"
)

func BenchmarkRenderOverview(b *testing.B) {
	m := readyB(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Render()
	}
}

func BenchmarkSnapshotUpdate(b *testing.B) {
	m := readyB(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm, _ := m.Update(snapshotMsg{snap: metrics.SampleSnapshot(fixedNow.Add(time.Duration(i)*time.Second), i)})
		m = tm.(Model)
	}
}

func readyB(b *testing.B) Model {
	t := &testing.T{}
	m := ready(t, 140, 44)
	for i := 0; i < 300; i++ {
		m.hist.Record(metrics.SampleSnapshot(fixedNow.Add(-time.Duration(300-i)*time.Second), i))
	}
	return m
}

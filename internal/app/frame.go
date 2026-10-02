package app

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
)

// RenderOnce drives the model synchronously: it loads host info, collects
// `samples` snapshots `gap` apart (so charts have history) and returns the
// rendered screen. It is used by `mifo -frame` and tests.
func RenderOnce(ctx context.Context, m Model, w, h, samples int, gap time.Duration) string {
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: w, Height: h})
	if m.src != nil {
		tm, _ = tm.Update(hostMsg{host: m.src.Host(ctx)})
		for i := 0; i < samples; i++ {
			if i > 0 {
				select {
				case <-time.After(gap):
				case <-ctx.Done():
				}
			}
			tm, _ = tm.Update(snapshotMsg{snap: m.src.Collect(ctx)})
		}
	}
	return tm.(Model).Render()
}

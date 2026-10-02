//go:build darwin

package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// System is the real macOS metrics source.
type System struct {
	mu        sync.Mutex // serialises Collect; samplers keep delta state
	cpu       cpuSampler
	disk      diskSampler
	net       netSampler
	proc      procSampler
	providers []SensorProvider
	primed    bool

	// Slow-changing, comparatively expensive readings are cached briefly.
	battery   BatteryStats
	batteryAt time.Time
	temps     []Temperature
	tempsErr  error
	tempsAt   time.Time
}

const (
	batteryTTL = 5 * time.Second
	tempsTTL   = 3 * time.Second
)

// NewSystem returns a collector using the default unprivileged providers.
func NewSystem() *System {
	return &System{providers: DefaultSensorProviders()}
}

// Host implements Source.
func (s *System) Host(ctx context.Context) HostInfo { return readHost(ctx) }

// safe runs f, converting a panic in a provider into an error so that one
// failing subsystem never takes down the app.
func safe(name string, f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s provider panicked: %v", name, r)
		}
	}()
	f()
	return nil
}

// Collect implements Source. Subsystems are sampled concurrently; each
// sampler's state is only touched by its own goroutine.
func (s *System) Collect(ctx context.Context) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.primed {
		// Establish baselines so the first visible snapshot has rates.
		now := time.Now()
		s.cpu.sample(ctx)
		s.net.sample(ctx, now)
		s.disk.sample(ctx, now)
		s.proc.sample(ctx, now)
		s.primed = true
		select {
		case <-time.After(400 * time.Millisecond):
		case <-ctx.Done():
		}
	}

	start := time.Now()
	snap := Snapshot{Time: start}
	var wg sync.WaitGroup
	run := func(name string, f func(), onPanic func(error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := safe(name, f); err != nil {
				onPanic(err)
			}
		}()
	}
	run("cpu", func() { snap.CPU = s.cpu.sample(ctx) }, func(e error) { snap.CPU = CPUStats{Err: e} })
	run("memory", func() { snap.Memory = readMemory() }, func(e error) { snap.Memory = MemoryStats{Err: e} })
	run("disk", func() { snap.Disk = s.disk.sample(ctx, start) }, func(e error) { snap.Disk = DiskStats{Err: e} })
	run("network", func() { snap.Network = s.net.sample(ctx, start) }, func(e error) { snap.Network = NetworkStats{Err: e} })
	run("battery", func() {
		if s.batteryAt.IsZero() || start.Sub(s.batteryAt) >= batteryTTL || !s.battery.Valid {
			s.battery, s.batteryAt = readBattery(ctx), start
		}
		snap.Battery = s.battery
	}, func(e error) { snap.Battery = BatteryStats{Err: e} })
	run("processes", func() { snap.Processes = s.proc.sample(ctx, start) }, func(e error) { snap.Processes = ProcessStats{Err: e} })
	run("sensors", func() { snap.Sensors = s.readSensors(ctx, start) }, func(e error) {
		snap.Sensors = SensorStats{Providers: []ProviderStatus{{Name: "sensors", Err: e}}}
	})
	wg.Wait()
	snap.Took = time.Since(start)
	return snap
}

func (s *System) readSensors(ctx context.Context, now time.Time) SensorStats {
	var st SensorStats
	for _, p := range s.providers {
		var err error
		if _, isTemp := p.(hidTempProvider); isTemp {
			// Temperature sensors are slow to query and change slowly.
			if s.tempsAt.IsZero() || now.Sub(s.tempsAt) >= tempsTTL {
				var fresh SensorStats
				if perr := safe(p.Name(), func() { err = p.Read(ctx, &fresh) }); perr != nil {
					err = perr
				}
				s.temps, s.tempsErr, s.tempsAt = fresh.Temps, err, now
			}
			st.Temps = append(st.Temps[:0:0], s.temps...)
			err = s.tempsErr
		} else if perr := safe(p.Name(), func() { err = p.Read(ctx, &st) }); perr != nil {
			err = perr
		}
		st.Providers = append(st.Providers, ProviderStatus{Name: p.Name(), OK: err == nil, Err: err})
	}
	summarizeTemps(&st)
	return st
}

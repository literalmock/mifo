//go:build darwin

package metrics

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/shirou/gopsutil/v4/sensors"

	"github.com/literalmock/mifo/internal/metrics/smc"
)

// hidTempProvider reads temperatures via IOKit HID (Apple Silicon) or SMC
// (Intel) through gopsutil.
type hidTempProvider struct{}

func (hidTempProvider) Name() string { return "Temperature sensors" }

func (hidTempProvider) Read(ctx context.Context, out *SensorStats) error {
	temps, err := sensors.TemperaturesWithContext(ctx)
	if err != nil && len(temps) == 0 {
		return err
	}
	for _, t := range temps {
		if !plausibleTemp(t.Temperature) {
			continue
		}
		out.Temps = append(out.Temps, Temperature{
			Name: t.SensorKey, Group: classifyTemperature(t.SensorKey), Celsius: t.Temperature,
		})
	}
	if len(out.Temps) == 0 {
		return errors.New("no temperature sensors exposed")
	}
	return nil
}

// smcProvider reads fans and power rails from the SMC. The connection is
// opened lazily and reused; access is serialised by the collector.
type smcProvider struct {
	mu   sync.Mutex
	conn *smc.Conn
	err  error
}

func (*smcProvider) Name() string { return "SMC (fans, power)" }

func (s *smcProvider) Read(_ context.Context, out *SensorStats) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil && s.err == nil {
		s.conn, s.err = smc.Open()
	}
	if s.err != nil {
		out.FanState = FansUnavailable
		return s.err
	}
	c := s.conn
	n, err := c.ReadFloat("FNum")
	switch {
	case err != nil:
		// Fanless Macs (MacBook Air) do not define FNum at all.
		out.FanState = FansNone
	case n == 0:
		out.FanState = FansNone
	default:
		out.FanState = FansOK
		for i := 0; i < int(n) && i < 8; i++ {
			rpm, err := c.ReadFloat(fmt.Sprintf("F%dAc", i))
			if err != nil || rpm < 0 {
				continue
			}
			f := Fan{Index: i, RPM: rpm}
			if v, err := c.ReadFloat(fmt.Sprintf("F%dMn", i)); err == nil {
				f.Min = Some(v)
			}
			if v, err := c.ReadFloat(fmt.Sprintf("F%dMx", i)); err == nil && v > 0 {
				f.Max = Some(v)
			}
			out.Fans = append(out.Fans, f)
		}
		if len(out.Fans) == 0 {
			out.FanState = FansUnavailable
		}
	}
	for _, k := range []struct{ key, name string }{
		{"PSTR", "System total"},
		{"PDTR", "DC input"},
		{"PPBR", "Battery rail"},
		{"PHPC", "Heatpipe / CPU"},
	} {
		if w, err := c.ReadFloat(k.key); err == nil && w >= 0.001 && w < 1000 {
			out.Power = append(out.Power, PowerReading{Name: k.name, Watts: w})
		}
	}
	return nil
}

// thermalProvider reads the system thermal pressure level published via
// notify(3) under com.apple.system.thermalpressurelevel.
type thermalProvider struct {
	once  sync.Once
	err   error
	get   func(token int32, state *uint64) uint32
	token int32
}

func (*thermalProvider) Name() string { return "Thermal pressure" }

func (t *thermalProvider) init() {
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		t.err = err
		return
	}
	var reg func(name string, token unsafe.Pointer) uint32
	purego.RegisterLibFunc(&reg, lib, "notify_register_check")
	purego.RegisterLibFunc(&t.get, lib, "notify_get_state")
	if r := reg("com.apple.system.thermalpressurelevel", unsafe.Pointer(&t.token)); r != 0 {
		t.err = fmt.Errorf("notify_register_check failed: %d", r)
	}
}

func (t *thermalProvider) Read(_ context.Context, out *SensorStats) error {
	t.once.Do(t.init)
	if t.err != nil {
		return t.err
	}
	var state uint64
	if r := t.get(t.token, &state); r != 0 {
		return fmt.Errorf("notify_get_state failed: %d", r)
	}
	out.Thermal = thermalFromState(state)
	return nil
}

func thermalFromState(s uint64) ThermalPressure {
	switch s {
	case 0:
		return ThermalNominal
	case 1:
		return ThermalModerate
	case 2:
		return ThermalHeavy
	case 3:
		return ThermalTrapping
	case 4:
		return ThermalSleeping
	}
	return ThermalUnknown
}

// DefaultSensorProviders returns the built-in, unprivileged backends.
func DefaultSensorProviders() []SensorProvider {
	return []SensorProvider{hidTempProvider{}, &smcProvider{}, &thermalProvider{}}
}

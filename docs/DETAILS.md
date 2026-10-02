# MIFO

*Your Mac. Every heartbeat.*

MIFO is a terminal system monitor for macOS, built with Go and Charm's
[Bubble Tea v2](https://github.com/charmbracelet/bubbletea),
[Bubbles v2](https://github.com/charmbracelet/bubbles) and
[Lip Gloss v2](https://github.com/charmbracelet/lipgloss). It reads live
metrics from macOS APIs without root privileges. When a metric can't be read
on your Mac, MIFO says so instead of showing a made-up number.

```text
 ◆ MIFO  Your Mac. Every heartbeat.                                       ● Warning · Low storage on Data   03:28:24
 MacBook Air · Apple M2 · macOS Tahoe 26.6.2 · up 9d 3h 23m                               updated just now · every 1s
──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
▍ 1  Overview    │ ╭─ CPU ────────────────────────╮ ╭─ Memory ─────────────────────╮ ╭─ Disk ───────────────────────╮
  2  CPU         │ │ 8.1%      4P + 4E · 3.50 GHz │ │ 11.4 GB           of 16.0 GB │ │ 11.7 GB       free of 245 GB │
  3  Memory      │ │ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │ │ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │ │ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
  4  Processes   │ │ User / System    4.2% / 3.9% │ │ Pressure            ● Normal │ │ Data                95% used │
  5  Disk        │ │ Cores E▂▁▁▁ P▁▁▁▁            │ │ Swap        690 MB / 2.00 GB │ │ R / W    220 KB/s / 886 KB/s │
  6  Network     │ │ ▄▂▂▁▂▂▄▂▂▂▂▂▁▂▁▂▁▂▁▁▂▂▂▃▃▃▂▁ │ │ ▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆ │ │ █▁▁▁▁▁▁▁▁▁▂▁▁▁▁▁▁▁▁▁▁▂▁▁▁▁▁▁ │
  7  Sensors     │ ╰──────────────────────────────╯ ╰──────────────────────────────╯ ╰──────────────────────────────╯
  8  Battery     │ ╭─ Network ────────────────────╮ ╭─ Battery ────────────────────╮ ╭─ Thermal ────────────────────╮
  9  Settings    │ │ ↓ 349 B/s   ↑ 611 B/s        │ │ 77%               On battery │ │ 45.2°C            max 46.2°C │
                 │ │ Interface      ● Wi-Fi (en0) │ │ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │ │ Thermal pressure   ● Nominal │
                 │ │ Received             49.3 GB │ │ Health      84% · 269 cycles │ │ Fans          none (fanless) │
                 │ │ Sent                 4.03 GB │ │ Time remaining        5h 12m │ │ System power          3.00 W │
                 │ │ ▅▁▇▁▄▆█▁▃▄▁▁▁▁▁▁▄▄▁▁▄▁▁▁▁▁▁▁ │ │ Battery power        -6.71 W │ │ ▅▄▅▄▄▄▆▆▇▆▆▆▆▆▅▆▅▆▅▅▅▅▅▅▅▅▅▄ │
                 │ ╰──────────────────────────────╯ ╰──────────────────────────────╯ ╰──────────────────────────────╯
                 │ ╭─ CPU ───────────────────────────── 5m · 8.1% ─╮ ╭─ Memory ────────────────────────── 5m · 71% ─╮
                 │ │                                          100% │ │                                         100% │
                 │ │           ⢀                                   │ │  ⢰⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶⣶      │
                 │ │   ⢠⡆      ⢸⡀          ⢀           ⣀⣀⣀         │ │  ⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿      │
                 │ │   ⢸⣷⣶⣶⣴⣶⣶⣶⣾⣷⣶⣶⣷⣷⣾⣦⣦⣦⣦⣴⣾⣴⣶⣴⣶⣤⣤⣷⣦⣷⣶⣿⣿⣿⣿⣷⣄⣄   0% │ │  ⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿   0% │
                 │ ╰───────────────────────────────────────────────╯ ╰──────────────────────────────────────────────╯
                 │ ╭─ Network ───────────────── ↓349 B/s ↑611 B/s ─╮ ╭─ Disk I/O ─────────── R 220 KB/s W 886 KB/s ─╮
                 │ │                                     5.00 MB/s │ │                                    50.0 MB/s │
                 │ │ ⣆⣀⣸⣄⣀⣤⣰⣶⣇⣀⣄⣀⣄⣀⣀⣀⣀⣀⣀⣀⣠⣀⣄⣀⣀⣄⣄⣀⣀⣀⣀⣀⣀⣀⣀           │ │ ⣀⣀⢀⣀⣀⣀⣀⣀⣀⣀⢀⣀⣀ ⣀⢀⣀ ⡀ ⡀ ⡀⡀⣀⡀⣀⢀⢀⣀⣀⣀⣀⢀           │
                 │ │ ⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉           │ │ ⢹⠉⠉⠁⠉⠉⠉⠉⠁⠉⠉⠉⠉⠁⠉⠉⠁⠁⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉           │
                 │ │                                     5.00 MB/s │ │                                    50.0 MB/s │
                 │ ╰───────────────────────────────────────────────╯ ╰──────────────────────────────────────────────╯
                 │ ╭─ Top Processes ─────────────────────────────────────────────────────────────────── enter open ─╮
                 │ │ Name                                                                 PID   CPU%    Memory  Thr │
                 │ │ Helium Helper (Renderer)                                           48242    6.0   1.58 GB   23 │
                 │ │ claude                                                              1310    6.0    427 MB   20 │
                 │ │ WindowServer                                                         166    6.0   92.1 MB   20 │
 CPU  ━━━━━━   8%│ │ mifo                                                               6369    5.0   41.2 MB   20 │
 MEM  ━━━━━━  71%│ │ Helium Helper (Renderer)                                           95419    2.0    221 MB   25 │
 BAT  ━━━━━━  77%│ │ Helium Helper                                                      48238    2.0    179 MB   20 │
                 │ ╰────────────────────────────────────────────────────────────────────────────────────────────────╯
──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
 ↑/k up · ↓/j down · enter open · tab next panel · ? help · q quit                           Overview · menu · ● live
```

The capture above is unedited (colours stripped), taken from `tmux
capture-pane` on a MacBook Air (M2) after about a minute of history. In a
terminal it renders in a deep charcoal theme with emerald accents. Use
`mifo -frame 140x44` to print a frame from your own Mac.

## Features

- **Overview dashboard**: CPU, memory, disk, network, battery and thermal
  cards, live braille charts for CPU, memory, network and disk I/O, the top
  five processes, and active alerts.
- **CPU**: large utilisation readout, user/system/nice/idle breakdown,
  per-core bars with P/E labels (read from the IOKit device tree), per-core
  sparklines, load average, max frequency and thermal pressure. You can
  change the sampling interval from this page.
- **Memory**: Activity Monitor's model (App + Wired + Compressed = Used),
  cached files shown as reclaimable rather than "used", the kernel's memory
  pressure level and memorystatus gauge, swap, raw VM counters, and the
  processes using the most memory. Each category has a one-line explanation.
- **Process manager**: Bubbles table with sorting (CPU, memory, PID, name,
  threads), live search, a "mine only" filter, and a detail panel with the
  executable path, parent, start time, CPU time and full command line.
  Terminating a process always asks for confirmation (SIGTERM or SIGKILL).
- **Disk**: mounted volumes with capacity bars, filesystem, internal/external,
  SSD, bus protocol, read-only flag and SMART status, plus read/write
  throughput and IOPS history. MIFO never scans the filesystem recursively.
- **Network**: interface table (primary, Wi-Fi, VPN, ...), IPv4/IPv6 with
  netmasks, MAC, MTU, errors, totals and a per-interface throughput chart.
  Diagnostics only run when you ask for them:
  - `t` runs a latency and packet-loss test (5 × ICMP ping to 1.1.1.1) and
    times a DNS lookup.
  - `n` takes a one-second per-app traffic sample via `nettop`. This is
    clearly labelled best-effort attribution.
- **Sensors**: thermal pressure, every readable temperature sensor grouped by
  SoC, storage and battery, fans (or "No fans" on fanless Macs), SMC power
  rails, and the health of each sensor backend.
- **Battery**: charge, state, power source, the time-remaining estimate only
  when macOS provides a reliable one, health (full-charge capacity vs.
  design), cycles, voltage, current, power flow, system load, temperature and
  charge history. Desktop Macs get a power and thermal view instead.
- **Alerts**: CPU (sustained), memory pressure, low storage, high temperature
  (only when the Mac has real sensors) and low battery. Alerts have a
  cooldown, a history, dismissal, and optional native macOS notifications.
  Missing or stale data never raises or clears an alert.
- **Settings**: refresh interval (1/2/5/10 s), theme (Dark / Light / System),
  graph window (1 m / 5 m / 15 m / 1 h), symbols (Unicode / ASCII), sidebar
  mode, start page, vim keys, dashboard widgets and alert thresholds. Changes
  are saved immediately.

## Install

MIFO needs Go 1.26+ and macOS 11 or newer. cgo is not required.

```bash
go run ./cmd/mifo                 # run from source
go build -o mifo ./cmd/mifo      # build for this Mac
make build                         # same, with version info and -trimpath
```

### Apple Silicon and Intel

MIFO talks to IOKit, the SMC and libSystem through
[purego](https://github.com/ebitengine/purego), so cross-compiling is a
plain `GOARCH` switch:

```bash
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o mifo-arm64 ./cmd/mifo   # Apple Silicon
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o mifo-amd64 ./cmd/mifo   # Intel
make release   # dist/: arm64, amd64, a universal binary (lipo) and SHA256SUMS
```

### Homebrew (optional, not yet published)

There is **no published Homebrew formula**. To distribute MIFO through a
personal tap, create a `homebrew-tap` repository with a formula along these
lines:

```ruby
class Mifo < Formula
  desc "Terminal system monitor for macOS"
  homepage "https://github.com/<you>/mifo"
  url "https://github.com/<you>/mifo/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "<sha256 of the tarball>"
  depends_on "go" => :build
  depends_on :macos

  def install
    ENV["CGO_ENABLED"] = "0"
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "./cmd/mifo"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/mifo -version")
  end
end
```

Then `brew install <you>/tap/mifo`.

## Usage

```text
mifo [flags]
  -interval N      refresh interval for this session (1, 2, 5, 10); not saved
  -page NAME       start page for this session: overview, cpu, memory, processes,
                   disk, network, sensors, battery, settings
  -config PATH     config file (default ~/Library/Application Support/MIFO/config.json)
  -frame WxH       render one frame with real data to stdout and exit
  -version         print version
```

`MIFO_CONFIG` can also point to an alternative config file.

### Keyboard shortcuts

| Key | Action |
| --- | --- |
| `1`–`9` | Jump to a section |
| `Tab` / `Shift+Tab` | Move focus between the sidebar and the page |
| `↑`/`↓`, `j`/`k` | Navigate (sidebar, tables, lists) |
| `Enter` | Open / inspect |
| `Esc` | Close a dialog, leave search or details, or go back |
| `p` | Process manager |
| `s` | Settings |
| `a` | Alerts panel (`d` dismiss, `D` dismiss all) |
| `w` | Cycle graph window (1 m → 5 m → 15 m → 1 h) |
| `r` | Refresh now |
| `?` | Help |
| `q`, `Ctrl+C` | Quit |

**Process manager:** `/` search · `c` `m` `i` `n` `t` sort by CPU, memory,
PID, name or threads (press again to reverse) · `u` mine only · `Enter`
details · `x` terminate (SIGTERM) · `X` force quit (SIGKILL) ·
`y`/`n` confirm or cancel.

**CPU:** `+` / `-` change the sampling interval · `j`/`k` scroll cores.
**Network:** `t` connectivity test · `n` per-app traffic · `A` show all
interfaces. **Settings:** `←`/`→` or `Enter` change a value, `Enter` to edit
numbers, `Esc` to cancel.

While a text field or confirmation dialog is open it receives every key, so
typing `q` or `s` in the search box never quits or navigates.

## Supported metrics and where they come from

| Metric | Source | Privileges |
| --- | --- | --- |
| CPU usage (total, per core, user/system/nice) | `host_processor_info` via gopsutil | none |
| P/E core labels | IOKit device tree (`cluster-type`) | none |
| Max CPU frequency | IOKit (Apple Silicon) / sysctl (Intel) via gopsutil | none |
| Memory categories | `host_statistics64` (`HOST_VM_INFO64`) via purego | none |
| Memory pressure | `kern.memorystatus_vm_pressure_level`, `kern.memorystatus_level` | none |
| Swap | `vm.swapusage` | none |
| Processes, CPU%, RSS | `/bin/ps` (setuid, so root processes are included) | none |
| Thread counts | `ps -M` | none |
| Volumes, SMART, internal/external | statfs, `diskutil info -plist` (cached) | none |
| Disk throughput | IOKit block storage statistics via gopsutil | none |
| Network interfaces and rates | `getifaddrs`, `route`, `networksetup` | none |
| Battery | `ioreg -rn AppleSmartBattery -a` | none |
| Temperatures | IOKit HID sensors (Apple Silicon), SMC keys (Intel) | none |
| Fans, power rails | AppleSMC user client via purego | none |
| Thermal pressure | `notify(3)` `com.apple.system.thermalpressurelevel` | none |
| Per-app network | `nettop` (only on request) | none |

## Limitations

- **Apple Silicon sensors.** Temperatures come from the HID sensor hub; the
  raw names (`PMU tdie1`, ...) are shown as macOS reports them. There is no
  separate GPU temperature sensor, so GPU temperature shows as unavailable.
  Disconnected or uncalibrated sensors that report negative or impossible
  values are hidden rather than shown as zeros.
- **Fans.** Fanless Macs (MacBook Air) show "No fans". If the SMC can't be
  opened, fans show as unavailable.
- **CPU frequency.** macOS doesn't expose the current frequency without
  `powermetrics` (root), so MIFO shows the maximum frequency, labelled "max".
- **Energy impact.** Per-process energy impact needs `powermetrics` (root) and
  isn't collected.
- **Per-process network attribution** is best-effort. Traffic that system
  daemons (e.g. `mDNSResponder`, `nsurlsessiond`) generate on an app's behalf
  is credited to the daemon.
- **Protected processes.** Signalling another user's processes, or processes
  protected by System Integrity Protection, fails with a permission error.
  MIFO explains why and never escalates privileges.
- **Battery time remaining** only appears when macOS has a stable estimate.

### Optional privileged integrations

MIFO has no privileged mode. Sensor backends implement the
`metrics.SensorProvider` interface (`internal/metrics/sensors.go`), so a
backend built on `powermetrics` (current CPU/GPU frequency, per-process
energy) could be added later. It would need `sudo` and should be opt-in.

## Startup behaviour

MIFO starts on the page set under Settings → General → Start page, or
`-page` for a single session. To open MIFO automatically:

- **iTerm2:** Settings → Profiles → General → Command: `/path/to/mifo`.
- **Ghostty:** `command = /path/to/mifo` in a dedicated config, launched with
  `ghostty --config-file=...`.
- **Kitty:** `kitty /path/to/mifo` from a launcher or a session file.
- **Terminal.app:** create a profile, then Shell → Run command: `/path/to/mifo`.
- **At login:** System Settings → General → Login Items, add a small `.command`
  script that runs `mifo`.

## Configuration

Settings are stored as JSON in
`~/Library/Application Support/MIFO/config.json` and written atomically.
Invalid values are repaired with a warning. A corrupted file is moved aside to
`config.json.corrupt` and defaults are used, so MIFO always starts.

## Architecture

```text
cmd/mifo/            entry point, flags, -frame renderer
internal/metrics/     collectors (cpu, memory, disk, network, battery, sensors,
                      processes, diagnostics), Source interface, bounded ring
                      buffers, MockSource for tests, smc/ (SMC client)
internal/alerts/      threshold engine, cooldowns, history, health summary
internal/config/      load / validate / atomic save
internal/styles/      palettes (dark, light) and Unicode/ASCII symbol sets
internal/components/  panel, bars, braille charts, sparklines, tables,
                      big digits, header, footer, sidebar (Bubbles list)
internal/pages/       one file per screen; each page is a small state machine
internal/app/         root Bubble Tea model: update loop, key routing,
                      focus/navigation, overlays (help, alerts)
```

- Collection runs in a `tea.Cmd`, off the UI goroutine. Subsystems are
  sampled concurrently. A panic in one provider is recovered and reported for
  that subsystem only.
- Only one collection can run at a time. Ticks from an old interval are
  ignored after you change the refresh rate.
- History lives in fixed-capacity ring buffers: one hour at 1 s, about 30 KB
  per series.
- External commands use a C locale and short timeouts, and are cancelled
  when MIFO exits.
- Process identity is (PID, start time). Before MIFO sends a signal it
  re-checks the start time, so a reused PID is never signalled.

## Development

```bash
gofmt -w .
go vet ./...
go test ./...            # unit + UI tests against MockSource
make race                # race detector
make live                # smoke test against this Mac's real hardware
```

The UI tests drive the root model with key messages and render every page at
sizes from 64×20 up to 220×60. They check that the output always fits the
terminal, that ASCII mode contains no Unicode glyphs, and that missing data
renders as "unavailable" (never as zeros or NaN).

## Troubleshooting

- **Garbled boxes or charts:** your font may lack box-drawing or braille
  glyphs. Set Settings → Symbols → ASCII.
- **Colours look wrong:** use a terminal with true-colour support (Ghostty,
  Kitty, iTerm2). Terminal.app ignores the background colour MIFO sets, so
  the Dark theme uses your profile's background there. The System theme
  follows the terminal's background.
- **"Terminal too small":** MIFO needs at least 64×20.
- **Native notifications don't appear:** the first one may trigger a macOS
  permission prompt for your terminal. Check System Settings → Notifications.
- **"Unavailable on this Mac":** the metric isn't exposed to unprivileged
  processes on this hardware. The Sensor Backends panel on the Sensors page
  (`7`) shows which backend failed and why.

## License

MIT, see [LICENSE](LICENSE).

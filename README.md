<div align="center">

# ◆ mifo

**your mac. every heartbeat.** uwu

terminal system monitor for macOS · pure Go · no root · no cgo · no fake numbers xd

[![ci](https://github.com/literalmock/mifo/actions/workflows/ci.yml/badge.svg)](https://github.com/literalmock/mifo/actions)
![macOS](https://img.shields.io/badge/macOS-11%2B-black?logo=apple)
![go](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)
![license](https://img.shields.io/badge/license-MIT-green)

</div>

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

> real capture, colours stripped. irl it's charcoal + emerald ✨
> peek your own: `mifo -frame 140x44`

## 📦 install

pick ur fighter:

```bash
# 1. one-liner (grabs the latest release, puts it in your PATH) <- easiest
curl -fsSL https://raw.githubusercontent.com/literalmock/mifo/main/scripts/install.sh | sh

# 2. go
go install github.com/literalmock/mifo/cmd/mifo@latest

# 3. from source
git clone https://github.com/literalmock/mifo && cd mifo && make install
```

or yoink a binary from [Releases](https://github.com/literalmock/mifo/releases)
(`arm64`, `amd64`, `universal`) and `mv mifo /usr/local/bin/`.

install script knobs:

| env | what | default |
| --- | --- | --- |
| `MIFO_BIN_DIR` | where it lands | `/usr/local/bin` (falls back to `~/.local/bin`) |
| `MIFO_VERSION` | pin a tag | `latest` |

nuke it: `rm "$(command -v mifo)"` · config lives in `~/Library/Application Support/MIFO/`

## 🚀 use

```bash
mifo                  # go brrr
mifo -page cpu        # start on a page
mifo -interval 2      # refresh every 2s (1/2/5/10)
mifo -version
```

## ⌨️ keys

| key | does |
| --- | --- |
| `1`-`9` | jump to page |
| `tab` | sidebar ↔ page |
| `j` `k` / `↑` `↓` | move |
| `p` | process manager (`/` search, `x` kill, `X` force kill) |
| `a` | alerts |
| `w` | graph window 1m → 5m → 15m → 1h |
| `?` | help |
| `q` | bye |

## ✨ what's inside

cpu · memory · processes · disk · network · sensors · battery · alerts · settings.
if a metric can't be read it says so instead of lying to you. promise.

## 🔧 hack

```bash
make          # fmt + vet + test + build
make install  # build + drop in PATH
make run
```

deep dive (architecture, metric sources, limits, troubleshooting) → [docs/DETAILS.md](docs/DETAILS.md)

## 📜 license

MIT. be nice. 💚

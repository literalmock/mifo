// Command mifo is a terminal system monitor for macOS.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/literalmock/mifo/internal/app"
	"github.com/literalmock/mifo/internal/config"
	"github.com/literalmock/mifo/internal/metrics"
)

var version = "dev"

func main() {
	var (
		interval = flag.Int("interval", 0, "refresh interval in seconds (1, 2, 5 or 10); overrides settings for this session")
		cfgPath  = flag.String("config", "", "path to config file (default ~/Library/Application Support/MIFO/config.json)")
		page     = flag.String("page", "", "start page: overview, cpu, memory, processes, disk, network, sensors, battery, settings")
		showVer  = flag.Bool("version", false, "print version and exit")
		frame    = flag.String("frame", "", "render a single frame at WIDTHxHEIGHT to stdout and exit (for screenshots and debugging)")
	)
	flag.Parse()
	if *showVer {
		fmt.Println("mifo", version)
		return
	}

	path := *cfgPath
	if path == "" {
		p, err := config.DefaultPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "mifo: cannot locate config directory:", err)
		}
		path = p
	}
	res := config.Load(path)
	cfg := res.Config
	if *interval != 0 && !slices.Contains(config.Intervals, *interval) {
		fmt.Fprintln(os.Stderr, "mifo: invalid -interval; use 1, 2, 5 or 10")
		os.Exit(2)
	}
	if *page != "" && !slices.Contains(config.Pages, *page) {
		fmt.Fprintln(os.Stderr, "mifo: unknown -page", *page)
		os.Exit(2)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()

	src := metrics.NewSystem()
	opts := app.Options{
		Ctx: ctx, Source: src, Config: cfg, ConfigPath: path, Warning: res.Warning,
		StartPage: *page, Interval: *interval,
	}

	if *frame != "" {
		os.Exit(renderFrame(ctx, opts, *frame))
	}

	p := tea.NewProgram(app.New(opts), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, "mifo:", err)
		os.Exit(1)
	}
}

// renderFrame collects real metrics and prints one frame without entering
// the alternate screen.
func renderFrame(ctx context.Context, opts app.Options, size string) int {
	w, h, ok := parseSize(size)
	if !ok {
		fmt.Fprintln(os.Stderr, "mifo: -frame expects WIDTHxHEIGHT, e.g. 140x42")
		return 2
	}
	opts.NoSave = true
	m := app.New(opts)
	frame := app.RenderOnce(ctx, m, w, h, 3, 700*time.Millisecond)
	fmt.Println(frame)
	return 0
}

func parseSize(s string) (int, int, bool) {
	a, b, ok := strings.Cut(strings.ToLower(s), "x")
	if !ok {
		return 0, 0, false
	}
	w, err1 := strconv.Atoi(a)
	h, err2 := strconv.Atoi(b)
	return w, h, err1 == nil && err2 == nil && w > 0 && h > 0
}

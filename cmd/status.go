package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/limits"
	"github.com/jmeiracorbal/claudio/internal/profile"
	"github.com/jmeiracorbal/claudio/internal/resolver"
	"github.com/spf13/cobra"
)

const probeTimeout = 10 * time.Second

var (
	statusJSON  bool
	statusWatch int
)

var statusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show plan usage limits for every profile, or one in detail",
	Long: `Show each profile's subscription usage: the 5-hour session and weekly
windows, with their reset times. With a profile name, show that profile in detail.

Usage is read from Anthropic with the token Claude Code stored for the profile.
claudio never refreshes tokens: an expired one is refreshed by launching the profile.

Examples:
  claudio status
  claudio status personal
  claudio status --watch        (refresh every 60s)
  claudio status --watch=30
  claudio status --json`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runStatus,
	SilenceUsage: true,
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "print JSON for scripts")
	statusCmd.Flags().IntVar(&statusWatch, "watch", 0, "refresh every N seconds (default 60 when given without a value)")
	statusCmd.Flags().Lookup("watch").NoOptDefVal = "60"
}

func runStatus(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles found. Run: claudio create <name>")
		return nil
	}

	names := make([]string, 0, len(cfg.Profiles))
	if len(args) == 1 {
		if _, ok := cfg.Profiles[args[0]]; !ok {
			return fmt.Errorf("profile %q not found", args[0])
		}
		names = append(names, args[0])
	} else {
		for n := range cfg.Profiles {
			names = append(names, n)
		}
		sort.Strings(names)
	}

	render := func() error {
		reports := probeAll(cfg, names)
		if statusJSON {
			return printStatusJSON(names, reports, len(args) == 1)
		}
		if len(args) == 1 {
			printStatusDetail(cfg, names[0], reports[names[0]])
		} else {
			printStatusTable(cfg, names, reports)
		}
		return nil
	}

	if statusWatch <= 0 {
		return render()
	}
	interval := time.Duration(statusWatch) * time.Second
	for {
		if !statusJSON {
			fmt.Print("\033[H\033[2J")
		}
		if err := render(); err != nil {
			return err
		}
		if !statusJSON {
			fmt.Printf("\n  updated %s · every %s · ctrl+c to exit\n", time.Now().Format("15:04:05"), interval)
		}
		time.Sleep(interval)
	}
}

func probeAll(cfg *config.Config, names []string) map[string]limits.Report {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	reports := make(map[string]limits.Report, len(names))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, n := range names {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := limits.Probe(ctx, config.ExpandPath(cfg.Profiles[n].ConfigDir))
			mu.Lock()
			reports[n] = r
			mu.Unlock()
		}()
	}
	wg.Wait()
	return reports
}

func printStatusJSON(names []string, reports map[string]limits.Report, single bool) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if single {
		return enc.Encode(reports[names[0]])
	}
	return enc.Encode(map[string]any{"profiles": reports})
}

func printStatusTable(cfg *config.Config, names []string, reports map[string]limits.Report) {
	active, _ := resolver.Resolve(cfg)

	maxLen := 4
	for _, n := range names {
		if len(n)+2 > maxLen {
			maxLen = len(n) + 2
		}
	}

	row := func(name, plan, state, five, fiveReset, week, weekReset string) {
		fmt.Printf("  %-*s  %-6s  %-11s  %-15s  %-10s  %-15s  %s\n", maxLen, name, plan, state, five, fiveReset, week, weekReset)
	}
	row("NAME", "PLAN", "STATE", "5H", "RESETS", "7D", "RESETS")
	row("----", "----", "-----", "--", "------", "--", "------")
	for _, n := range names {
		r := reports[n]
		label := n
		if n == active {
			label += " *"
		}
		five, fiveReset := windowCells(r.Window("session"))
		week, weekReset := windowCells(r.Window("weekly"))
		row(label, orDash(r.Plan), string(r.State), five, fiveReset, week, weekReset)
	}

	var hints []string
	for _, n := range names {
		switch reports[n].State {
		case limits.StateExpired:
			hints = append(hints, fmt.Sprintf("%s: token expired, launch it to refresh (claudio %s)", n, n))
		case limits.StateNone:
			hints = append(hints, fmt.Sprintf("%s: not logged in (claudio login %s)", n, n))
		case limits.StateUnavailable:
			hints = append(hints, fmt.Sprintf("%s: %s", n, reports[n].Error))
		}
	}
	if len(hints) > 0 {
		fmt.Println()
		for _, h := range hints {
			fmt.Println("  " + h)
		}
	}
}

func printStatusDetail(cfg *config.Config, name string, r limits.Report) {
	p := cfg.Profiles[name]
	configDir := config.ExpandPath(p.ConfigDir)

	header := name
	if active, reason := resolver.ResolveEffective(cfg); active == name {
		header += fmt.Sprintf("  (active · %s)", reason)
	}
	field := func(label, value string) {
		fmt.Printf("  %-9s %s\n", label, value)
	}

	fmt.Println()
	fmt.Println("  " + header)
	fmt.Println()
	field("email", orDash(profile.Email(configDir)))
	field("config", configDir)
	plan := orDash(r.Plan)
	if r.Tier != "" {
		plan += " (" + r.Tier + ")"
	}
	field("plan", plan)
	field("model", orDefault(p.DefaultModel))

	state := string(r.State)
	switch r.State {
	case limits.StateOK:
		if !r.TokenExpiresAt.IsZero() {
			state += " · token expires in " + formatDuration(time.Until(r.TokenExpiresAt))
		}
	case limits.StateExpired:
		state += fmt.Sprintf(" · launch the profile to refresh it (claudio %s)", name)
	case limits.StateNone:
		state += fmt.Sprintf(" · not logged in (claudio login %s)", name)
	case limits.StateUnavailable:
		state += " · " + r.Error
	}
	field("state", state)

	if len(r.Windows) > 0 {
		fmt.Println()
		for _, w := range r.Windows {
			line := fmt.Sprintf("%3d%%  %s", w.Percent, bar(w.Percent, 20))
			if !w.ResetsAt.IsZero() {
				line += fmt.Sprintf("  resets in %s (%s)", formatDuration(time.Until(w.ResetsAt)), formatReset(w.ResetsAt))
			}
			if w.Severity != "" && w.Severity != "normal" {
				line += "  " + w.Severity
			}
			field(windowLabel(w.Kind), line)
		}
	}

	if e := r.Extra; e != nil && (e.Enabled || e.MonthlyLimit > 0) {
		state := "disabled"
		if e.Enabled {
			state = "enabled"
		}
		fmt.Println()
		field("extra", fmt.Sprintf("%s · %.2f / %.2f %s", state, e.UsedCredits, e.MonthlyLimit, e.Currency))
	}
	fmt.Println()
}

func windowCells(w *limits.Window) (usage, reset string) {
	if w == nil {
		return "—", ""
	}
	usage = fmt.Sprintf("%3d%% %s", w.Percent, bar(w.Percent, 10))
	if !w.ResetsAt.IsZero() {
		reset = formatReset(w.ResetsAt)
	}
	return usage, reset
}

func windowLabel(kind string) string {
	switch kind {
	case "session":
		return "5h"
	case "weekly_all":
		return "7d"
	}
	return strings.ReplaceAll(kind, "_", " ")
}

func bar(percent, width int) string {
	filled := min(max(percent*width/100, 0), width)
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// formatReset shows the time of day for resets within a day, the weekday otherwise.
func formatReset(t time.Time) string {
	t = t.Local().Round(time.Minute)
	if time.Until(t) < 24*time.Hour {
		return t.Format("15:04")
	}
	return t.Format("Mon 15:04")
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	d = d.Round(time.Minute)
	days, hours, mins := int(d.Hours())/24, int(d.Hours())%24, int(d.Minutes())%60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd%dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh%02dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func orDefault(s string) string {
	if s == "" {
		return "default"
	}
	return s
}

package tui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/jmeiracorbal/claudio/internal/config"
	"github.com/jmeiracorbal/claudio/internal/launcher"
	"github.com/jmeiracorbal/claudio/internal/profile"
	resolverpkg "github.com/jmeiracorbal/claudio/internal/resolver"
)

type model struct {
	styles    Styles
	w, h      int
	screen    string
	selected  int
	profiles  []string
	cfg       *config.Config
	cwd       string
	active    string
	reason    string
	message   string
	inputMode string
	input     string
	confirm   bool
	demoFrame bool
}

func newModel(theme Theme, ansi16 bool) (model, error) {
	cfg, err := config.Load()
	if err != nil {
		return model{}, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return model{}, err
	}
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	active, reason := resolverpkg.ResolveEffective(cfg)
	return model{styles: NewStyles(theme, ansi16), cfg: cfg, cwd: cwd, profiles: names,
		active: active, reason: reason, screen: "home"}, nil
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case processResult:
		if msg.err != nil {
			m.message = fmt.Sprintf("Claude session for %q ended: %v", msg.profile, msg.err)
		} else {
			m.message = "Claude session for " + msg.profile + " ended"
		}
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.inputMode != "" {
			return m, m.updateInput(key)
		}
		if m.confirm {
			switch key {
			case "y", "enter":
				if m.selected < len(m.profiles) {
					name := m.profiles[m.selected]
					if err := profile.Remove(name); err != nil {
						m.message = err.Error()
					} else {
						m.message = "Removed profile " + name
						m.refresh()
					}
				}
				m.confirm = false
			case "n", "esc":
				m.confirm = false
			}
			return m, nil
		}
		switch key {
		case "q", "esc":
			if m.screen != "home" {
				m.screen = "home"
				m.selected = 0
			} else {
				return m, tea.Quit
			}
		case "1":
			m.screen, m.selected = "home", 0
		case "2":
			m.screen, m.selected = "profiles", 0
		case "3":
			m.screen, m.selected = "projects", 0
		case "4":
			m.screen, m.selected = "setup", 0
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < m.maxSelection() {
				m.selected++
			}
		case "enter", " ":
			return m.activate()
		case "c":
			if m.screen == "profiles" {
				m.inputMode, m.input = "create", ""
			}
		case "r":
			if m.screen == "profiles" && m.selected < len(m.profiles) {
				m.inputMode, m.input = "rename", m.profiles[m.selected]
			}
		case "l":
			if m.screen == "profiles" && m.selected < len(m.profiles) {
				return m.launch(m.profiles[m.selected])
			}
		case "x":
			if m.screen == "profiles" && m.selected < len(m.profiles) {
				m.confirm = true
			}
		case "p":
			if m.screen == "projects" && len(m.profiles) > 0 {
				m.inputMode, m.selected = "pin-profile", 0
			}
		case "u":
			if m.screen == "projects" {
				m.unpin()
			}
		case "a":
			if m.screen == "projects" {
				m.inputMode, m.input = "rule-path", ""
			}
		case "d":
			if m.screen == "projects" {
				m.removeRule()
			}
		case "?":
			m.screen = "help"
		}
	}
	return m, nil
}

func (m model) maxSelection() int {
	switch m.screen {
	case "home":
		return 3
	case "launch":
		return max(0, len(m.profiles)-1)
	case "profiles":
		return max(0, len(m.profiles)-1)
	case "projects":
		return max(0, len(m.cfg.Rules)-1)
	default:
		return 0
	}
}

func (m model) activate() (tea.Model, tea.Cmd) {
	switch m.screen {
	case "home":
		switch m.selected {
		case 0:
			m.screen = "launch"
			m.selected = 0
		case 1:
			m.screen = "profiles"
			m.selected = 0
		case 2:
			m.screen = "projects"
			m.selected = 0
		case 3:
			m.screen = "setup"
			m.selected = 0
		}
	case "launch":
		if m.selected < len(m.profiles) {
			return m.launch(m.profiles[m.selected])
		}
	case "profiles":
		if m.selected < len(m.profiles) {
			return m.launch(m.profiles[m.selected])
		}
	case "projects":
		if m.selected < len(m.cfg.Rules) {
			m.message = "Rule: " + m.cfg.Rules[m.selected].Path + " → " + m.cfg.Rules[m.selected].Profile
		}
	case "setup":
		m.message = "Run `claudio doctor` for the complete setup report."
	case "help":
		m.screen = "home"
	}
	return m, nil
}

func (m model) launch(name string) (tea.Model, tea.Cmd) {
	dir, err := m.cfg.ProfileConfigDir(name)
	if err != nil {
		m.message = err.Error()
		return m, nil
	}
	cmd, err := launcher.Command(dir, nil)
	if err != nil {
		m.message = err.Error()
		return m, nil
	}
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return processResult{err: err, profile: name} })
}

type processResult struct {
	err     error
	profile string
}

func (m model) updateInput(key string) tea.Cmd {
	switch key {
	case "esc":
		m.inputMode = ""
		m.message = "Cancelled"
	case "enter":
		switch m.inputMode {
		case "create":
			name := strings.TrimSpace(m.input)
			if name == "" {
				m.message = "Profile name is required"
				break
			}
			if err := profile.Create(name); err != nil {
				m.message = err.Error()
			} else {
				m.message = "Created profile " + name
				m.refresh()
			}
			m.inputMode = ""
		case "rename":
			if m.selected >= len(m.profiles) {
				m.inputMode = ""
				break
			}
			old, name := m.profiles[m.selected], strings.TrimSpace(m.input)
			if name == "" {
				m.message = "Profile name is required"
				break
			}
			if err := profile.Rename(old, name); err != nil {
				m.message = err.Error()
			} else {
				m.message = "Renamed " + old + " to " + name
				m.refresh()
			}
			m.inputMode = ""
		case "rule-path":
			path := strings.TrimSpace(m.input)
			if path == "" {
				m.message = "Rule path is required"
				m.inputMode = ""
				break
			}
			m.inputMode = "rule-profile"
			m.input = path
			m.selected = 0
		case "rule-profile":
			if len(m.profiles) == 0 {
				m.message = "Create a profile first"
				m.inputMode = ""
				break
			}
			path, name := m.input, m.profiles[min(max(m.selected, 0), len(m.profiles)-1)]
			m.cfg.Rules = append(m.cfg.Rules, config.Rule{Path: path, Profile: name})
			if err := m.cfg.Save(); err != nil {
				m.message = err.Error()
			} else {
				m.message = "Added route " + path + " → " + name
			}
			m.inputMode = ""
			m.refresh()
		case "pin-profile":
			m.pinSelected()
			m.inputMode = ""
		}
	case "up":
		if (m.inputMode == "rule-profile" || m.inputMode == "pin-profile") && m.selected > 0 {
			m.selected--
		}
	case "down":
		if (m.inputMode == "rule-profile" || m.inputMode == "pin-profile") && m.selected < len(m.profiles)-1 {
			m.selected++
		}
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	default:
		if len(key) == 1 && key >= " " {
			m.input += key
		}
	}
	return nil
}

func (m *model) refresh() {
	cfg, err := config.Load()
	if err != nil {
		m.message = err.Error()
		return
	}
	m.cfg = cfg
	m.profiles = m.profiles[:0]
	for name := range cfg.Profiles {
		m.profiles = append(m.profiles, name)
	}
	sort.Strings(m.profiles)
	m.selected = min(m.selected, max(0, m.maxSelection()))
	m.active, m.reason = resolverpkg.ResolveEffective(cfg)
}

func (m *model) pinSelected() {
	if len(m.profiles) == 0 {
		m.message = "Create a profile first"
		return
	}
	name := m.profiles[min(max(m.selected, 0), len(m.profiles)-1)]
	if err := os.WriteFile(filepath.Join(m.cwd, ".claudio-account"), []byte(name+"\n"), 0644); err != nil {
		m.message = err.Error()
	} else {
		m.message = "Pinned this project to " + name
		m.refresh()
	}
}

func (m *model) unpin() {
	path := m.nearestPinPath()
	if path == "" {
		m.message = "No project pin found in this project"
		return
	}
	if err := os.Remove(path); err != nil {
		m.message = err.Error()
		return
	}
	m.message = "Removed project pin"
	m.refresh()
}

func (m *model) removeRule() {
	if m.selected >= len(m.cfg.Rules) {
		m.message = "Select a path rule first"
		return
	}
	rule := m.cfg.Rules[m.selected]
	m.cfg.Rules = append(m.cfg.Rules[:m.selected], m.cfg.Rules[m.selected+1:]...)
	if err := m.cfg.Save(); err != nil {
		m.message = err.Error()
		return
	}
	m.message = "Removed route " + rule.Path
	m.refresh()
}

func (m model) View() tea.View {
	v := tea.NewView(m.view())
	v.AltScreen = true
	v.BackgroundColor = m.styles.Base.GetBackground()
	return v
}

func (m model) view() string {
	if m.w < 68 || m.h < 18 {
		return lipgloss.Place(max(1, m.w), max(1, m.h), lipgloss.Center, lipgloss.Center, m.styles.Warning.Render("Terminal too small · need 68×18"))
	}
	var lines []string
	lines = append(lines, m.header(), "")
	switch m.screen {
	case "home":
		lines = append(lines, m.home()...)
	case "launch":
		lines = append(lines, m.launchScreen()...)
	case "profiles":
		lines = append(lines, m.profileLibrary()...)
	case "projects":
		lines = append(lines, m.projectRouting()...)
	case "setup":
		lines = append(lines, m.setup()...)
	case "help":
		lines = append(lines, "  Keyboard guide", "", "  1–4  open workspace      ↑/↓  move", "  enter select / launch    c    create profile", "  r    rename profile      x    remove profile", "  p    pin project         u    remove pin", "  a    add path rule       ?    close help", "  esc  back                q    quit")
	}
	if m.inputMode != "" {
		lines = append(lines, "", m.inputPrompt())
	}
	if m.confirm && m.selected < len(m.profiles) {
		name := m.profiles[m.selected]
		dir, _ := m.cfg.ProfileConfigDir(name)
		lines = append(lines, "", m.styles.Warning.Render("  Remove profile "+name+" and delete "+dir+"? [y/N]"))
	}
	if m.message != "" {
		lines = append(lines, "", m.styles.Muted.Render("  "+m.message))
	}
	lines = append(lines, "", m.footer())
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return lipgloss.NewStyle().Width(m.w).Height(m.h).MaxWidth(m.w).MaxHeight(m.h).Padding(1, 2).Render(content)
}

func (m model) header() string {
	left := m.styles.HeaderApp.Render(" claudio ") + m.styles.HeaderSep.Render(" / ") + m.styles.Title.Render(map[string]string{"home": "Task Hub", "launch": "Launch Claude", "profiles": "Profile Library", "projects": "Project Routing", "setup": "Setup", "help": "Help"}[m.screen])
	right := m.styles.Muted.Render("1 Hub  2 Profiles  3 Projects")
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", max(1, m.w-lipgloss.Width(left)-lipgloss.Width(right)-4)), right)
}

func (m model) home() []string {
	return []string{"  Manage Claude Code accounts and choose the profile for each project.", "",
		m.homeRow(0, "Launch Claude", "Choose a profile and start a Claude session"),
		m.homeRow(1, "Profile Library", fmt.Sprintf("%d profiles · create, rename, authenticate, remove", len(m.profiles))),
		m.homeRow(2, "Project Routing", fmt.Sprintf("%d path rules · pin this project or route folders", len(m.cfg.Rules))),
		m.homeRow(3, "Check setup", "Review Claude, profiles and configuration"), "",
		"  Current project", "  " + m.styles.Muted.Render(m.cwd),
		"  Resolved profile  " + m.resolution()}
}

func (m model) homeRow(i int, title, desc string) string {
	prefix := "   "
	style := m.styles.Name
	if i == m.selected {
		prefix = "  › "
		style = m.styles.RowSel
	}
	return prefix + style.Render(title) + m.styles.Muted.Render("  "+desc)
}

func (m model) launchScreen() []string {
	lines := []string{"  CHOOSE A PROFILE TO START CLAUDE", ""}
	if len(m.profiles) == 0 {
		return append(lines, "  No profiles yet. Press 2 to open Profile Library.")
	}
	for i, name := range m.profiles {
		prefix := "   "
		style := m.styles.Name
		if i == m.selected {
			prefix = "  › "
			style = m.styles.RowSel
		}
		marker := ""
		if name == m.active {
			marker = "  current here"
		}
		lines = append(lines, prefix+style.Render(name)+m.styles.Muted.Render(marker))
	}
	return lines
}

func (m model) profileLibrary() []string {
	lines := []string{"  PROFILES", ""}
	if len(m.profiles) == 0 {
		lines = append(lines, "  No profiles yet. Press c to create one.")
	}
	start, end := selectionWindow(len(m.profiles), m.selected, max(1, m.h-17))
	if start > 0 {
		lines = append(lines, m.styles.Faint.Render(fmt.Sprintf("  … %d earlier", start)))
	}
	for i, name := range m.profiles[start:end] {
		i += start
		prefix := "   "
		style := m.styles.Name
		if i == m.selected {
			prefix = "  › "
			style = m.styles.RowSel
		}
		dir, _ := m.cfg.ProfileConfigDir(name)
		status := "ready"
		if !m.demoFrame {
			status = profile.Status(dir)
		}
		mark := "○"
		if status == "ready" {
			mark = "✓"
		}
		if status == "missing" {
			mark = "!"
		}
		active := ""
		if name == m.active {
			active = "  current"
		}
		lines = append(lines, prefix+mark+"  "+style.Render(name)+m.styles.Muted.Render("  "+status+active))
	}
	if end < len(m.profiles) {
		lines = append(lines, m.styles.Faint.Render(fmt.Sprintf("  … %d more", len(m.profiles)-end)))
	}
	lines = append(lines, "", "  Selected profile details")
	if m.selected < len(m.profiles) {
		name := m.profiles[m.selected]
		dir, _ := m.cfg.ProfileConfigDir(name)
		typeLabel := "independent profile"
		if m.cfg.Profiles[name].OriginAccount {
			typeLabel = "origin account (copied from ~/.claude)"
		}
		lines = append(lines, "    Config   "+m.styles.Muted.Render(dir), "    Type     "+typeLabel)
	}
	return lines
}

func (m model) projectRouting() []string {
	lines := []string{"  PROJECT WORKSPACE", "", "  Working directory  " + m.styles.Muted.Render(m.cwd), "  Resolved profile    " + m.resolution(), "  Local pin           " + m.pinSummary(), "", "  PATH RULES"}
	if len(m.cfg.Rules) == 0 {
		lines = append(lines, "  No path rules configured.")
	}
	start, end := selectionWindow(len(m.cfg.Rules), m.selected, max(1, m.h-15))
	if start > 0 {
		lines = append(lines, m.styles.Faint.Render(fmt.Sprintf("  … %d earlier routes", start)))
	}
	for i, rule := range m.cfg.Rules[start:end] {
		i += start
		prefix := "   "
		style := m.styles.Name
		if i == m.selected {
			prefix = "  › "
			style = m.styles.RowSel
		}
		lines = append(lines, prefix+style.Render(rule.Path)+m.styles.Muted.Render("  →  "+rule.Profile))
	}
	if end < len(m.cfg.Rules) {
		lines = append(lines, m.styles.Faint.Render(fmt.Sprintf("  … %d more routes", len(m.cfg.Rules)-end)))
	}
	lines = append(lines, "", "  Rules are checked in order after the nearest .claudio-account pin.")
	return lines
}

func selectionWindow(total, selected, visible int) (int, int) {
	if total <= visible {
		return 0, total
	}
	start := selected - visible/2
	if start < 0 {
		start = 0
	}
	if start+visible > total {
		start = total - visible
	}
	return start, start + visible
}

func (m model) setup() []string {
	if m.demoFrame {
		return []string{"  SETUP OVERVIEW", "", "  ✓ Claude binary  /usr/local/bin/claude", "  ✓ Config file    ~/.claudio/config.json", "  ✓ Profile work   ready", "  ✓ Profile personal  ready", "", "  Resolved profile  work · path rule (~/projects/work/**)", "", "  Run `claudio doctor` for profile and project-conflict checks."}
	}
	claude, err := exec.LookPath("claude")
	claudeState := m.styles.Error.Render("✗ not found in PATH")
	if err == nil {
		claudeState = m.styles.Success.Render("✓ " + claude)
	}
	cfgPath, _ := config.Path()
	cfgState := m.styles.Success.Render("✓ readable")
	if _, err := os.Stat(cfgPath); err != nil {
		if os.IsNotExist(err) {
			cfgState = m.styles.Warning.Render("! not created yet")
		} else {
			cfgState = m.styles.Error.Render("✗ " + err.Error())
		}
	}
	lines := []string{"  SETUP OVERVIEW", "", "  Claude binary  " + claudeState, "  Config file    " + cfgState + m.styles.Muted.Render("  "+cfgPath), ""}
	if len(m.profiles) == 0 {
		lines = append(lines, "  "+m.styles.Warning.Render("! No profiles configured · press 2 to create one"))
	}
	for _, name := range m.profiles {
		dir, _ := m.cfg.ProfileConfigDir(name)
		status := profile.Status(dir)
		mark, style := "✓", m.styles.Success
		if status == "missing" || status == "error" {
			mark, style = "✗", m.styles.Error
		}
		lines = append(lines, "  "+style.Render(mark+" Profile "+name)+m.styles.Muted.Render("  "+status+" · "+dir))
	}
	if m.active == "" {
		lines = append(lines, "", "  "+m.styles.Warning.Render("! No profile resolves here · choose one in Project Routing"))
	} else {
		lines = append(lines, "", "  Resolved profile  "+m.resolution())
	}
	if ext := os.Getenv("CLAUDE_CONFIG_DIR"); ext != "" {
		lines = append(lines, "  "+m.styles.Warning.Render("! CLAUDE_CONFIG_DIR is set externally · "+ext))
	}
	lines = append(lines, "", "  Run `claudio doctor` for profile and project-conflict checks.")
	return lines
}

func (m model) resolution() string {
	if m.active == "" {
		return m.styles.Warning.Render("none · select a profile")
	}
	return m.styles.Success.Render(m.active) + m.styles.Muted.Render(" · "+m.reason)
}

func (m model) pinSummary() string {
	if m.demoFrame {
		return "work  " + m.styles.Muted.Render("(~/projects/work)")
	}
	dir := m.nearestPinDir()
	if dir != "" {
		data, _ := os.ReadFile(filepath.Join(dir, ".claudio-account"))
		return strings.TrimSpace(string(data)) + m.styles.Muted.Render("  ("+dir+")")
	}
	return m.styles.Faint.Render("not pinned")
}

func (m model) nearestPinDir() string {
	path, err := resolverpkg.ProjectOverridePath()
	if err != nil {
		return ""
	}
	return filepath.Dir(path)
}
func (m model) nearestPinPath() string {
	if m.demoFrame {
		return ""
	}
	path, err := resolverpkg.ProjectOverridePath()
	if err != nil {
		return ""
	}
	return path
}

func (m model) inputPrompt() string {
	switch m.inputMode {
	case "create":
		return fmt.Sprintf("  New profile name: %s_   [enter save · esc cancel]", m.input)
	case "rename":
		return fmt.Sprintf("  Rename to: %s_   [enter save · esc cancel]", m.input)
	case "rule-path":
		return fmt.Sprintf("  Route path or glob (e.g. ~/work/**): %s_", m.input)
	case "rule-profile":
		name := "none"
		if len(m.profiles) > 0 {
			name = m.profiles[min(max(m.selected, 0), len(m.profiles)-1)]
		}
		return fmt.Sprintf("  Profile: %s   [↑/↓ choose · enter save · esc cancel]", name)
	case "pin-profile":
		name := "none"
		if len(m.profiles) > 0 {
			name = m.profiles[min(max(m.selected, 0), len(m.profiles)-1)]
		}
		return fmt.Sprintf("  Pin project to: %s   [↑/↓ choose · enter save · esc cancel]", name)
	}
	return ""
}

func (m model) footer() string {
	var hints string
	switch m.screen {
	case "home":
		hints = "  ↑↓ move   enter open   1–4 workspaces   ? help   q quit"
	case "launch":
		hints = "  ↑↓ select profile   enter launch Claude   esc back"
	case "profiles":
		hints = "  ↑↓ select  enter launch  l login  c new  r rename  x delete  esc back"
	case "projects":
		hints = "  p pin  u unpin  a add route  d remove route  esc back"
	default:
		hints = "  esc back   q quit"
	}
	if m.inputMode != "" {
		hints = "  type value   enter continue/save   esc cancel"
	}
	return m.styles.FooterBar.Render(hints)
}

func Run() error {
	theme, err := LoadTheme("")
	if err != nil {
		return err
	}
	m, err := newModel(theme, false)
	if err != nil {
		return err
	}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}
	if result, ok := final.(model); ok && result.message != "" {
		fmt.Fprintln(os.Stderr, result.message)
	}
	return nil
}

func RenderFrame(cols, rows int, depth, themePath, screen string, selected int, out io.Writer) error {
	theme, err := LoadTheme(themePath)
	if err != nil {
		return err
	}
	ansi16 := depth == "16"
	if screen == "" {
		screen = "home"
	}
	if screen != "home" && screen != "launch" && screen != "profiles" && screen != "projects" && screen != "setup" && screen != "help" {
		return fmt.Errorf("screen must be home, launch, profiles, projects, setup or help")
	}
	m := model{styles: NewStyles(theme, ansi16), w: cols, h: rows, screen: screen, cwd: "~/projects/work", demoFrame: true,
		profiles: []string{"personal", "work"}, cfg: &config.Config{Profiles: map[string]config.Profile{
			"personal": {ConfigDir: "/home/demo/.claudio/profiles/personal/claude"}, "work": {ConfigDir: "/home/demo/.claudio/profiles/work/claude"},
		}, Rules: []config.Rule{{Path: "~/projects/work/**", Profile: "work"}}}, active: "work", reason: "path rule (~/projects/work/**)"}
	if screen == "profiles" || screen == "launch" {
		m.selected = max(0, min(selected, len(m.profiles)-1))
	} else if screen == "projects" {
		m.selected = max(0, min(selected, len(m.cfg.Rules)-1))
	} else {
		m.selected = max(0, min(selected, 3))
	}
	update, _ := m.Update(tea.WindowSizeMsg{Width: cols, Height: rows})
	content := update.(model).View().Content
	p := map[string]colorprofile.Profile{"truecolor": colorprofile.TrueColor, "256": colorprofile.ANSI256, "16": colorprofile.ANSI}[depth]
	if p == colorprofile.Unknown {
		return fmt.Errorf("depth must be truecolor, 256 or 16")
	}
	writer := &colorprofile.Writer{Forward: out, Profile: p}
	_, err = fmt.Fprintln(writer, strings.TrimRight(content, "\n"))
	return err
}

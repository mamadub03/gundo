package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/manifoldco/promptui"
	"golang.org/x/term"
)

// Presentation stays separate from GitHub retrieval and checkpoint ownership.
type operatorConsole struct {
	reader      *bufio.Reader
	out         io.Writer
	interactive bool
	color       bool
}

func newOperatorConsole() *operatorConsole {
	tty := term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("TERM") != "dumb" && os.Getenv("GUNDO_PLAIN") != "1"
	_, noColor := os.LookupEnv("NO_COLOR")
	return &operatorConsole{bufio.NewReader(os.Stdin), os.Stdout, tty, tty && !noColor}
}

func (c *operatorConsole) style(code, text string) string {
	if !c.color {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// Remote output is data, not terminal instructions (including OSC/CSI sequences).
var terminalSequence = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\))`)

func consoleText(value string) string {
	value = terminalSequence.ReplaceAllString(value, "")
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, value)
}
func consoleLabel(value string) string { return strings.Join(strings.Fields(consoleText(value)), " ") }

func (c *operatorConsole) banner() {
	art := `
       ______  __  __  _   __  ____   ____
      / ____/ / / / / / | / / / __ \ / __ \
     / / __  / / / / /  |/ / / / / // / / /
    / /_/ / / /_/ / / /|  / / /_/ // /_/ /
    \____/  \____/ /_/ |_/ /_____/ \____/

              [ G U N D O ]
`
	fmt.Fprint(c.out, c.style("1;32", art))
	fmt.Fprintf(c.out, "\n   OPERATOR CONSOLE  %s\n\n", c.style("90", "v"+clientVersion))
}

// Redraw only actual interactive terminals; redirected output stays readable.
func (c *operatorConsole) screen() {
	if c.interactive {
		fmt.Fprint(c.out, "\x1b[2J\x1b[H")
	}
	c.banner()
}

func (c *operatorConsole) failure(err error) {
	fmt.Fprintln(c.out, c.style("31", "  Error: "+consoleLabel(err.Error())))
}
func (c *operatorConsole) line(prompt string) (string, error) {
	fmt.Fprint(c.out, prompt)
	value, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

// A select returns an index only; callers retain the underlying agent/action.
func (c *operatorConsole) selectItem(label string, items []string) (int, error) {
	if len(items) == 0 {
		return -1, fmt.Errorf("no choices available")
	}
	if c.interactive {
		active := "> {{ . }}"
		if c.color {
			active = `{{ ">" | cyan }} {{ . }}`
		}
		prompt := promptui.Select{
			Label: label, Items: items, Size: 10, HideSelected: true, HideHelp: true,
			Templates: &promptui.SelectTemplates{Label: "{{ . }}", Active: active, Inactive: "  {{ . }}", Selected: "{{ . }}"},
			Searcher: func(input string, index int) bool {
				return strings.Contains(strings.ToLower(consoleText(items[index])), strings.ToLower(input))
			},
		}
		fmt.Fprintln(c.out, c.style("90", "  ↑↓ Navigate · Enter Select · / Search · Ctrl+C Back/Exit"))
		index, _, err := prompt.Run()
		return index, err
	}
	fmt.Fprintln(c.out, "\n  "+label)
	for i, item := range items {
		fmt.Fprintf(c.out, "  %d. %s\n", i+1, item)
	}
	for {
		input, err := c.line("  Select number (0 to back/exit): ")
		if err != nil {
			return -1, err
		}
		if input == "0" {
			return -1, io.EOF
		}
		index, err := strconv.Atoi(input)
		if err == nil && index >= 1 && index <= len(items) {
			return index - 1, nil
		}
		fmt.Fprintln(c.out, "  Enter one of the listed numbers.")
	}
}

func heartbeatAge(stamp, now time.Time) string {
	if stamp.IsZero() {
		return "no heartbeat"
	}
	age := now.Sub(stamp)
	if age < 0 {
		return "just now"
	}
	if age < time.Minute {
		return fmt.Sprintf("%ds ago", int(age.Seconds()))
	}
	if age < time.Hour {
		return fmt.Sprintf("%dm ago", int(age.Minutes()))
	}
	if age < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(age.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(age.Hours()/24))
}
func visibleAgents(clients []ManagedClient, showOffline bool) []ManagedClient {
	var visible []ManagedClient
	for _, client := range clients {
		if showOffline || client.Online {
			visible = append(visible, client)
		}
	}
	sort.SliceStable(visible, func(i, j int) bool {
		if visible[i].Online != visible[j].Online {
			return visible[i].Online
		}
		return visible[i].IssueNumber < visible[j].IssueNumber
	})
	return visible
}
func (c *operatorConsole) agentLabels(clients []ManagedClient) []string {
	labels := make([]string, len(clients))
	for i, client := range clients {
		status := "○ Offline"
		if client.Online {
			status = "● Online"
		}
		// Keep the list compact; full identity and heartbeat age appear after selection.
		labels[i] = fmt.Sprintf("%s   Agent #%d", c.style("32", fmt.Sprintf("%-9s", status)), client.IssueNumber)
	}
	return labels
}
func (c *operatorConsole) agentHeading(client ManagedClient) {
	fmt.Fprintf(c.out, "\n  %s\n", c.style("1;36", "AGENT · Issue #"+strconv.Itoa(client.IssueNumber)))
	fmt.Fprintf(c.out, "  ClientID  %s\n", consoleLabel(client.ClientID))
	fmt.Fprintf(c.out, "  %s · Last heartbeat: %s\n", c.style("32", "Online"), heartbeatAge(client.LastSeen, time.Now()))
}
func (c *operatorConsole) result(action, taskID string, result GundoMessage, elapsed time.Duration) {
	code := "32"
	if result.Status != "success" {
		code = "31"
	}
	fmt.Fprintf(c.out, "\n  %s\n", c.style(code, consoleLabel(action)+" · "+consoleLabel(result.Status)))
	fmt.Fprintf(c.out, "  Task     %s\n  Elapsed  %s\n", consoleLabel(taskID), elapsed.Round(time.Millisecond))
	fmt.Fprintln(c.out, "\n  OUTPUT\n  ────────────────────────────────────────")
	fmt.Fprintln(c.out, consoleText(result.Output))
	fmt.Fprintln(c.out, "  ────────────────────────────────────────")
}

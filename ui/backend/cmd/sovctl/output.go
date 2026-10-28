package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

var useColor bool

// initOutput turns colour on only for a terminal, and never with --no-color,
// NO_COLOR or -o json.
func initOutput() {
	fi, err := os.Stdout.Stat()
	tty := err == nil && fi.Mode()&os.ModeCharDevice != 0
	useColor = tty && !noColor && os.Getenv("NO_COLOR") == "" && outputMode != "json"
}

func paint(code, s string) string {
	if !useColor {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func red(s string) string    { return paint("31", s) }
func green(s string) string  { return paint("32", s) }
func yellow(s string) string { return paint("33", s) }
func blue(s string) string   { return paint("34", s) }
func cyan(s string) string   { return paint("36", s) }
func bold(s string) string   { return paint("1", s) }
func dim(s string) string    { return paint("2", s) }

// statusColor colours common Kubernetes-ish status words.
func statusColor(s string) string {
	switch strings.ToLower(s) {
	case "ready", "running", "succeeded", "active", "true", "connected", "provisioned", "programmed", "ok", "up", "yes", "allowed", "accepted", "live":
		return green(s)
	case "pending", "importinprogress", "provisioning", "scheduling", "starting", "waking…", "mixed", "deleting…", "unknown":
		return yellow(s)
	case "failed", "error", "notready", "false", "disconnected", "crashloopbackoff", "no", "denied", "fallback":
		return red(s)
	}
	return s
}

func yesNo(b bool) string {
	if b {
		return green("yes")
	}
	return red("no")
}

func printRawJSON(raw []byte) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		_, err = os.Stdout.Write(raw)
		return err
	}
	buf.WriteByte('\n')
	_, err := buf.WriteTo(os.Stdout)
	return err
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// table prints aligned columns. Colour codes are zero-width for the eye but
// not for tabwriter, so cells are padded on the plain text first.
type table struct {
	headers []string
	rows    [][]string
}

func newTable(headers ...string) *table { return &table{headers: headers} }

func (t *table) add(cells ...string) { t.rows = append(t.rows, cells) }

func (t *table) print() {
	if len(t.rows) == 0 {
		fmt.Println(dim("(none)"))
		return
	}
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = len(h)
	}
	for _, r := range t.rows {
		for i, c := range r {
			if i < len(widths) {
				widths[i] = max(widths[i], visibleLen(c))
			}
		}
	}
	w := bufio.NewWriter(os.Stdout)
	line := func(cells []string, style func(string) string) {
		for i, c := range cells {
			if i > 0 {
				w.WriteString("  ")
			}
			if style != nil {
				c = style(c)
			}
			w.WriteString(c)
			if i < len(cells)-1 {
				w.WriteString(strings.Repeat(" ", widths[i]-visibleLen(c)))
			}
		}
		w.WriteString("\n")
	}
	line(t.headers, bold)
	for _, r := range t.rows {
		line(r, nil)
	}
	w.Flush()
}

// visibleLen is the printed width of s, ignoring ANSI colour codes.
func visibleLen(s string) int {
	n, esc := 0, false
	for _, r := range s {
		switch {
		case esc:
			if r == 'm' {
				esc = false
			}
		case r == '\x1b':
			esc = true
		default:
			n++
		}
	}
	return n
}

// kv prints "key: value" pairs, aligned.
func kv(pairs ...string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for i := 0; i+1 < len(pairs); i += 2 {
		key := pairs[i]
		if key != "" {
			key += ":"
		}
		fmt.Fprintf(w, "%s\t%s\n", dim(key), pairs[i+1])
	}
	w.Flush()
}

func section(title string) { fmt.Println("\n" + bold(title)) }

func notes(source string, ns []string) {
	if source != "" && source != "live" {
		fmt.Println(dim("source: ") + statusColor(source))
	}
	for _, n := range ns {
		fmt.Println(dim("note: " + n))
	}
}

func join(xs []string) string {
	if len(xs) == 0 {
		return "-"
	}
	return strings.Join(xs, ",")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// confirm asks a yes/no question on the terminal. -y skips it.
func confirm(prompt string, yes bool) bool {
	if yes {
		return true
	}
	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)
	r := bufio.NewReader(os.Stdin)
	ans, _ := r.ReadString('\n')
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes"
}

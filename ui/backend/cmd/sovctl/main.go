// sovctl is the command-line twin of the KubeUI dashboard. It talks to the
// same backend HTTP API (ui/backend, :8080), so anything the dashboard can do
// is one command away: clusters and their on-demand idle timer, VMs, images,
// the registry, the Istio mesh, cross-cluster connections, security scans, AI
// agents and the web terminal.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

var version = "dev"

// command is one node of the command tree: either a leaf (run) or a group
// (sub). Groups with a run fall back to it when no subcommand is given.
type command struct {
	name    string
	summary string
	usage   string // "[flags] <arg>", shown after the full name
	run     func(args []string) error
	sub     []*command
}

func (c *command) find(name string) *command {
	for _, s := range c.sub {
		if s.name == name {
			return s
		}
	}
	return nil
}

var root = &command{name: "sovctl", summary: "command-line control for the KubeUI platform"}

func init() {
	root.sub = []*command{
		statusCmd, clustersCmd, clusterCmd, timerCmd,
		nodesCmd, podsCmd, namespacesCmd, eventsCmd,
		vmsCmd, vmCmd, imagesCmd, registryCmd,
		meshCmd, connectionsCmd, securityCmd, aiCmd,
		configCmd, shellCmd, completionCmd, versionCmd, helpCmd,
	}
}

var helpCmd = &command{name: "help", summary: "show help for a command", usage: "[command...]",
	run: func(args []string) error {
		c, path := root, "sovctl"
		for _, a := range args {
			n := c.find(a)
			if n == nil {
				return fmt.Errorf("unknown command %q", strings.Join(args, " "))
			}
			c, path = n, path+" "+a
		}
		printHelp(os.Stdout, c, path)
		return nil
	}}

var versionCmd = &command{name: "version", summary: "print the sovctl version",
	run: func(args []string) error { fmt.Println("sovctl", version); return nil }}

func printHelp(w io.Writer, c *command, path string) {
	fmt.Fprintf(w, "%s — %s\n\n", path, c.summary)
	if c.run != nil || len(c.sub) == 0 {
		fmt.Fprintf(w, "Usage:\n  %s %s\n\n", path, c.usage)
	}
	if len(c.sub) > 0 {
		fmt.Fprintln(w, "Commands:")
		width := 0
		for _, s := range c.sub {
			width = max(width, len(s.name))
		}
		for _, s := range c.sub {
			fmt.Fprintf(w, "  %-*s  %s\n", width, s.name, s.summary)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "Global flags (anywhere on the line):")
	fmt.Fprintln(w, "  --server URL   backend address (env SOVCTL_SERVER, default http://localhost:8080)")
	fmt.Fprintln(w, "  -o table|json  output format (default table)")
	fmt.Fprintln(w, "  --no-color     disable colour")
	if path == "sovctl" {
		fmt.Fprintln(w, "\nRun 'sovctl help <command>' for details.")
	}
}

// errUsage marks an error that should be followed by the command's help.
var errUsage = errors.New("usage")

func usageErr(format string, a ...any) error {
	return fmt.Errorf("%w: %s", errUsage, fmt.Sprintf(format, a...))
}

func main() {
	args, err := extractGlobals(os.Args[1:])
	if err != nil {
		fail(err)
	}
	initOutput()
	if len(args) == 0 {
		args = []string{"status"}
	}

	c, path := root, "sovctl"
	for len(args) > 0 && len(c.sub) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			printHelp(os.Stdout, c, path)
			return
		}
		n := c.find(args[0])
		if n == nil {
			if c.run != nil {
				break
			}
			fmt.Fprintf(os.Stderr, "sovctl: unknown command %q\n\n", strings.Join(append(strings.Fields(path)[1:], args[0]), " "))
			printHelp(os.Stderr, c, path)
			os.Exit(2)
		}
		c, path, args = n, path+" "+args[0], args[1:]
	}
	if c.run == nil {
		printHelp(os.Stdout, c, path)
		return
	}
	for _, a := range args {
		if a == "-h" || a == "--help" {
			printHelp(os.Stdout, c, path)
			return
		}
	}
	if err := c.run(args); err != nil {
		if errors.Is(err, errUsage) {
			fmt.Fprintf(os.Stderr, "sovctl: %s\n\n", strings.TrimPrefix(err.Error(), errUsage.Error()+": "))
			printHelp(os.Stderr, c, path)
			os.Exit(2)
		}
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, red("error: ")+err.Error())
	os.Exit(1)
}

// Global options, settable anywhere on the command line.
var (
	serverURL  = envDefault("SOVCTL_SERVER", "http://localhost:8080")
	outputMode = "table"
	noColor    bool
)

func envDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// extractGlobals pulls --server, -o/--output and --no-color out of args,
// wherever they appear, and returns the rest in order.
func extractGlobals(args []string) ([]string, error) {
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, val, hasVal := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if !strings.HasPrefix(a, "-") || a == "-" || a == "--" {
			rest = append(rest, a)
			continue
		}
		takeVal := func() (string, error) {
			if hasVal {
				return val, nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("flag %s needs a value", a)
			}
			i++
			return args[i], nil
		}
		switch name {
		case "server":
			v, err := takeVal()
			if err != nil {
				return nil, err
			}
			serverURL = v
		case "o", "output":
			v, err := takeVal()
			if err != nil {
				return nil, err
			}
			if v != "table" && v != "json" {
				return nil, fmt.Errorf("-o must be table or json, not %q", v)
			}
			outputMode = v
		case "no-color":
			noColor = true
		default:
			rest = append(rest, a)
		}
	}
	serverURL = strings.TrimRight(serverURL, "/")
	return rest, nil
}

// parseFlags parses a subcommand's flags allowing them before, between or
// after positional arguments (Go's flag package stops at the first one).
func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	fs.SetOutput(io.Discard)
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, usageErr("%v", err)
		}
		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}
		if args[0] == "--" {
			return append(pos, args[1:]...), nil
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
}

func commandNames(c *command) []string {
	var out []string
	for _, s := range c.sub {
		out = append(out, s.name)
	}
	sort.Strings(out)
	return out
}

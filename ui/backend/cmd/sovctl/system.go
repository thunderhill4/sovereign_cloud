package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var configCmd = &command{name: "config", summary: "backend settings: pool mode, control plane, images, profiles, registry, AI",
	run: func(args []string) error {
		var c backendConfig
		return show("/api/v1/config", &c, func() {
			fmt.Println(bold("Target cluster"))
			kv("control plane", c.TargetCPMode+dim(" (TARGET_CP_MODE)"),
				"pool", fmt.Sprintf("%s · %s", map[bool]string{true: "enabled", false: "disabled"}[c.Pool.Enabled], c.Pool.Mode)+dim(" (POOL_ENABLED, POOL_MODE)"),
				"idle auto-delete", fmtTTL(c.Pool.IdleTTLSeconds)+dim(" ("+c.Pool.IdleTTLSource+"; change with 'sovctl timer set')"),
				"standby manifest", c.Pool.StandbyManifest,
				"front door", c.Pool.FrontDoorAddr+dim(" (published as 172.18.255.223:6443)"),
				"repo", c.RepoDir)
			section("Deploy images (sovctl cluster deploy --image)")
			t := newTable("KEY", "DEFAULT", "ALLOWED", "IMAGE", "DESCRIPTION")
			for _, i := range c.Images {
				t.add(i.Key, map[bool]string{true: green("*"), false: ""}[i.Default], yesNo(i.Allowed), dim(i.Image), i.Label)
			}
			t.print()
			section("Profiles (sovctl cluster deploy --profile)")
			p := newTable("KEY", "DESCRIPTION")
			for _, x := range c.Profiles {
				p.add(x.Key, x.Label)
			}
			p.print()
			section("Services")
			kv("registry", c.RegistryURL+dim(" (shown as "+c.RegistryAlias+")"), "security agent", c.SecurityAgentURL)
			section("AI mode")
			printAIConfig(c.AI)
		})
	}}

// shellCmd attaches to the dashboard's web terminal: a shell on the backend
// host, started in the repo. Same WebSocket protocol as TerminalView.tsx.
var shellCmd = &command{name: "shell", summary: "open the dashboard's web terminal (a shell on the backend host)",
	run: func(args []string) error {
		u, err := url.Parse(serverURL + "/api/v1/terminal")
		if err != nil {
			return err
		}
		u.Scheme = map[string]string{"https": "wss"}[u.Scheme]
		if u.Scheme == "" {
			u.Scheme = "ws"
		}
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			return connErr(err)
		}
		defer conn.Close()

		restore := rawTerminal()
		defer restore()

		send := func(v any) {
			b, _ := json.Marshal(v)
			_ = conn.WriteMessage(websocket.TextMessage, b)
		}
		resize := func() {
			if cols, rows, ok := termSize(); ok {
				send(map[string]any{"type": "resize", "cols": cols, "rows": rows})
			}
		}
		resize()
		winch := make(chan os.Signal, 1)
		signal.Notify(winch, syscall.SIGWINCH)
		go func() {
			for range winch {
				resize()
			}
		}()
		// zsh discards typeahead while it starts, so hold stdin until the shell
		// has printed something (its prompt), or 3s at most.
		ready := make(chan struct{})
		go func() {
			select {
			case <-ready:
			case <-time.After(3 * time.Second):
			}
			buf := make([]byte, 4096)
			for {
				n, err := os.Stdin.Read(buf)
				if n > 0 {
					send(map[string]string{"type": "input", "data": string(buf[:n])})
				}
				if err != nil {
					// EOF (piped input ran out): end the shell and keep reading its
					// output until it exits and the server hangs up. zsh drops a
					// Ctrl-D sent as typeahead, so say exit.
					send(map[string]string{"type": "input", "data": "\nexit\n"})
					return
				}
			}
		}()
		var once sync.Once
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return nil // shell exited or connection closed
			}
			os.Stdout.Write(msg)
			once.Do(func() { close(ready) })
		}
	}}

// rawTerminal puts the local tty in raw mode via stty and returns the restore.
func rawTerminal() func() {
	saved, err := stty("-g")
	if err != nil {
		return func() {}
	}
	if _, err := stty("raw", "-echo"); err != nil {
		return func() {}
	}
	return func() { _, _ = stty(strings.TrimSpace(saved)) }
}

func stty(args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	return string(out), err
}

func termSize() (cols, rows int, ok bool) {
	out, err := stty("size")
	if err != nil {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(out, "%d %d", &rows, &cols); err != nil {
		return 0, 0, false
	}
	return cols, rows, true
}

var completionCmd = &command{name: "completion", summary: "print a shell completion script", usage: "bash|zsh",
	run: func(args []string) error {
		if len(args) != 1 || (args[0] != "bash" && args[0] != "zsh") {
			return usageErr("give bash or zsh")
		}
		var b strings.Builder
		// One case per command path that has subcommands, generated from the tree.
		var walk func(c *command, path string)
		walk = func(c *command, path string) {
			if len(c.sub) == 0 {
				return
			}
			fmt.Fprintf(&b, "    %q) words=%q ;;\n", path, strings.Join(commandNames(c), " "))
			for _, s := range c.sub {
				walk(s, strings.TrimSpace(path+" "+s.name))
			}
		}
		walk(root, "")
		cases := b.String()
		if args[0] == "zsh" {
			fmt.Println("autoload -U +X bashcompinit && bashcompinit")
		}
		fmt.Printf(`_sovctl() {
  local cur path="" i words
  cur="${COMP_WORDS[COMP_CWORD]}"
  for ((i=1; i<COMP_CWORD; i++)); do
    case "${COMP_WORDS[i]}" in
      -o|--output|--server|-n|--namespace|--agent|-a|--profile|--image|--tool|--count|--timeout|--interval|--username) ((i++)) ;;
      -*) ;;
      *) path="${path:+$path }${COMP_WORDS[i]}" ;;
    esac
  done
  case "$path" in
%s    *) words="" ;;
  esac
  COMPREPLY=($(compgen -W "$words" -- "$cur"))
}
complete -F _sovctl sovctl
`, cases)
		return nil
	}}

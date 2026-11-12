package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

var aiCmd = &command{name: "ai", summary: "AI mode: Sympozium agents, chat, approvals, configuration, console"}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type envelope struct {
	Type     string    `json:"type"` // text | proposal | tool_step | error | done
	Text     string    `json:"text"`
	Proposal *proposal `json:"proposal"`
	Step     *logEntry `json:"step"`
	Error    string    `json:"error"`
}

type proposal struct {
	ID      string `json:"id"`
	Action  string `json:"action"`
	Profile string `json:"profile"`
	Reason  string `json:"reason"`
}

// chatOnce sends the conversation, streams the reply to stdout and returns the
// full reply text plus any action the agent proposed.
func chatOnce(agent string, msgs []chatMessage) (string, *proposal, error) {
	body := map[string]any{"messages": msgs}
	if agent != "" {
		body["agent"] = agent
	}
	var reply strings.Builder
	var prop *proposal
	var chatErr string
	err := stream(http.MethodPost, "/api/ai/chat", body, func(data []byte) error {
		if outputMode == "json" {
			fmt.Println(string(data))
		}
		var e envelope
		if err := json.Unmarshal(data, &e); err != nil {
			return nil
		}
		switch e.Type {
		case "text":
			reply.WriteString(e.Text)
			if outputMode != "json" {
				fmt.Print(e.Text)
			}
		case "proposal":
			prop = e.Proposal
		case "error":
			chatErr = e.Error
		}
		return nil
	})
	if outputMode != "json" && reply.Len() > 0 && !strings.HasSuffix(reply.String(), "\n") {
		fmt.Println()
	}
	if err != nil {
		return "", nil, err
	}
	if chatErr != "" {
		return reply.String(), nil, fmt.Errorf("%s", chatErr)
	}
	return reply.String(), prop, nil
}

// runAction runs an approved action (today only deploy_target_cluster),
// streaming its deploy log.
func runAction(action, profile string) error {
	failed := false
	err := stream(http.MethodPost, "/api/ai/action", map[string]string{"action": action, "profile": profile}, func(data []byte) error {
		if outputMode == "json" {
			fmt.Println(string(data))
			return nil
		}
		var e envelope
		if err := json.Unmarshal(data, &e); err != nil {
			return nil
		}
		switch e.Type {
		case "text":
			fmt.Println(e.Text)
		case "tool_step":
			if e.Step != nil {
				printLogEntry(*e.Step)
				if e.Step.Type == "done" && e.Step.Message == "failed" {
					failed = true
				}
			}
		case "error":
			fmt.Println(red("✗ " + e.Error))
			failed = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if failed {
		return fmt.Errorf("action %s failed", action)
	}
	return nil
}

func handleProposal(p *proposal, autoApprove bool) error {
	if p == nil {
		return nil
	}
	fmt.Printf("\n%s %s (profile %s)", yellow("Proposed action:"), bold(p.Action), orDash(p.Profile))
	if p.Reason != "" {
		fmt.Printf(" — %s", p.Reason)
	}
	fmt.Println()
	if !confirm("Approve and run it?", autoApprove) {
		fmt.Println(dim("not approved"))
		return nil
	}
	return runAction(p.Action, p.Profile)
}

func init() {
	aiCmd.sub = []*command{
		{name: "agents", summary: "list the Sympozium agents chat can use",
			run: func(args []string) error {
				var r struct {
					Agents []struct {
						Name        string `json:"name"`
						Description string `json:"description"`
					} `json:"agents"`
					Default string `json:"default"`
				}
				var cfg backendConfig
				_, _ = get("/api/v1/config", &cfg) // optional: adds env-configured agents + URLs
				return show("/api/ai/agents", &r, func() {
					urls := map[string]string{}
					seen := map[string]bool{}
					for _, a := range cfg.AI.Agents {
						urls[a.Name] = a.URL
					}
					t := newTable("AGENT", "DEFAULT", "URL", "DESCRIPTION")
					for _, a := range r.Agents {
						seen[a.Name] = true
						t.add(a.Name, map[bool]string{true: green("*"), false: ""}[a.Name == r.Default], dim(orDash(urls[a.Name])), a.Description)
					}
					// The backend lists SympoziumInstance CRs, a CRD newer charts dropped,
					// so agents set up only by env URL would otherwise be invisible.
					for _, a := range cfg.AI.Agents {
						if !seen[a.Name] {
							t.add(a.Name, "", dim(a.URL), dim("(configured by SYMPOZIUM_AGENT_URL_*)"))
						}
					}
					t.print()
				})
			}},
		{name: "chat", summary: "chat with an agent: one message, or an interactive session with none", usage: "[--agent NAME] [--yes] [message...]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("chat", flag.ContinueOnError)
				agent := fs.String("agent", "", "agent name (default: the backend's SYMPOZIUM_DEFAULT_AGENT); see 'sovctl ai agents'")
				fs.StringVar(agent, "a", "", "agent name")
				yes := fs.Bool("yes", false, "auto-approve proposed actions")
				pos, err := parseFlags(fs, args)
				if err != nil {
					return err
				}
				if len(pos) > 0 {
					_, p, err := chatOnce(*agent, []chatMessage{{Role: "user", Content: strings.Join(pos, " ")}})
					if err != nil {
						return err
					}
					return handleProposal(p, *yes)
				}
				// Interactive: keep the history so the agent sees the whole conversation.
				name := *agent
				if name == "" {
					name = "default agent"
				}
				fmt.Println(dim("chatting with " + name + " — /new clears the history, /exit or Ctrl+D quits"))
				var history []chatMessage
				in := bufio.NewScanner(os.Stdin)
				in.Buffer(make([]byte, 0, 64<<10), 1<<20)
				for {
					fmt.Print(bold("you> "))
					if !in.Scan() {
						fmt.Println()
						return nil
					}
					line := strings.TrimSpace(in.Text())
					switch line {
					case "":
						continue
					case "/exit", "/quit":
						return nil
					case "/new":
						history = nil
						fmt.Println(dim("history cleared"))
						continue
					}
					history = append(history, chatMessage{Role: "user", Content: line})
					fmt.Print(cyan("agent> "))
					reply, p, err := chatOnce(*agent, history)
					if err != nil {
						fmt.Println(red("✗ " + err.Error()))
						history = history[:len(history)-1]
						continue
					}
					history = append(history, chatMessage{Role: "assistant", Content: reply})
					if err := handleProposal(p, *yes); err != nil {
						fmt.Println(red("✗ " + err.Error()))
					}
				}
			}},
		{name: "approve", summary: "run an action an agent can propose (deploy_target_cluster)", usage: "deploy [--profile lite|full] [-y]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("approve", flag.ContinueOnError)
				profile := fs.String("profile", "lite", "lite | full")
				yes := fs.Bool("y", false, "do not ask for confirmation")
				pos, err := parseFlags(fs, args)
				if err != nil {
					return err
				}
				if len(pos) != 1 || (pos[0] != "deploy" && pos[0] != "deploy_target_cluster") {
					return usageErr("the only allow-listed action is: deploy")
				}
				if !confirm("Deploy the target cluster as the AI action would (profile "+*profile+", image noble)?", *yes) {
					return fmt.Errorf("aborted")
				}
				return runAction("deploy_target_cluster", *profile)
			}},
		{name: "config", summary: "AI mode configuration: agents and URLs, default agent, guard, console",
			run: func(args []string) error {
				var cfg backendConfig
				raw, err := get("/api/v1/config", &cfg)
				if err != nil {
					return err
				}
				if outputMode == "json" {
					var m map[string]json.RawMessage
					_ = json.Unmarshal(raw, &m)
					return printRawJSON(m["ai"])
				}
				printAIConfig(cfg.AI)
				return nil
			}},
		{name: "console", summary: "the AI tab's Sympozium console URL (runs, schedules, policies)", usage: "[--open]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("console", flag.ContinueOnError)
				open := fs.Bool("open", false, "open it in the browser")
				if _, err := parseFlags(fs, args); err != nil {
					return err
				}
				var cfg backendConfig
				if _, err := get("/api/v1/config", &cfg); err != nil {
					return err
				}
				u, err := url.Parse(serverURL)
				if err != nil {
					return err
				}
				console := fmt.Sprintf("%s://%s:%s/", u.Scheme, u.Hostname(), cfg.AI.ConsolePort)
				fmt.Println(console)
				if *open {
					return exec.Command("xdg-open", console).Start()
				}
				return nil
			}},
	}
}

func printAIConfig(a aiConfig) {
	guard := red("off") + dim(" (LAYA_URL unset)")
	if a.GuardEnabled {
		guard = green("on") + fmt.Sprintf(" · %s · block at ≥ %.2f · key %s", a.GuardURL, a.GuardThreshold, setUnset(a.GuardKeySet))
	}
	kv("namespace", a.Namespace,
		"default agent", bold(a.DefaultAgent),
		"API token", setUnset(a.APITokenSet)+dim(" (SYMPOZIUM_API_TOKEN; per-agent <agent>-web-proxy-key Secrets win)"),
		"chat timeout", (time.Duration(a.ChatTimeoutSecs)*time.Second).String()+dim(" (AI_CHAT_TIMEOUT)"),
		"pre-model guard", guard,
		"console", fmt.Sprintf(":%s → %s", a.ConsolePort, a.ConsoleUpstream))
	section("Agents")
	t := newTable("AGENT", "DEFAULT", "URL")
	for _, ag := range a.Agents {
		t.add(ag.Name, map[bool]string{true: green("*"), false: ""}[ag.Default], ag.URL)
	}
	t.print()
	fmt.Println(dim("\nThese come from the backend's environment (run-ui.sh); change them there and restart `make ui`."))
}

func setUnset(b bool) string {
	if b {
		return green("set")
	}
	return dim("unset")
}

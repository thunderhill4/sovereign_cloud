package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// targetState is the CLI's twin of describeTarget in
// ui/frontend/src/hooks/usePoolStatus.ts: same states, same wording.
func targetState(p *poolStatus) (label, detail string) {
	if p == nil {
		return "Unknown", "backend not reachable"
	}
	ondemand := p.Mode == "ondemand"
	switch {
	case p.Deleting && p.DeletingSeconds >= 180:
		return red("Deleting (stuck)"), fmt.Sprintf("teardown running for %d min; check the CAPK controller: kubectl -n capk-system get pods", p.DeletingSeconds/60)
	case p.Deleting:
		return yellow("Deleting…"), "tearing the cluster down"
	case p.Stopped && p.State == "none":
		return red("Stopped"), "deleted from the UI; stays down until the next Deploy (sovctl cluster deploy)"
	case p.State == "building" || ((p.State == "warm" || p.State == "claimed") && !p.ClusterReady):
		if ondemand {
			return yellow("Waking…"), "accessed while asleep, building the cluster"
		}
		return yellow("Building…"), "building the standby cluster"
	case p.State == "warm" || p.State == "claimed":
		if !ondemand {
			if p.State == "warm" {
				return green("Standby ready"), "Deploy claims it instantly"
			}
			return green("Up"), "standby pool mode"
		}
		if p.IdleTTLSeconds <= 0 {
			return green("Up"), "on-demand · auto-delete off"
		}
		left := max(0, p.IdleTTLSeconds-p.IdleSeconds)
		return green("Up"), fmt.Sprintf("on-demand · auto-deletes after %s idle · %s left", fmtTTL(p.IdleTTLSeconds), fmtDur(left))
	case ondemand:
		return blue("Asleep"), "wakes on first access: Deploy, kubectl via 172.18.255.223, or chat to target-cluster-agent"
	}
	return red("Down"), "no target cluster"
}

func fmtTTL(s int) string {
	if s <= 0 {
		return "never"
	}
	return fmtDur(s)
}

func fmtDur(s int) string {
	d := time.Duration(s) * time.Second
	switch {
	case d >= time.Hour && d%time.Hour == 0:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d >= time.Hour:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	case d >= time.Minute && d%time.Minute == 0:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d >= time.Minute:
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%ds", s)
}

var statusCmd = &command{name: "status", summary: "platform overview: management cluster, target cluster state, idle timer",
	run: func(args []string) error {
		var cs clusterStatus
		var ts targetStatus
		var ps poolStatus
		csRaw, err := get("/api/v1/cluster/status", &cs)
		if err != nil {
			return err
		}
		tsRaw, err := get("/api/v1/cluster/target-status", &ts)
		if err != nil {
			return err
		}
		psRaw, err := get("/api/v1/cluster/pool-status", &ps)
		if err != nil {
			return err
		}
		if outputMode == "json" {
			return printJSON(map[string]json.RawMessage{"management": csRaw, "target": tsRaw, "pool": psRaw})
		}
		label, detail := targetState(&ps)
		fmt.Println(bold("Management cluster (cluster2)"))
		kv("nodes", fmt.Sprintf("%d/%d ready", cs.Nodes.Ready, cs.Nodes.Total),
			"pods", fmt.Sprintf("%d total · %d running · %d pending · %d failed", cs.Pods.Total, cs.Pods.Running, cs.Pods.Pending, cs.Pods.Failed),
			"VMs", fmt.Sprintf("%d total · %d running · %d stopped", cs.VMs.Total, cs.VMs.Running, cs.VMs.Stopped),
			"namespaces", fmt.Sprint(cs.Namespaces))
		section("Target cluster (target-cluster)")
		kv("state", label,
			"", dim(detail),
			"mode", ps.Mode,
			"phase", orDash(ts.ClusterPhase),
			"API answers", yesNo(ts.APIReady)+dim(" (hosted k3s server on .215; up even while asleep)"),
			"Istio", yesNo(ts.IstioReady),
			"operation", orDash(ts.Operation))
		if ps.LastError != "" {
			fmt.Println(red("last error: ") + ps.LastError)
		}
		return nil
	}}

var clustersCmd = &command{name: "clusters", summary: "list clusters with their machines and VMs",
	run: func(args []string) error {
		var cs clusterStatus
		var ts targetStatus
		var ps poolStatus
		if _, err := get("/api/v1/cluster/status", &cs); err != nil {
			return err
		}
		tsRaw, err := get("/api/v1/cluster/target-status", &ts)
		if err != nil {
			return err
		}
		psRaw, err := get("/api/v1/cluster/pool-status", &ps)
		if err != nil {
			return err
		}
		if outputMode == "json" {
			return printJSON(map[string]any{"management": cs, "target": json.RawMessage(tsRaw), "pool": json.RawMessage(psRaw)})
		}
		label, _ := targetState(&ps)
		t := newTable("CLUSTER", "ROLE", "STATE", "NODES", "DETAIL")
		t.add("cluster2", "management (Kind: CAPI, KubeVirt, Sympozium)", green("Up"), fmt.Sprintf("%d/%d", cs.Nodes.Ready, cs.Nodes.Total), fmt.Sprintf("%d VMs running", cs.VMs.Running))
		t.add("target-cluster", "workload (k3s on KubeVirt VMs, "+ps.Mode+")", label, fmt.Sprint(len(ts.VMIs)), "phase "+orDash(ts.ClusterPhase))
		t.print()
		if len(ts.Machines) > 0 {
			section("Machines")
			mt := newTable("NAME", "ROLE", "PHASE")
			for _, m := range ts.Machines {
				mt.add(m.Name, orDash(m.Role), statusColor(m.Phase))
			}
			mt.print()
		}
		if len(ts.VMIs) > 0 {
			section("VM instances")
			vt := newTable("NAME", "PHASE", "IP")
			for _, v := range ts.VMIs {
				vt.add(v.Name, statusColor(v.Phase), orDash(v.IP))
			}
			vt.print()
		}
		return nil
	}}

// printLogEntry renders one deploy-family log line.
func printLogEntry(e logEntry) {
	switch e.Type {
	case "step":
		fmt.Println(cyan(bold(e.Message)))
	case "success":
		fmt.Println(green("✓ " + e.Message))
	case "error":
		fmt.Println(red("✗ " + e.Message))
	case "warn":
		fmt.Println(yellow("! " + e.Message))
	case "done":
	default:
		fmt.Println("  " + e.Message)
	}
}

// followLog streams a deploy-family SSE endpoint to the terminal and fails
// when the run's final state is "failed".
func followLog(method, path string) error {
	final := ""
	err := stream(method, path, nil, func(data []byte) error {
		var e logEntry
		if err := json.Unmarshal(data, &e); err != nil {
			return nil
		}
		if outputMode == "json" {
			fmt.Println(string(data))
		} else {
			printLogEntry(e)
		}
		if e.Type == "done" {
			final = e.Message
			return errStreamDone
		}
		return nil
	})
	if err != nil {
		return err
	}
	switch final {
	case "failed":
		return fmt.Errorf("operation failed (see the log above)")
	case "":
		return nil
	}
	if outputMode != "json" {
		fmt.Println(dim("finished: " + final))
	}
	return nil
}

var clusterCmd = &command{name: "cluster", summary: "deploy, delete, follow and wait for the target cluster"}

func init() {
	clusterCmd.sub = []*command{
		{name: "deploy", summary: "deploy the target cluster (claims a warm standby when one is ready)",
			usage: "[--profile full|lite] [--image hosted|warm|noble|minimal] [--detach]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
				profile := fs.String("profile", "full", "full | lite")
				image := fs.String("image", "", "hosted | warm | noble | minimal (default: backend's default for its TARGET_CP_MODE)")
				detach := fs.Bool("detach", false, "start it and return; follow later with 'sovctl cluster logs'")
				if _, err := parseFlags(fs, args); err != nil {
					return err
				}
				if *profile != "full" && *profile != "lite" {
					return usageErr("--profile must be full or lite")
				}
				q := url.Values{"profile": {*profile}}
				if *image != "" {
					q.Set("image", *image)
				}
				path := "/api/v1/cluster/deploy?" + q.Encode()
				if !*detach {
					return followLog(http.MethodPost, path)
				}
				// The deploy runs server-side regardless of the client: read the
				// first line to know it started, then hang up.
				return stream(http.MethodPost, path, nil, func(data []byte) error {
					var e logEntry
					_ = json.Unmarshal(data, &e)
					printLogEntry(e)
					fmt.Println(dim("detached; follow with: sovctl cluster logs"))
					return errStreamDone
				})
			}},
		{name: "delete", summary: "delete the target cluster; it stays down (Stopped) until the next deploy", usage: "[-y]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("delete", flag.ContinueOnError)
				yes := fs.Bool("y", false, "do not ask for confirmation")
				if _, err := parseFlags(fs, args); err != nil {
					return err
				}
				if !confirm("Delete target-cluster? It stays down until you deploy again.", *yes) {
					return fmt.Errorf("aborted")
				}
				return followLog(http.MethodPost, "/api/v1/cluster/delete")
			}},
		{name: "logs", summary: "follow (or replay) the current deploy/delete/istio run",
			run: func(args []string) error { return followLog(http.MethodGet, "/api/v1/cluster/deploy/logs") }},
		{name: "istio", summary: "install Istio + nginx demo on the target cluster",
			run: func(args []string) error { return followLog(http.MethodPost, "/api/v1/cluster/istio") }},
		{name: "wait", summary: "wait until the target cluster is Up (e.g. after an on-demand wake)", usage: "[--timeout 3m]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("wait", flag.ContinueOnError)
				timeout := fs.Duration("timeout", 3*time.Minute, "give up after this long")
				if _, err := parseFlags(fs, args); err != nil {
					return err
				}
				deadline := time.Now().Add(*timeout)
				last := ""
				for {
					var ps poolStatus
					if _, err := get("/api/v1/cluster/pool-status", &ps); err != nil {
						return err
					}
					label, detail := targetState(&ps)
					if label != last {
						fmt.Printf("%s  %s  %s\n", dim(time.Now().Format("15:04:05")), label, dim(detail))
						last = label
					}
					if (ps.State == "warm" || ps.State == "claimed") && ps.ClusterReady && !ps.Deleting {
						return nil
					}
					if time.Now().After(deadline) {
						return fmt.Errorf("not up after %s", *timeout)
					}
					time.Sleep(2 * time.Second)
				}
			}},
	}
}

// parseTTL accepts a Go duration (15m, 1h30m, 90m) or never/off/0.
func parseTTL(s string) (int, error) {
	switch strings.ToLower(s) {
	case "never", "off", "0", "none":
		return 0, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		if n, aerr := parseInt(s); aerr == nil {
			d = time.Duration(n) * time.Minute // bare number = minutes, like the UI's custom field
		} else {
			return 0, fmt.Errorf("%q is not a duration (examples: 15m, 1h30m, 90, never)", s)
		}
	}
	if d < time.Minute || d > 24*time.Hour {
		return 0, fmt.Errorf("idle time must be between 1m and 24h, or never (got %s)", d)
	}
	if d%time.Second != 0 {
		return 0, fmt.Errorf("idle time must be whole seconds")
	}
	return int(d.Seconds()), nil
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil || fmt.Sprint(n) != s {
		return 0, fmt.Errorf("not an integer")
	}
	return n, nil
}

func printTimer(ps poolStatus) {
	src := map[string]string{"ui": "set from UI/CLI", "env": "POOL_IDLE_TTL", "default": "built-in default"}[ps.IdleTTLSource]
	pairs := []string{"auto-delete after", fmtTTL(ps.IdleTTLSeconds) + dim(" ("+src+")"), "mode", ps.Mode}
	if ps.Mode != "ondemand" {
		pairs = append(pairs, "", yellow("idle deletion only runs in on-demand mode (POOL_MODE=ondemand)"))
	} else if ps.State == "warm" || ps.State == "claimed" {
		pairs = append(pairs, "idle for", fmtDur(ps.IdleSeconds))
		if ps.IdleTTLSeconds > 0 {
			pairs = append(pairs, "deleted in", fmtDur(max(0, ps.IdleTTLSeconds-ps.IdleSeconds)))
		}
	} else {
		label, _ := targetState(&ps)
		pairs = append(pairs, "cluster", label)
	}
	kv(pairs...)
}

var timerCmd = &command{name: "timer", summary: "show or change the on-demand idle auto-delete timer",
	run: func(args []string) error {
		var ps poolStatus
		return show("/api/v1/cluster/pool-status", &ps, func() { printTimer(ps) })
	}}

func init() {
	timerCmd.sub = []*command{
		{name: "show", summary: "show the idle timer (same as 'sovctl timer')", run: timerCmd.run},
		{name: "set", summary: "set the idle time after which the cluster is deleted", usage: "<15m|1h30m|90|never> [-y]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("set", flag.ContinueOnError)
				yes := fs.Bool("y", false, "do not ask when this deletes the cluster right away")
				pos, err := parseFlags(fs, args)
				if err != nil {
					return err
				}
				if len(pos) != 1 {
					return usageErr("give exactly one duration")
				}
				secs, err := parseTTL(pos[0])
				if err != nil {
					return usageErr("%v", err)
				}
				var cur poolStatus
				if _, err := get("/api/v1/cluster/pool-status", &cur); err != nil {
					return err
				}
				up := cur.State == "warm" || cur.State == "claimed"
				if up && cur.Mode == "ondemand" && secs > 0 && secs <= cur.IdleSeconds {
					if !confirm(fmt.Sprintf("The cluster has been idle %s already; %s deletes it within ~5s. Continue?", fmtDur(cur.IdleSeconds), fmtTTL(secs)), *yes) {
						return fmt.Errorf("aborted")
					}
				}
				var ps poolStatus
				raw, err := fetch(http.MethodPut, "/api/v1/cluster/pool-settings", map[string]int{"idleTTLSeconds": secs}, &ps)
				if err != nil {
					return err
				}
				if outputMode == "json" {
					return printRawJSON(raw)
				}
				fmt.Println(green("✓ ") + "idle timer set to " + fmtTTL(ps.IdleTTLSeconds))
				printTimer(ps)
				return nil
			}},
		{name: "reset", summary: "drop the UI/CLI setting and go back to POOL_IDLE_TTL",
			run: func(args []string) error {
				var ps poolStatus
				raw, err := fetch(http.MethodDelete, "/api/v1/cluster/pool-settings", nil, &ps)
				if err != nil {
					return err
				}
				if outputMode == "json" {
					return printRawJSON(raw)
				}
				fmt.Println(green("✓ ") + "idle timer reset")
				printTimer(ps)
				return nil
			}},
	}
}

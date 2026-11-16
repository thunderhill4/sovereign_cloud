package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

var meshCmd = &command{name: "mesh", summary: "Istio ambient mesh: topology, gateways, waypoint, multicluster, probes"}

type probeEvent struct {
	Type    string `json:"type"`
	Seq     int    `json:"seq"`
	Version string `json:"version"`
	Code    int    `json:"code"`
	V1      int    `json:"v1"`
	V2      int    `json:"v2"`
	Message string `json:"message"`
}

// followProbe streams an Istio action (probe / failover) to the terminal.
func followProbe(path string, body any, label1, label2 string) error {
	return stream(http.MethodPost, path, body, func(data []byte) error {
		if outputMode == "json" {
			fmt.Println(string(data))
			return nil
		}
		var e probeEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return nil
		}
		switch e.Type {
		case "step":
			fmt.Println(cyan(bold(e.Message)))
		case "result":
			v := e.Version
			if v == "" {
				v = red("no answer")
			}
			code := fmt.Sprint(e.Code)
			if e.Code >= 200 && e.Code < 300 {
				code = green(code)
			} else {
				code = red(code)
			}
			fmt.Printf("  #%-3d %s  %-10s %s=%d %s=%d\n", e.Seq, code, v, label1, e.V1, label2, e.V2)
		case "summary":
			fmt.Println(green(bold(fmt.Sprintf("%s=%d %s=%d", label1, e.V1, label2, e.V2))) + "  " + e.Message)
		case "error":
			fmt.Println(red("✗ " + e.Message))
		}
		return nil
	})
}

func init() {
	meshCmd.sub = []*command{
		{name: "topology", summary: "service mesh graph: nodes and connections",
			run: func(args []string) error {
				var g struct {
					Nodes []struct {
						ID, Label, Type, Status, Cluster string
						Meta                             map[string]string
					}
					Edges []struct {
						From, To, Label, Protocol string
						Dashed                    bool
					}
					Source string
				}
				return show("/api/v1/mesh/topology", &g, func() {
					t := newTable("NODE", "TYPE", "CLUSTER", "STATUS", "ID")
					for _, n := range g.Nodes {
						t.add(n.Label, n.Type, orDash(n.Cluster), statusColor(n.Status), dim(n.ID))
					}
					t.print()
					section("Connections")
					e := newTable("FROM", "TO", "PROTOCOL", "LABEL")
					for _, x := range g.Edges {
						to := x.To
						if x.Dashed {
							to += dim(" (planned/idle)")
						}
						e.add(x.From, to, x.Protocol, x.Label)
					}
					e.print()
					notes(g.Source, nil)
				})
			}},
		{name: "overview", summary: "per-cluster Istio health: istiod, ztunnel, CNI, root CA",
			run: func(args []string) error {
				var o struct {
					Clusters []struct {
						Name, Context, IstiodVersion, Ztunnel, CNI, Network, RootCaSha, Error string
						Reachable, IstiodReady, ZtunnelReady, CniReady                        bool
						AmbientNamespaces                                                     int
					}
					SharedRootCa bool
					Source       string
					Notes        []string
				}
				return show("/api/v1/istio/overview", &o, func() {
					t := newTable("CLUSTER", "REACHABLE", "ISTIOD", "ZTUNNEL", "CNI", "NETWORK", "AMBIENT NS", "ROOT CA")
					for _, c := range o.Clusters {
						istiod := orDash(c.IstiodVersion)
						if c.IstiodReady {
							istiod = green(istiod)
						} else {
							istiod = red(istiod)
						}
						zt, cni := orDash(c.Ztunnel), orDash(c.CNI)
						if !c.ZtunnelReady {
							zt = red(zt)
						}
						if !c.CniReady {
							cni = red(cni)
						}
						sha := c.RootCaSha
						if len(sha) > 12 {
							sha = sha[:12]
						}
						t.add(c.Name, yesNo(c.Reachable), istiod, zt, cni, orDash(c.Network), fmt.Sprint(c.AmbientNamespaces), dim(orDash(sha)))
						if c.Error != "" {
							t.add("", red(c.Error), "", "", "", "", "", "")
						}
					}
					t.print()
					fmt.Println(dim("shared root CA: ") + yesNo(o.SharedRootCa))
					notes(o.Source, o.Notes)
				})
			}},
		{name: "gateways", summary: "Gateway API gateways, routes and listener sets",
			run: func(args []string) error {
				var g struct {
					Gateways []struct {
						Name, Namespace, Cluster, Class, Address, Act string
						Programmed                                    bool
						Listeners                                     []string
					}
					Routes []struct {
						Name, Namespace, Cluster, Parent, Reason string
						Hostnames, Backends                      []string
						Accepted                                 bool
					}
					ListenerSets []struct {
						Name, Namespace, Cluster, Parent string
						Accepted                         bool
					}
					Source string
					Notes  []string
				}
				return show("/api/v1/istio/gateways", &g, func() {
					t := newTable("CLUSTER", "NAMESPACE", "GATEWAY", "CLASS", "ADDRESS", "PROGRAMMED", "LISTENERS", "ACT")
					for _, x := range g.Gateways {
						t.add(x.Cluster, x.Namespace, x.Name, x.Class, orDash(x.Address), yesNo(x.Programmed), join(x.Listeners), orDash(x.Act))
					}
					t.print()
					section("Routes")
					r := newTable("CLUSTER", "NAMESPACE", "ROUTE", "PARENT", "HOSTNAMES", "BACKENDS", "ACCEPTED")
					for _, x := range g.Routes {
						acc := yesNo(x.Accepted)
						if x.Reason != "" && !x.Accepted {
							acc += " " + dim(x.Reason)
						}
						r.add(x.Cluster, x.Namespace, x.Name, orDash(x.Parent), join(x.Hostnames), join(x.Backends), acc)
					}
					r.print()
					if len(g.ListenerSets) > 0 {
						section("Listener sets")
						l := newTable("CLUSTER", "NAMESPACE", "NAME", "PARENT", "ACCEPTED")
						for _, x := range g.ListenerSets {
							l.add(x.Cluster, x.Namespace, x.Name, orDash(x.Parent), yesNo(x.Accepted))
						}
						l.print()
					}
					notes(g.Source, g.Notes)
				})
			}},
		{name: "waypoint", summary: "L7 waypoint enrollment and authorization policies",
			run: func(args []string) error {
				var w struct {
					EnrolledNamespaces []string
					WaypointProgrammed bool
					WaypointName       string
					Policies           []struct {
						Name, Namespace, Action, TargetKind, TargetName string
						Principals, Methods, Paths                      []string
					}
					Pods []struct {
						Name       string
						Containers []string
						Ready      bool
					}
					SidecarFree bool
					Source      string
					Notes       []string
				}
				return show("/api/v1/istio/waypoint", &w, func() {
					kv("waypoint", orDash(w.WaypointName), "programmed", yesNo(w.WaypointProgrammed),
						"enrolled namespaces", join(w.EnrolledNamespaces), "sidecar-free", yesNo(w.SidecarFree))
					section("Authorization policies")
					t := newTable("NAMESPACE", "NAME", "ACTION", "TARGET", "PRINCIPALS", "METHODS", "PATHS")
					for _, p := range w.Policies {
						target := "-"
						if p.TargetKind != "" {
							target = p.TargetKind + "/" + p.TargetName
						}
						t.add(p.Namespace, p.Name, p.Action, target, join(p.Principals), join(p.Methods), join(p.Paths))
					}
					t.print()
					section("Pods")
					pt := newTable("POD", "READY", "CONTAINERS")
					for _, p := range w.Pods {
						pt.add(p.Name, yesNo(p.Ready), join(p.Containers))
					}
					pt.print()
					notes(w.Source, w.Notes)
				})
			}},
		{name: "multicluster", summary: "east-west gateways, remote secrets, federated services",
			run: func(args []string) error {
				var m struct {
					Clusters []struct {
						Name, Network, EastWestIp, RemoteSecret string
						EastWestProgrammed, Reachable           bool
						GlobalServices, LocalEndpoints          int
					}
					SharedRootCa, Federated bool
					Source                  string
					Notes                   []string
				}
				return show("/api/v1/istio/multicluster", &m, func() {
					t := newTable("CLUSTER", "REACHABLE", "NETWORK", "EAST-WEST IP", "EW PROGRAMMED", "REMOTE SECRET", "GLOBAL SVCS", "LOCAL EPS")
					for _, c := range m.Clusters {
						t.add(c.Name, yesNo(c.Reachable), orDash(c.Network), orDash(c.EastWestIp), yesNo(c.EastWestProgrammed), orDash(c.RemoteSecret), fmt.Sprint(c.GlobalServices), fmt.Sprint(c.LocalEndpoints))
					}
					t.print()
					kv("shared root CA", yesNo(m.SharedRootCa), "federated", yesNo(m.Federated))
					notes(m.Source, m.Notes)
				})
			}},
		{name: "ai-gateway", summary: "agentgateway (AI gateway) status and models",
			run: func(args []string) error {
				var a struct {
					Present, ClassExists, Programmed bool
					Address, Backend                 string
					Models                           []string
					Source                           string
					Notes                            []string
				}
				return show("/api/v1/istio/ai-gateway", &a, func() {
					kv("present", yesNo(a.Present), "class exists", yesNo(a.ClassExists), "programmed", yesNo(a.Programmed),
						"address", orDash(a.Address), "backend", orDash(a.Backend), "models", join(a.Models))
					notes(a.Source, a.Notes)
				})
			}},
		{name: "observability", summary: "Prometheus and Kiali reachability",
			run: func(args []string) error {
				var o struct {
					PrometheusUp, KialiUp bool
					KialiUrl              string
					RequestSeries         int
					Source                string
					Notes                 []string
				}
				return show("/api/v1/istio/observability", &o, func() {
					kv("Prometheus", yesNo(o.PrometheusUp), "Kiali", yesNo(o.KialiUp), "Kiali URL", orDash(o.KialiUrl), "request series", fmt.Sprint(o.RequestSeries))
					notes(o.Source, o.Notes)
				})
			}},
		{name: "probe", summary: "Act 1: send requests through the north-south gateway, count v1/v2", usage: "[--count 100] [--internal]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("probe", flag.ContinueOnError)
				count := fs.Int("count", 100, "requests to send (max 200)")
				internal := fs.Bool("internal", false, "send x-demo-user: internal (routes to v2)")
				if _, err := parseFlags(fs, args); err != nil {
					return err
				}
				return followProbe("/api/v1/istio/probe", map[string]any{"count": *count, "internal": *internal}, "v1", "v2")
			}},
		{name: "identity-probe", summary: "Act 2: trusted vs untrusted client through the waypoint",
			run: func(args []string) error {
				var r struct {
					Results []struct {
						Identity, Path string
						Code           int
						Allowed        bool
					}
					Xfcc   string
					Source string
					Notes  []string
				}
				raw, err := fetch(http.MethodPost, "/api/v1/istio/identity-probe", nil, &r)
				if err != nil {
					return err
				}
				if outputMode == "json" {
					return printRawJSON(raw)
				}
				t := newTable("IDENTITY", "PATH", "CODE", "ALLOWED")
				for _, x := range r.Results {
					allowed := green("allowed")
					if !x.Allowed {
						allowed = red("denied")
					}
					t.add(x.Identity, x.Path, fmt.Sprint(x.Code), allowed)
				}
				t.print()
				if r.Xfcc != "" {
					fmt.Println(dim("x-forwarded-client-cert: ") + r.Xfcc)
				}
				notes(r.Source, r.Notes)
				return nil
			}},
		{name: "failover", summary: "Act 3: scale cluster1's echo to 0 and watch cross-cluster failover (auto-restored)", usage: "[-y]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("failover", flag.ContinueOnError)
				yes := fs.Bool("y", false, "do not ask for confirmation")
				if _, err := parseFlags(fs, args); err != nil {
					return err
				}
				if !confirm("This scales demo-apps/echo-v1 and echo-v2 on cluster1 to 0 for ~1 min (restored afterwards). Continue?", *yes) {
					return fmt.Errorf("aborted")
				}
				return followProbe("/api/v1/istio/failover", nil, "ok", "failed")
			}},
	}
}

var connectionsCmd = &command{name: "connections", summary: "cross-cluster connection probes (cluster1 ↔ target-cluster)", usage: "[--watch [--interval 3s]]",
	run: func(args []string) error {
		fs := flag.NewFlagSet("connections", flag.ContinueOnError)
		watch := fs.Bool("watch", false, "refresh until Ctrl+C")
		fs.BoolVar(watch, "w", false, "refresh until Ctrl+C")
		interval := fs.Duration("interval", 3*time.Second, "refresh interval with --watch")
		if _, err := parseFlags(fs, args); err != nil {
			return err
		}
		once := func() error {
			var s struct {
				Probes []struct {
					ID, From, To, URL, Error, Timestamp, Direction string
					StatusCode, LatencyMs                          int
				}
				HttpbinAlive, NginxAlive, FailoverActive bool
				Timestamp                                string
			}
			return show("/api/v1/cross-cluster/probe", &s, func() {
				t := newTable("FROM", "TO", "DIRECTION", "STATUS", "LATENCY", "URL")
				for _, p := range s.Probes {
					st := green(fmt.Sprint(p.StatusCode))
					if p.StatusCode == 0 || p.StatusCode >= 400 {
						st = red(orDash(p.Error))
						if p.StatusCode != 0 {
							st = red(fmt.Sprint(p.StatusCode))
						}
					}
					t.add(p.From, p.To, p.Direction, st, fmt.Sprintf("%dms", p.LatencyMs), dim(p.URL))
				}
				t.print()
				kv("httpbin (cluster1)", yesNo(s.HttpbinAlive), "nginx (target-cluster)", yesNo(s.NginxAlive), "failover active", yesNo(s.FailoverActive))
			})
		}
		if !*watch {
			return once()
		}
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		for {
			fmt.Print("\x1b[H\x1b[2J")
			fmt.Println(dim(time.Now().Format("15:04:05") + "  every " + interval.String() + " — Ctrl+C to stop"))
			if err := once(); err != nil {
				fmt.Println(red(err.Error()))
			}
			select {
			case <-sig:
				return nil
			case <-time.After(*interval):
			}
		}
	}}

func init() {
	connectionsCmd.sub = []*command{
		{name: "scale", summary: "scale a demo deployment to break/restore a connection", usage: "<cluster1|target-cluster> <namespace>/<deployment> <replicas 0-10> [-y]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("scale", flag.ContinueOnError)
				yes := fs.Bool("y", false, "do not ask for confirmation")
				pos, err := parseFlags(fs, args)
				if err != nil {
					return err
				}
				if len(pos) != 3 || !strings.Contains(pos[1], "/") {
					return usageErr("give <cluster> <namespace>/<deployment> <replicas>")
				}
				if pos[0] != "cluster1" && pos[0] != "target-cluster" {
					return usageErr("cluster must be cluster1 or target-cluster")
				}
				n, err := parseInt(pos[2])
				if err != nil || n < 0 || n > 10 {
					return usageErr("replicas must be 0-10")
				}
				ns, dep, _ := strings.Cut(pos[1], "/")
				if !confirm(fmt.Sprintf("Scale %s/%s on %s to %d?", ns, dep, pos[0], n), *yes) {
					return fmt.Errorf("aborted")
				}
				var r struct {
					OK     bool   `json:"ok"`
					Output string `json:"output"`
				}
				raw, err := fetch(http.MethodPost, "/api/v1/cross-cluster/scale", map[string]any{"deployment": dep, "namespace": ns, "cluster": pos[0], "replicas": n}, &r)
				if err != nil {
					return err
				}
				if outputMode == "json" {
					return printRawJSON(raw)
				}
				fmt.Println(green("✓ ") + strings.TrimSpace(r.Output))
				return nil
			}},
	}
}

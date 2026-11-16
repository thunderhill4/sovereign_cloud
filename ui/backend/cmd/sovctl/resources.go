package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// nsQuery parses -n/--namespace (and -A for all) and returns "?namespace=x" or "".
func nsQuery(name string, args []string) (string, []string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	ns := fs.String("n", "", "namespace (default: all)")
	fs.StringVar(ns, "namespace", "", "namespace (default: all)")
	fs.Bool("A", false, "all namespaces (the default)")
	pos, err := parseFlags(fs, args)
	if err != nil {
		return "", nil, err
	}
	if *ns == "" || *ns == "all" {
		return "", pos, nil
	}
	return "?namespace=" + url.QueryEscape(*ns), pos, nil
}

var nodesCmd = &command{name: "nodes", summary: "management cluster nodes",
	run: func(args []string) error {
		var nodes []struct {
			Name, Status, Roles, CPU, Memory, Age, KubeletVersion string
		}
		return show("/api/v1/nodes", &nodes, func() {
			t := newTable("NAME", "STATUS", "ROLES", "CPU", "MEMORY", "AGE", "VERSION")
			for _, n := range nodes {
				t.add(n.Name, statusColor(n.Status), n.Roles, n.CPU, n.Memory, n.Age, n.KubeletVersion)
			}
			t.print()
		})
	}}

var podsCmd = &command{name: "pods", summary: "pods on the management cluster", usage: "[-n namespace]",
	run: func(args []string) error {
		q, _, err := nsQuery("pods", args)
		if err != nil {
			return err
		}
		var pods []struct {
			Name, Namespace, Status, Node, Age    string
			Restarts, Containers, ReadyContainers int
		}
		return show("/api/v1/pods"+q, &pods, func() {
			t := newTable("NAMESPACE", "NAME", "READY", "STATUS", "RESTARTS", "NODE", "AGE")
			for _, p := range pods {
				t.add(p.Namespace, p.Name, fmt.Sprintf("%d/%d", p.ReadyContainers, p.Containers), statusColor(p.Status), fmt.Sprint(p.Restarts), p.Node, p.Age)
			}
			t.print()
		})
	}}

var namespacesCmd = &command{name: "namespaces", summary: "namespaces on the management cluster",
	run: func(args []string) error {
		var nss []struct{ Name, Status string }
		return show("/api/v1/namespaces", &nss, func() {
			t := newTable("NAME", "STATUS")
			for _, n := range nss {
				t.add(n.Name, statusColor(n.Status))
			}
			t.print()
		})
	}}

var eventsCmd = &command{name: "events", summary: "recent Kubernetes events", usage: "[-n namespace] [--warnings]",
	run: func(args []string) error {
		warnOnly := false
		var rest []string
		for _, a := range args {
			if a == "--warnings" || a == "-w" {
				warnOnly = true
			} else {
				rest = append(rest, a)
			}
		}
		q, _, err := nsQuery("events", rest)
		if err != nil {
			return err
		}
		var evs []struct {
			Type, Reason, Message, InvolvedObject, Namespace, Age string
			Count                                                 int
		}
		return show("/api/v1/events"+q, &evs, func() {
			t := newTable("AGE", "TYPE", "NAMESPACE", "OBJECT", "REASON", "COUNT", "MESSAGE")
			for _, e := range evs {
				if warnOnly && e.Type != "Warning" {
					continue
				}
				typ := e.Type
				if typ == "Warning" {
					typ = yellow(typ)
				}
				t.add(e.Age, typ, e.Namespace, e.InvolvedObject, e.Reason, fmt.Sprint(e.Count), e.Message)
			}
			t.print()
		})
	}}

type vm struct {
	Name, Namespace, Status, Memory, Node, Age, IPAddress string
	CPU                                                   int
}

var vmsCmd = &command{name: "vms", summary: "KubeVirt virtual machines", usage: "[-n namespace]",
	run: func(args []string) error {
		q, _, err := nsQuery("vms", args)
		if err != nil {
			return err
		}
		var vms []vm
		return show("/api/v1/virtualmachines"+q, &vms, func() {
			t := newTable("NAMESPACE", "NAME", "STATUS", "CPU", "MEMORY", "IP", "NODE", "AGE")
			for _, v := range vms {
				t.add(v.Namespace, v.Name, statusColor(v.Status), fmt.Sprint(v.CPU), v.Memory, orDash(v.IPAddress), orDash(v.Node), v.Age)
			}
			t.print()
		})
	}}

var vmCmd = &command{name: "vm", summary: "one virtual machine", usage: "<namespace>/<name>",
	run: func(args []string) error {
		if len(args) != 1 || !strings.Contains(args[0], "/") {
			return usageErr("give <namespace>/<name>")
		}
		ns, name, _ := strings.Cut(args[0], "/")
		var v vm
		return show("/api/v1/virtualmachines/"+url.PathEscape(ns)+"/"+url.PathEscape(name), &v, func() {
			kv("name", v.Name, "namespace", v.Namespace, "status", statusColor(v.Status), "CPU", fmt.Sprint(v.CPU),
				"memory", v.Memory, "IP", orDash(v.IPAddress), "node", orDash(v.Node), "age", v.Age)
		})
	}}

var imagesCmd = &command{name: "images", summary: "golden images (CDI DataVolumes)", usage: "[-n namespace]",
	run: func(args []string) error {
		q, _, err := nsQuery("images", args)
		if err != nil {
			return err
		}
		var dvs []struct {
			Name, Namespace, Phase, Progress, SourceType, Size, ClaimName, Age, ClusterName string
		}
		return show("/api/v1/images"+q, &dvs, func() {
			t := newTable("NAMESPACE", "NAME", "PHASE", "PROGRESS", "SOURCE", "SIZE", "CLUSTER", "AGE")
			for _, d := range dvs {
				t.add(d.Namespace, d.Name, statusColor(d.Phase), d.Progress, d.SourceType, d.Size, orDash(d.ClusterName), d.Age)
			}
			t.print()
		})
	}}

func humanBytes(n int64) string {
	if n <= 0 {
		return "-" // the registry API did not report a size
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

var registryCmd = &command{name: "registry", summary: "local container registry: repositories, tags, delete"}

func init() {
	ls := &command{name: "ls", summary: "list repositories and tags",
		run: func(args []string) error {
			var repos []struct {
				Name    string   `json:"name"`
				Tags    []string `json:"tags"`
				TagInfo []struct {
					Tag, Digest, Created string
					Size                 int64
				} `json:"tagInfo"`
			}
			return show("/api/v1/registry/images", &repos, func() {
				t := newTable("REPOSITORY", "TAG", "SIZE", "CREATED", "DIGEST")
				for _, r := range repos {
					if len(r.TagInfo) == 0 {
						for _, tag := range r.Tags {
							t.add(r.Name, tag, "-", "-", "-")
						}
						continue
					}
					for _, ti := range r.TagInfo {
						dg := ti.Digest
						if len(dg) > 19 {
							dg = dg[:19]
						}
						t.add(r.Name, ti.Tag, humanBytes(ti.Size), orDash(ti.Created), dim(dg))
					}
				}
				t.print()
			})
		}}
	registryCmd.run = ls.run
	registryCmd.sub = []*command{
		ls,
		{name: "config", summary: "registry address and connection status",
			run: func(args []string) error {
				var c map[string]any
				return show("/api/v1/registry/config", &c, func() {
					var pairs []string
					for _, k := range []string{"url", "name", "status"} {
						if v, ok := c[k]; ok {
							s := fmt.Sprint(v)
							if k == "status" {
								s = statusColor(s)
							}
							pairs = append(pairs, k, s)
						}
					}
					for k, v := range c {
						if k != "url" && k != "name" && k != "status" {
							pairs = append(pairs, k, fmt.Sprint(v))
						}
					}
					kv(pairs...)
				})
			}},
		{name: "rm", summary: "delete an image tag from the registry", usage: "<name>:<tag> [-y]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("rm", flag.ContinueOnError)
				yes := fs.Bool("y", false, "do not ask for confirmation")
				pos, err := parseFlags(fs, args)
				if err != nil {
					return err
				}
				if len(pos) != 1 || strings.Count(pos[0], ":") != 1 {
					return usageErr("give exactly one <name>:<tag>")
				}
				if !confirm("Delete "+pos[0]+" from the registry?", *yes) {
					return fmt.Errorf("aborted")
				}
				name, tag, _ := strings.Cut(pos[0], ":")
				if _, err := call(http.MethodDelete, "/api/v1/registry/images/"+name+":"+url.PathEscape(tag), nil); err != nil {
					return err
				}
				fmt.Println(green("✓ ") + "deleted " + pos[0])
				return nil
			}},
	}
}

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var securityCmd = &command{name: "security", summary: "IaC and registry security scans (security agent)"}

// securityErr explains the backend's 502 when the security agent is not running.
func securityErr(err error) error {
	var ae *apiError
	if errors.As(err, &ae) && ae.Status == http.StatusBadGateway {
		return fmt.Errorf("%w; the security agent is not running (start the UI with RUN_SECURITY_AGENT=1 make ui)", err)
	}
	return err
}

type finding struct {
	ID          string `json:"id"`
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Remediation string `json:"remediation"`
	Resource    string `json:"resource"`
	Filename    string `json:"filename"`
	Line        int    `json:"line"`
	LLMAnalysis string `json:"llm_analysis"`
}

var severityRank = map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3, "info": 4}

func severity(s string) string {
	switch strings.ToLower(s) {
	case "critical", "high":
		return red(s)
	case "medium":
		return yellow(s)
	}
	return s
}

func printFindings(fs []finding, verbose bool) {
	sort.SliceStable(fs, func(i, j int) bool { return severityRank[fs[i].Severity] < severityRank[fs[j].Severity] })
	t := newTable("SEVERITY", "RULE", "WHERE", "RESOURCE", "TITLE")
	counts := map[string]int{}
	for _, f := range fs {
		counts[f.Severity]++
		where := f.Filename
		if f.Line > 0 {
			where = fmt.Sprintf("%s:%d", f.Filename, f.Line)
		}
		t.add(severity(f.Severity), f.RuleID, orDash(where), orDash(f.Resource), f.Title)
	}
	t.print()
	if verbose {
		for _, f := range fs {
			fmt.Printf("\n%s %s\n  fix: %s\n", severity(f.Severity), bold(f.Title), f.Remediation)
			if f.LLMAnalysis != "" {
				fmt.Println("  AI: " + f.LLMAnalysis)
			}
		}
	}
	var parts []string
	for _, s := range []string{"critical", "high", "medium", "low", "info"} {
		if counts[s] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[s], s))
		}
	}
	if len(parts) > 0 {
		fmt.Println(dim("findings: ") + strings.Join(parts, ", "))
	}
}

// Long timeout: an LLM-enriched scan calls the model per finding.
var scanClientTimeout = 10 * time.Minute

func init() {
	securityCmd.sub = []*command{
		{name: "rules", summary: "list the scanner's policy rules", usage: "[--tool terraform|kubernetes|kubevirt|ansible]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("rules", flag.ContinueOnError)
				tool := fs.String("tool", "", "only rules for this tool")
				if _, err := parseFlags(fs, args); err != nil {
					return err
				}
				var r struct {
					Rules []struct{ ID, Title, Policy, Tool string } `json:"rules"`
					Total int                                        `json:"total"`
				}
				return securityErr(show("/api/v1/security/rules", &r, func() {
					t := newTable("ID", "TOOL", "POLICY", "TITLE")
					for _, x := range r.Rules {
						if *tool == "" || x.Tool == *tool {
							t.add(x.ID, x.Tool, x.Policy, x.Title)
						}
					}
					t.print()
				}))
			}},
		{name: "scan", summary: "scan IaC files (Terraform, Kubernetes, KubeVirt, Ansible)", usage: "<file|dir>... --tool terraform|kubernetes|kubevirt|ansible [--llm] [-v]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("scan", flag.ContinueOnError)
				tool := fs.String("tool", "", "terraform | kubernetes | kubevirt | ansible (required)")
				llm := fs.Bool("llm", false, "enrich findings with the local LLM (slow)")
				verbose := fs.Bool("v", false, "print remediation for each finding")
				pos, err := parseFlags(fs, args)
				if err != nil {
					return err
				}
				switch *tool {
				case "terraform", "kubernetes", "kubevirt", "ansible":
				default:
					return usageErr("--tool must be terraform, kubernetes, kubevirt or ansible")
				}
				if len(pos) == 0 {
					return usageErr("give at least one file or directory")
				}
				type file struct {
					Filename string `json:"filename"`
					Content  string `json:"content"`
				}
				var files []file
				for _, p := range pos {
					err := filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
						if err != nil || d.IsDir() {
							return err
						}
						switch strings.ToLower(filepath.Ext(path)) {
						case ".tf", ".yaml", ".yml", ".json", ".hcl":
						default:
							if path != p {
								return nil // in a directory, only IaC-looking files
							}
						}
						b, err := os.ReadFile(path)
						if err != nil {
							return err
						}
						files = append(files, file{Filename: path, Content: string(b)})
						return nil
					})
					if err != nil {
						return err
					}
				}
				if len(files) == 0 {
					return fmt.Errorf("no .tf/.yaml/.yml/.json/.hcl files found")
				}
				httpClient.Timeout = scanClientTimeout
				var results []struct {
					Tool       string    `json:"tool"`
					Filename   string    `json:"filename"`
					Findings   []finding `json:"findings"`
					DurationMs int       `json:"duration_ms"`
				}
				raw, err := fetch(http.MethodPost, "/api/v1/security/scan", map[string]any{"files": files, "tool": *tool, "use_llm": *llm}, &results)
				if err != nil {
					return securityErr(err)
				}
				if outputMode == "json" {
					return printRawJSON(raw)
				}
				var all []finding
				for _, r := range results {
					all = append(all, r.Findings...)
				}
				fmt.Println(dim(fmt.Sprintf("scanned %d file(s) with %s", len(files), *tool)))
				printFindings(all, *verbose)
				return nil
			}},
		{name: "scan-registry", summary: "scan a container registry's images", usage: "<registry-url> [--insecure] [--username U --password-stdin] [-v]",
			run: func(args []string) error {
				fs := flag.NewFlagSet("scan-registry", flag.ContinueOnError)
				insecure := fs.Bool("insecure", false, "plain HTTP / skip TLS verification")
				user := fs.String("username", "", "registry username")
				pwStdin := fs.Bool("password-stdin", false, "read the registry password from stdin")
				verbose := fs.Bool("v", false, "print remediation for each finding")
				pos, err := parseFlags(fs, args)
				if err != nil {
					return err
				}
				if len(pos) != 1 {
					return usageErr("give exactly one registry URL")
				}
				body := map[string]any{"registry_url": pos[0], "insecure": *insecure}
				if *user != "" {
					body["username"] = *user
				}
				if *pwStdin {
					pw, _ := bufio.NewReader(os.Stdin).ReadString('\n')
					body["password"] = strings.TrimRight(pw, "\r\n")
				}
				httpClient.Timeout = scanClientTimeout
				var r struct {
					RegistryURL   string    `json:"registry_url"`
					Findings      []finding `json:"findings"`
					ImagesScanned int       `json:"images_scanned"`
					DurationMs    int       `json:"duration_ms"`
				}
				raw, err := fetch(http.MethodPost, "/api/v1/security/scan/registry", body, &r)
				if err != nil {
					return securityErr(err)
				}
				if outputMode == "json" {
					return printRawJSON(raw)
				}
				fmt.Println(dim(fmt.Sprintf("%s: %d image(s) scanned in %dms", r.RegistryURL, r.ImagesScanned, r.DurationMs)))
				printFindings(r.Findings, *verbose)
				return nil
			}},
	}
}

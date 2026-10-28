package handlers

import (
	"net/http"
	"os"
	"sort"
	"strings"
)

// GET /api/v1/config: the backend's effective, env-driven settings, for the
// sovctl CLI ("sovctl config", "sovctl ai config"). Read-only. Secrets are
// reported only as set/unset, never their values.

type ConfigImage struct {
	Key     string `json:"key"`
	Image   string `json:"image"`
	Label   string `json:"label"`
	Allowed bool   `json:"allowed"` // deployable in the current TARGET_CP_MODE
	Default bool   `json:"default"`
}

type ConfigProfile struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type ConfigAgent struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Default bool   `json:"default"`
}

type ConfigAI struct {
	Namespace       string        `json:"namespace"`
	DefaultAgent    string        `json:"defaultAgent"`
	Agents          []ConfigAgent `json:"agents"` // agents with a URL configured by env
	APITokenSet     bool          `json:"apiTokenSet"`
	ChatTimeoutSecs int           `json:"chatTimeoutSeconds"`
	ConsolePort     string        `json:"consolePort"`
	ConsoleUpstream string        `json:"consoleUpstream"`
	GuardEnabled    bool          `json:"guardEnabled"`
	GuardURL        string        `json:"guardURL"`
	GuardThreshold  float64       `json:"guardThreshold"`
	GuardKeySet     bool          `json:"guardKeySet"`
}

type ConfigPool struct {
	Enabled         bool   `json:"enabled"`
	Mode            string `json:"mode"` // standby | ondemand
	IdleTTLSeconds  int    `json:"idleTTLSeconds"`
	IdleTTLSource   string `json:"idleTTLSource"`
	StandbyManifest string `json:"standbyManifest"`
	FrontDoorAddr   string `json:"frontDoorAddr"`
}

type Config struct {
	TargetCPMode     string          `json:"targetCPMode"` // hosted | vm
	Pool             ConfigPool      `json:"pool"`
	Images           []ConfigImage   `json:"images"`
	Profiles         []ConfigProfile `json:"profiles"`
	RegistryURL      string          `json:"registryURL"`
	RegistryAlias    string          `json:"registryAlias"`
	SecurityAgentURL string          `json:"securityAgentURL"`
	RepoDir          string          `json:"repoDir"`
	AI               ConfigAI        `json:"ai"`
}

func HandleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, currentConfig())
}

func currentConfig() Config {
	ttl, src := poolIdleTTLWithSource()
	manifest := os.Getenv("POOL_STANDBY_MANIFEST")
	if manifest == "" {
		manifest = defaultStandbyManifest()
	}
	c := Config{
		TargetCPMode: cpMode(),
		Pool: ConfigPool{
			Enabled:         poolEnabled(),
			Mode:            poolMode(),
			IdleTTLSeconds:  int(ttl.Seconds()),
			IdleTTLSource:   src,
			StandbyManifest: manifest,
			FrontDoorAddr:   frontDoorAddr(),
		},
		RegistryURL:      registryURL,
		RegistryAlias:    registryAlias,
		SecurityAgentURL: envOrDefault("SECURITY_AGENT_URL", "http://security-agent.kubeui.svc.cluster.local:8082"),
		RepoDir:          claudeDir(),
		AI: ConfigAI{
			Namespace:       sympoziumNamespace(),
			DefaultAgent:    getDefaultAgent(),
			APITokenSet:     os.Getenv("SYMPOZIUM_API_TOKEN") != "",
			ChatTimeoutSecs: int(aiChatTimeout().Seconds()),
			ConsolePort:     envOrDefault("SYMPOZIUM_CONSOLE_PORT", "8081"),
			ConsoleUpstream: envOrDefault("SYMPOZIUM_DASHBOARD_URL", "http://172.18.255.212:8080"),
			GuardEnabled:    os.Getenv("LAYA_URL") != "",
			GuardURL:        os.Getenv("LAYA_URL"),
			GuardThreshold:  guardThreshold(),
			GuardKeySet:     os.Getenv("LAYA_API_KEY") != "",
		},
	}

	def := defaultImageKey()
	for k, v := range imageVariants {
		c.Images = append(c.Images, ConfigImage{Key: k, Image: v.Image, Label: v.Label, Allowed: imageAllowed(cpMode(), k), Default: k == def})
	}
	sort.Slice(c.Images, func(i, j int) bool { return c.Images[i].Key < c.Images[j].Key })
	for k, v := range profileSpecs {
		c.Profiles = append(c.Profiles, ConfigProfile{Key: k, Label: v.Label})
	}
	sort.Slice(c.Profiles, func(i, j int) bool { return c.Profiles[i].Key < c.Profiles[j].Key })

	c.AI.Agents = configuredAgents(os.Environ(), getDefaultAgent())
	return c
}

// configuredAgents lists the default agent plus every agent with a
// SYMPOZIUM_AGENT_URL_<NAME> override, resolved the same way chat resolves them.
func configuredAgents(environ []string, defaultAgent string) []ConfigAgent {
	const prefix = "SYMPOZIUM_AGENT_URL_"
	names := map[string]bool{defaultAgent: true}
	for _, kv := range environ {
		k, _, ok := strings.Cut(kv, "=")
		if !ok || !strings.HasPrefix(k, prefix) || k == prefix {
			continue
		}
		names[strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(k, prefix), "_", "-"))] = true
	}
	var out []ConfigAgent
	for n := range names {
		out = append(out, ConfigAgent{Name: n, URL: getSympoziumAgentURL(n), Default: n == defaultAgent})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

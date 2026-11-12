package main

// Response shapes of the backend API, mirrored here (like
// ui/frontend/src/lib/types.ts) rather than imported from handlers, which would
// pull client-go and the backend's init side effects into the CLI.

type logEntry struct {
	Type    string `json:"type"` // step | info | success | error | warn | done
	Message string `json:"message"`
	Time    string `json:"time"`
}

type poolStatus struct {
	State           string `json:"state"` // none | building | warm | claimed
	ClusterReady    bool   `json:"clusterReady"`
	LastError       string `json:"lastError"`
	Mode            string `json:"mode"` // standby | ondemand
	IdleSeconds     int    `json:"idleSeconds"`
	IdleTTLSeconds  int    `json:"idleTTLSeconds"` // 0 = never
	IdleTTLSource   string `json:"idleTTLSource"`  // default | env | ui
	Stopped         bool   `json:"stopped"`
	Deleting        bool   `json:"deleting"`
	DeletingSeconds int    `json:"deletingSeconds"`
}

type targetStatus struct {
	State        string `json:"state"`
	Operation    string `json:"operation"`
	ClusterPhase string `json:"clusterPhase"`
	Machines     []struct {
		Name  string `json:"name"`
		Phase string `json:"phase"`
		Role  string `json:"role"`
	} `json:"machines"`
	VMIs []struct {
		Name  string `json:"name"`
		Phase string `json:"phase"`
		IP    string `json:"ip"`
	} `json:"vmis"`
	APIReady   bool `json:"apiReady"`
	IstioReady bool `json:"istioReady"`
}

type clusterStatus struct {
	Nodes struct{ Total, Ready int } `json:"nodes"`
	Pods  struct {
		Total, Running, Pending, Failed int
	} `json:"pods"`
	VMs struct {
		Total, Running, Stopped int
	} `json:"vms"`
	Namespaces int `json:"namespaces"`
}

type backendConfig struct {
	TargetCPMode string `json:"targetCPMode"`
	Pool         struct {
		Enabled         bool   `json:"enabled"`
		Mode            string `json:"mode"`
		IdleTTLSeconds  int    `json:"idleTTLSeconds"`
		IdleTTLSource   string `json:"idleTTLSource"`
		StandbyManifest string `json:"standbyManifest"`
		FrontDoorAddr   string `json:"frontDoorAddr"`
	} `json:"pool"`
	Images []struct {
		Key     string `json:"key"`
		Image   string `json:"image"`
		Label   string `json:"label"`
		Allowed bool   `json:"allowed"`
		Default bool   `json:"default"`
	} `json:"images"`
	Profiles []struct {
		Key   string `json:"key"`
		Label string `json:"label"`
	} `json:"profiles"`
	RegistryURL      string   `json:"registryURL"`
	RegistryAlias    string   `json:"registryAlias"`
	SecurityAgentURL string   `json:"securityAgentURL"`
	RepoDir          string   `json:"repoDir"`
	AI               aiConfig `json:"ai"`
}

type aiConfig struct {
	Namespace    string `json:"namespace"`
	DefaultAgent string `json:"defaultAgent"`
	Agents       []struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Default bool   `json:"default"`
	} `json:"agents"`
	APITokenSet     bool    `json:"apiTokenSet"`
	ChatTimeoutSecs int     `json:"chatTimeoutSeconds"`
	ConsolePort     string  `json:"consolePort"`
	ConsoleUpstream string  `json:"consoleUpstream"`
	GuardEnabled    bool    `json:"guardEnabled"`
	GuardURL        string  `json:"guardURL"`
	GuardThreshold  float64 `json:"guardThreshold"`
	GuardKeySet     bool    `json:"guardKeySet"`
}

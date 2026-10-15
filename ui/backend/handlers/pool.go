package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "kubeui/backend/k8s"
)

// Pool state machine label + values on the target Cluster object.
const (
	poolStateLabel   = "pool.local/state"
	poolStateWarm    = "WARM"
	poolStateClaimed = "CLAIMED"
)

// PoolAction is the controller's per-tick decision.
type PoolAction int

const (
	ActionNone PoolAction = iota
	ActionBuild
	ActionRebuild
)

func (a PoolAction) String() string {
	switch a {
	case ActionBuild:
		return "ActionBuild"
	case ActionRebuild:
		return "ActionRebuild"
	default:
		return "ActionNone"
	}
}

// reconcileDecision is the controller's pure decision: given the observed
// cluster + in-flight flags, what should happen this tick. Honors the
// single-cluster invariant — never builds while any cluster or op exists.
// In on-demand mode nothing is built ahead of use: wakeTarget does that.
func reconcileDecision(mode string, clusterExists bool, state string, ready bool, opInFlight bool, building bool) PoolAction {
	if opInFlight || building {
		return ActionNone
	}
	if !clusterExists {
		if mode == poolModeOnDemand {
			return ActionNone
		}
		return ActionBuild
	}
	if state == poolStateClaimed {
		return ActionNone
	}
	if state == poolStateWarm {
		if !ready {
			return ActionRebuild
		}
		return ActionNone
	}
	// Unlabeled existing cluster: a Ready one is left alone (operator-managed);
	// a NotReady one is a half-built orphan (e.g. crashed mid-build) → rebuild.
	if !ready {
		return ActionRebuild
	}
	return ActionNone
}

// ClaimAction is the deploy handler's pure decision.
type ClaimAction int

const (
	ClaimLiveBuild ClaimAction = iota
	ClaimStandby
)

func (a ClaimAction) String() string {
	if a == ClaimStandby {
		return "ClaimStandby"
	}
	return "ClaimLiveBuild"
}

// claimDecision: claim the standby only if a WARM cluster exists AND is ready.
func claimDecision(warmExists bool, ready bool) ClaimAction {
	if warmExists && ready {
		return ClaimStandby
	}
	return ClaimLiveBuild
}

// PoolStatus is the JSON shape returned by GET /api/v1/cluster/pool-status.
type PoolStatus struct {
	State        string `json:"state"` // none | building | warm | claimed
	ClusterReady bool   `json:"clusterReady"`
	LastError    string `json:"lastError"`
	// On-demand mode: how long the cluster has gone unused, and when it is deleted.
	Mode           string `json:"mode"` // standby | ondemand
	IdleSeconds    int    `json:"idleSeconds"`
	IdleTTLSeconds int    `json:"idleTTLSeconds"`
}

// poolState is the controller's in-memory view, surfaced via pool-status.
// The live Cluster object remains the source of truth; this is a cache for UI.
type poolState struct {
	mu       sync.Mutex
	state    string // none | building | warm | claimed
	ready    bool
	lastErr  string
	building bool
}

var pool = &poolState{state: "none"}

func (p *poolState) setBuildState(state, errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
	p.lastErr = errMsg
}

func (p *poolState) setReady(ready bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ready = ready
}

func (p *poolState) setBuilding(b bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.building = b
}

func (p *poolState) isBuilding() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.building
}

func (p *poolState) claimAndSetBuilding() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.building {
		return false
	}
	p.building = true
	return true
}

func (p *poolState) snapshot() PoolStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	return PoolStatus{State: p.state, ClusterReady: p.ready, LastError: p.lastErr}
}

// observeCluster reads the live target Cluster: whether it exists, its
// pool.local/state label, and whether 2 nodes are Ready.
func observeCluster(ctx context.Context) (exists bool, state string, ready bool) {
	obj, err := k8sclient.DynamicClient.Resource(capiClusterGVR).Namespace("default").Get(ctx, targetClusterName, metav1.GetOptions{})
	if err != nil {
		return false, "", false
	}
	state = obj.GetLabels()[poolStateLabel]
	ready = targetNodesReady(2)
	return true, state, ready
}

// targetNodesReady reports whether >= expected nodes are Ready on the target
// cluster (via its kubeconfig at /tmp/<name>-kubeconfig).
//
// Counts by NAME, not by raw count: the :warm golden image carries a ghost
// Node object (named after the bake VM, e.g. "ubuntu-bake-vm-warm") left over
// from baking, still marked Ready until an async cleanup step removes it. A
// bare count of Ready nodes can hit `expected` from control-plane + ghost
// before the real worker ever joins — the same bug Chapter 7 of the book
// found in phase-timings.sh/time-to-ready.sh. Only nodes named
// "<cluster>-cp-*" or "<cluster>-workers-*" count.
func targetNodesReady(expected int) bool {
	kubeconfig := "/tmp/" + targetClusterName + "-kubeconfig"
	if _, err := os.Stat(kubeconfig); err != nil {
		// /tmp does not survive a reboot. Without this, a healthy cluster reads
		// as NotReady and reconcileDecision tears it down to rebuild it.
		if runShell(fmt.Sprintf("clusterctl get kubeconfig %s > %s.new 2>/dev/null && mv %s.new %s",
			targetClusterName, kubeconfig, kubeconfig, kubeconfig)) != nil {
			return false
		}
	}
	out := kubectlGet("--kubeconfig="+kubeconfig, "get", "nodes",
		"-o", "jsonpath={range .items[*]}{.metadata.name}={.status.conditions[?(@.type==\"Ready\")].status}{\"\\n\"}{end}")
	cpPrefix := targetClusterName + "-cp-"
	workerPrefix := targetClusterName + "-workers-"
	ready := 0
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, status, found := strings.Cut(line, "=")
		if !found || status != "True" {
			continue
		}
		if strings.HasPrefix(name, cpPrefix) || strings.HasPrefix(name, workerPrefix) {
			ready++
		}
	}
	return ready >= expected
}

// labelTargetCluster sets pool.local/state=<state> on the target Cluster.
func labelTargetCluster(state string) error {
	return runShell("kubectl label cluster " + targetClusterName +
		" " + poolStateLabel + "=" + state + " --overwrite")
}

func poolPollInterval() time.Duration {
	if v := os.Getenv("POOL_POLL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 5 * time.Second
}

func poolEnabled() bool { return os.Getenv("POOL_ENABLED") != "false" }

// HandlePoolStatus: GET /api/v1/cluster/pool-status
func HandlePoolStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s := pool.snapshot()
	s.Mode = poolMode()
	s.IdleSeconds = int(targetIdleFor().Seconds())
	s.IdleTTLSeconds = int(poolIdleTTL().Seconds())
	writeJSON(w, s)
}

// claimPending is set when a user claims while a standby build is in flight, so
// the build labels the cluster CLAIMED (not WARM) when it completes.
var claimPending atomic.Bool

// RunPoolController is the reconcile loop. Start once from main() in a goroutine.
func RunPoolController(ctx context.Context) {
	if !poolEnabled() {
		pool.setBuildState("none", "")
		return
	}
	ticker := time.NewTicker(poolPollInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			poolReconcileTick(ctx)
		}
	}
}

func poolReconcileTick(ctx context.Context) {
	exists, state, ready := observeCluster(ctx)
	// Refresh the UI cache from observed truth (unless mid-build).
	if !pool.isBuilding() {
		switch {
		case !exists:
			pool.setBuildState("none", "")
		case state == poolStateClaimed:
			pool.setBuildState("claimed", "")
		case state == poolStateWarm:
			pool.setBuildState("warm", "")
		}
		pool.setReady(ready)
	}

	opInFlight := getCurrentOp() != ""
	if idleDecision(poolMode(), exists, state, opInFlight, pool.isBuilding(), targetIdleFor(), poolIdleTTL()) {
		setCurrentOp("idle-delete")
		go runIdleDeletion()
		return
	}
	if wakePending.Load() {
		wakeTarget()
	}
	switch reconcileDecision(poolMode(), exists, state, ready, opInFlight, pool.isBuilding()) {
	case ActionBuild:
		if pool.claimAndSetBuilding() {
			go buildStandby(false, false)
		}
	case ActionRebuild:
		if pool.claimAndSetBuilding() {
			go buildStandby(false, true)
		}
	case ActionNone:
		// nothing
	}
}

// runClaim hands over an existing WARM standby as the user's cluster: relabel
// CLAIMED and emit synthetic progress. Resolves in seconds (cluster already up).
func runClaim() {
	defer setCurrentOp("")
	deploy.addLog("step", "━━ Claiming pre-built standby cluster ━━")
	deploy.addLog("info", "A warm standby was ready — handing it over instead of building.")

	if err := labelTargetCluster(poolStateClaimed); err != nil {
		deploy.addLog("warn", "could not set claimed label: "+err.Error())
	}
	claimPending.Store(false)
	// Refresh the kubeconfig copies for downstream scripts and target-cluster-agent.
	kubeconfigPath := "/tmp/" + targetClusterName + "-kubeconfig"
	_ = runShell(fmt.Sprintf("clusterctl get kubeconfig %s > %s 2>/dev/null || true", targetClusterName, kubeconfigPath))
	publishTargetKubeconfig(kubeconfigPath)
	touchTarget()

	pool.setBuildState("claimed", "")
	pool.setReady(true)
	deploy.addLog("success", "✓ Kubeconfig ready → "+kubeconfigPath)
	deploy.addLog("success", "✓ Standby claimed — cluster is up and Ready")
	deploy.addLog("step", "━━ Done! ━━")
	deploy.finish("done")
}

// runIdleDeletion deletes a cluster nobody has used for POOL_IDLE_TTL. The
// caller has already set the current op.
func runIdleDeletion() {
	deploy.resetForNewRun()
	deploy.addLog("step", fmt.Sprintf("━━ Target cluster unused for %s — deleting ━━", poolIdleTTL()))
	runDeletion()
	pool.setBuildState("none", "")
	pool.setReady(false)
	// An access that arrived mid-teardown gets its build now, not a tick later.
	if wakePending.Load() {
		wakeTarget()
	}
}

// buildStandby builds the standby cluster off the user's clock. If claimAfter
// (or claimPending becomes set during the build) it labels CLAIMED, else WARM.
// predelete=true tears down any existing (degraded) cluster before applying.
func buildStandby(claimAfter bool, predelete bool) {
	setCurrentOp("pool-build")
	defer func() {
		setCurrentOp("")
		pool.setBuilding(false)
		if r := recover(); r != nil {
			pool.setBuildState("none", fmt.Sprintf("panic: %v", r))
			deploy.finish("failed")
		}
	}()

	deploy.resetForNewRun()
	pool.setBuildState("building", "")
	if predelete {
		deploy.addLog("step", "━━ Rebuilding standby — tearing down degraded cluster ━━")
		_ = runShell("kubectl delete cluster " + targetClusterName + " --ignore-not-found --timeout=120s")
	}
	deploy.addLog("step", "━━ Building warm standby cluster (background) ━━")

	manifest := os.Getenv("POOL_STANDBY_MANIFEST")
	if manifest == "" {
		manifest = "03-target-cluster/target-cluster-warm.yaml"
	}
	manifestPath := filepath.Join(claudeDir(), manifest)
	// Warm manifest needs the fixed CA/token secrets seeded first so KThrees
	// adopts the material already baked into the :warm image (no CA conflict,
	// no --cluster-reset) — same order as the on-demand deploy path.
	if strings.Contains(manifest, "target-cluster-warm") {
		deploy.addLog("info", "Warm standby — seeding fixed CA + token secrets so KThrees adopts the baked material...")
		if err := runShell("bash " + filepath.Join(claudeDir(), "scripts/seed-cluster-secrets.sh")); err != nil {
			pool.setBuildState("none", "failed to seed warm-path secrets: "+err.Error())
			claimPending.Store(false)
			deploy.finish("failed")
			return
		}
	}
	if err := runShell("kubectl apply -f " + manifestPath); err != nil {
		pool.setBuildState("none", "apply failed: "+err.Error())
		claimPending.Store(false)
		deploy.finish("failed")
		return
	}
	if err := waitForTargetReady(2); err != nil {
		pool.setBuildState("none", err.Error())
		claimPending.Store(false)
		deploy.finish("failed")
		return
	}

	label := poolStateWarm
	uiState := "warm"
	if claimAfter || claimPending.Load() {
		label = poolStateClaimed
		uiState = "claimed"
		claimPending.Store(false)
	}
	if err := labelTargetCluster(label); err != nil {
		deploy.addLog("warn", "could not set pool label: "+err.Error())
	}
	pool.setReady(true)
	pool.setBuildState(uiState, "")
	if uiState == "warm" {
		deploy.addLog("success", "✓ Warm standby ready — Deploy will claim it instantly")
	} else {
		deploy.addLog("success", "✓ Cluster ready")
	}
	deploy.finish("done")
}

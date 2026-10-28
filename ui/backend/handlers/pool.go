package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
// In on-demand mode nothing is built ahead of use: wakeTarget does that. A
// cluster the user deleted from the UI (stopped) is not rebuilt in either mode.
func reconcileDecision(mode string, stopped bool, clusterExists bool, state string, ready bool, opInFlight bool, building bool) PoolAction {
	if opInFlight || building {
		return ActionNone
	}
	if !clusterExists {
		if mode == poolModeOnDemand || stopped {
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
	IdleTTLSeconds int    `json:"idleTTLSeconds"` // 0 = never auto-delete
	IdleTTLSource  string `json:"idleTTLSource"`  // default | env | ui
	// Deleted from the UI: stays down until the next Deploy.
	Stopped bool `json:"stopped"`
	// The Cluster has a deletionTimestamp; how long it has been deleting. A
	// teardown that runs for minutes is stuck (e.g. the CAPK controller is down).
	Deleting        bool `json:"deleting"`
	DeletingSeconds int  `json:"deletingSeconds"`
}

// poolState is the controller's in-memory view, surfaced via pool-status.
// The live Cluster object remains the source of truth; this is a cache for UI.
type poolState struct {
	mu       sync.Mutex
	state    string // none | building | warm | claimed
	ready    bool
	lastErr  string
	building bool
	// deletingSince is the observed Cluster's deletionTimestamp; zero if none.
	deletingSince time.Time
}

var pool = &poolState{state: "none"}

func (p *poolState) setBuildState(state, errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
	p.lastErr = errMsg
}

// setStateKeepError changes state but leaves lastErr alone, so a retry that
// starts right after a failure keeps the cause on screen until it succeeds.
func (p *poolState) setStateKeepError(state string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
}

func (p *poolState) setReady(ready bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ready = ready
}

func (p *poolState) setDeletingSince(t time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deletingSince = t
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
	s := PoolStatus{State: p.state, ClusterReady: p.ready, LastError: p.lastErr}
	if !p.deletingSince.IsZero() {
		s.Deleting = true
		s.DeletingSeconds = int(time.Since(p.deletingSince).Seconds())
	}
	return s
}

// observeCluster reads the live target Cluster: whether it exists, its
// pool.local/state label, and whether 2 nodes are Ready.
func observeCluster(ctx context.Context) (exists bool, state string, ready bool) {
	exists, state, ready, _ = observeClusterDeletion(ctx)
	return exists, state, ready
}

// observeClusterDeletion is observeCluster plus the Cluster's deletionTimestamp
// (zero if it is not being deleted). A Cluster being deleted is never ready:
// its VMs, and so its Nodes, can stay up for as long as the teardown is stuck.
func observeClusterDeletion(ctx context.Context) (exists bool, state string, ready bool, deletingSince time.Time) {
	obj, err := k8sclient.DynamicClient.Resource(capiClusterGVR).Namespace("default").Get(ctx, targetClusterName, metav1.GetOptions{})
	if err != nil {
		return false, "", false, time.Time{}
	}
	state = obj.GetLabels()[poolStateLabel]
	if ts := obj.GetDeletionTimestamp(); ts != nil {
		return true, state, false, ts.Time
	}
	return true, state, targetNodesReady(2), time.Time{}
}

// targetNodesReady reports whether >= expected nodes are Ready on the target
// cluster (via its kubeconfig at /tmp/<name>-kubeconfig, refetched from the
// <cluster>-kubeconfig Secret when /tmp lost it).
//
// Counts by NAME, not by raw count (countReadyTargetNodes): the :warm golden
// image carries a ghost Node object named after the bake VM, still marked Ready
// until an async cleanup removes it, so a bare count could hit `expected` before
// the real worker joins. On the hosted path both VMs are "<cluster>-workers-*"
// agents and nothing else is a Node, so the same rule holds.
func targetNodesReady(expected int) bool {
	kubeconfig, err := directTargetKubeconfig()
	if err != nil {
		// /tmp does not survive a reboot. Without the refetch, a healthy cluster
		// reads as NotReady and reconcileDecision tears it down to rebuild it.
		return false
	}
	out := kubectlGet("--kubeconfig="+kubeconfig, "get", "nodes",
		"-o", "jsonpath={range .items[*]}{.metadata.name}={.status.conditions[?(@.type==\"Ready\")].status}{\"\\n\"}{end}")
	return countReadyTargetNodes(out, targetClusterName) >= expected
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
	writeJSON(w, currentPoolStatus())
}

func currentPoolStatus() PoolStatus {
	s := pool.snapshot()
	s.Mode = poolMode()
	s.IdleSeconds = int(targetIdleFor().Seconds())
	ttl, src := poolIdleTTLWithSource()
	s.IdleTTLSeconds = int(ttl.Seconds())
	s.IdleTTLSource = src
	s.Stopped = targetStoppedByUser()
	return s
}

// HandlePoolSettings: PUT /api/v1/cluster/pool-settings {"idleTTLSeconds": N}
// sets the on-demand idle TTL (0 = never). It is a setting change, not use, so
// the idle clock is not reset: a TTL below the current idle time deletes the
// cluster on the next tick. DELETE drops the override (back to POOL_IDLE_TTL).
func HandlePoolSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		clearIdleTTLOverride()
		log.Printf("pool settings: idle TTL override cleared from the UI/CLI")
		writeJSON(w, currentPoolStatus())
		return
	}
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		IdleTTLSeconds *int `json:"idleTTLSeconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.IdleTTLSeconds == nil {
		writeError(w, http.StatusBadRequest, "body must be {\"idleTTLSeconds\": N}")
		return
	}
	d := time.Duration(*body.IdleTTLSeconds) * time.Second
	if !validIdleTTL(d) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("idleTTLSeconds must be 0 (never) or %d-%d", int(idleTTLMin.Seconds()), int(idleTTLMax.Seconds())))
		return
	}
	setIdleTTLOverride(d)
	log.Printf("pool settings: idle TTL set to %s from the UI", d)
	writeJSON(w, currentPoolStatus())
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
	exists, state, ready, deletingSince := observeClusterDeletion(ctx)
	pool.setDeletingSince(deletingSince)
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

	// A teardown is already under way: wait for it. Rebuilding now would only
	// time out on the same delete and apply onto a Cluster that is going away.
	if !deletingSince.IsZero() {
		return
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
	switch reconcileDecision(poolMode(), targetStoppedByUser(), exists, state, ready, opInFlight, pool.isBuilding()) {
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
	runDeletion(false)
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
	pool.setStateKeepError("building")
	if predelete {
		deploy.addLog("step", "━━ Rebuilding standby — tearing down degraded cluster ━━")
		if err := runShell("kubectl delete cluster " + targetClusterName + " --ignore-not-found --timeout=120s"); err != nil {
			// Still deleting: applying now would land on a Cluster that is going
			// away. The next tick sees the deletionTimestamp and waits.
			pool.setBuildState("none", "teardown did not finish in 120s; is the CAPK controller running? "+err.Error())
			claimPending.Store(false)
			deploy.finish("failed")
			return
		}
	}
	deploy.addLog("step", "━━ Building warm standby cluster (background) ━━")

	manifest := os.Getenv("POOL_STANDBY_MANIFEST")
	if manifest == "" {
		manifest = defaultStandbyManifest()
	}
	manifestPath := filepath.Join(claudeDir(), manifest)
	if strings.Contains(manifest, "target-cluster-hosted") {
		// Hosted path: the resident server must be up and the kubeconfig Secret
		// seeded before the agents boot. Clear Nodes a failed teardown may have
		// left, or the new Machines would never match "2 Ready" by name.
		deploy.addLog("info", "Hosted control plane — making sure the resident k3s server is up...")
		if err := ensureHostedCP(deploy.addLog); err != nil {
			pool.setBuildState("none", err.Error())
			claimPending.Store(false)
			deploy.finish("failed")
			return
		}
		if err := deleteTargetNodes(); err != nil {
			deploy.addLog("warn", "could not clear stale target Nodes: "+err.Error())
		}
	} else if strings.Contains(manifest, "target-cluster-warm") {
		// Warm manifest needs the fixed CA/token secrets seeded first so KThrees
		// adopts the material already baked into the :warm image (no CA conflict,
		// no --cluster-reset) — same order as the on-demand deploy path.
		deploy.addLog("info", "Warm standby — seeding fixed CA + token secrets so KThrees adopts the baked material...")
		if err := runShell("bash " + filepath.Join(claudeDir(), "scripts/seed-cluster-secrets.sh")); err != nil {
			pool.setBuildState("none", "failed to seed warm-path secrets: "+err.Error())
			claimPending.Store(false)
			deploy.finish("failed")
			return
		}
	}
	ensureRegistryPulls(deploy.addLog)
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

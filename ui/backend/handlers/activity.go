package handlers

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "kubeui/backend/k8s"
)

// On-demand mode: no target cluster exists while nobody uses it. The first
// access builds one (wakeTarget), and the pool controller deletes it again once
// it has gone POOL_IDLE_TTL without use (idleDecision). "Use" is anything that
// calls touchTarget: the front door (activator.go), Deploy, and chat to
// target-cluster-agent.
const (
	poolModeStandby  = "standby"
	poolModeOnDemand = "ondemand"

	targetAgentName = "target-cluster-agent"
)

// poolMode: POOL_MODE=ondemand opts in; anything else is the standby pool.
func poolMode() string {
	if os.Getenv("POOL_MODE") == poolModeOnDemand {
		return poolModeOnDemand
	}
	return poolModeStandby
}

func poolIdleTTL() time.Duration {
	if v := os.Getenv("POOL_IDLE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return time.Hour
}

// lastActivity is in memory only and starts at process start, so a backend
// restart can delay an idle delete but never bring one forward.
var lastActivity atomic.Int64

func init() { lastActivity.Store(time.Now().UnixNano()) }

func touchTarget() { lastActivity.Store(time.Now().UnixNano()) }

func targetIdleFor() time.Duration {
	return time.Since(time.Unix(0, lastActivity.Load()))
}

// idleDecision is the pure "delete the idle cluster now?" decision. Only
// clusters this backend labeled are ever reaped; an unlabeled one is
// operator-managed and left alone, same rule as reconcileDecision.
func idleDecision(mode string, clusterExists bool, state string, opInFlight bool, building bool, idle, ttl time.Duration) bool {
	if mode != poolModeOnDemand || !clusterExists || opInFlight || building {
		return false
	}
	if state != poolStateWarm && state != poolStateClaimed {
		return false
	}
	return idle >= ttl
}

// wakePending is set when an access arrives while a teardown (or any other op)
// is in flight; the build starts as soon as that op is done.
var wakePending atomic.Bool

var (
	wakeMu      sync.Mutex
	lastWakeTry time.Time
)

// targetClusterState is the cheap existence check (one API GET, no kubectl).
func targetClusterState(ctx context.Context) (exists bool, state string) {
	if k8sclient.DynamicClient == nil {
		return false, ""
	}
	obj, err := k8sclient.DynamicClient.Resource(capiClusterGVR).Namespace("default").Get(ctx, targetClusterName, metav1.GetOptions{})
	if err != nil {
		return false, ""
	}
	return true, obj.GetLabels()[poolStateLabel]
}

// wakeTarget records use and, in on-demand mode, starts a build if no target
// cluster exists. Returns immediately; safe to call on every access.
func wakeTarget() {
	touchTarget()
	if poolMode() != poolModeOnDemand || !poolEnabled() {
		return
	}
	wakeMu.Lock()
	defer wakeMu.Unlock()
	if time.Since(lastWakeTry) < 500*time.Millisecond {
		return
	}
	lastWakeTry = time.Now()

	if pool.isBuilding() {
		return
	}
	if getCurrentOp() != "" || deploy.getState() == "running" {
		wakePending.Store(true)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if exists, _ := targetClusterState(ctx); exists {
		wakePending.Store(false)
		return
	}
	if pool.claimAndSetBuilding() {
		wakePending.Store(false)
		go buildStandby(true, false)
	}
}

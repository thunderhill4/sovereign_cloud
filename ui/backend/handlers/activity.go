package handlers

import (
	"context"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// poolIdleTTL is the idle time after which the target is deleted. A value set
// from the UI (setIdleTTLOverride) wins over POOL_IDLE_TTL; 0 means never.
func poolIdleTTL() time.Duration {
	d, _ := poolIdleTTLWithSource()
	return d
}

const (
	idleTTLSourceDefault = "default"
	idleTTLSourceEnv     = "env"
	idleTTLSourceUI      = "ui"

	idleTTLMin = time.Minute
	idleTTLMax = 24 * time.Hour
)

func poolIdleTTLWithSource() (time.Duration, string) {
	if v := idleTTLOverride.Load(); v >= 0 {
		return time.Duration(v), idleTTLSourceUI
	}
	if v := os.Getenv("POOL_IDLE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d, idleTTLSourceEnv
		}
	}
	return time.Hour, idleTTLSourceDefault
}

// idleTTLOverride holds the UI-set TTL in nanoseconds; -1 means unset.
var idleTTLOverride atomic.Int64

func init() { idleTTLOverride.Store(-1) }

// validIdleTTL: 0 (never) or between idleTTLMin and idleTTLMax.
func validIdleTTL(d time.Duration) bool {
	return d == 0 || (d >= idleTTLMin && d <= idleTTLMax)
}

// Kept apart from kubeui-target-state, whose existence is the stop flag.
const poolSettingsConfigMap = "kubeui-pool-settings"

// setIdleTTLOverride applies the TTL now and, best effort, persists it.
func setIdleTTLOverride(d time.Duration) {
	idleTTLOverride.Store(int64(d))
	if k8sclient.Clientset == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cms := k8sclient.Clientset.CoreV1().ConfigMaps(targetStateNamespace)
	data := map[string]string{"idleTTL": d.String()}
	cm, err := cms.Get(ctx, poolSettingsConfigMap, metav1.GetOptions{})
	switch {
	case apierrors.IsNotFound(err):
		cm = &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: poolSettingsConfigMap}, Data: data}
		_, err = cms.Create(ctx, cm, metav1.CreateOptions{})
	case err == nil:
		cm.Data = data
		_, err = cms.Update(ctx, cm, metav1.UpdateOptions{})
	}
	if err != nil {
		log.Printf("pool settings: could not persist the idle TTL: %v", err)
	}
}

// clearIdleTTLOverride drops the UI-set TTL (back to POOL_IDLE_TTL or the
// default) and, best effort, its ConfigMap.
func clearIdleTTLOverride() {
	idleTTLOverride.Store(-1)
	if k8sclient.Clientset == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := k8sclient.Clientset.CoreV1().ConfigMaps(targetStateNamespace).Delete(ctx, poolSettingsConfigMap, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Printf("pool settings: could not delete %s: %v", poolSettingsConfigMap, err)
	}
}

// LoadPoolSettings restores a UI-set idle TTL at startup. Call after k8s Init.
func LoadPoolSettings() {
	if k8sclient.Clientset == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cm, err := k8sclient.Clientset.CoreV1().ConfigMaps(targetStateNamespace).Get(ctx, poolSettingsConfigMap, metav1.GetOptions{})
	if err != nil {
		return
	}
	d, err := time.ParseDuration(cm.Data["idleTTL"])
	if err != nil || !validIdleTTL(d) {
		log.Printf("pool settings: ignoring idleTTL %q in %s", cm.Data["idleTTL"], poolSettingsConfigMap)
		return
	}
	idleTTLOverride.Store(int64(d))
	log.Printf("pool settings: idle TTL %s (set from the UI)", d)
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
	if mode != poolModeOnDemand || !clusterExists || opInFlight || building || ttl <= 0 {
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

// Stopped by the user. A Delete from the UI means "I am done with it", unlike
// an idle delete, which only frees resources until the next access. While
// stopped nothing brings the cluster back on its own: no front-door wake, no
// wake from target-cluster-agent chat, no standby rebuild. Only Deploy clears
// it. Kept in a ConfigMap so a backend restart does not forget it.
const (
	targetStateConfigMap = "kubeui-target-state"
	targetStateNamespace = "default"
)

var targetStopped atomic.Bool

func targetStoppedByUser() bool { return targetStopped.Load() }

// setTargetStopped records the stop flag in memory and, best effort, in the
// ConfigMap.
func setTargetStopped(stopped bool) {
	targetStopped.Store(stopped)
	if stopped {
		wakePending.Store(false)
	}
	if k8sclient.Clientset == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cms := k8sclient.Clientset.CoreV1().ConfigMaps(targetStateNamespace)
	if !stopped {
		if err := cms.Delete(ctx, targetStateConfigMap, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			log.Printf("target state: could not clear the stop flag: %v", err)
		}
		return
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: targetStateConfigMap},
		Data:       map[string]string{"stoppedAt": time.Now().UTC().Format(time.RFC3339)},
	}
	if _, err := cms.Create(ctx, cm, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		log.Printf("target state: could not persist the stop flag: %v", err)
	}
}

// LoadTargetStopped restores the stop flag at startup. Call after k8s Init.
func LoadTargetStopped() {
	if k8sclient.Clientset == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := k8sclient.Clientset.CoreV1().ConfigMaps(targetStateNamespace).Get(ctx, targetStateConfigMap, metav1.GetOptions{}); err == nil {
		targetStopped.Store(true)
		log.Printf("target-cluster was deleted from the UI; it stays down until the next Deploy")
	}
}

// wakeTarget records use and, in on-demand mode, starts a build if no target
// cluster exists and the user has not stopped it. Returns immediately; safe to
// call on every access.
func wakeTarget() {
	touchTarget()
	if poolMode() != poolModeOnDemand || !poolEnabled() || targetStoppedByUser() {
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

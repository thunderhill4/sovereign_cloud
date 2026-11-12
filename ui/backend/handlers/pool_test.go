package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestReconcileDecision(t *testing.T) {
	cases := []struct {
		name         string
		clusterExists bool
		state        string
		ready        bool
		opInFlight   bool
		building     bool
		want         PoolAction
	}{
		{"no cluster idle -> build", false, "", false, false, false, ActionBuild},
		{"no cluster but op in flight -> none", false, "", false, true, false, ActionNone},
		{"no cluster but already building -> none", false, "", false, false, true, ActionNone},
		{"warm healthy -> none", true, poolStateWarm, true, false, false, ActionNone},
		{"warm not ready -> rebuild", true, poolStateWarm, false, false, false, ActionRebuild},
		{"warm not ready but op in flight -> none", true, poolStateWarm, false, true, false, ActionNone},
		{"claimed -> none", true, poolStateClaimed, true, false, false, ActionNone},
		{"claimed not ready -> none", true, poolStateClaimed, false, false, false, ActionNone},
		{"unlabeled existing cluster -> none", true, "", true, false, false, ActionNone},
		{"unlabeled existing NotReady -> rebuild", true, "", false, false, false, ActionRebuild},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := reconcileDecision(poolModeStandby, false, c.clusterExists, c.state, c.ready, c.opInFlight, c.building)
			if got != c.want {
				t.Fatalf("reconcileDecision(%v) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

func TestReconcileDecisionOnDemand(t *testing.T) {
	// No cluster: on-demand never builds ahead of use.
	if got := reconcileDecision(poolModeOnDemand, false, false, "", false, false, false); got != ActionNone {
		t.Fatalf("on-demand, no cluster = %v, want ActionNone", got)
	}
	// A half-built orphan is still repaired: someone asked for that cluster.
	if got := reconcileDecision(poolModeOnDemand, false, true, "", false, false, false); got != ActionRebuild {
		t.Fatalf("on-demand, unlabeled NotReady = %v, want ActionRebuild", got)
	}
}

func TestReconcileDecisionStoppedByUser(t *testing.T) {
	// Deleted from the UI: the standby pool must not bring it back.
	if got := reconcileDecision(poolModeStandby, true, false, "", false, false, false); got != ActionNone {
		t.Fatalf("standby, stopped, no cluster = %v, want ActionNone", got)
	}
	if got := reconcileDecision(poolModeOnDemand, true, false, "", false, false, false); got != ActionNone {
		t.Fatalf("on-demand, stopped, no cluster = %v, want ActionNone", got)
	}
}

func TestWakeTargetWhileStoppedDoesNotBuild(t *testing.T) {
	t.Setenv("POOL_MODE", poolModeOnDemand)
	t.Setenv("POOL_ENABLED", "true")
	targetStopped.Store(true)
	defer targetStopped.Store(false)
	wakeMu.Lock()
	lastWakeTry = time.Time{}
	wakeMu.Unlock()
	wakeTarget()
	if pool.isBuilding() || wakePending.Load() {
		t.Fatal("a wake while stopped must not start or queue a build")
	}
}

func TestIdleDecision(t *testing.T) {
	const ttl = time.Hour
	cases := []struct {
		name       string
		mode       string
		exists     bool
		state      string
		opInFlight bool
		building   bool
		idle       time.Duration
		want       bool
	}{
		{"claimed past ttl -> delete", poolModeOnDemand, true, poolStateClaimed, false, false, ttl, true},
		{"warm past ttl -> delete", poolModeOnDemand, true, poolStateWarm, false, false, 2 * ttl, true},
		{"claimed under ttl -> keep", poolModeOnDemand, true, poolStateClaimed, false, false, ttl - time.Second, false},
		{"unlabeled past ttl -> keep (operator-managed)", poolModeOnDemand, true, "", false, false, 2 * ttl, false},
		{"standby mode never reaps", poolModeStandby, true, poolStateClaimed, false, false, 2 * ttl, false},
		{"op in flight -> keep", poolModeOnDemand, true, poolStateClaimed, true, false, 2 * ttl, false},
		{"building -> keep", poolModeOnDemand, true, poolStateClaimed, false, true, 2 * ttl, false},
		{"no cluster -> nothing to do", poolModeOnDemand, false, "", false, false, 2 * ttl, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := idleDecision(c.mode, c.exists, c.state, c.opInFlight, c.building, c.idle, ttl); got != c.want {
				t.Fatalf("idleDecision = %v, want %v", got, c.want)
			}
		})
	}
	if idleDecision(poolModeOnDemand, true, poolStateClaimed, false, false, 100*ttl, 0) {
		t.Fatal("ttl 0 means never: must not delete")
	}
}

func TestPoolIdleTTLPrecedence(t *testing.T) {
	defer idleTTLOverride.Store(-1)
	t.Setenv("POOL_IDLE_TTL", "")
	if d, src := poolIdleTTLWithSource(); d != time.Hour || src != idleTTLSourceDefault {
		t.Fatalf("default = %s/%s, want 1h/default", d, src)
	}
	t.Setenv("POOL_IDLE_TTL", "20m")
	if d, src := poolIdleTTLWithSource(); d != 20*time.Minute || src != idleTTLSourceEnv {
		t.Fatalf("env = %s/%s, want 20m/env", d, src)
	}
	idleTTLOverride.Store(int64(5 * time.Minute))
	if d, src := poolIdleTTLWithSource(); d != 5*time.Minute || src != idleTTLSourceUI {
		t.Fatalf("ui = %s/%s, want 5m/ui", d, src)
	}
	idleTTLOverride.Store(0)
	if d, src := poolIdleTTLWithSource(); d != 0 || src != idleTTLSourceUI {
		t.Fatalf("ui never = %s/%s, want 0/ui", d, src)
	}
}

func TestHandlePoolSettings(t *testing.T) {
	defer idleTTLOverride.Store(-1)
	cases := []struct {
		body string
		code int
	}{
		{`{"idleTTLSeconds": 900}`, http.StatusOK},
		{`{"idleTTLSeconds": 0}`, http.StatusOK},
		{`{"idleTTLSeconds": 30}`, http.StatusBadRequest},
		{`{"idleTTLSeconds": 90000}`, http.StatusBadRequest},
		{`{}`, http.StatusBadRequest},
		{`nope`, http.StatusBadRequest},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		HandlePoolSettings(rec, httptest.NewRequest(http.MethodPut, "/api/v1/cluster/pool-settings", strings.NewReader(c.body)))
		if rec.Code != c.code {
			t.Fatalf("%s: code %d, want %d", c.body, rec.Code, c.code)
		}
	}
	idleTTLOverride.Store(-1)
	rec := httptest.NewRecorder()
	HandlePoolSettings(rec, httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"idleTTLSeconds": 600}`)))
	var s PoolStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil || s.IdleTTLSeconds != 600 || s.IdleTTLSource != idleTTLSourceUI {
		t.Fatalf("response = %+v (%v), want 600s from ui", s, err)
	}
}

func TestClaimDecision(t *testing.T) {
	if claimDecision(true, true) != ClaimStandby {
		t.Fatal("warm+ready should claim standby")
	}
	if claimDecision(true, false) != ClaimLiveBuild {
		t.Fatal("warm-but-not-ready should live build")
	}
	if claimDecision(false, false) != ClaimLiveBuild {
		t.Fatal("no standby should live build")
	}
	if claimDecision(false, true) != ClaimLiveBuild {
		t.Fatal("no standby (ready=true) should live build")
	}
}

func TestPoolSnapshotDefault(t *testing.T) {
	p := &poolState{state: "none"}
	s := p.snapshot()
	if s.State != "none" || s.ClusterReady || s.LastError != "" {
		t.Fatalf("unexpected default snapshot: %+v", s)
	}
}

func TestPoolSetBuildState(t *testing.T) {
	p := &poolState{state: "none"}
	p.setBuildState("warm", "")
	if s := p.snapshot(); s.State != "warm" || s.LastError != "" {
		t.Fatalf("after warm: %+v", s)
	}
	p.setBuildState("building", "boom")
	if s := p.snapshot(); s.State != "building" || s.LastError != "boom" {
		t.Fatalf("after error: %+v", s)
	}
}

func TestPoolRetryKeepsLastError(t *testing.T) {
	p := &poolState{state: "none"}
	p.setBuildState("none", "apply failed: webhook timeout")
	p.setStateKeepError("building")
	if s := p.snapshot(); s.State != "building" || s.LastError != "apply failed: webhook timeout" {
		t.Fatalf("retry cleared the error: %+v", s)
	}
	p.setBuildState("warm", "")
	if s := p.snapshot(); s.LastError != "" {
		t.Fatalf("success kept the error: %+v", s)
	}
}

func TestClaimAndSetBuildingExclusive(t *testing.T) {
	p := &poolState{state: "none"}
	const n = 50
	var wins int64
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if p.claimAndSetBuilding() {
				atomic.AddInt64(&wins, 1)
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("expected exactly 1 winner, got %d", wins)
	}
	if !p.isBuilding() {
		t.Fatal("building flag should be set after a win")
	}
}

func TestSnapshotDeleting(t *testing.T) {
	p := &poolState{state: "warm", ready: true}
	if s := p.snapshot(); s.Deleting {
		t.Fatal("no deletionTimestamp -> not deleting")
	}
	p.setDeletingSince(time.Now().Add(-5 * time.Minute))
	s := p.snapshot()
	if !s.Deleting || s.DeletingSeconds < 299 {
		t.Fatalf("snapshot = %+v, want deleting for ~300s", s)
	}
}

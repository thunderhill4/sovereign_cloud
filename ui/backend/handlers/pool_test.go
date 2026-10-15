package handlers

import (
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
			got := reconcileDecision(poolModeStandby, c.clusterExists, c.state, c.ready, c.opInFlight, c.building)
			if got != c.want {
				t.Fatalf("reconcileDecision(%v) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

func TestReconcileDecisionOnDemand(t *testing.T) {
	// No cluster: on-demand never builds ahead of use.
	if got := reconcileDecision(poolModeOnDemand, false, "", false, false, false); got != ActionNone {
		t.Fatalf("on-demand, no cluster = %v, want ActionNone", got)
	}
	// A half-built orphan is still repaired: someone asked for that cluster.
	if got := reconcileDecision(poolModeOnDemand, true, "", false, false, false); got != ActionRebuild {
		t.Fatalf("on-demand, unlabeled NotReady = %v, want ActionRebuild", got)
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

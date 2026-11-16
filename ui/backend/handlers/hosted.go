package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Hosted control plane (docs/sub-60s-cluster-strategy.md §12).
//
// The target cluster's k3s server runs as a resident pod on cluster2
// (03-target-cluster/hosted-cp/, namespace target-cp) and owns the API VIP
// targetLBIP. Both KubeVirt VMs are k3s agents booted from the :agent image, so
// a build is VM boot + agent join (~9s) with no control plane to boot.
//
// What changes for the backend:
//   - builds ensure the server pod is up and the kubeconfig Secret is seeded
//     (scripts/seed-hosted-cp.sh), then apply target-cluster-hosted.yaml
//   - deleting the Cluster does NOT delete its Nodes: CAPI skips workload Node
//     deletion when the Cluster is being deleted and when there are no
//     control-plane Machines. The resident server would keep them (still
//     marked Ready), so every teardown deletes them here.
//   - the API answers even when no cluster exists, so the front door's "up"
//     means "the target's Nodes are Ready", not "the API answers". The server
//     also answers 401 to unauthenticated /readyz, which the VM path probed.
//
// TARGET_CP_MODE=vm restores the VM control plane (target-cluster-warm). The
// two cannot run at once: both claim targetLBIP.

const (
	cpModeHosted = "hosted"
	cpModeVM     = "vm"

	// hostedImageKey is the imageVariants key for the hosted path.
	hostedImageKey = "hosted"

	hostedCPNamespace  = "target-cp"
	hostedCPDeployment = "k3s-server"
)

// cpMode: TARGET_CP_MODE=vm selects the VM control plane; anything else is hosted.
func cpMode() string {
	if os.Getenv("TARGET_CP_MODE") == cpModeVM {
		return cpModeVM
	}
	return cpModeHosted
}

func hostedCP() bool { return cpMode() == cpModeHosted }

// defaultImageKey is the Deploy image when the request names none.
func defaultImageKey() string {
	if hostedCP() {
		return hostedImageKey
	}
	return warmImageKey
}

// imageAllowed reports whether an image variant can be deployed in the current
// control-plane mode. VM-control-plane images would claim the VIP the hosted
// server owns, and the hosted image needs the hosted server.
func imageAllowed(mode, image string) bool {
	if mode == cpModeHosted {
		return image == hostedImageKey
	}
	return image != hostedImageKey
}

// defaultStandbyManifest is the pool's build manifest unless
// POOL_STANDBY_MANIFEST overrides it.
func defaultStandbyManifest() string {
	if hostedCP() {
		return "03-target-cluster/target-cluster-hosted.yaml"
	}
	return "03-target-cluster/target-cluster-warm.yaml"
}

// ensureHostedCP makes the resident control plane ready: seeds its CA Secret and
// the <cluster>-kubeconfig Secret (kept if already valid), applies the server
// manifests, and waits for the server pod. Idempotent and quick when it is
// already running. seed-hosted-cp.sh refuses while a legacy VM-control-plane
// cluster holds the VIP.
func ensureHostedCP(logf func(level, msg string)) error {
	dir := claudeDir()
	if err := runShell("bash " + filepath.Join(dir, "scripts/seed-hosted-cp.sh")); err != nil {
		return fmt.Errorf("seeding the hosted control plane: %w", err)
	}
	if err := runShell("kubectl apply -k " + filepath.Join(dir, "03-target-cluster/hosted-cp")); err != nil {
		return fmt.Errorf("applying the hosted control plane: %w", err)
	}
	if err := runShell(fmt.Sprintf("kubectl -n %s rollout status deploy/%s --timeout=120s",
		hostedCPNamespace, hostedCPDeployment)); err != nil {
		return fmt.Errorf("hosted control plane did not become ready: %w", err)
	}
	logf("success", "✓ Hosted control plane ready at "+targetLBIP+":6443")
	return nil
}

// directTargetKubeconfig returns the backend's direct kubeconfig, fetching it from
// the <cluster>-kubeconfig Secret when /tmp lost it. In hosted mode that Secret
// outlives the Cluster.
func directTargetKubeconfig() (string, error) {
	kc := "/tmp/" + targetClusterName + "-kubeconfig"
	if _, err := os.Stat(kc); err == nil {
		return kc, nil
	}
	if err := runShell(fmt.Sprintf("clusterctl get kubeconfig %s > %s.new 2>/dev/null && mv %s.new %s",
		targetClusterName, kc, kc, kc)); err != nil {
		return "", err
	}
	return kc, nil
}

// deleteTargetNodes removes the agents' Node objects from the resident server
// after their VMs are gone (see the file comment). Best effort.
func deleteTargetNodes() error {
	kc, err := directTargetKubeconfig()
	if err != nil {
		return fmt.Errorf("no target kubeconfig: %w", err)
	}
	return runShell(fmt.Sprintf("kubectl --kubeconfig=%s delete nodes --all --wait=true --timeout=30s", kc))
}

// countReadyTargetNodes counts Ready nodes in `name=ReadyStatus` lines whose
// names belong to the target cluster's VMs. Counting by name, not raw count,
// keeps ghost or foreign Node objects (e.g. the :warm image's baked bake-VM
// node) from satisfying "N nodes Ready".
func countReadyTargetNodes(out, cluster string) int {
	cpPrefix := cluster + "-cp-"
	workerPrefix := cluster + "-workers-"
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
	return ready
}

// targetUsable is the front door's "up" in hosted mode: both target Nodes are
// Ready. Cached 1s while usable and 200ms while not, like targetAPIUp, so a
// finished build is noticed quickly without a kubectl call per connection.
var usableProbe struct {
	mu sync.Mutex
	at time.Time
	ok bool
}

func targetUsable() bool {
	usableProbe.mu.Lock()
	defer usableProbe.mu.Unlock()
	ttl := 200 * time.Millisecond
	if usableProbe.ok {
		ttl = time.Second
	}
	if time.Since(usableProbe.at) < ttl {
		return usableProbe.ok
	}
	usableProbe.ok, usableProbe.at = targetNodesReady(2), time.Now()
	return usableProbe.ok
}

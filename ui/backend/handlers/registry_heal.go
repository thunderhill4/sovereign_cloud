package handlers

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	k8sclient "kubeui/backend/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Registry self-heal. Images are referenced by the registry ALIAS
// (172.18.0.2:5000), which containerd maps to the registry's real IP via a
// hosts.toml on each node. A host restart can move the registry's IP, and
// then every pull from the alias fails: the CAPK controller (a locally built
// image) goes ImagePullBackOff, its webhook stops answering, and every
// target-cluster apply fails. Fixing that used to need `make registry-fix` on
// the host, which a user on a remote browser cannot run. The backend runs on
// the host, so it re-runs scripts/fix-registry-hosts.sh itself.

const (
	registryHealInterval = 30 * time.Second
	registryHealCooldown = 2 * time.Minute
)

type stuckPod struct {
	Namespace, Name, Image string
	// restartable: owned by a ReplicaSet, so deleting it brings a fresh pod
	// at once instead of waiting out kubelet's pull backoff (up to 5 min).
	// Other owners (virt-launcher pods, bare pods) are left to kubelet.
	restartable bool
}

// stuckOnRegistryAlias returns the pods with a container waiting on a failed
// pull of an image from alias.
func stuckOnRegistryAlias(pods []corev1.Pod, alias string) []stuckPod {
	var out []stuckPod
	for _, p := range pods {
		statuses := append(append([]corev1.ContainerStatus{}, p.Status.InitContainerStatuses...), p.Status.ContainerStatuses...)
		for _, cs := range statuses {
			w := cs.State.Waiting
			if w == nil || (w.Reason != "ImagePullBackOff" && w.Reason != "ErrImagePull") {
				continue
			}
			if !strings.HasPrefix(cs.Image, alias+"/") {
				continue
			}
			restartable := false
			for _, o := range p.OwnerReferences {
				if o.Kind == "ReplicaSet" {
					restartable = true
				}
			}
			out = append(out, stuckPod{Namespace: p.Namespace, Name: p.Name, Image: cs.Image, restartable: restartable})
			break
		}
	}
	return out
}

type registryHealer struct {
	mu      sync.Mutex
	lastRun time.Time
}

var regHealer = &registryHealer{}

// mayRun reports whether a heal may run now, and if so records it. The
// cooldown keeps a fix that does not help (registry container down) from
// re-running every tick.
func (h *registryHealer) mayRun(now time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.lastRun.IsZero() && now.Sub(h.lastRun) < registryHealCooldown {
		return false
	}
	h.lastRun = now
	return true
}

// healRegistryPulls fixes the alias mapping if any pod is stuck pulling from
// it, and restarts the stuck Deployment pods. It reports whether it acted.
func healRegistryPulls(ctx context.Context, logf func(typ, msg string)) bool {
	if k8sclient.Clientset == nil {
		return false
	}
	pods, err := k8sclient.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return false
	}
	stuck := stuckOnRegistryAlias(pods.Items, registryAlias)
	if len(stuck) == 0 || !regHealer.mayRun(time.Now()) {
		return false
	}
	logf("warn", fmt.Sprintf("Registry self-heal: %d pod(s) cannot pull from %s (e.g. %s/%s, %s); re-mapping the alias to the registry's current IP",
		len(stuck), registryAlias, stuck[0].Namespace, stuck[0].Name, stuck[0].Image))
	if err := runShellQuiet("REGISTRY_ALIAS=" + registryAlias + " bash " + filepath.Join(claudeDir(), "scripts/fix-registry-hosts.sh")); err != nil {
		logf("error", "Registry self-heal failed: "+err.Error())
		return true
	}
	for _, s := range stuck {
		if !s.restartable {
			continue
		}
		if err := k8sclient.Clientset.CoreV1().Pods(s.Namespace).Delete(ctx, s.Name, metav1.DeleteOptions{}); err != nil {
			logf("warn", fmt.Sprintf("Registry self-heal: could not restart %s/%s: %v", s.Namespace, s.Name, err))
		} else {
			logf("info", fmt.Sprintf("Registry self-heal: restarted %s/%s", s.Namespace, s.Name))
		}
	}
	return true
}

// ensureRegistryPulls runs a heal before a target-cluster apply and, if it
// acted, waits for the CAPK controller whose webhook the apply goes through.
func ensureRegistryPulls(logf func(typ, msg string)) {
	if !healRegistryPulls(context.Background(), logf) {
		return
	}
	logf("info", "Waiting for the CAPK controller to come back...")
	if err := runShellQuiet("kubectl -n capk-system rollout status deploy/capk-controller-manager --timeout=90s"); err != nil {
		logf("warn", "CAPK controller not ready yet: "+err.Error())
		return
	}
	logf("success", "✓ CAPK controller ready")
}

// RunRegistryHealer checks for registry pull failures every 30s. Start once
// from main() in a goroutine.
func RunRegistryHealer(ctx context.Context) {
	logf := func(typ, msg string) { log.Printf("[registry-heal] %s: %s", typ, msg) }
	ticker := time.NewTicker(registryHealInterval)
	defer ticker.Stop()
	for {
		healRegistryPulls(ctx, logf)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

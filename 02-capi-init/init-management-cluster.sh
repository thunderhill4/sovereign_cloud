#!/usr/bin/env bash
set -euo pipefail

echo "==> Initializing CAPI management cluster..."
echo "    Infrastructure: KubeVirt (CAPK)"
echo "    Bootstrap:      k3s"
echo "    Control Plane:  k3s"

# Enable experimental features for ClusterResourceSet
export EXP_CLUSTER_RESOURCE_SET=true

clusterctl init \
  --infrastructure kubevirt \
  --bootstrap k3s \
  --control-plane k3s

# Found 2026-09-16 (cluster2 rebuild for the Kubernetes 1.35 -> 1.37 upgrade):
# the k3s bootstrap/control-plane provider manifests (still v0.3.0, unchanged)
# ship their kube-rbac-proxy sidecar as `gcr.io/kubebuilder/kube-rbac-proxy:v0.16.0`,
# a reference that no longer resolves (ImagePullBackOff, "not found") --
# apparently a dead/retired gcr.io path, though clusterctl fetches the release
# manifest fresh on every `init` so it's unclear whether this was ever pullable
# from this repo's install path. Whatever the history, the *deployment* that
# was actually running before this rebuild used
# `quay.io/brancz/kube-rbac-proxy:v0.16.0` (confirmed pullable), so that's the
# known-good pin here. Without this, both provider Deployments sit at 1/2
# ready forever and the `kubectl wait --for=condition=available` calls below
# silently time out into their `|| true` fallback -- so `make capi-init`
# reports success while KThreesControlPlane's mutating webhook is dead,
# and the first `target-cluster` deploy fails opaquely on
# "failed calling webhook ...kthreescontrolplane: connection refused".
for ns in capi-k3s-bootstrap-system capi-k3s-control-plane-system; do
  deploy="$(kubectl get deploy -n "$ns" -o name 2>/dev/null | head -1)"
  [[ -n "$deploy" ]] || continue
  if kubectl get "$deploy" -n "$ns" -o jsonpath='{.spec.template.spec.containers[?(@.name=="kube-rbac-proxy")].image}' 2>/dev/null \
      | grep -q '^gcr.io/kubebuilder/'; then
    echo "==> Patching $deploy ($ns): kube-rbac-proxy gcr.io image -> quay.io/brancz (known pullable)"
    kubectl patch "$deploy" -n "$ns" --type=strategic \
      -p='{"spec":{"template":{"spec":{"containers":[{"name":"kube-rbac-proxy","image":"quay.io/brancz/kube-rbac-proxy:v0.16.0"}]}}}}'
  fi
done

# CAPK v0.11.2 watches v1beta1 types that CAPI v1.11's helpers never match, so
# it misses the "bootstrap data ready" event and stalls control-plane VM
# creation 2-30s per deploy. Swap in the patched build; see patch-capk.sh.
bash "$(dirname "${BASH_SOURCE[0]}")/patch-capk.sh" || echo "WARN: patch-capk.sh failed; CAPK left unpatched" >&2

echo "==> Waiting for CAPI core controller..."
kubectl wait --for=condition=available deployment/capi-controller-manager \
  -n capi-system --timeout=300s

echo "==> Waiting for CAPK infrastructure controller..."
kubectl wait --for=condition=available deployment/capk-controller-manager \
  -n capk-system --timeout=300s

echo "==> Waiting for k3s bootstrap controller..."
kubectl wait --for=condition=available deployment/capi-k3s-bootstrap-controller-manager \
  -n capi-k3s-bootstrap-system --timeout=300s 2>/dev/null || \
kubectl wait --for=condition=available deployment -l cluster.x-k8s.io/provider=bootstrap-k3s \
  --all-namespaces --timeout=300s 2>/dev/null || true

echo "==> Waiting for k3s control plane controller..."
kubectl wait --for=condition=available deployment/capi-k3s-control-plane-controller-manager \
  -n capi-k3s-control-plane-system --timeout=300s 2>/dev/null || \
kubectl wait --for=condition=available deployment -l cluster.x-k8s.io/provider=control-plane-k3s \
  --all-namespaces --timeout=300s 2>/dev/null || true

echo ""
echo "==> Management cluster initialized. Provider pods:"
kubectl get pods -n capi-system
kubectl get pods -n capk-system
kubectl get pods -A -l cluster.x-k8s.io/provider=bootstrap-k3s 2>/dev/null || true
kubectl get pods -A -l cluster.x-k8s.io/provider=control-plane-k3s 2>/dev/null || true

# Raise KubeVirt's support-container CPU limits (default 10m/15m throttles the
# containerDisk init container to ~8s per VM start). Measured: time-to-ready
# median 42.7s -> 34.5s. Idempotent; see scripts/configure-kubevirt-perf.sh.
bash "$(dirname "${BASH_SOURCE[0]}")/../scripts/configure-kubevirt-perf.sh" || true

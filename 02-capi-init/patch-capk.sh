#!/usr/bin/env bash
# patch-capk.sh — run a patched CAPK controller that reacts to CAPI events.
#
# WHY: CAPK v0.11.2 is built against CAPI v1.11 but still WATCHES the v1beta1
# Machine/Cluster types, while CAPI v1.11's util map funcs and predicates
# type-assert *v1beta2 objects. Every assertion fails, so all three watches
# (KubevirtMachine<-Machine, KubevirtMachine<-Cluster, KubevirtCluster<-Cluster)
# silently drop every event. CAPK then only notices that the control plane's
# bootstrap data is ready when an unrelated event or a 10-20s requeue happens to
# fire — and, before that, it dials the target API VIP (no backend yet) with
# client-go's 30s default dial timeout, blocking the reconcile.
#
# Measured 2026-09-23 (upstream v0.11.2): "bootstrap data ready -> CP VM created"
# took 2.0s / 11.2s / 30.0s across three runs, the whole of the run-to-run
# spread (time-to-ready 31.5 / 41.8 / 62.9s). The patch
# (capk-patches/<tag>.patch) watches the v1beta2 types, skips the node
# providerID lookup while no VM can exist, and bounds workload-API calls to 5s.
#
# The patch is keyed to the installed CAPK tag, because `clusterctl init`
# installs whatever release is current. On an unpatched tag this warns and
# leaves CAPK alone — check whether upstream has fixed the watches before
# porting the patch.
#
# Idempotent. Needs: go, docker, git, the local registry on localhost:5000.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NS=capk-system
DEPLOY=capk-controller-manager
REGISTRY_PUSH="${REGISTRY_PUSH:-localhost:5000}"   # where docker pushes
REGISTRY_PULL="${REGISTRY_PULL:-172.18.0.2:5000}"  # what containerd resolves
SUFFIX=kubeui2

current="$(kubectl get deploy "$DEPLOY" -n "$NS" \
  -o jsonpath='{.spec.template.spec.containers[?(@.name=="manager")].image}')"
tag="${current##*:}"
tag="${tag%-"$SUFFIX"}"
patch="$HERE/capk-patches/$tag.patch"
target="$REGISTRY_PULL/capk-manager:$tag-$SUFFIX"

if [ "$current" = "$target" ]; then
  echo "==> CAPK already patched ($current)"
  exit 0
fi
if [ ! -f "$patch" ]; then
  echo "WARN: no CAPK patch for $tag (image $current) — leaving CAPK unpatched." >&2
  echo "      Control-plane VM creation may stall 2-30s per deploy; see $0." >&2
  exit 0
fi

if ! docker manifest inspect --insecure "$REGISTRY_PUSH/capk-manager:$tag-$SUFFIX" >/dev/null 2>&1; then
  echo "==> Building patched CAPK $tag"
  work="$(mktemp -d)"
  trap 'rm -rf "$work"' EXIT
  git clone -q --depth 1 --branch "$tag" \
    https://github.com/kubernetes-sigs/cluster-api-provider-kubevirt "$work/capk"
  git -C "$work/capk" apply "$patch"
  (cd "$work/capk" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" -o bin/manager .)
  # Layer only the binary onto the official image: same base, user, entrypoint.
  printf 'FROM quay.io/capk/capk-manager:%s\nCOPY bin/manager /manager\n' "$tag" \
    > "$work/capk/Dockerfile.kubeui"
  docker build -q -f "$work/capk/Dockerfile.kubeui" \
    -t "$REGISTRY_PUSH/capk-manager:$tag-$SUFFIX" "$work/capk" >/dev/null
  docker push -q "$REGISTRY_PUSH/capk-manager:$tag-$SUFFIX"
fi

echo "==> Switching $DEPLOY to $target"
kubectl set image "deploy/$DEPLOY" -n "$NS" "manager=$target"
kubectl rollout status "deploy/$DEPLOY" -n "$NS" --timeout=180s

#!/usr/bin/env bash
#
# time-to-ready-hosted.sh — time apply -> N agent Nodes Ready for the
# hosted-control-plane target cluster (docs/sub-60s-cluster-strategy.md §12).
#
# time-to-ready-by-name.sh requires a `-cp-` Node, which this cluster does not
# have. This script waits for N Ready Nodes, all of whose names start with the
# cluster's worker prefix, through the hosted API (the server is resident, so
# the kubeconfig is valid before any VM exists).
#
# Each run, from a clean slate:
#   delete the Cluster, wait for VMs, VMIs and launcher pods to be gone, then
#   delete stale Nodes in the hosted API. CAPI does not delete them here.
# Clock: t0 = kubectl apply. Reports VMIs Running and Nodes Ready per run,
# then the median.
#
# Usage: ./scripts/time-to-ready-hosted.sh [--runs N] [--nodes N] [--manifest FILE]
set -uo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

RUNS=3; NODES=2; MANIFEST=03-target-cluster/target-cluster-hosted.yaml
CLUSTER_NAME="${CLUSTER_NAME:-target-cluster}"; TIMEOUT="${TIMEOUT:-180}"
while [ $# -gt 0 ]; do
  case "$1" in
    --runs) RUNS="$2"; shift 2 ;; --nodes) NODES="$2"; shift 2 ;;
    --manifest) MANIFEST="$2"; shift 2 ;; *) echo "unknown arg $1" >&2; exit 2 ;;
  esac
done
KC=$(mktemp); trap 'rm -f "$KC"' EXIT
clusterctl get kubeconfig "$CLUSTER_NAME" >"$KC" 2>/dev/null ||
  { echo "no ${CLUSTER_NAME}-kubeconfig — run 'make hosted-cp' first" >&2; exit 1; }
tk() { KUBECONFIG="$KC" timeout 5 kubectl "$@"; }

teardown() {
  kubectl delete cluster "$CLUSTER_NAME" --ignore-not-found --wait=true --timeout=180s >/dev/null 2>&1
  local i
  for i in $(seq 1 120); do
    [ -z "$(kubectl get vm,vmi,pods -n default -o name 2>/dev/null | grep -E "$CLUSTER_NAME|virt-launcher-$CLUSTER_NAME")" ] && break
    sleep 1
  done
  tk delete nodes --all --wait=true >/dev/null 2>&1
}

totals=()
for r in $(seq 1 "$RUNS"); do
  teardown
  t0=$(date +%s.%N); vmrun=""
  kubectl apply -f "$MANIFEST" >/dev/null 2>&1 || { echo "apply failed" >&2; exit 1; }
  while :; do
    el=$(awk -v a="$(date +%s.%N)" -v b="$t0" 'BEGIN{printf "%.1f", a-b}')
    if [ -z "$vmrun" ] && [ "$(kubectl get vmi -n default --no-headers 2>/dev/null | grep "^$CLUSTER_NAME" | grep -c ' Running ')" -ge "$NODES" ]; then
      vmrun=$el
    fi
    ready=$(tk get nodes --no-headers 2>/dev/null | awk -v p="$CLUSTER_NAME-" 'index($1,p)==1 && $2=="Ready"' | wc -l)
    if [ "$ready" -ge "$NODES" ]; then
      echo "run $r: VMIs Running t+${vmrun:-?}s, $NODES Nodes Ready t+${el}s"; totals+=("$el"); break
    fi
    if [ "${el%.*}" -ge "$TIMEOUT" ]; then echo "run $r: TIMEOUT (${ready}/${NODES} Ready)"; break; fi
    sleep 0.25
  done
done
[ "${#totals[@]}" -gt 0 ] && printf 'median: %s s  (%s)\n' \
  "$(printf '%s\n' "${totals[@]}" | sort -n | awk '{a[NR]=$1} END{print (NR%2)?a[(NR+1)/2]:(a[NR/2]+a[NR/2+1])/2}')" "${totals[*]}"

#!/usr/bin/env bash
# time-to-ready-by-name.sh — measure target-cluster boot time, counting nodes
# BY NAME rather than by number.
#
# WHY THIS EXISTS
# ---------------
# `scripts/time-to-ready.sh` and `scripts/phase-timings.sh` both wait for
# "2 nodes Ready" by COUNT. The :warm golden image carries a ghost Node object
# left over from the bake VM (`ubuntu-bake-vm-warm`), still marked Ready, which
# the control plane's async `warm-ghost-node-cleanup` step only deletes after
# startup. So for the first seconds of every warm build the count reaches 2 from
# control-plane + ghost, before the real worker has joined — and the clock stops
# early. On 2026-09-17 that produced a reported 23.9s "both nodes" time whose
# real value was 33.7s: the instrument was reporting the control plane alone.
#
# This script waits for a node whose name contains `-cp-` AND a node whose name
# contains `-workers-`, each Ready, and reports them separately. A node matching
# neither pattern (the ghost) is ignored and named in the output so it is visible
# rather than silently counted.
#
# It also FAILS LOUDLY on a missing timestamp instead of treating it as zero,
# which is the second half of the same bug in phase-timings.sh.
#
# Usage:
#   ./scripts/time-to-ready-by-name.sh [<manifest>] [--runs N]
#
# Defaults: 03-target-cluster/target-cluster-warm.yaml, 1 run.
# For the warm manifest the fixed CA/token secrets are seeded first, as
# `make target-cluster-warm` does.
#
# Requires: kubectl, clusterctl.
set -euo pipefail

# Accept the manifest and --runs in any order, so that both of these work:
#   ./scripts/time-to-ready-by-name.sh --runs 3
#   ./scripts/time-to-ready-by-name.sh 03-target-cluster/target-cluster-warm.yaml --runs 3
# The earlier version took the manifest positionally first, so `--runs 3` alone
# was read as a filename and died with "manifest not found: --runs".
MANIFEST="03-target-cluster/target-cluster-warm.yaml"
RUNS=1
while [ $# -gt 0 ]; do
  case "$1" in
    --runs) RUNS="${2:-1}"; shift 2 ;;
    --runs=*) RUNS="${1#*=}"; shift ;;
    *) MANIFEST="$1"; shift ;;
  esac
done

CLUSTER_NAME="${CLUSTER_NAME:-target-cluster}"
TIMEOUT="${TIMEOUT:-300}"
POLL="${POLL:-0.5}"
CP_MATCH="${CP_MATCH:--cp-}"
WORKER_MATCH="${WORKER_MATCH:--workers-}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

[ -f "$MANIFEST" ] || { echo "ERROR: manifest not found: $MANIFEST" >&2; exit 1; }

# ── one run ────────────────────────────────────────────────────────────────
run_once() {
  local kc; kc="$(mktemp)"
  # shellcheck disable=SC2064
  trap "rm -f '$kc'" RETURN

  echo "==> Tearing down any existing $CLUSTER_NAME"
  kubectl delete cluster "$CLUSTER_NAME" --ignore-not-found --timeout=180s >/dev/null 2>&1 || true
  # Wait until the previous run's VMs and their launcher pods are really gone.
  # `delete cluster` returns once the Cluster object is gone, while the old
  # virt-launcher pods are still terminating and holding 8Gi + 6Gi. The next
  # control-plane pod then sits in FailedScheduling "Insufficient memory" for up
  # to 12s (found 2026-09-30; every run that day was affected, the slow CAPI
  # chain only partly hid it). A real deploy onto a clean node never waits on
  # this, so it must not be in the number.
  local w=0
  while [ "$w" -lt 180 ]; do
    [ -z "$(kubectl get vmi,vm -o name 2>/dev/null | grep "$CLUSTER_NAME" || true)" ] &&
    [ -z "$(kubectl get pods -o name 2>/dev/null | grep "virt-launcher-$CLUSTER_NAME" || true)" ] && break
    sleep 1; w=$((w+1))
  done
  echo "    previous VMs and launcher pods gone after ${w}s"

  if [[ "$MANIFEST" == *target-cluster-warm* ]]; then
    echo "==> Seeding warm CA/token secrets"
    ./scripts/seed-cluster-secrets.sh >/dev/null
  fi

  echo "==> Applying $MANIFEST"
  local start_ns; start_ns=$(date +%s%N)
  kubectl apply -f "$MANIFEST" >/dev/null

  local cp_s="" worker_s="" ghosts=""
  local deadline=$(( $(date +%s) + TIMEOUT ))

  while :; do
    if clusterctl get kubeconfig "$CLUSTER_NAME" > "$kc" 2>/dev/null && [ -s "$kc" ]; then
      # name=ReadyStatus, one per line
      # --request-timeout bounds the poll. Until the CP VM exists, the VIP has no
      # backend and a dial to it can blackhole for client-go's 30s default, so an
      # unbounded poll overstated time-to-ready by ~4s (measured 2026-09-23: node
      # Ready timestamps +24s/+31s, script reported 34.8s for both).
      local lines
      lines=$(kubectl --kubeconfig="$kc" --request-timeout=2s get nodes \
        -o 'jsonpath={range .items[*]}{.metadata.name}={.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}' \
        2>/dev/null || true)

      while IFS= read -r line; do
        [ -n "$line" ] || continue
        local name="${line%%=*}" status="${line##*=}"
        [ "$status" = "True" ] || continue
        local now_s
        now_s=$(echo "scale=1; ($(date +%s%N) - $start_ns)/1000000000" | bc)
        case "$name" in
          *"$CP_MATCH"*)     [ -z "$cp_s" ]     && cp_s="$now_s"     && echo "    control plane Ready: ${cp_s}s  ($name)" ;;
          *"$WORKER_MATCH"*) [ -z "$worker_s" ] && worker_s="$now_s" && echo "    worker Ready:        ${worker_s}s  ($name)" ;;
          *)                 case "$ghosts" in *"$name"*) ;; *) ghosts="$ghosts $name"
                               echo "    IGNORED (matches neither pattern): $name — Ready, not counted" ;; esac ;;
        esac
      done <<< "$lines"
    fi

    [ -n "$cp_s" ] && [ -n "$worker_s" ] && break

    if [ "$(date +%s)" -ge "$deadline" ]; then
      echo "ERROR: timed out after ${TIMEOUT}s." >&2
      [ -z "$cp_s" ]     && echo "ERROR: control-plane node (*${CP_MATCH}*) never reported Ready." >&2
      [ -z "$worker_s" ] && echo "ERROR: worker node (*${WORKER_MATCH}*) never reported Ready." >&2
      echo "       A missing timestamp is a failure, not a zero — not reporting a total." >&2
      return 1
    fi
    sleep "$POLL"
  done

  [ -n "$ghosts" ] && echo "    (ghost nodes ignored:$ghosts)"
  echo "RESULT cp=${cp_s} worker=${worker_s}"
}

# ── main ───────────────────────────────────────────────────────────────────
echo "=== time-to-ready, by name ==="
echo "manifest: $MANIFEST   runs: $RUNS"
echo "patterns: control plane *${CP_MATCH}*  worker *${WORKER_MATCH}*"
echo "host load before: $(cut -d' ' -f1-3 /proc/loadavg)"
echo

cps=(); workers=()
for i in $(seq 1 "$RUNS"); do
  echo "--- run $i of $RUNS ---"
  out="$(run_once)" || exit 1
  echo "$out" | grep -v '^RESULT'
  line="$(echo "$out" | grep '^RESULT')"
  cps+=("$(echo "$line"     | sed 's/.*cp=\([0-9.]*\).*/\1/')")
  workers+=("$(echo "$line" | sed 's/.*worker=\([0-9.]*\).*/\1/')")
  echo
done

median() { printf '%s\n' "$@" | sort -n | awk '{a[NR]=$1} END{print (NR%2)?a[(NR+1)/2]:(a[NR/2]+a[NR/2+1])/2}'; }

echo "=== summary ==="
echo "control plane Ready : ${cps[*]}    median $(median "${cps[@]}")s"
echo "both nodes Ready    : ${workers[*]}    median $(median "${workers[@]}")s"
echo
echo "Note: 'both nodes' is the worker's time because the worker joins last."
echo "Host load after: $(cut -d' ' -f1-3 /proc/loadavg)"

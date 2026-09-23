#!/usr/bin/env bash
#
# ch-recovery-drill.sh — time the full "replace instead of repair" cycle.
#
#   destroy → infrastructure gone → seed → rebuild → workload → first HTTP 200
#
# WHY THIS EXISTS, AND WHY IT IS NOT scripts/time-to-ready-by-name.sh
# -------------------------------------------------------------------
# The book publishes a build time (33.7 s) and a teardown time (18.9 s, N=1)
# but has never timed the cycle a reader actually cares about: how long from
# "destroy it" to "it is serving again". time-to-ready-by-name.sh cannot do it:
# it discards the delete, starts its build clock while VMs may still be
# terminating, has no workload phase, and captures host load once per
# invocation rather than once per run. It IS correct about the ghost node, and
# the by-name matching below is copied from it.
#
# THE BOUNDARIES ARE PRE-REGISTERED (see measurements/README.md, committed
# before the first measured run). The estimate straddles one minute. Moving a
# boundary after seeing the data in order to land under 60 s is exactly the
# failure this book exists to avoid, so the marks are fixed in advance.
#
#   T0  destroy command issued
#   T1  delete returns
#   T2  infra gone: no target-cluster API objects AND no qemu process
#   T3  secrets seeded and all four confirmed
#   T4  build clock origin (kubectl apply)
#   T5  control-plane node Ready, matched by NAME
#   T6  worker node Ready, matched by NAME
#   T7  workload applied (negative control has already failed)
#   T8  workload pod Ready
#   T9  first HTTP 200 through the Service
#
#   Headline MTTR = T9 - T0.
#
# Usage:  ./measurements/ch-recovery-drill.sh [runs]     (default 5)
#         REHEARSE=1 ./measurements/ch-recovery-drill.sh 1   # discarded run
#
set -uo pipefail

REPO="${REPO:-/mnt/mil}"
BOOK="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNS="${1:-5}"
MANIFEST="03-target-cluster/target-cluster-warm.yaml"
WORKLOAD="$BOOK/measurements/drill-workload.yaml"
KC="/tmp/drill-kubeconfig"
STAMP="$(date +%Y%m%d-%H%M%S)"
TSV="$BOOK/measurements/ch17-recovery-drill-${STAMP}.tsv"
CTX="kind-cluster2"

cd "$REPO" || exit 1
now_ns() { date +%s%N; }
el() { echo "scale=2; ($2 - $1)/1000000000" | bc; }   # elapsed seconds
say() { printf '%s\n' "$*" >&2; }

# ── pre-flight gates — abort, never adjust ────────────────────────────────
gate_fail=0
gate() { if eval "$2" >/dev/null 2>&1; then say "  PASS  $1"; else say "  FAIL  $1"; gate_fail=1; fi; }

say "=== Pre-flight gates ==="
gate "context is $CTX" "[ \"\$(kubectl config current-context)\" = $CTX ]"

PERF="$(kubectl get kubevirt kubevirt -n kubevirt -o jsonpath='{.spec.configuration.supportContainerResources}' 2>/dev/null)"
case "$PERF" in
  *'"cpu":"1"'*) say "  PASS  kubevirt-perf applied (supportContainerResources cpu:1)" ;;
  *) say "  FAIL  kubevirt-perf missing or degraded — worth ~8s per VM start"; gate_fail=1 ;;
esac

WARM_ID="$(docker exec cluster2-control-plane crictl images 2>/dev/null | awk '/ubuntu-noble-k3s.*warm/{print $3}' | head -1)"
if [ -n "$WARM_ID" ]; then
  say "  PASS  :warm image cached, id=$WARM_ID"
else
  say "  FAIL  :warm image NOT cached — ensure-warm-image would bake for 5-8 min. Aborting rather than baking."
  gate_fail=1
fi

if ss -ltn 2>/dev/null | grep -qE ':(8080|8081|5173)\b'; then
  say "  FAIL  UI backend / warm-pool controller appears to be running — it would rebuild a standby mid-drill"; gate_fail=1
else
  say "  PASS  no UI backend or warm-pool controller listening"
fi

MEMAV="$(awk '/MemAvailable/{print int($2/1024)}' /proc/meminfo)"
say "  INFO  MemAvailable ${MEMAV} MiB (two launchers request 14920 MiB; the running cluster's share frees on delete)"

OLLAMA_LAST="$(kubectl get sympoziumschedule ollama-warm -n sympozium-system -o jsonpath='{.status.lastRunTime}' 2>/dev/null || echo none)"
say "  INFO  ollama-warm schedule lastRunTime=$OLLAMA_LAST (4-min cadence; flag any run where this moves)"

[ "$gate_fail" -eq 0 ] || { say ""; say "ABORTING: a pre-flight gate failed. Fix it; do not adjust the drill."; exit 1; }

# ── output ────────────────────────────────────────────────────────────────
printf 'run\tstatus\tmttr_T9_T0\tdelete_returns_T1_T0\tinfra_gone_T2_T0\tseed_T3_T2\tbuild_both_T6_T4\tbuild_cp_T5_T4\tworkload_T9_T7\tcp_node\tworker_node\tghosts\tworkload_node\tpulled\tkubesystem_created\tload_before\tload_after\tmemav_before\n' > "$TSV"

run_one() {
  local run="$1" ghosts="" cp_s="" wk_s="" cp_name="" wk_name=""
  local load_before load_after memav_before
  load_before="$(cut -d' ' -f1 /proc/loadavg)"
  memav_before="$(awk '/MemAvailable/{print int($2/1024)}' /proc/meminfo)"

  say ""; say "--- run $run of $RUNS ---"

  # T0: destroy
  local T0 T1 T2 T3 T4 T5 T6 T7 T8 T9
  T0=$(now_ns)
  kubectl delete cluster target-cluster --ignore-not-found --timeout=240s >/dev/null 2>&1
  T1=$(now_ns)
  say "    delete returned        $(el "$T0" "$T1")s"

  # T2: objects gone AND qemu gone. Poll at 0.25s: 2s polling quantises an
  # 18.9s figure by ~10%.
  local deadline=$(( $(date +%s) + 240 ))
  while :; do
    local objs qemu
    objs="$(kubectl get vm,vmi,kubevirtmachine,machine,machineset,machinedeployment,kthreescontrolplane,cluster -n default -o name 2>/dev/null | grep -c target-cluster)"
    # ps, not pgrep: `pgrep -f` matches the shell whose own command line
    # contains the pattern, so it can never reach zero. Anchoring on the
    # process NAME via ps cannot self-match (our comm is ps/awk).
    qemu="$(ps -eo comm=,args= | awk '/^qemu/ && /guest=default_target-cluster/' | wc -l)"
    [ "$objs" -eq 0 ] && [ "$qemu" -eq 0 ] && break
    [ "$(date +%s)" -ge "$deadline" ] && { say "    TIMEOUT waiting for teardown (objs=$objs qemu=$qemu)"; echo "FAILED"; return 1; }
    sleep 0.25
  done
  T2=$(now_ns)
  say "    infrastructure gone    $(el "$T0" "$T2")s"

  # T3: seed. The token secret is garbage-collected with the cluster; the CA
  # secrets survive. Seeding every run is mandatory, not merely idempotent.
  ./scripts/seed-cluster-secrets.sh >/dev/null 2>&1
  local missing=0
  for s in ca cca etcd token; do
    kubectl get secret "target-cluster-$s" -n default >/dev/null 2>&1 || missing=1
  done
  [ "$missing" -eq 0 ] || { say "    seed incomplete"; echo "FAILED"; return 1; }
  T3=$(now_ns)

  # T4: build clock origin
  rm -f "$KC"
  T4=$(now_ns)
  kubectl apply -f "$MANIFEST" >/dev/null 2>&1

  # T5/T6: nodes Ready, BY NAME. A node matching neither pattern is the ghost
  # baked into the :warm image — recorded and named, never counted.
  deadline=$(( $(date +%s) + 300 ))
  while :; do
    if [ ! -s "$KC" ]; then
      clusterctl get kubeconfig target-cluster > "$KC" 2>/dev/null || true
    fi
    if [ -s "$KC" ]; then
      local lines
      lines="$(kubectl --kubeconfig="$KC" get nodes -o 'jsonpath={range .items[*]}{.metadata.name}={.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}' 2>/dev/null)"
      while IFS= read -r line; do
        [ -n "$line" ] || continue
        local name="${line%%=*}" st="${line##*=}"
        [ "$st" = "True" ] || continue
        case "$name" in
          *-cp-*)      [ -z "$cp_s" ] && { cp_s=$(now_ns); cp_name="$name"; say "    control plane Ready    $(el "$T4" "$cp_s")s  ($name)"; } ;;
          *-workers-*) [ -z "$wk_s" ] && { wk_s=$(now_ns); wk_name="$name"; say "    worker Ready           $(el "$T4" "$wk_s")s  ($name)"; } ;;
          *)           case "$ghosts" in *"$name"*) ;; *) ghosts="$ghosts $name"; say "    GHOST IGNORED: $name (Ready, not counted)";; esac ;;
        esac
      done <<< "$lines"
    fi
    [ -n "$cp_s" ] && [ -n "$wk_s" ] && break
    [ "$(date +%s)" -ge "$deadline" ] && { say "    TIMEOUT waiting for nodes (cp=${cp_s:-none} worker=${wk_s:-none})"; echo "FAILED"; return 1; }
    sleep 0.5
  done
  T5="$cp_s"; T6="$wk_s"

  # the artifact that stops this being read as provision-from-scratch
  local ks_created
  ks_created="$(kubectl --kubeconfig="$KC" get ns kube-system -o jsonpath='{.metadata.creationTimestamp}' 2>/dev/null)"

  # negative control MUST fail before the workload exists. Never piped: a pipe
  # converts exit 1 into exit 0 (verified live; same class as the tail bug).
  kubectl --kubeconfig="$KC" get --raw "/api/v1/namespaces/drill/services/drill-web:80/proxy/index.html" >/dev/null 2>&1
  if [ $? -eq 0 ]; then say "    NEGATIVE CONTROL PASSED — stale service present, run invalid"; echo "FAILED"; return 1; fi

  # T7-T9: workload
  T7=$(now_ns)
  kubectl --kubeconfig="$KC" apply -f "$WORKLOAD" >/dev/null 2>&1
  deadline=$(( $(date +%s) + 180 ))
  while :; do
    kubectl --kubeconfig="$KC" get --raw "/api/v1/namespaces/drill/services/drill-web:80/proxy/index.html" >/dev/null 2>&1
    [ $? -eq 0 ] && break
    [ "$(date +%s)" -ge "$deadline" ] && { say "    TIMEOUT waiting for HTTP 200"; echo "FAILED"; return 1; }
    sleep 0.25
  done
  T9=$(now_ns)
  T8="$T9"

  local wl_node pulled
  wl_node="$(kubectl --kubeconfig="$KC" -n drill get pod -l app=drill-web -o jsonpath='{.items[0].spec.nodeName}' 2>/dev/null)"
  pulled="$(kubectl --kubeconfig="$KC" -n drill get events --field-selector reason=Pulling --no-headers 2>/dev/null | wc -l)"

  say "    FIRST HTTP 200         $(el "$T7" "$T9")s after apply"
  say "    >>> MTTR (T9-T0)       $(el "$T0" "$T9")s"

  load_after="$(cut -d' ' -f1 /proc/loadavg)"
  printf '%s\tOK\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$run" "$(el "$T0" "$T9")" "$(el "$T0" "$T1")" "$(el "$T0" "$T2")" "$(el "$T2" "$T3")" \
    "$(el "$T4" "$T6")" "$(el "$T4" "$T5")" "$(el "$T7" "$T9")" \
    "$cp_name" "$wk_name" "${ghosts:-none}" "$wl_node" "$pulled" "$ks_created" \
    "$load_before" "$load_after" "$memav_before" >> "$TSV"

  # keep run 1's credential to test the shared-identity claim at the end
  if [ "$run" = "1" ]; then cp "$KC" /tmp/drill-kubeconfig-run1; fi
  echo "OK"
}

say ""; say "=== Recovery drill: $RUNS runs ==="
for i in $(seq 1 "$RUNS"); do
  res="$(run_one "$i" | tail -1)"
  if [ "$res" != "OK" ]; then
    printf '%s\tFAILED\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n' "$i" >> "$TSV"
    say "    run $i FAILED — recorded as a failed row, not dropped"
  fi
  sleep 20   # settle
done

say ""; say "=== Raw data: $TSV ==="
column -t -s$'\t' "$TSV" >&2

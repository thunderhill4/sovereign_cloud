#!/usr/bin/env bash
#
# verify-hosted-cp.sh — deterministic pass/fail checks for the hosted control
# plane and its agents-only target cluster. These are the Phase 0 spike
# criteria from docs/sub-60s-cluster-strategy.md §12. Never trust a single
# "nodes Ready" count: this checks identity, CAPI wiring and the
# apiserver->kubelet tunnel separately.
#
#   1. server pod Ready, Service holds the VIP, /v1-k3s/readyz 200 through it
#   2. Cluster has no controlPlaneRef and its infrastructure is provisioned
#   3. every Machine is Running with a NodeRef
#   4. every target Node is Ready, has a providerID, and matches a Machine
#   5. on EVERY node: a pod runs, and `kubectl logs` + `kubectl exec` work
#      (both go apiserver -> kubelet, i.e. through the k3s agent tunnel,
#      the path --disable-agent could break)
#   6. CoreDNS is Running on an agent VM
#
# Usage: ./scripts/verify-hosted-cp.sh [expected-node-count]   (default 2)
set -uo pipefail

EXPECTED="${1:-2}"
CLUSTER_NAME="${CLUSTER_NAME:-target-cluster}"
NS="${NAMESPACE:-default}"
CP_NS="${CP_NAMESPACE:-target-cp}"
VIP="${API_LB_IP:-172.18.255.215}"
KC="$(mktemp)"; trap 'rm -f "$KC"' EXIT

pass=0; fail=0
ok()  { echo "  PASS  $*"; pass=$((pass+1)); }
bad() { echo "  FAIL  $*"; fail=$((fail+1)); }
check() { local d="$1"; shift; if "$@" >/dev/null 2>&1; then ok "$d"; else bad "$d"; fi; }
tk() { KUBECONFIG="$KC" timeout 30 kubectl "$@"; }

echo "== hosted control plane"
check "server pod Ready" kubectl -n "$CP_NS" wait --for=condition=Ready pod -l app=k3s-server --timeout=5s
lb=$(kubectl -n "$CP_NS" get svc k3s-server -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
[ "$lb" = "$VIP" ] && ok "Service k3s-server holds $VIP" || bad "Service k3s-server LB IP is '${lb:-none}', want $VIP"
# Unauthenticated /readyz is 401 on this server, so probe the supervisor's
# /v1-k3s/readyz with the join token, as the agents and the pod's probe do.
code=$(curl -ks -o /dev/null -w '%{http_code}' --max-time 3 -u "node:${WARM_TOKEN:-f00dcafef00dcafef00dcafef00dcafe}" "https://$VIP:6443/v1-k3s/readyz")
[ "$code" = 200 ] && ok "/v1-k3s/readyz on $VIP = 200" || bad "/v1-k3s/readyz on $VIP = ${code:-no answer}"

echo "== CAPI wiring"
if clusterctl get kubeconfig "$CLUSTER_NAME" >"$KC" 2>/dev/null && [ -s "$KC" ]; then
  ok "clusterctl get kubeconfig"
  iss=$(sed -n 's/^ *client-certificate-data: //p' "$KC" | base64 -d | openssl x509 -noout -issuer 2>/dev/null)
  [ "$iss" = "issuer=CN=k3s-client-ca" ] && ok "kubeconfig client cert is client-ca signed" \
    || bad "kubeconfig client cert issuer: ${iss:-unreadable} (CAPI regenerated it? re-run FORCE_KUBECONFIG=1 seed-hosted-cp.sh)"
else
  bad "clusterctl get kubeconfig (is ${CLUSTER_NAME}-kubeconfig seeded?)"
fi
cpref=$(kubectl -n "$NS" get cluster "$CLUSTER_NAME" -o jsonpath='{.spec.controlPlaneRef.name}' 2>/dev/null)
[ -z "$cpref" ] && ok "Cluster has no controlPlaneRef" || bad "Cluster has controlPlaneRef '$cpref' (legacy manifest?)"
infra=$(kubectl -n "$NS" get cluster "$CLUSTER_NAME" -o jsonpath='{.status.initialization.infrastructureProvisioned}{.status.infrastructureReady}' 2>/dev/null)
[[ "$infra" == *true* ]] && ok "infrastructure provisioned" || bad "infrastructure not provisioned ('$infra')"

mapfile -t machines < <(kubectl -n "$NS" get machines -l "cluster.x-k8s.io/cluster-name=$CLUSTER_NAME" \
  -o jsonpath='{range .items[*]}{.metadata.name} {.status.phase} {.status.nodeRef.name}{"\n"}{end}' 2>/dev/null)
[ "${#machines[@]}" -eq "$EXPECTED" ] && ok "${#machines[@]} Machines" || bad "${#machines[@]} Machines, want $EXPECTED"
declare -A noderef=()
for m in "${machines[@]}"; do
  read -r name phase node <<<"$m"
  [ "$phase" = Running ] && [ -n "$node" ] && ok "Machine $name Running, nodeRef $node" \
    || bad "Machine $name phase=${phase:-?} nodeRef=${node:-none}"
  [ -n "$node" ] && noderef[$node]=1
done

echo "== target nodes"
mapfile -t nodes < <(tk get nodes -o jsonpath='{range .items[*]}{.metadata.name} {.status.conditions[?(@.type=="Ready")].status} {.spec.providerID}{"\n"}{end}' 2>/dev/null)
[ "${#nodes[@]}" -eq "$EXPECTED" ] && ok "${#nodes[@]} Nodes" || bad "${#nodes[@]} Nodes, want $EXPECTED (stale Nodes from a previous wake?)"
nodenames=()
for n in "${nodes[@]}"; do
  read -r name ready pid <<<"$n"
  nodenames+=("$name")
  [ "$ready" = True ] && ok "Node $name Ready" || bad "Node $name Ready=${ready:-?}"
  [ -n "$pid" ] && ok "Node $name providerID $pid" || bad "Node $name has no providerID (CAPK could not match it by VM name)"
  [ -n "${noderef[$name]:-}" ] && ok "Node $name is a Machine's nodeRef" || bad "Node $name matches no Machine"
done

echo "== apiserver -> kubelet (agent tunnel), per node"
img=$(tk get nodes -o jsonpath='{range .items[*].status.images[*]}{.names[*]}{"\n"}{end}' 2>/dev/null |
      tr ' ' '\n' | grep -E 'busybox' | grep -v '@sha256' | head -1)
if [ -z "$img" ]; then
  bad "no busybox image on the nodes to test with"
else
  for n in "${nodenames[@]}"; do
    pod="hosted-cp-verify-$(echo "$n" | tail -c 6)"
    tk delete pod "$pod" -n default --ignore-not-found --wait=true >/dev/null 2>&1
    tk run "$pod" -n default --image="$img" --image-pull-policy=IfNotPresent --restart=Never \
      --overrides="{\"spec\":{\"nodeName\":\"$n\"}}" -- sh -c 'echo hosted-cp-ok; sleep 300' >/dev/null 2>&1
    if tk wait --for=condition=Ready pod/"$pod" -n default --timeout=60s >/dev/null 2>&1; then
      ok "pod Running on $n"
      tk logs "$pod" -n default 2>/dev/null | grep -q hosted-cp-ok && ok "kubectl logs via $n" || bad "kubectl logs via $n"
      [ "$(tk exec "$pod" -n default -- echo exec-ok 2>/dev/null)" = exec-ok ] && ok "kubectl exec via $n" || bad "kubectl exec via $n"
    else
      bad "pod did not start on $n"
    fi
    tk delete pod "$pod" -n default --ignore-not-found --wait=false >/dev/null 2>&1
  done
fi

echo "== CoreDNS"
dnsnode=$(tk -n kube-system get pods -l k8s-app=kube-dns -o jsonpath='{.items[0].spec.nodeName} {.items[0].status.phase}' 2>/dev/null)
[[ "$dnsnode" == *" Running" ]] && ok "CoreDNS Running on ${dnsnode% *}" || bad "CoreDNS: '${dnsnode:-not found}'"

echo
echo "RESULT: $pass passed, $fail failed"
[ "$fail" -eq 0 ]

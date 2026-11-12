#!/usr/bin/env bash
#
# seed-hosted-cp.sh — create the Secrets the hosted control plane and the
# agents-only target cluster need. Run it before `kubectl apply -k
# 03-target-cluster/hosted-cp` and before applying target-cluster-hosted.yaml.
# `make hosted-cp` does both.
#
#   target-cp/hosted-cp-ca       the 11 fixed warm CA files, which the k3s server
#                                pod copies into its tls/ dir before every start
#   default/<cluster>-kubeconfig an admin kubeconfig for https://<VIP>:6443, with
#                                a client cert signed by warm-ca/client-ca
#
# Why we write the kubeconfig Secret ourselves: with no controlPlaneRef, CAPI's
# cluster controller creates <cluster>-kubeconfig on its own if it is missing.
# That cert would be signed by the <cluster>-ca (SERVER) CA, which k3s does not
# accept for clients. CAPI only creates the Secret when it is missing, and it
# rotates it only when the cert is close to expiry, so a long-lived cert seeded
# here is the one every consumer gets:
#   clusterctl get kubeconfig, CAPK's providerID patching, CAPI's NodeRef
#   lookup, publishTargetKubeconfig, 06-sympozium/refresh-target-kubeconfig.sh.
# The Secret has no ownerReference, so it survives `kubectl delete cluster`.
# That is what lets an on-demand wake reuse it.
#
# Idempotent. An existing kubeconfig Secret is kept unless FORCE_KUBECONFIG=1.
# DEMO ONLY: same fixed-credential posture as the warm path (warm-ca/).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CA_DIR="${CA_DIR:-${SCRIPT_DIR}/../03-target-cluster/warm-ca}"
CLUSTER_NAME="${CLUSTER_NAME:-target-cluster}"
NAMESPACE="${NAMESPACE:-default}"
CP_NAMESPACE="${CP_NAMESPACE:-target-cp}"
API_LB_IP="${API_LB_IP:-172.18.255.215}"
FORCE_KUBECONFIG="${FORCE_KUBECONFIG:-0}"
DRY_RUN="${DRY_RUN:-0}"   # 1 = print the manifests instead of applying them

CA_FILES=(server-ca.crt server-ca.key client-ca.crt client-ca.key
          request-header-ca.crt request-header-ca.key
          etcd/server-ca.crt etcd/server-ca.key etcd/peer-ca.crt etcd/peer-ca.key
          service.key)
for f in "${CA_FILES[@]}"; do
  [ -f "${CA_DIR}/${f}" ] || { echo "ERROR: ${CA_DIR}/${f} missing — run scripts/gen-warm-ca.sh first." >&2; exit 1; }
done

apply() { if [ "$DRY_RUN" = 1 ]; then cat; echo "---"; else kubectl apply -f -; fi; }

# The legacy warm cluster's CAPK control-plane Service owns the same VIP. Two
# owners of one MetalLB IP silently split traffic, so refuse instead.
if [ "$DRY_RUN" != 1 ] &&
   kubectl get kthreescontrolplane "${CLUSTER_NAME}-control-plane" -n "$NAMESPACE" >/dev/null 2>&1; then
  echo "ERROR: ${CLUSTER_NAME} is a legacy VM-control-plane cluster and holds ${API_LB_IP}." >&2
  echo "       Delete it first: kubectl delete cluster ${CLUSTER_NAME}" >&2
  exit 1
fi

b64() { base64 -w0 "$1"; }

echo "==> ${CP_NAMESPACE}/hosted-cp-ca (warm CA files)" >&2
{
  cat <<YAML
apiVersion: v1
kind: Namespace
metadata:
  name: ${CP_NAMESPACE}
---
apiVersion: v1
kind: Secret
metadata:
  name: hosted-cp-ca
  namespace: ${CP_NAMESPACE}
type: Opaque
data:
YAML
  # Secret keys cannot contain '/', so etcd/x.crt becomes etcd-x.crt; the
  # deployment's init container maps the names back.
  for f in "${CA_FILES[@]}"; do
    printf '  %s: %s\n' "${f//\//-}" "$(b64 "${CA_DIR}/${f}")"
  done
} | apply

if [ "$DRY_RUN" != 1 ] && [ "$FORCE_KUBECONFIG" != 1 ] &&
   kubectl get secret "${CLUSTER_NAME}-kubeconfig" -n "$NAMESPACE" >/dev/null 2>&1; then
  issuer=$(kubectl get secret "${CLUSTER_NAME}-kubeconfig" -n "$NAMESPACE" -o jsonpath='{.data.value}' |
           base64 -d | sed -n 's/^ *client-certificate-data: //p' | base64 -d |
           openssl x509 -noout -issuer 2>/dev/null || true)
  if [ "$issuer" = "issuer=CN=k3s-client-ca" ]; then
    echo "==> ${NAMESPACE}/${CLUSTER_NAME}-kubeconfig exists (client-ca signed) — kept" >&2
    exit 0
  fi
  # Anything else (e.g. one CAPI generated from the server CA before this ran)
  # is unusable against k3s: replace it.
  echo "==> ${NAMESPACE}/${CLUSTER_NAME}-kubeconfig exists but is not client-ca signed (${issuer:-unreadable}) — replacing" >&2
fi

echo "==> ${NAMESPACE}/${CLUSTER_NAME}-kubeconfig (admin, client-ca signed)" >&2
work=$(mktemp -d); trap 'rm -rf "$work"' EXIT
openssl ecparam -name prime256v1 -genkey -noout -out "$work/admin.key" 2>/dev/null
openssl req -new -key "$work/admin.key" -subj "/O=system:masters/CN=system:admin" -out "$work/admin.csr"
printf 'keyUsage=critical,digitalSignature,keyEncipherment\nextendedKeyUsage=clientAuth\nbasicConstraints=critical,CA:FALSE\n' >"$work/ext"
# 3500 days (~9.6 years) stays well clear of CAPI's rotation window and ends
# before the warm CA itself does (Jun 2036).
openssl x509 -req -in "$work/admin.csr" -CA "${CA_DIR}/client-ca.crt" -CAkey "${CA_DIR}/client-ca.key" \
  -CAcreateserial -CAserial "$work/ca.srl" -days 3500 -sha256 -extfile "$work/ext" -out "$work/admin.crt" 2>/dev/null

# One `server:` line per cluster, on its own line: frontDoorKubeconfig in
# ui/backend/handlers/cluster_deploy.go rewrites exactly that shape.
cat >"$work/kubeconfig" <<KC
apiVersion: v1
kind: Config
clusters:
- name: ${CLUSTER_NAME}
  cluster:
    certificate-authority-data: $(b64 "${CA_DIR}/server-ca.crt")
    server: https://${API_LB_IP}:6443
contexts:
- name: ${CLUSTER_NAME}-admin@${CLUSTER_NAME}
  context:
    cluster: ${CLUSTER_NAME}
    user: ${CLUSTER_NAME}-admin
current-context: ${CLUSTER_NAME}-admin@${CLUSTER_NAME}
users:
- name: ${CLUSTER_NAME}-admin
  user:
    client-certificate-data: $(b64 "$work/admin.crt")
    client-key-data: $(b64 "$work/admin.key")
KC

apply <<YAML
apiVersion: v1
kind: Secret
metadata:
  name: ${CLUSTER_NAME}-kubeconfig
  namespace: ${NAMESPACE}
  labels:
    cluster.x-k8s.io/cluster-name: ${CLUSTER_NAME}
type: cluster.x-k8s.io/secret
data:
  value: $(b64 "$work/kubeconfig")
YAML
[ "${KEEP_KUBECONFIG_FILE:-}" != "" ] && cp "$work/kubeconfig" "$KEEP_KUBECONFIG_FILE"
echo "==> Done." >&2

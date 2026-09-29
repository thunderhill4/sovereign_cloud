#!/usr/bin/env bash
# Installs Sympozium (AI agent control plane) on cluster2.
# Replaces kagent as the backend for the AI tab.
#
# Prerequisites: kubectl context pointing at cluster2, helm, internet access.
# Safe to re-run; cert-manager and Helm install are idempotent.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYMPOZIUM_NS="${SYMPOZIUM_NAMESPACE:-sympozium-system}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.16.2}"
# Pinned to the version these manifests/labs are verified against.
# Upgraded 0.10.38 -> 0.10.47 (2026-08-21) -> 0.10.57 (2026-09-09) -> 0.10.75
# (2026-09-16) -> 0.10.87 (2026-09-30; see the 0.10.75 -> 0.10.87 note below).
# Earlier history:
# (2026-09-16, latest published at time of upgrade). The 0.10.38 pin's
# original rationale (newer charts dropped the SympoziumInstance CRD) no
# longer applies: the SympoziumInstance objects were removed from
# cluster2-agent.yaml and target-cluster-agent.yaml, which now carry only
# Agent CRs.
#
# 0.10.47 -> 0.10.57: manual upgrade, not this script (helm never upgrades
# CRDs in crds/, and the controller mutates its own built-in SkillPacks after
# install, so a plain re-run of this script's install path does not apply to
# an existing release -- see the WARNING block below for the real procedure).
# Two things broke on the way, both worth knowing before touching this again:
#
#   1. New chart-templated `sympozium-agent` ServiceAccount collided with an
#      existing one the controller had created out-of-band (managed-by:
#      sympozium, no Helm ownership annotations). `helm upgrade` refuses to
#      adopt it. Fix (idempotent, apply before upgrading if this is still
#      the installed base):
#        kubectl label sa sympozium-agent -n sympozium-system \
#          app.kubernetes.io/managed-by=Helm --overwrite
#        kubectl annotate sa sympozium-agent -n sympozium-system \
#          meta.helm.sh/release-name=sympozium \
#          meta.helm.sh/release-namespace=sympozium-system --overwrite
#
#   2. `--wait` polls EVERY MCPServer's custom status for readiness, including
#      ones deliberately left Suspended (e.g. `postgres`, unconfigured demo
#      example -- Ready:false is its correct steady state, not a fault). Any
#      `--wait` upgrade on this chart times out on it regardless of version.
#      Drop `--wait` and verify readiness manually instead (Deployments +
#      Agent phase + a real chat completion, not the helm exit code).
#
#   3. 0.10.57 defaults `nats.auth.enabled: true` and wires NATS_USERNAME/
#      PASSWORD into the controller and llmfit-daemonset -- but NOT into the
#      built-in web-endpoint SkillPack's sidecar env (still bare
#      EVENT_BUS_URL, no credentials). Every served agent's web-proxy then
#      dials NATS unauthenticated, gets rejected ("authentication error" in
#      the nats pod's own log), and crash-loops with exit 2 and zero log
#      output -- same signature as the :latest tag-skew landmine below,
#      different cause. `06-sympozium/values.yaml` sets `nats.auth.enabled:
#      false` to restore the known-working behavior; see the comment there
#      for why that's an acceptable trade-off on this cluster and when to
#      revisit it.
#
# After ANY chart upgrade, re-run fix-web-proxy-image.sh (image tag reverts
# to the mutable `:latest` on every upgrade -- Helm re-applies the chart's
# own SkillPack default) and fix-missing-builtin-skillpacks.sh (checks for
# the admission-webhook race; harmless no-op if nothing is missing), then
# sympozium-lb-setup.sh (the AgentRun controller recreates Services as
# ClusterIP when it regenerates the serving Deployments).
#
# 0.10.75 -> 0.10.87: CRDs server-side, then helm upgrade without --wait. Values
# diff is additive only (Celln mediation/fleet settings, all off by default);
# nothing in values.yaml changed. Two breakages, both in the post-upgrade step,
# both now handled by fix-web-proxy-image.sh:
#   1. Recreated serving Deployments come back with readOnlyRootFilesystem:
#      true, so every web-proxy crash-loops (exit 2, no logs). mesh-sre-agent
#      was never in any fix-web-proxy-rootfs.sh caller list and had been
#      crash-looping since 0.10.75 for this reason.
#   2. cluster2-agent's recreated serving AgentRun raced its Deployment and
#      went Failed ("server Deployment not found") with no retry; deleting it
#      once more fixes it.
# Verified: istio-adv-verify 22/22, all three agents Serving with 0 restarts,
# live chat completions with real kubectl tool calls through all three.
#
# 0.10.57 -> 0.10.75: clean helm upgrade (SA already Helm-adopted from the
# last upgrade; no new collisions). CRDs/RBAC diff against 0.10.57 is
# additive only (new Celln-fleet + model-gateway CRDs, both `enabled: false`
# by default and untouched by this repo's values.yaml) and a real bug fix
# (generated NATS passwords could start with a digit and break nats.conf
# parsing) -- nothing here required a values.yaml change. One real breaking
# change found by testing, not by the diff (it's a controller behavior
# change, not a template one): AgentRun task pods now run as a per-run
# ServiceAccount (`sympozium-run-<agentrun-name>`, bound to a matching
# per-run Role/RoleBinding the controller generates and deletes with the
# run) instead of the shared `sympozium-agent` SA every skill sidecar used
# to run as. That silently broke agent-istio-rbac.yaml and
# agent-kubevirt-rbac.yaml, which bound only the literal `sympozium-agent`
# subject -- mesh-sre-agent's live tool calls degraded to "permission
# issue" confabulation while `kubectl get gateways --as=...sympozium-agent`
# still looked fine (wrong identity to check). Fixed by rebinding both
# ClusterRoleBindings to the `system:serviceaccounts:sympozium-system`
# Group instead of one SA name, so every present and future per-run SA
# inherits the grant automatically. Verified with
# `--as=system:serviceaccount:sympozium-system:sympozium-run-<real-run-name>`
# (not the old `sympozium-agent` name) and a live chat completion whose
# answer matched real cluster state. Re-run this same check after any future
# upgrade -- the per-run identity model is new enough here that it could
# change again.
SYMPOZIUM_CHART_VERSION="${SYMPOZIUM_CHART_VERSION:-0.10.87}"

# LLM credentials. For Ollama-compatible endpoints that don't require auth,
# any non-empty value works. Override LLM_API_KEY if using OpenAI/Anthropic.
LLM_API_KEY="${LLM_API_KEY:-ollama-dummy-key}"

echo "== 1/5 Installing cert-manager (${CERT_MANAGER_VERSION}) =="
if kubectl get deployment cert-manager -n cert-manager &>/dev/null; then
    echo "cert-manager already installed, skipping."
else
    kubectl apply -f "https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml"
    echo "Waiting for cert-manager pods to be Ready..."
    kubectl wait --for=condition=Available --timeout=180s \
        -n cert-manager deployment/cert-manager \
        deployment/cert-manager-cainjector \
        deployment/cert-manager-webhook
fi

echo ""
echo "== 2/5 Installing Sympozium Helm chart =="
helm repo add sympozium https://deploy.sympozium.ai/charts 2>/dev/null || true
helm repo update sympozium
# The Sympozium controller mutates some fields on its own built-in SkillPacks
# (e.g. web-endpoint.spec.sidecar.mountWorkspace) after install. Helm upgrade
# then conflicts. Only run install if the release doesn't exist yet.
if helm status sympozium -n "$SYMPOZIUM_NS" &>/dev/null; then
    CURRENT_VER="$(helm list -n "$SYMPOZIUM_NS" -f '^sympozium$' -o json 2>/dev/null \
                   | sed -n 's/.*"chart":"sympozium-\([^"]*\)".*/\1/p')"
    echo "Helm release 'sympozium' already exists (chart ${CURRENT_VER:-unknown}), skipping install."
    if [[ -n "$CURRENT_VER" && "$CURRENT_VER" != "$SYMPOZIUM_CHART_VERSION" ]]; then
        echo ""
        echo "  WARNING: installed chart ${CURRENT_VER} != pinned ${SYMPOZIUM_CHART_VERSION}."
        echo "  This script does NOT upgrade in place. Helm never upgrades CRDs in crds/,"
        echo "  and the controller mutates its own built-in SkillPacks after install"
        echo "  (spec.sidecar.mountWorkspace), so a plain 'helm upgrade' conflicts."
        echo "  To upgrade, apply the CRDs first, then upgrade server-side:"
        echo ""
        echo "    helm template sympozium-crds sympozium/sympozium-crds \\"
        echo "      --version $SYMPOZIUM_CHART_VERSION | kubectl apply --server-side --force-conflicts -f -"
        echo "    helm upgrade sympozium sympozium/sympozium --version $SYMPOZIUM_CHART_VERSION \\"
        echo "      -n $SYMPOZIUM_NS -f $SCRIPT_DIR/values.yaml \\"
        echo "      --server-side=true --force-conflicts --wait --timeout 12m"
        echo ""
        echo "  Note --server-side=true (with a value): Helm 4 made --server-side take an"
        echo "  argument, so a bare '--server-side --force-conflicts' fails with"
        echo "  'invalid/unknown release server-side apply method: --force-conflicts'."
        echo "  Afterwards, pin the web-proxy sidecar (see fix-web-proxy-image.sh)."
        echo ""
    fi
else
    helm install sympozium sympozium/sympozium \
        --version "$SYMPOZIUM_CHART_VERSION" \
        -n "$SYMPOZIUM_NS" --create-namespace \
        -f "$SCRIPT_DIR/values.yaml" \
        --wait --timeout 5m
fi

echo ""
echo "== 2a/5 Ensuring built-in SkillPacks (helm install can silently drop some) =="
bash "$SCRIPT_DIR/fix-missing-builtin-skillpacks.sh"

echo ""
echo "== 2b/5 Fixing node-probe loopback (Ollama runs on the Docker host, not in-cluster) =="
# One-shot immediate fix; the kustomize bundle applied in step 4 also installs
# node-probe-loopback-ds.yaml, which keeps the rule re-applied after restarts.
bash "$SCRIPT_DIR/fix-node-probe-loopback.sh"

echo ""
echo "== 2c/5 Injecting nvidia-smi shim into llmfit-daemon (GPU otherwise reported as AMD) =="
bash "$SCRIPT_DIR/fix-llmfit-nvidia-smi.sh"

echo ""
echo "== 3/5 Creating LLM credentials Secret =="
kubectl create secret generic llm-credentials \
    --from-literal=apiKey="$LLM_API_KEY" \
    -n "$SYMPOZIUM_NS" \
    --dry-run=client -o yaml | kubectl apply -f -

echo ""
echo "== 4/5 Applying Phase-0 core bundle (policies + warm schedule + cluster2-agent) =="
kubectl apply -k "$SCRIPT_DIR"
bash "$SCRIPT_DIR/fix-web-proxy-rootfs.sh" cluster2-agent-web-endpoint-server

echo ""
echo "== 4b/5 Deploying skills-webhook (injects per-agent SkillPack into chat AgentRuns) =="
# On 0.10.38, AgentRuns created via POST /api/v1/runs (dashboard + kubeui chat)
# don't inherit their agent's skills, so tool calls (kubectl/virtctl) silently
# dead-end. This admission webhook injects the right SkillPack per agent. Needs
# a local docker+kind (cluster2 is a Kind cluster); non-fatal if unavailable.
if command -v kind &>/dev/null && command -v docker &>/dev/null; then
    bash "$SCRIPT_DIR/skills-webhook/build-and-deploy.sh" || \
        echo "WARNING: skills-webhook deploy failed — agent tool calls will not work until it is deployed."
else
    echo "WARNING: kind/docker not found — skipping skills-webhook. Deploy later with:"
    echo "  bash $SCRIPT_DIR/skills-webhook/build-and-deploy.sh"
fi

# target-cluster-agent needs the target-cluster kubeconfig as a Secret;
# applying the SympoziumInstance before that Secret exists would create a pod
# stuck waiting on the mount.
TARGET_KUBECONFIG="${TARGET_KUBECONFIG:-$SCRIPT_DIR/../target-cluster-kubeconfig}"
if [[ -f "$TARGET_KUBECONFIG" ]]; then
    echo ""
    echo "== 5/5 Applying target-cluster-agent (kubeconfig found at $TARGET_KUBECONFIG) =="
    kubectl create secret generic target-cluster-kubeconfig \
        --from-file=kubeconfig="$TARGET_KUBECONFIG" \
        -n "$SYMPOZIUM_NS" \
        --dry-run=client -o yaml | kubectl apply -f -
    kubectl apply -f "$SCRIPT_DIR/target-cluster-agent.yaml"
    bash "$SCRIPT_DIR/fix-web-proxy-rootfs.sh" target-cluster-agent-web-endpoint-server
else
    echo ""
    echo "== 5/5 SKIPPED =="
    echo "WARNING: $TARGET_KUBECONFIG not found — target-cluster-agent NOT applied."
    echo "After running 'make verify' to generate target-cluster-kubeconfig, re-run this script."
fi

echo ""
echo "Sympozium installed. Next steps:"
echo "  1. ./sympozium-lb-setup.sh        # expose served agents via MetalLB"
echo "  2. make ui                         # launch the UI"

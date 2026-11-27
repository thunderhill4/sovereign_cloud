#!/usr/bin/env bash
# Installs Sympozium (AI agent control plane) and its agents on cluster2.
#
# Prerequisites: kubectl context pointing at cluster2, helm, internet access.
# Safe to re-run: every step is idempotent. Does NOT upgrade an existing release
# (see UPGRADE-NOTES.md).
#
# Env overrides: SYMPOZIUM_NAMESPACE, CERT_MANAGER_VERSION, SYMPOZIUM_CHART_VERSION,
#                LLM_API_KEY, TARGET_KUBECONFIG

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYMPOZIUM_NS="${SYMPOZIUM_NAMESPACE:-sympozium-system}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.21.3}"
# Pinned to the version these manifests/labs are verified against. Upgrade
# history and procedure: UPGRADE-NOTES.md. verify-sympozium.sh parses this
# line with sed; keep its shape.
SYMPOZIUM_CHART_VERSION="${SYMPOZIUM_CHART_VERSION:-0.12.0}"
# Ollama needs no auth; Sympozium only requires the Secret to exist and be
# non-empty. Local endpoint only (Architecture Constraint 1).
LLM_API_KEY="${LLM_API_KEY:-ollama-dummy-key}"
TARGET_KUBECONFIG="${TARGET_KUBECONFIG:-$SCRIPT_DIR/../target-cluster-kubeconfig}"

# Agents in the kustomize bundle (kustomization.yaml).
CORE_AGENTS=(cluster2-agent mesh-sre-agent)
# 0.11.0+: an Agent may not share a key Secret with another Agent (the admission
# webhook refuses the serving run). cluster2-agent keeps llm-credentials; these
# get their own copy of the same key.
SEPARATE_KEY_AGENTS=(mesh-sre-agent target-cluster-agent incident-responder cost-analyzer)

TOTAL_STEPS=8
STEP=0
step() {
    STEP=$((STEP + 1))
    echo ""
    echo "== ${STEP}/${TOTAL_STEPS} $* =="
}

# apply_secret <name> <create-secret args...>: create-or-update a generic Secret.
apply_secret() {
    local name="$1"; shift
    kubectl create secret generic "$name" "$@" -n "$SYMPOZIUM_NS" \
        --dry-run=client -o yaml | kubectl apply -f -
}

# fix_rootfs <agent...>: patch each agent's serving Deployment (waits for it).
fix_rootfs() {
    local deploys=()
    local a
    for a in "$@"; do deploys+=("${a}-web-endpoint-server"); done
    bash "$SCRIPT_DIR/fix-web-proxy-rootfs.sh" "${deploys[@]}"
}

install_cert_manager() {
    step "Installing cert-manager (${CERT_MANAGER_VERSION})"
    if kubectl get deployment cert-manager -n cert-manager &>/dev/null; then
        echo "cert-manager already installed, skipping."
        return
    fi
    kubectl apply -f "https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml"
    echo "Waiting for cert-manager pods to be Ready..."
    kubectl wait --for=condition=Available --timeout=180s \
        -n cert-manager deployment/cert-manager \
        deployment/cert-manager-cainjector \
        deployment/cert-manager-webhook
}

install_chart() {
    step "Installing Sympozium Helm chart (${SYMPOZIUM_CHART_VERSION})"
    helm repo add sympozium https://deploy.sympozium.ai/charts 2>/dev/null || true
    helm repo update sympozium

    if ! helm status sympozium -n "$SYMPOZIUM_NS" &>/dev/null; then
        helm install sympozium sympozium/sympozium \
            --version "$SYMPOZIUM_CHART_VERSION" \
            -n "$SYMPOZIUM_NS" --create-namespace \
            -f "$SCRIPT_DIR/values.yaml" \
            --wait --timeout 5m
        return
    fi

    # The controller mutates its own built-in SkillPacks after install, so a
    # plain helm upgrade conflicts: only install when the release is absent.
    local current
    current="$(helm list -n "$SYMPOZIUM_NS" -f '^sympozium$' -o json 2>/dev/null \
               | sed -n 's/.*"chart":"sympozium-\([^"]*\)".*/\1/p')"
    echo "Helm release 'sympozium' already exists (chart ${current:-unknown}), skipping install."
    if [[ -n "$current" && "$current" != "$SYMPOZIUM_CHART_VERSION" ]]; then
        cat <<EOF

  WARNING: installed chart ${current} != pinned ${SYMPOZIUM_CHART_VERSION}.
  This script does NOT upgrade in place. Apply the CRDs first, then upgrade
  server-side:

    helm template sympozium-crds sympozium/sympozium-crds \\
      --version $SYMPOZIUM_CHART_VERSION | kubectl apply --server-side --force-conflicts -f -
    helm upgrade sympozium sympozium/sympozium --version $SYMPOZIUM_CHART_VERSION \\
      -n $SYMPOZIUM_NS -f $SCRIPT_DIR/values.yaml \\
      --server-side=true --force-conflicts --timeout 12m

  Then run the post-upgrade fixes. Details: $SCRIPT_DIR/UPGRADE-NOTES.md

EOF
    fi
}

run_post_install_fixes() {
    step "Ensuring built-in SkillPacks (helm install can silently drop some)"
    bash "$SCRIPT_DIR/fix-missing-builtin-skillpacks.sh"

    step "Fixing node-probe loopback (Ollama runs on the Docker host, not in-cluster)"
    # One-shot immediate fix; the kustomize bundle also installs
    # node-probe-loopback-ds.yaml, which keeps the rule re-applied after restarts.
    bash "$SCRIPT_DIR/fix-node-probe-loopback.sh"

    step "Injecting nvidia-smi shim into llmfit-daemon (GPU otherwise reported as AMD)"
    bash "$SCRIPT_DIR/fix-llmfit-nvidia-smi.sh"
}

create_llm_secrets() {
    step "Creating LLM credentials Secrets"
    apply_secret llm-credentials --from-literal=apiKey="$LLM_API_KEY"
    local agent
    for agent in "${SEPARATE_KEY_AGENTS[@]}"; do
        apply_secret "${agent}-llm-credentials" --from-literal=apiKey="$LLM_API_KEY"
    done
}

apply_core_bundle() {
    step "Applying core bundle (policies + ${CORE_AGENTS[*]})"
    kubectl apply -k "$SCRIPT_DIR"
    fix_rootfs "${CORE_AGENTS[@]}"

    # skills-webhook retired: since 0.10.47 the apiserver copies an Agent's
    # spec.skills onto the runs it creates, so the webhook only no-ops. Remove a
    # previously deployed one so nothing sits in the AgentRun admission path.
    if [[ -f "$SCRIPT_DIR/skills-webhook/deploy.yaml" ]]; then
        kubectl delete -f "$SCRIPT_DIR/skills-webhook/deploy.yaml" --ignore-not-found >/dev/null 2>&1 || true
    fi
    kubectl delete mutatingwebhookconfiguration agentrun-skills-injector --ignore-not-found >/dev/null 2>&1 || true
}

apply_target_agent() {
    # target-cluster-agent mounts the target-cluster kubeconfig Secret; applying
    # the Agent before that Secret exists leaves its pod stuck on the mount.
    if [[ ! -f "$TARGET_KUBECONFIG" ]]; then
        step "Applying target-cluster-agent: SKIPPED"
        echo "WARNING: $TARGET_KUBECONFIG not found — target-cluster-agent NOT applied."
        echo "After running 'make verify' to generate target-cluster-kubeconfig, re-run this script."
        return
    fi
    step "Applying target-cluster-agent (kubeconfig found at $TARGET_KUBECONFIG)"
    apply_secret target-cluster-kubeconfig --from-file=kubeconfig="$TARGET_KUBECONFIG"
    kubectl apply -f "$SCRIPT_DIR/target-cluster-agent.yaml"
    fix_rootfs target-cluster-agent
}

print_next_steps() {
    cat <<'EOF'

Sympozium installed. Next steps:
  1. ./sympozium-lb-setup.sh        # expose served agents via MetalLB
  2. make sympozium-warm            # load the agent model (model.env)
  3. make sympozium-verify          # deterministic check + one golden case
  4. make ui                         # launch the UI
EOF
}

main() {
    install_cert_manager
    install_chart
    run_post_install_fixes
    create_llm_secrets
    apply_core_bundle
    apply_target_agent
    print_next_steps
}

main "$@"

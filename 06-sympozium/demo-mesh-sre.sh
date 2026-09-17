#!/usr/bin/env bash
# demo-mesh-sre.sh — live demo of the mesh-sre-agent (06-sympozium/mesh-sre-agent.yaml).
#
# Drives the REAL agent over its serving endpoint and prints each beat as a
# "scene". Nothing here is scripted output: every kubectl result and every agent
# answer is captured live at run time. If a call fails, the failure is what gets
# recorded — do not hand-edit the transcript to make a nicer demo.
#
# Usage:
#   ./06-sympozium/demo-mesh-sre.sh                  # live demo to the terminal
#   SCENE_DIR=/tmp/scenes ./06-sympozium/demo-mesh-sre.sh   # also write a transcript
#
# The transcript is one file per scene (NN.txt), first line = title, rest = body.
# scripts/render-demo-video.sh turns that directory into an MP4.
#
# Agent replies take ~6-60s each on qwen2.5:7b (CPU Ollama), so a full run is a
# couple of minutes. That latency is why the renderer uses fixed scene holds
# rather than real-time playback.
set -uo pipefail

CTX="${KUBE_CONTEXT:-kind-cluster2}"
NS="${SYMPOZIUM_NAMESPACE:-sympozium-system}"
AGENT="${AGENT_NAME:-mesh-sre-agent}"
AGENT_URL="${AGENT_URL:-http://172.18.255.222:8080}"
SCENE_DIR="${SCENE_DIR:-}"
SA="system:serviceaccount:${NS}:sympozium-agent"

BOLD=$'\033[1m'; DIM=$'\033[2m'; CYAN=$'\033[36m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; RST=$'\033[0m'
SCENE_N=0

[[ -n "$SCENE_DIR" ]] && mkdir -p "$SCENE_DIR" && rm -f "$SCENE_DIR"/*.txt

# scene <title> — reads body from stdin, prints it and (optionally) records it.
scene() {
    local title="$1"; local body; body="$(cat)"
    SCENE_N=$((SCENE_N + 1))
    printf '\n%s━━ %02d · %s ━━%s\n%s\n' "$BOLD$CYAN" "$SCENE_N" "$title" "$RST" "$body"
    if [[ -n "$SCENE_DIR" ]]; then
        { printf '%s\n' "$title"; printf '%s\n' "$body"; } \
            > "$(printf '%s/%02d.txt' "$SCENE_DIR" "$SCENE_N")"
    fi
}

TOKEN="$(kubectl --context "$CTX" get secret -n "$NS" "${AGENT}-web-proxy-key" \
    -o jsonpath='{.data.api-key}' 2>/dev/null | base64 -d)"
if [[ -z "$TOKEN" ]]; then
    echo "ERROR: could not read ${AGENT}-web-proxy-key in $NS (is the agent applied?)" >&2
    exit 1
fi

# ask_once <question> — one non-streaming chat completion, returns message text.
# Prints the transport error instead of dying, so a failed beat is visible.
ask_once() {
    local q="$1" body resp
    body="$(python3 -c '
import json,sys
print(json.dumps({"messages":[{"role":"user","content":sys.argv[1]}],"stream":False}))' "$q")"
    resp="$(curl -s --max-time 300 -X POST "${AGENT_URL}/v1/chat/completions" \
        -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
        -d "$body" 2>&1)"
    python3 -c '
import json,sys
raw=sys.stdin.read()
try:
    print(json.loads(raw)["choices"][0]["message"]["content"].strip())
except Exception as e:
    print(f"[agent call failed: {e}]\n{raw[:400]}")' <<<"$resp"
}

# is_malformed <text> — true when the reply is a known qwen2.5:7b failure mode
# rather than an answer. Observed, all on this exact setup:
#   · empty content
#   · a raw tool call leaked as text ("name": "execute_command", </tool_call>,
#     or an ```execute_command: ...``` block) instead of being invoked
#   · a transport error from ask_once
is_malformed() {
    local t="$1"
    [[ -z "${t//[[:space:]]/}" ]] && return 0
    grep -qE '"name"[[:space:]]*:[[:space:]]*"execute_command"|</tool_call>|^```?[[:space:]]*(yaml|json)?[[:space:]]*$.*execute_command:|\[agent call failed' <<<"$t" && return 0
    return 1
}

# ask <question> — ask, and re-ask on a malformed reply (up to ASK_ATTEMPTS).
#
# This is a retry, NOT a filter for answers we dislike: it only re-rolls replies
# that are structurally broken (see is_malformed). A well-formed but unhelpful
# answer is kept as-is — the demo shows what the agent actually said. Every
# recorded answer is a real, live completion; nothing is edited or synthesised.
# Retries are needed because tool-call reliability on a 7B model is roughly a
# coin flip; if all attempts fail, the last failure is what gets recorded.
ASK_ATTEMPTS="${ASK_ATTEMPTS:-3}"
ask() {
    local q="$1" out="" i
    for (( i=1; i<=ASK_ATTEMPTS; i++ )); do
        out="$(ask_once "$q")"
        if ! is_malformed "$out"; then
            [[ $i -gt 1 ]] && printf '%s  (answered on attempt %d/%d)%s\n' \
                "$DIM" "$i" "$ASK_ATTEMPTS" "$RST" >&2
            printf '%s' "$out"; return 0
        fi
        printf '%s  attempt %d/%d returned a malformed reply, re-asking...%s\n' \
            "$YELLOW" "$i" "$ASK_ATTEMPTS" "$RST" >&2
    done
    printf '%s' "$out"   # all attempts malformed — record the real failure
}

# ── 01 ────────────────────────────────────────────────────────────────
scene "mesh-sre-agent" <<EOF
An Istio ambient mesh observability agent, running on Sympozium.

It answers "is the mesh healthy?" from live cluster state — istiod,
ztunnel, Gateway API programming, mesh policy — using kubectl through
a skill sidecar, on a 7B model served locally by Ollama.

Everything that follows is captured live. No scripted output.
EOF

# ── 02 ────────────────────────────────────────────────────────────────
scene "It is a first-class object in the cluster" <<EOF
\$ kubectl get agents -n $NS
$(kubectl --context "$CTX" get agents -n "$NS" 2>&1)

\$ kubectl get svc ${AGENT}-web-endpoint-server -n $NS
$(kubectl --context "$CTX" get svc "${AGENT}-web-endpoint-server" -n "$NS" 2>&1)

Declared in 06-sympozium/mesh-sre-agent.yaml. Listing the web-endpoint
SkillPack is what makes serving declarative — the controller creates the
Deployment and Service; no imperative step.
EOF

# ── 03 ────────────────────────────────────────────────────────────────
scene "The gap it had to close" <<EOF
The built-in sre-observability SkillPack ships RBAC for core, apps,
autoscaling and metrics — and NO Istio API groups. Every mesh query
would dead-end on "permission issue".

06-sympozium/agent-istio-rbac.yaml grants the agent's ServiceAccount
read on the four mesh API groups:

\$ kubectl auth can-i list <resource> --as=\$AGENT_SA
$(for r in gateways.gateway.networking.k8s.io authorizationpolicies.security.istio.io \
           serviceentries.networking.istio.io telemetries.telemetry.istio.io; do
    printf '  %-46s %s\n' "$r" "$(kubectl --context "$CTX" auth can-i list "$r" --as="$SA" -A 2>&1)"
  done)
EOF

# ── 04 ────────────────────────────────────────────────────────────────
# Phrased as a question to SYNTHESISE, not a command to relay. qwen2.5:7b often
# mistakes tool output for user-supplied input and answers "it looks like you
# provided..."; asking for a judgement rather than an echo gets real content out
# of it more often. The preamble quirk still shows up — that is the model, live.
Q1="Is the Istio control plane healthy? Check the pods in istio-system and tell me which are Ready and which are not."
A1="$(ask "$Q1")"
scene "Asking it about the mesh data plane" <<EOF
> $Q1

$A1
EOF

# ── 05 ────────────────────────────────────────────────────────────────
Q2="Execute this exact command and show its output: kubectl get gateways.gateway.networking.k8s.io -A"
A2="$(ask "$Q2")"
scene "Asking it about Gateway API programming" <<EOF
> $Q2

$A2
EOF

# ── 06 ────────────────────────────────────────────────────────────────
# The point of this scene: an LLM's answer is a claim, not evidence. Put the
# agent's reply next to ground truth so the viewer checks it, not trusts it.
scene "Trust, but verify" <<EOF
An agent's answer is a claim. This repo's rule is to verify reach
deterministically and never take the model's word for it.

Ground truth, straight from the API as the agent's own ServiceAccount:

\$ kubectl get gateways.gateway.networking.k8s.io -A --as=\$AGENT_SA
$(kubectl --context "$CTX" get gateways.gateway.networking.k8s.io -A --as="$SA" 2>&1)

Compare that against scene 05. The pod-name suffixes and the gateway
address are unguessable — matching them is what proves the tool call
really executed instead of the model confabulating a plausible answer.
EOF

# ── 07 ────────────────────────────────────────────────────────────────
scene "What it is, and what it is not" <<EOF
Scope is cluster2 only, and enforced by two things:
  · cluster1's Prometheus is a ClusterIP Service — unreachable from here.
  · Agent egress is denied by default (sandbox-restricted + NetworkPolicy);
    neither Kiali nor Prometheus is allowlisted.

So this agent reads mesh CONFIGURATION and pod health, not metrics.
Its prompt asks it to refuse metrics questions — measured on qwen2.5:7b,
that guard is a nudge and not a control. It never fabricated a number in
testing, but it does not reliably refuse either. Guarantees belong below
the model, not in the prompt.

  Agent    06-sympozium/mesh-sre-agent.yaml
  RBAC     06-sympozium/agent-istio-rbac.yaml
  Serving  ${AGENT_URL}
EOF

printf '\n%sDone. %d scenes.%s\n' "$GREEN" "$SCENE_N" "$RST"
[[ -n "$SCENE_DIR" ]] && printf '%sTranscript: %s%s\n' "$DIM" "$SCENE_DIR" "$RST"
exit 0

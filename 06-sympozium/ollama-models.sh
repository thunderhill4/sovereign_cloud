#!/usr/bin/env bash
# Create the Ollama model variants the agents name in their Agent CRs.
#
# The agents reach Ollama through its OpenAI-compatible /v1 endpoint, which
# has no per-request context setting, so every model runs at Ollama's default
# 4096-token context. The agents' prompts (system prompt + 9 tool definitions
# + tool output) are 3-6k tokens, and Ollama drops the START of a prompt that
# does not fit -- the system prompt and the tool definitions -- logging
# "truncating input prompt limit=2050 prompt=5399". Each variant here is a
# base model with a larger num_ctx baked in, which /v1 then honours.
#
# Measured 2026-10-24 (same 3 checkable questions x 5 rounds, toolCalls read
# from the AgentRun): see CLAUDE.md "Chat works, but ...".
#
# Idempotent: pulls a missing base model, skips a variant that already has the
# right num_ctx. Safe to re-run.

set -euo pipefail

OLLAMA_URL="${OLLAMA_URL:-http://172.18.0.1:11434}"

# name  base  num_ctx
VARIANTS=(
    "qwen3-16k:8b qwen3:8b 16384"
)

if ! curl -sf --max-time 5 "${OLLAMA_URL}/api/tags" > /dev/null; then
    echo "[models] ERROR: Ollama not reachable at ${OLLAMA_URL}" >&2
    echo "[models] Hint: run 'ollama serve' on the host, or set OLLAMA_URL=..." >&2
    exit 1
fi

has_model() {
    curl -sf --max-time 10 "${OLLAMA_URL}/api/show" -d "{\"model\":\"$1\"}" 2>/dev/null
}

for v in "${VARIANTS[@]}"; do
    read -r name base ctx <<< "$v"

    if has_model "$name" | grep -Eq "num_ctx[[:space:]]+${ctx}\b"; then
        echo "[models] ${name}: present (num_ctx ${ctx})"
        continue
    fi

    if ! has_model "$base" > /dev/null; then
        echo "[models] ${name}: pulling base ${base} (several GB, may take minutes)..."
        curl -sf --max-time 3600 "${OLLAMA_URL}/api/pull" \
            -d "{\"model\":\"${base}\",\"stream\":false}" | grep -q '"success"' \
            || { echo "[models] ERROR: pull of ${base} failed" >&2; exit 1; }
    fi

    echo "[models] ${name}: creating from ${base} with num_ctx ${ctx}"
    curl -sf --max-time 300 "${OLLAMA_URL}/api/create" \
        -d "{\"model\":\"${name}\",\"from\":\"${base}\",\"parameters\":{\"num_ctx\":${ctx}},\"stream\":false}" \
        | grep -q '"success"' \
        || { echo "[models] ERROR: create of ${name} failed" >&2; exit 1; }
done

echo "[models] OK"

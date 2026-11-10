#!/usr/bin/env bash
# Stop (not delete) every Kind cluster's node containers, keeping cluster state.
# Usage: scripts/stop-kind-clusters.sh [cluster ...]   (default: all clusters)
# Restart later with: docker start $(docker ps -aq --filter label=io.x-k8s.kind.cluster)
# The local registry container is not a Kind node and is left running.
set -euo pipefail

if [ "$#" -gt 0 ]; then
  clusters=("$@")
else
  mapfile -t clusters < <(kind get clusters 2>/dev/null | grep -v '^No kind clusters' || true)
fi

if [ "${#clusters[@]}" -eq 0 ]; then
  echo "No Kind clusters found."
  exit 0
fi

for c in "${clusters[@]}"; do
  mapfile -t ids < <(docker ps -q --filter "label=io.x-k8s.kind.cluster=${c}")
  if [ "${#ids[@]}" -eq 0 ]; then
    echo "• ${c}: already stopped"
    continue
  fi
  echo "• ${c}: stopping ${#ids[@]} node container(s)"
  docker stop "${ids[@]}" >/dev/null
done

echo "Done."

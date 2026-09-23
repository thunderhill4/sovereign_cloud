#!/usr/bin/env bash
# fetch-k3s-artifacts.sh — download the k3s release artifacts the golden-image
# bake needs into a local cache, once, so bakes do not depend on GitHub.
#
# WHY: the bake VM used to curl the k3s binary and the airgap image tarball
# straight from GitHub releases. On 2026-09-23 GitHub was degraded from this host
# (~15s per connect, ~670KB/s at best) and Step 1 alone took ~19 min — past the
# bake's 900s wait, so the script gave up while the VM carried on. bake-common.sh
# now serves this cache to the bake VM over the Kind network gateway instead.
#
# Resumable (curl -C -) and retried, so a flaky connection only has to succeed
# once. Verifies the binary and the airgap tarball against the release's own
# sha256sum-amd64.txt. Idempotent: a complete, verified cache is a no-op.
#
# Usage: ./scripts/fetch-k3s-artifacts.sh [K3S_VERSION]
#   default version: v1.37.0+k3s1 (keep in sync with bake-common.sh)
#   cache dir:       ${K3S_CACHE_DIR:-<repo>/.cache/k3s/<version>}
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-${K3S_VERSION:-v1.37.0+k3s1}}"
DIR="${K3S_CACHE_DIR:-$REPO_ROOT/.cache/k3s/$VERSION}"
BASE="https://github.com/k3s-io/k3s/releases/download/${VERSION/+/%2B}"
mkdir -p "$DIR"

verify() {  # verify <file>: check it against the release checksum file
  local f=$1 want got
  want=$(awk -v f="$f" '$2 == f {print $1}' "$DIR/sha256sum-amd64.txt")
  [ -n "$want" ] || { echo "ERROR: no checksum for $f in sha256sum-amd64.txt" >&2; return 1; }
  got=$(sha256sum "$DIR/$f" | awk '{print $1}')
  [ "$want" = "$got" ]
}

fetch() {  # fetch <url> <file>
  echo "==> $2"
  curl -fL --retry 20 --retry-all-errors --retry-delay 5 --connect-timeout 30 \
    -C - -o "$DIR/$2.part" "$1"
  mv "$DIR/$2.part" "$DIR/$2"
}

[ -s "$DIR/sha256sum-amd64.txt" ] || fetch "$BASE/sha256sum-amd64.txt" sha256sum-amd64.txt
[ -s "$DIR/install.sh" ]          || fetch "https://get.k3s.io" install.sh

for f in k3s k3s-airgap-images-amd64.tar.zst; do
  if [ -s "$DIR/$f" ] && verify "$f"; then
    echo "==> $f already cached and verified"
    continue
  fi
  rm -f "$DIR/$f"   # a complete-but-wrong file would make -C - append garbage
  fetch "$BASE/$f" "$f"
  verify "$f" || { echo "ERROR: $f failed checksum verification" >&2; rm -f "$DIR/$f"; exit 1; }
  echo "==> $f verified"
done
chmod 0755 "$DIR/k3s"
echo "==> k3s $VERSION artifacts ready in $DIR"

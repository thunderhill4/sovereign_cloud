#!/usr/bin/env bash
#
# bake-common.sh — Shared logic for baking pre-initialized k3s golden images.
#
# Sourced by bake-golden-image.sh (Noble, general-purpose) and
# bake-golden-image-minimal.sh (Ubuntu Minimal, lean). Callers set env vars
# before sourcing:
#
#   Required:
#     DV_SOURCE      — source DataVolume with the base OS image
#                      (e.g. ubuntu-noble-dv, ubuntu-minimal-noble-dv)
#     DV_TARGET      — output DataVolume name for the baked image
#                      (e.g. ubuntu-noble-k3s-preinit)
#     VM_NAME        — transient bake VM name
#
#   Optional:
#     PVC_SIZE       — output PVC size (default: 20Gi)
#     EXTRA_PACKAGES — space-separated apt packages to install via cloud-init
#                      packages: section before bake.sh runs. Use for lean
#                      base images (Ubuntu Minimal) that lack k3s deps.
#     K3S_VERSION    — DOES NOT actually control the bake (found 2026-09-16,
#                      upgrading v1.31.4+k3s1 -> v1.37.0+k3s1): the version
#                      that reaches the VM is a SEPARATE hardcoded literal
#                      inside the embedded /usr/local/bin/bake.sh cloud-init
#                      content below (search this file for the second
#                      K3S_VERSION= assignment) — that content block is
#                      written to the VM verbatim, so this outer bash
#                      variable never reaches it. Keep both literals in sync
#                      by hand until someone wires the interpolation through.
#                      (default here: v1.37.0+k3s1)
#     TARGET_IMAGE   — final containerDisk image ref printed in the end banner
#                      (default derived from DV_TARGET)
#
# See:
#   - bake-golden-image.sh          (Noble defaults)
#   - bake-golden-image-minimal.sh  (Minimal defaults)

set -euo pipefail

K3S_VERSION="${K3S_VERSION:-v1.37.0+k3s1}"
DV_SOURCE="${DV_SOURCE:?bake-common: DV_SOURCE must be set}"
DV_TARGET="${DV_TARGET:?bake-common: DV_TARGET must be set}"
VM_NAME="${VM_NAME:?bake-common: VM_NAME must be set}"
PVC_SIZE="${PVC_SIZE:-20Gi}"
EXTRA_PACKAGES="${EXTRA_PACKAGES:-}"
TARGET_IMAGE="${TARGET_IMAGE:-localhost:5000/${DV_TARGET%-preinit}:preinit}"

# ── Bake mode ───────────────────────────────────────────────
#   preinit (default) — keep k3s state, run `--cluster-reset` on first boot for a
#                       fresh etcd identity. CAPK injects per-cluster CAs that
#                       conflict with the baked leaves, so leaves are purged and
#                       regenerated. Floor ~100s+ (see docs/sub-60s-...).
#   warm              — bake k3s against a FIXED custom CA set (03-target-cluster/
#                       warm-ca/) and a fixed token, with the API VIP in the
#                       serving cert SAN. The management cluster pre-seeds the
#                       SAME CAs (scripts/seed-cluster-secrets.sh), so on first
#                       boot the CAs KThrees writes already match the baked
#                       leaves — no purge, no `--cluster-reset`. Single-cluster
#                       only (reused etcd identity is safe). Target <40s.
BAKE_MODE="${BAKE_MODE:-preinit}"
WARM_CA_DIR="${WARM_CA_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/03-target-cluster/warm-ca}"
# Fixed join token for the warm image. MUST match scripts/seed-cluster-secrets.sh
# and the worker bootstrap in target-cluster-warm.yaml.
WARM_TOKEN="${WARM_TOKEN:-f00dcafef00dcafef00dcafef00dcafe}"
# API server VIP baked into the serving cert SAN. MUST match API_LB_IP in the
# warm cluster manifest.
WARM_API_LB_IP="${WARM_API_LB_IP:-172.18.255.215}"

# ── k3s artifact mirror ─────────────────────────────────────
# The bake VM used to download the k3s binary and the airgap image tarball from
# GitHub itself. On 2026-09-23 GitHub was degraded from this host and Step 1 alone
# took ~19 min, past this script's 900s wait. Now the artifacts are fetched ONCE
# into a host cache (scripts/fetch-k3s-artifacts.sh: resumable, retried,
# checksum-verified) and served to the bake VM for the duration of the bake over
# the Kind network gateway, which pods already reach (host Ollama uses it).
# Measured: 81MB binary VM <- host in 0.03s. K3S_MIRROR=off restores the old
# download-from-GitHub-in-the-VM behaviour; so does any failure to fetch or serve.
K3S_MIRROR="${K3S_MIRROR:-on}"
K3S_MIRROR_HOST="${K3S_MIRROR_HOST:-172.18.0.1}"
K3S_MIRROR_PORT="${K3S_MIRROR_PORT:-18080}"
K3S_CACHE_DIR="${K3S_CACHE_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/.cache/k3s/${K3S_VERSION}}"
K3S_MIRROR_URL=""   # set by start_k3s_mirror; empty = VM downloads from GitHub

if [ "${BAKE_MODE}" = "warm" ]; then
  for f in server-ca.crt server-ca.key client-ca.crt client-ca.key \
           request-header-ca.crt request-header-ca.key \
           etcd/server-ca.crt etcd/server-ca.key etcd/peer-ca.crt etcd/peer-ca.key \
           service.key; do
    if [ ! -f "${WARM_CA_DIR}/${f}" ]; then
      echo "bake-common: BAKE_MODE=warm but ${WARM_CA_DIR}/${f} is missing." >&2
      echo "             Run scripts/gen-warm-ca.sh first." >&2
      exit 1
    fi
  done
fi

# ── Colors ──────────────────────────────────────────────────
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

banner() {
  echo ""
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${CYAN}  $1${NC}"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

info() {
  echo -e "${BOLD}==> $1${NC}"
}

# emit_warm_ca_writefiles — emit cloud-init write_files entries that drop the
# fixed custom CA set into the k3s tls/ dir BEFORE bake.sh starts k3s, so the
# warm-up signs every leaf from these CAs. base64-encoded so binary keys survive.
emit_warm_ca_writefiles() {
  _wf() {  # <srcfile> <dstpath> <perm>
    printf '  - path: %s\n'        "$2"
    printf '    encoding: b64\n'
    printf '    owner: root:root\n'
    printf "    permissions: '%s'\n" "$3"
    printf '    content: %s\n'      "$(base64 -w0 "$1")"
  }
  local d="${WARM_CA_DIR}" t=/var/lib/rancher/k3s/server/tls
  _wf "$d/server-ca.crt"         "$t/server-ca.crt"          0644
  _wf "$d/server-ca.key"         "$t/server-ca.key"          0600
  _wf "$d/client-ca.crt"         "$t/client-ca.crt"          0644
  _wf "$d/client-ca.key"         "$t/client-ca.key"          0600
  _wf "$d/request-header-ca.crt" "$t/request-header-ca.crt"  0644
  _wf "$d/request-header-ca.key" "$t/request-header-ca.key"  0600
  _wf "$d/etcd/server-ca.crt"    "$t/etcd/server-ca.crt"     0644
  _wf "$d/etcd/server-ca.key"    "$t/etcd/server-ca.key"     0600
  _wf "$d/etcd/peer-ca.crt"      "$t/etcd/peer-ca.crt"       0644
  _wf "$d/etcd/peer-ca.key"      "$t/etcd/peer-ca.key"       0600
  _wf "$d/service.key"           "$t/service.key"            0600
}

# start_k3s_mirror — fetch the k3s artifacts into the host cache (no-op when
# already cached) and serve them to the bake VM until this script exits. Sets
# K3S_MIRROR_URL on success; on any failure warns and leaves it empty so the
# bake falls back to downloading from GitHub inside the VM.
start_k3s_mirror() {
  [ "${K3S_MIRROR}" = "on" ] || return 0
  # The version actually installed is the literal inside the embedded bake.sh
  # (see the K3S_VERSION note at the top of this file). Serving a cache for a
  # different version would silently bake the wrong k3s, so refuse.
  local baked
  baked=$(sed -n 's/^ *K3S_VERSION="\(v[^"]*\)"$/\1/p' "${BASH_SOURCE[0]}" | tail -1)
  if [ "${baked}" != "${K3S_VERSION}" ]; then
    echo -e "    ${YELLOW}!${NC} K3S_VERSION=${K3S_VERSION} but bake.sh installs ${baked}; not using the mirror." >&2
    return 0
  fi
  if ! K3S_CACHE_DIR="${K3S_CACHE_DIR}" \
       "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/scripts/fetch-k3s-artifacts.sh" "${K3S_VERSION}"; then
    echo -e "    ${YELLOW}!${NC} Could not fetch k3s artifacts; the bake VM will download from GitHub." >&2
    return 0
  fi
  python3 -m http.server --bind "${K3S_MIRROR_HOST}" "${K3S_MIRROR_PORT}" \
    --directory "${K3S_CACHE_DIR}" >/dev/null 2>&1 &
  local pid=$!
  trap 'kill '"${pid}"' 2>/dev/null || true' EXIT
  local i
  for i in $(seq 1 20); do
    if curl -fs -o /dev/null --max-time 2 "http://${K3S_MIRROR_HOST}:${K3S_MIRROR_PORT}/sha256sum-amd64.txt"; then
      K3S_MIRROR_URL="http://${K3S_MIRROR_HOST}:${K3S_MIRROR_PORT}"
      echo -e "    ${GREEN}✓${NC} Serving k3s ${K3S_VERSION} artifacts at ${K3S_MIRROR_URL} (pid ${pid})"
      return 0
    fi
    sleep 0.5
  done
  echo -e "    ${YELLOW}!${NC} Mirror did not come up on ${K3S_MIRROR_HOST}:${K3S_MIRROR_PORT}; the bake VM will download from GitHub." >&2
  kill "${pid}" 2>/dev/null || true
}

# emit_cloudinit — writes cloud-init userdata to stdout.
# Prepends a `packages:` section from EXTRA_PACKAGES, emits the `write_files:`
# header + (warm only) the bake-mode marker and fixed CA set, then the static
# body from a single-quoted heredoc so `$VAR` references inside bake.sh and its
# nested heredocs are preserved verbatim.
emit_cloudinit() {
  echo "#cloud-config"
  if [ -n "${EXTRA_PACKAGES}" ]; then
    echo "package_update: true"
    echo "packages:"
    for pkg in ${EXTRA_PACKAGES}; do
      echo "  - ${pkg}"
    done
  fi
  echo "write_files:"
  # bake.sh reads /etc/bake-mode to choose warm vs preinit first-boot behaviour.
  printf '  - path: /etc/bake-mode\n    content: %s\n    permissions: %s\n' \
    "${BAKE_MODE}" "'0644'"
  # bake.sh reads /etc/bake-k3s-mirror; absent = download from GitHub.
  if [ -n "${K3S_MIRROR_URL}" ]; then
    printf '  - path: /etc/bake-k3s-mirror\n    content: %s\n    permissions: %s\n' \
      "${K3S_MIRROR_URL}" "'0644'"
  fi
  if [ "${BAKE_MODE}" = "warm" ]; then
    emit_warm_ca_writefiles
  fi
  cat << 'CLOUDINIT'
  - path: /etc/systemd/network/10-enp1s0.network
    content: |
      [Match]
      Name=enp1s0

      [Network]
      DHCP=yes
      # IPv4-only cluster. With IPv6 link-local on, networkd keeps the link
      # "configuring" until IPv6LL finishes DAD, and wait-online (which gates
      # cloud-init's network stage) waited ~1.3s for it after DHCPv4 had already
      # landed in 11ms. Measured: wait-online 1.3-1.9s -> 12ms.
      LinkLocalAddressing=no
      IPv6AcceptRA=no

      [DHCP]
      RouteMetric=100
      UseMTU=true
    permissions: '0644'
  - path: /usr/local/bin/bake.sh
    content: |
      #!/bin/bash
      set -e
      K3S_VERSION="v1.37.0+k3s1"
      BAKE_MODE="$(cat /etc/bake-mode 2>/dev/null || echo preinit)"
      echo "[bake] mode: ${BAKE_MODE}"

      # Host-side mirror of the k3s release artifacts (see start_k3s_mirror in
      # bake-common.sh); empty = download from GitHub as before.
      K3S_MIRROR="$(cat /etc/bake-k3s-mirror 2>/dev/null || true)"
      AIRGAP=/var/lib/rancher/k3s/agent/images/k3s-airgap-images-amd64.tar.zst
      mkdir -p /var/lib/rancher/k3s/agent/images

      echo "[bake] Step 1: Installing k3s binary..."
      if [ -n "$K3S_MIRROR" ]; then
        echo "[bake]   from host mirror $K3S_MIRROR"
        curl -fsS --retry 5 "$K3S_MIRROR/sha256sum-amd64.txt" -o /tmp/k3s-sha256sum.txt
        curl -fsS --retry 5 "$K3S_MIRROR/install.sh" -o /tmp/k3s-install.sh
        curl -fsS --retry 5 "$K3S_MIRROR/k3s" -o /usr/local/bin/k3s
        curl -fsS --retry 5 "$K3S_MIRROR/k3s-airgap-images-amd64.tar.zst" -o "$AIRGAP"
        # Same checksums the host verified; catches a truncated transfer.
        echo "$(awk '$2 == "k3s" {print $1}' /tmp/k3s-sha256sum.txt)  /usr/local/bin/k3s" | sha256sum -c -
        echo "$(awk '$2 == "k3s-airgap-images-amd64.tar.zst" {print $1}' /tmp/k3s-sha256sum.txt)  $AIRGAP" | sha256sum -c -
        chmod 0755 /usr/local/bin/k3s
        INSTALL_K3S_SKIP_DOWNLOAD=true \
          INSTALL_K3S_VERSION="$K3S_VERSION" \
          INSTALL_K3S_SKIP_START=true \
          INSTALL_K3S_SKIP_ENABLE=true \
          sh /tmp/k3s-install.sh
      else
        curl -sfL https://get.k3s.io | \
          INSTALL_K3S_VERSION="$K3S_VERSION" \
          INSTALL_K3S_SKIP_START=true \
          INSTALL_K3S_SKIP_ENABLE=true \
          sh -
      fi

      echo "[bake] Step 2: Downloading k3s airgap images..."
      if [ -n "$K3S_MIRROR" ]; then
        echo "[bake]   already fetched from host mirror"
      else
        curl -L "https://github.com/k3s-io/k3s/releases/download/${K3S_VERSION/+/%2B}/k3s-airgap-images-amd64.tar.zst" \
          -o "$AIRGAP"
      fi

      echo "[bake] Step 3: Creating PRODUCTION k3s.service (no install script needed)..."
      # Create the FINAL k3s.service that reads config from /etc/rancher/k3s/
      mkdir -p /etc/rancher/k3s
      cat > /etc/systemd/system/k3s.service << 'SVCEOF'
      [Unit]
      Description=Lightweight Kubernetes
      Documentation=https://k3s.io
      Wants=network-online.target
      After=network-online.target

      [Install]
      WantedBy=multi-user.target

      [Service]
      Type=notify
      EnvironmentFile=-/etc/default/%N
      EnvironmentFile=-/etc/sysconfig/%N
      EnvironmentFile=-/etc/systemd/system/k3s.service.env
      KillMode=process
      Delegate=yes
      LimitNOFILE=1048576
      LimitNPROC=infinity
      LimitCORE=infinity
      TasksMax=infinity
      TimeoutStartSec=0
      Restart=always
      RestartSec=5s
      ExecStartPre=/bin/sh -xc '! /usr/bin/systemctl is-enabled --quiet nm-cloud-setup.service'
      ExecStart=/usr/local/bin/k3s server
      SVCEOF

      # Create k3s-agent.service for workers
      cat > /etc/systemd/system/k3s-agent.service << 'SVCEOF'
      [Unit]
      Description=Lightweight Kubernetes Agent
      Documentation=https://k3s.io
      Wants=network-online.target
      After=network-online.target

      [Install]
      WantedBy=multi-user.target

      [Service]
      Type=notify
      EnvironmentFile=-/etc/default/%N
      EnvironmentFile=-/etc/sysconfig/%N
      EnvironmentFile=-/etc/systemd/system/k3s-agent.service.env
      KillMode=process
      Delegate=yes
      LimitNOFILE=1048576
      LimitNPROC=infinity
      LimitCORE=infinity
      TasksMax=infinity
      TimeoutStartSec=0
      Restart=always
      RestartSec=5s
      ExecStartPre=/bin/sh -xc '! /usr/bin/systemctl is-enabled --quiet nm-cloud-setup.service'
      ExecStart=/usr/local/bin/k3s agent
      SVCEOF

      echo "[bake] Step 4: Enabling systemd-networkd..."
      systemctl enable systemd-networkd

      echo "[bake] Step 4b: Pre-loading kernel modules for k3s..."
      modprobe br_netfilter 2>/dev/null || true
      modprobe overlay 2>/dev/null || true
      cat > /etc/modules-load.d/k3s.conf << 'MODEOF'
      br_netfilter
      overlay
      MODEOF

      echo "[bake] Step 4c: Pre-configuring sysctl for k3s..."
      cat > /etc/sysctl.d/99-k3s.conf << 'SYSCTLEOF'
      net.bridge.bridge-nf-call-iptables = 1
      net.bridge.bridge-nf-call-ip6tables = 1
      net.ipv4.ip_forward = 1
      vm.swappiness = 0
      vm.overcommit_memory = 1
      vm.panic_on_oom = 0
      SYSCTLEOF
      sysctl --system >/dev/null 2>&1

      echo "[bake] Step 4d: Masking slow/unnecessary services..."
      systemctl mask \
        snapd.service snapd.socket snapd.seeded.service \
        multipathd.service multipathd.socket \
        apt-daily.service apt-daily-upgrade.service \
        apt-daily.timer apt-daily-upgrade.timer \
        motd-news.service motd-news.timer \
        unattended-upgrades.service 2>/dev/null || true

      # Boot-path trims, each measured in place on a live target VM (reboot, then
      # systemd-analyze + console-log timestamps); see
      # docs/sub-60s-cluster-strategy.md §10. Together: firmware+GRUB 1.73s ->
      # 0.8s, kernel+initramfs 3.2s -> 1.4s, wait-online 1.4s -> 12ms.
      echo "[bake] Step 4e: Boot-path trims..."
      # Belt and braces with LinkLocalAddressing=no above: IPv4-only readiness.
      # Named 99-* because netplan's generated 10-netplan.conf drop-in sorts after
      # any 10-* name and would reset ExecStart back.
      mkdir -p /etc/systemd/system/systemd-networkd-wait-online.service.d
      cat > /etc/systemd/system/systemd-networkd-wait-online.service.d/99-ipv4-only.conf << 'WOEOF'
      [Service]
      ExecStart=
      ExecStart=/usr/lib/systemd/systemd-networkd-wait-online --ipv4
      WOEOF
      # btrfs-progs/mdadm initramfs hooks load btrfs + raid456, whose raid6
      # benchmark cost ~0.44s per boot. Root is ext4, no RAID.
      DEBIAN_FRONTEND=noninteractive apt-get -y -qq remove btrfs-progs mdadm || true
      # Only the modules this (KubeVirt virtio) hardware needs: initrd 30.7MB ->
      # 17.5MB. It is now only the fallback, see GRUB_FORCE_PARTUUID below.
      sed -i 's/^MODULES=.*/MODULES=dep/' /etc/initramfs-tools/initramfs.conf
      update-initramfs -u
      # GRUB_FORCE_PARTUUID: Ubuntu's initrd-less boot. GRUB boots root=PARTUUID
      # with no initrd (virtio_blk/virtio_pci/ext4 are built in to the generic
      # kernel) and falls back to the initrd if that boot fails. Skips the ~1.2s
      # initramfs and the BIOS-speed load of the initrd itself. The PARTUUID is
      # the bake disk's, which is the disk the containerDisk image is cut from.
      # modprobe.blacklist: q35 AHCI with no disks (SATA link probing, ~0.31s)
      # and the PS/2 mouse probe (~0.72s).
      ROOT_PARTUUID="$(blkid -s PARTUUID -o value "$(findmnt -no SOURCE /)")"
      cat > /etc/default/grub.d/99-kubeui-fastboot.cfg << GRUBEOF
      GRUB_CMDLINE_LINUX_DEFAULT="\$GRUB_CMDLINE_LINUX_DEFAULT modprobe.blacklist=ahci,libahci,psmouse"
      GRUB_FORCE_PARTUUID=${ROOT_PARTUUID}
      GRUBEOF
      update-grub

      if [ "${BAKE_MODE}" = "warm" ]; then
        echo "[bake] Step 5-warm: pre-writing config.yaml (fixed token + VIP SAN + disabled components)..."
        # CA files are already in tls/ (placed by cloud-init write_files). Writing
        # config.yaml BEFORE the warm-up start makes k3s sign every leaf from those
        # fixed CAs, bake the VIP into the serving cert SAN, and use the fixed token
        # — so first boot needs no cert purge and no --cluster-reset.
        mkdir -p /etc/rancher/k3s
        {
          echo "token: f00dcafef00dcafef00dcafef00dcafe"
          echo "tls-san:"
          echo "  - 172.18.255.215"
          echo "disable:"
          echo "  - servicelb"
          echo "  - traefik"
        } > /etc/rancher/k3s/config.yaml
        chmod 600 /var/lib/rancher/k3s/server/tls/*.key \
                  /var/lib/rancher/k3s/server/tls/etcd/*.key 2>/dev/null || true
        # Stash the injected CA so we can later prove k3s adopted it (vs. regen).
        mkdir -p /etc/warm-ca-injected
        cp /var/lib/rancher/k3s/server/tls/server-ca.crt /etc/warm-ca-injected/server-ca.crt
        cp /var/lib/rancher/k3s/server/tls/server-ca.key /etc/warm-ca-injected/server-ca.key
      fi

      echo "[bake] Step 5: PRE-INITIALIZING K3S (loading images into containerd)..."
      # Temporarily start k3s to load images
      systemctl daemon-reload
      systemctl start k3s

      echo "[bake] Waiting for k3s to initialize (up to 3 min)..."
      for i in $(seq 1 180); do
        if /usr/local/bin/k3s kubectl get nodes 2>/dev/null | grep -q " Ready"; then
          echo "[bake] k3s node Ready after ${i}s"
          break
        fi
        sleep 1
      done

      sleep 20
      echo "[bake] Cached images:"
      /usr/local/bin/k3s crictl images 2>/dev/null | head -15 || true

      if [ "${BAKE_MODE}" = "warm" ]; then
        # Prove the core assumption: did k3s KEEP our pre-placed server-ca, or
        # regenerate its own? If kept, the baked leaves are signed by the seeded
        # CA → first boot needs no purge/reset. If changed → warm path is invalid.
        INJ=$(sha256sum /etc/warm-ca-injected/server-ca.crt | awk '{print $1}')
        CUR=$(sha256sum /var/lib/rancher/k3s/server/tls/server-ca.crt | awk '{print $1}')
        if [ "$INJ" = "$CUR" ]; then RESULT=PASS; else RESULT=FAIL; fi
        {
          echo "result: ${RESULT}"
          echo "injected_server_ca_sha256: ${INJ}"
          echo "baked_server_ca_sha256: ${CUR}"
        } > /etc/warm-ca-check
        echo "============================================================"
        echo "[bake] WARM-CA-CHECK: ${RESULT}  (k3s kept pre-placed CA = ${RESULT})"
        echo "[bake]   injected=${INJ}"
        echo "[bake]   baked   =${CUR}"
        echo "============================================================"
      fi

      echo "[bake] Step 6: Stopping k3s and PRESERVING full state for pre-init image..."
      systemctl stop k3s || true
      /usr/local/bin/k3s-killall.sh 2>/dev/null || true

      # PRE-INIT MODE: keep ALL k3s state (etcd DB, TLS, cred, agent kubeconfigs).
      # firstboot-regen.sh issues a fresh etcd cluster identity on first boot via
      # `k3s server --cluster-reset`, while keeping kube state (CRDs, RBAC, etc.).
      # We deliberately do NOT remove /var/lib/rancher/k3s/server/db.

      echo "[bake] Step 6b: Installing firstboot script (mode=${BAKE_MODE}) and systemd drop-in..."
      if [ "${BAKE_MODE}" = "warm" ]; then
      # WARM: the CAs CAPK writes on first boot already match the baked leaves
      # (both come from 03-target-cluster/warm-ca/), so there is NOTHING to purge
      # and NO need for --cluster-reset. The light firstboot only clears the bake
      # VM's stale node identity so this VM's kubelet registers cleanly.
      cat > /usr/local/bin/k3s-firstboot-regen.sh << 'REGEN_EOF'
      #!/bin/bash
      set -euo pipefail
      SENTINEL="/var/lib/rancher/k3s/.firstboot-done"
      log() { echo "[firstboot-warm] $*"; }
      [ -f "$SENTINEL" ] && { log "Sentinel present — skipping."; exit 0; }
      log "Warm first boot: no cluster-reset, no cert purge."
      # DELETE (not truncate) node-passwd: an empty-but-present file makes k3s run
      # nodepassword.MigrateFile against a not-yet-synced secrets cache → nil-ptr
      # panic (SIGSEGV) in startOnAPIServerReady → crash loop. Removing the file
      # makes k3s skip migration; the new node registers a fresh node-passwd.
      rm -f /var/lib/rancher/k3s/server/cred/node-passwd
      rm -f /etc/rancher/k3s/k3s.yaml
      mkdir -p "$(dirname "$SENTINEL")"
      touch "$SENTINEL"
      log "Done (warm)."
      REGEN_EOF
      else
      # PREINIT: keep k3s state but issue a fresh etcd identity. Canonical source:
      # 03-target-cluster/firstboot-regen.sh — keep in sync. CP preK3sCommands MUST
      # delete /var/lib/rancher/k3s/server/token before k3s starts so CAPI's
      # config.yaml token wins over the bake-time random token.
      cat > /usr/local/bin/k3s-firstboot-regen.sh << 'REGEN_EOF'
      #!/bin/bash
      set -euo pipefail
      SENTINEL="/var/lib/rancher/k3s/.firstboot-done"
      log() { echo "[firstboot-regen] $*"; }

      if [ -f "$SENTINEL" ]; then
        log "Sentinel present — already regenerated. Skipping."
        exit 0
      fi

      log "Running k3s server --cluster-reset (preserves kube state, fresh etcd identity)..."
      if ! /usr/local/bin/k3s server --cluster-reset >/var/log/k3s-firstboot-reset.log 2>&1; then
        log "FATAL: k3s --cluster-reset failed. Tail of log:"
        tail -50 /var/log/k3s-firstboot-reset.log >&2 || true
        exit 1
      fi

      NODE_PASSWD="/var/lib/rancher/k3s/server/cred/node-passwd"
      if [ -f "$NODE_PASSWD" ]; then
        log "Clearing stale node-passwd entries (bake-time hostname)."
        : > "$NODE_PASSWD"
      fi
      rm -f /etc/rancher/k3s/k3s.yaml
      mkdir -p "$(dirname "$SENTINEL")"
      touch "$SENTINEL"
      log "Done."
      REGEN_EOF
      fi
      chmod 0755 /usr/local/bin/k3s-firstboot-regen.sh

      mkdir -p /etc/systemd/system/k3s.service.d
      cat > /etc/systemd/system/k3s.service.d/10-firstboot.conf << 'DROPIN_EOF'
      [Service]
      ExecStartPre=/usr/local/bin/k3s-firstboot-regen.sh
      DROPIN_EOF
      chmod 0644 /etc/systemd/system/k3s.service.d/10-firstboot.conf

      echo "[bake] Step 6c: Leaving k3s DISABLED at bake time..."
      # CAPI's KThreesConfig postK3sCommands enables+starts k3s (server on CP,
      # agent on workers) after preK3sCommands have written config.yaml with the
      # per-cluster token. If we auto-enabled k3s here, systemd would race against
      # cloud-init, firstboot-regen would run with the wrong token, and workers
      # would fail to join.
      systemctl disable k3s 2>/dev/null || true
      systemctl disable k3s-agent 2>/dev/null || true

      echo "[bake] Preserved state summary:"
      ls -la /var/lib/rancher/k3s/server/ 2>/dev/null || true
      du -sh /var/lib/rancher/k3s/server/db/ 2>/dev/null | sed 's/^/  etcd: /' || echo "  (no etcd db)"
      du -sh /var/lib/rancher/k3s/server/tls/ 2>/dev/null | sed 's/^/  tls:  /' || echo "  (no tls)"

      echo "[bake] Step 7: Pre-creating ubuntu user with SSH..."
      useradd -m -s /bin/bash -G sudo ubuntu 2>/dev/null || true
      echo 'ubuntu ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/ubuntu
      mkdir -p /home/ubuntu/.ssh
      echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCoeg4clbqLvjstCxtDiDm2hJymy5EeT75HsTMBguKHKPAWcYJDqhakiFF/cxhBoDTLOv6OpfDzCeRoxy0BLFJjQdVNwe7tDl51w+lGG+UR48xATIEEfGxyw8wCjZvML5BnVwdDGyQKFJijZqTlf51hY1FS9x7jg4pVGgNPhf815JHTRsEmzVFgAFK+C5YVl0EYCfII9qpDR7EPECoZngZ5SaMTHLYOVxYBnqrPovuzHD04iemnIuDQKLy4hBYzFMygKkbiKNOYUsuoSsubhtYCtj5KzmV+DpSSIG9YCPC53mjxJ7QiS5/QV9aBEM/0qfVb0aXGhQdjlQ1NoGjeNAMDYsRKDoFNESwYhS39AnXr/ke9nk+4kS0SPYIGqsOMrffJ2e4qzuHOsVjzhCe6rsEwoUufmgzseE+RPVFNxu948cBG6haJde6uqMXe2eq1tKvczYCS9sN8bM8Pb/SmEJDFy7S5I4oaZvsxzotwEEEmgZ+cP92sBZeZefY8LTUmfB0= mahipal@mahipal-proart13' > /home/ubuntu/.ssh/authorized_keys
      chown -R ubuntu:ubuntu /home/ubuntu/.ssh
      chmod 700 /home/ubuntu/.ssh
      chmod 600 /home/ubuntu/.ssh/authorized_keys
      sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config

      echo "[bake] Step 8: Creating airgapped install script at /opt/install.sh..."
      # This is used by CAPI when airGapped: true is set
      cat > /opt/install.sh << 'INSTALLEOF'
      #!/bin/bash
      # Airgapped k3s install script - skips download, uses pre-baked binary
      set -e

      # Parse arguments
      INSTALL_K3S_EXEC=""
      while [ $# -gt 0 ]; do
        case "$1" in
          server|agent)
            INSTALL_K3S_EXEC="$1"
            shift
            ;;
          *)
            INSTALL_K3S_EXEC="$INSTALL_K3S_EXEC $1"
            shift
            ;;
        esac
      done

      # Determine if server or agent
      if echo "$INSTALL_K3S_EXEC" | grep -q "agent"; then
        SERVICE_NAME="k3s-agent"
        EXEC_CMD="/usr/local/bin/k3s agent"
      else
        SERVICE_NAME="k3s"
        EXEC_CMD="/usr/local/bin/k3s server"
      fi

      echo "[airgap-install] Mode: ${SERVICE_NAME}"
      echo "[airgap-install] k3s binary already at /usr/local/bin/k3s"

      # Service files already exist from bake, just update ExecStart if needed
      if [ -f /etc/systemd/system/${SERVICE_NAME}.service ]; then
        echo "[airgap-install] Service file exists"
      fi

      # Enable and start service
      systemctl daemon-reload
      systemctl enable ${SERVICE_NAME}
      systemctl start ${SERVICE_NAME}

      echo "[airgap-install] ${SERVICE_NAME} started"
      INSTALLEOF
      chmod +x /opt/install.sh

      echo "[bake] Step 9: (Legacy) Creating fast-start script..."
      # This script bypasses the k3s install script entirely
      cat > /usr/local/bin/k3s-fast-start.sh << 'FASTEOF'
      #!/bin/bash
      # Fast k3s start - bypasses install script
      set -e
      CONFIG_FILE="${1:-/etc/rancher/k3s/config.yaml}"
      MODE="${2:-server}"

      if [ "$MODE" = "server" ]; then
        # Write config to environment file for systemd
        if [ -f "$CONFIG_FILE" ]; then
          echo "K3S_CONFIG_FILE=$CONFIG_FILE" > /etc/systemd/system/k3s.service.env
        fi
        systemctl daemon-reload
        systemctl enable k3s
        systemctl start k3s
      else
        if [ -f "$CONFIG_FILE" ]; then
          echo "K3S_CONFIG_FILE=$CONFIG_FILE" > /etc/systemd/system/k3s-agent.service.env
        fi
        systemctl daemon-reload
        systemctl enable k3s-agent
        systemctl start k3s-agent
      fi
      FASTEOF
      chmod +x /usr/local/bin/k3s-fast-start.sh

      echo "[bake] Step 10: Resetting cloud-init and machine-id..."
      # Bake-time only: the host mirror address means nothing on a real cluster.
      rm -f /etc/bake-k3s-mirror /tmp/k3s-sha256sum.txt /tmp/k3s-install.sh
      cloud-init clean --logs
      truncate -s 0 /etc/machine-id
      rm -f /var/lib/dbus/machine-id /etc/ssh/ssh_host_*

      systemctl daemon-reload
      sync

      echo "[bake] Done — fast-start k3s image ready. Powering off."
      poweroff
    permissions: '0755'
runcmd:
  - /usr/local/bin/bake.sh
CLOUDINIT
}

# ─────────────────────────────────────────────────────────────
banner "Bake Golden Image: ${DV_TARGET}"
# ─────────────────────────────────────────────────────────────

echo ""
echo "  Pre-bakes a FULLY-INITIALIZED k3s server into a golden VM image."
echo "  First boot runs 'k3s server --cluster-reset' for a fresh etcd identity"
echo "  while keeping all kube state — target time-to-ready ~50–70s."
echo ""
echo "  Source DV:   ${DV_SOURCE}"
echo "  Target DV:   ${DV_TARGET}  (${PVC_SIZE})"
echo "  Bake VM:     ${VM_NAME}"
if [ -n "${EXTRA_PACKAGES}" ]; then
  echo "  Extra pkgs:  ${EXTRA_PACKAGES}"
fi
echo ""
echo "  What gets baked in:"
echo "    - k3s binary            (~60 MB)   at /usr/local/bin/k3s"
echo "    - k3s airgap images     (~134 MB)  at /var/lib/rancher/k3s/agent/images/"
echo "    - PRE-INITIALIZED state  (~30-80MB) at /var/lib/rancher/k3s/server/{db,tls,cred}"
echo "    - firstboot-regen.sh + systemd drop-in at /usr/local/bin and k3s.service.d/"
echo "    - systemd-networkd config          at /etc/systemd/network/10-enp1s0.network"
echo "    - bake-time placeholder token      at /var/lib/rancher/k3s/server/token"
echo "      (CAPI preK3sCommands MUST overwrite before first k3s start)"
echo "    - ubuntu user + sudoers            pre-created"
echo ""

# ─────────────────────────────────────────────────────────────
banner "Step 1/4: Clone DataVolume → ${DV_TARGET}"
# ─────────────────────────────────────────────────────────────
echo ""

if kubectl get dv "${DV_TARGET}" &>/dev/null; then
  PHASE=$(kubectl get dv "${DV_TARGET}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
  echo -e "    ${YELLOW}!${NC} DataVolume '${DV_TARGET}' already exists (phase: ${PHASE})."
  if [ "${PHASE}" = "Succeeded" ]; then
    echo -e "    ${GREEN}✓${NC} Skipping clone — DV is already ready."
    # The bake ends with `cloud-init clean`, so an already-BAKED DV looks like a
    # fresh instance and bake.sh re-runs ON TOP of the previous bake instead of
    # on the clean base image. Seen 2026-09-23: a "rebake" that finished in ~4
    # min and was never built from ${DV_SOURCE}.
    echo -e "    ${YELLOW}!${NC} If ${DV_TARGET} holds a PREVIOUS bake, this bakes on top of it,"
    echo -e "      not on ${DV_SOURCE}. For a clean rebake: kubectl delete dv ${DV_TARGET}"
  elif [ "${PHASE}" = "WaitForFirstConsumer" ]; then
    echo -e "    ${YELLOW}!${NC} DV is waiting for first consumer — VM creation in Step 2 will unblock it."
  else
    echo -e "    ${RED}✗${NC} DV exists but not Succeeded. Delete it and re-run:"
    echo "      kubectl delete dv ${DV_TARGET}"
    exit 1
  fi
else
  info "Creating DataVolume ${DV_TARGET} (clone of ${DV_SOURCE}, ${PVC_SIZE})..."
  kubectl apply -f - << YAML
apiVersion: cdi.kubevirt.io/v1beta1
kind: DataVolume
metadata:
  name: ${DV_TARGET}
  namespace: default
spec:
  source:
    pvc:
      namespace: default
      name: ${DV_SOURCE}
  pvc:
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: ${PVC_SIZE}
YAML

  info "Waiting for DataVolume to finish cloning..."
  MAX_WAIT=600
  ELAPSED=0
  while true; do
    PHASE=$(kubectl get dv "${DV_TARGET}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Pending")
    printf "\r    DataVolume phase: %-12s  [%ds]" "${PHASE}" "$ELAPSED"

    if [ "${PHASE}" = "Succeeded" ]; then
      echo ""
      echo -e "    ${GREEN}✓${NC} DataVolume ${DV_TARGET} is ready!"
      break
    fi

    if [ "${PHASE}" = "WaitForFirstConsumer" ]; then
      echo ""
      echo -e "    ${YELLOW}!${NC} DV waiting for first consumer — proceeding to VM creation to unblock clone."
      break
    fi

    if [ "$ELAPSED" -ge "$MAX_WAIT" ]; then
      echo ""
      echo -e "    ${RED}✗${NC} Timed out waiting for DataVolume after ${MAX_WAIT}s"
      echo "    Check: kubectl describe dv ${DV_TARGET}"
      exit 1
    fi

    sleep 10
    ELAPSED=$((ELAPSED + 10))
  done
fi

echo ""

# ─────────────────────────────────────────────────────────────
banner "Step 2/4: Create Bake VM"
# ─────────────────────────────────────────────────────────────
echo ""

if kubectl get vm "${VM_NAME}" &>/dev/null; then
  echo -e "    ${YELLOW}!${NC} VM '${VM_NAME}' already exists — skipping creation."
  echo -e "    ${DIM:-}    (already baking or finished; waiting for VMI in step 3)${NC}"
else
  info "Preparing k3s artifact mirror..."
  start_k3s_mirror
  info "Creating cloud-init Secret..."
  kubectl delete secret bake-cloudinit --ignore-not-found 2>/dev/null
  emit_cloudinit | kubectl create secret generic bake-cloudinit --from-file=userdata=/dev/stdin

  info "Creating bake VM ${VM_NAME}..."
  kubectl apply -f - << YAML
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: ${VM_NAME}
  namespace: default
spec:
  runStrategy: Once
  template:
    spec:
      domain:
        cpu:
          cores: 4
        memory:
          guest: 8Gi
        devices:
          disks:
            - disk:
                bus: virtio
              name: systemdisk
            - disk:
                bus: virtio
              name: cloudinitdisk
          interfaces:
            - masquerade: {}
              name: default
      networks:
        - name: default
          pod: {}
      volumes:
        - dataVolume:
            name: ${DV_TARGET}
          name: systemdisk
        - cloudInitNoCloud:
            secretRef:
              name: bake-cloudinit
          name: cloudinitdisk
YAML
fi

echo ""

# ─────────────────────────────────────────────────────────────
banner "Step 3/4: Wait for Bake VM to Complete"
# ─────────────────────────────────────────────────────────────
echo ""
echo "  The VM is downloading and installing:"
echo "    - k3s binary       (~60 MB  from get.k3s.io)"
echo "    - k3s airgap images (~134 MB tar.zst from GitHub releases)"
if [ -n "${EXTRA_PACKAGES}" ]; then
  echo "    - extra apt pkgs   (${EXTRA_PACKAGES})"
fi
echo ""
echo "  ETA: 5–8 minutes depending on network speed."
echo "  Watch VM console: virtctl console ${VM_NAME}"
echo ""

VMI_SEEN=false
MAX_WAIT=900  # 15 minutes
ELAPSED=0

while true; do
  PHASE=$(kubectl get vmi "${VM_NAME}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")

  if [ -n "${PHASE}" ]; then
    VMI_SEEN=true
  fi

  printf "\r    VMI phase: %-12s  [%ds]" "${PHASE:-Pending}" "$ELAPSED"

  # Success: VMI reached Succeeded, or VMI disappeared after we saw it (powered off + cleaned up)
  if [ "${PHASE}" = "Succeeded" ]; then
    echo ""
    echo -e "    ${GREEN}✓${NC} Bake VM completed successfully!"
    break
  fi

  if [ "${VMI_SEEN}" = "true" ] && [ -z "${PHASE}" ]; then
    echo ""
    echo -e "    ${GREEN}✓${NC} Bake VM powered off (VMI gone)."
    break
  fi

  if [ "${PHASE}" = "Failed" ]; then
    echo ""
    echo -e "    ${RED}✗${NC} Bake VM failed!"
    echo "    Debug: virtctl console ${VM_NAME}"
    echo "    Logs:  kubectl get events --field-selector involvedObject.name=${VM_NAME}"
    exit 1
  fi

  if [ "$ELAPSED" -ge "$MAX_WAIT" ]; then
    echo ""
    echo -e "    ${RED}✗${NC} Timed out waiting for bake VM after ${MAX_WAIT}s"
    echo "    Debug: virtctl console ${VM_NAME}"
    exit 1
  fi

  sleep 10
  ELAPSED=$((ELAPSED + 10))
done

echo ""

# ─────────────────────────────────────────────────────────────
banner "Step 4/4: Delete Bake VM"
# ─────────────────────────────────────────────────────────────
echo ""

info "Deleting bake VM (DataVolume ${DV_TARGET} is preserved)..."
kubectl delete vm "${VM_NAME}" --ignore-not-found
echo -e "    ${GREEN}✓${NC} Bake VM deleted."

echo ""

# ─────────────────────────────────────────────────────────────
banner "Done!"
# ─────────────────────────────────────────────────────────────
echo ""
echo -e "  ${BOLD}Golden image ready: ${DV_TARGET}${NC}"
echo ""
echo "  Verify:"
echo "    kubectl get dv ${DV_TARGET}"
echo ""
echo "  Next steps:"
echo "    1. Build containerDisk:"
echo "         DV_SOURCE=${DV_TARGET} IMAGE_NAME=${TARGET_IMAGE} \\"
echo "           ./build-containerdisk.sh"
echo "    2. Pre-pull on Kind nodes (see Makefile pre-pull-* targets)"
echo "    3. Measure boot time:"
echo "         ./scripts/time-to-ready.sh"
echo ""
echo "  Target time-to-ready: ~50–70s (down from ~90–150s)."
echo ""

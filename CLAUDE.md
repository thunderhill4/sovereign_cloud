# CLAUDE.md — KubeVirt Multi-Cluster Platform

## Project Overview

# Sovereign Cloud × Sympozium: Agentic AI Strategy
## Active Plan
Refer to `Sovereign_Cloud_Agentic_Strategy.md` for the current implementation roadmap. 
Always verify changes against the "Architecture Constraints" section in that file.
This repo provisions Kubernetes target clusters as KubeVirt VMs on a Kind-based management cluster, with Istio ambient mesh for cross-cluster service discovery and a full-stack web UI for cluster management, AI-powered operations, and service mesh visualization.

**Clusters:**
- `cluster1` (Kind) — demo/mesh cluster. Bare by default; `07-istio-advanced/` installs
  MetalLB + Istio 1.31 ambient + Gateway API on it. (It previously hosted an ad-hoc
  ambient mesh with `httpbin`/`sleep`; that was lost in a rebuild and is not recreated
  by any script.)
- `cluster2` (Kind) — Management cluster running CAPI, KubeVirt, CDI, MetalLB, Sympozium
- `target-cluster` — k3s cluster provisioned as KubeVirt VMs on cluster2 via CAPI

## Repository Structure

```
.
├── 00-prereqs/            # clusterctl installation script
├── 01-metallb/            # MetalLB L2 LoadBalancer config + install script
├── 02-capi-init/          # Cluster API provider initialization
├── 03-target-cluster/     # Target cluster YAML manifests + generator
│   ├── target-cluster.yaml          # Full profile (4 CPU, 8Gi CP / 6Gi worker)
│   ├── target-cluster-lite.yaml     # Lite profile (2 CPU, 4Gi)
│   └── target-cluster-parallel.yaml # Full profile, worker boots in parallel (~61s)
├── 04-verify/             # Cluster health verification script
├── 05-istio/              # Istio ambient mode install + cross-cluster demo
├── ui/                    # Web dashboard (React frontend + Go backend)
│   ├── backend/           # Go HTTP API server (port 8080)
│   ├── frontend/          # React 19 + TypeScript + Tailwind v4 (port 5173)
│   └── k8s/               # Kubernetes deployment manifests for the UI
├── Makefile               # All automation targets
├── 06-sympozium/          # Sympozium install + SympoziumInstance manifests
├── run-ui.sh              # Launch UI dev servers with Sympozium env vars
├── sympozium-lb-setup.sh  # Patch Sympozium serving Services to LoadBalancer with MetalLB IPs
├── bake-golden-image.sh   # Bake Ubuntu k3s golden VM image
├── build-containerdisk.sh # Build container disk image
├── demo.sh                # Interactive cluster demo
├── show-cluster.sh        # Display cluster status
└── target-cluster-kubeconfig  # kubeconfig for target cluster
```

## MetalLB IP Assignments

Pool: `172.18.255.200–210` (cluster1, installed by
`07-istio-advanced/00-prereqs/install-metallb-cluster1.sh`), `172.18.255.211–220`
(cluster2, installed by `01-metallb/install-metallb.sh`)

**Do not run `01-metallb/install-metallb.sh` against cluster1.** It computes an IP range
then `sed`s for `IP_RANGE_PLACEHOLDER`, which does not exist in
`01-metallb/metallb-config.yaml` (the pool is hardcoded to `.211-.220`). The sed is a
no-op, so cluster1 would advertise cluster2's range on the shared L2 segment.

| IP              | Service                        | Set In                              |
|-----------------|--------------------------------|-------------------------------------|
| 172.18.255.200  | httpbin-lb (mc-demo, cluster1) | cross-cluster demo manifests        |
| 172.18.255.201  | Act 1 north-south Gateway (cluster1) | `07-istio-advanced/act1-gateway/03-gateway.yaml` |
| 172.18.255.202  | Act 3 east-west gateway (cluster1)  | `07-istio-advanced/act3-multicluster/01-...yaml` |
| 172.18.255.203  | Act 4 agentgateway (cluster1)       | `07-istio-advanced/act4-ai-gateway/01-...yaml` |
| 172.18.255.204  | Kiali (cluster1)                    | `07-istio-advanced/act5-observability/run.sh` |
| 172.18.255.211  | kubeui-frontend                | `ui/k8s/kubeui.yaml`                |
| 172.18.255.212  | sympozium-apiserver (UI)       | `sympozium-lb-setup.sh`             |
| 172.18.255.213  | cluster2-agent (Sympozium)     | `sympozium-lb-setup.sh`             |
| 172.18.255.214  | target-cluster-agent (Sympozium) | `sympozium-lb-setup.sh`           |
| 172.18.255.215  | target-cluster API server      | `03-target-cluster/target-cluster.yaml` |
| 172.18.255.216  | target-cluster-nginx proxy (legacy `05-istio` demo, allocated only while it runs) | cross-cluster demo |
| 172.18.255.217  | security-agent                 | `ui/k8s/security-agent.yaml`        |
| 172.18.255.218  | cost-analyzer (Sympozium)      | `sympozium-lb-setup.sh`             |
| 172.18.255.219  | incident-responder (Sympozium) | `sympozium-lb-setup.sh`             |
| 172.18.255.220  | host-ollama-lb (optional)      | `snippets/host-ollama/` (`WITH_LB=1`) |
| 172.18.255.221  | Act 3 east-west gateway (cluster2) | `07-istio-advanced/act3-multicluster/02-...yaml` |
| 172.18.255.222  | mesh-sre-agent (Sympozium)     | `sympozium-lb-setup.sh`             |

**Note:** `.221` is outside the cluster2 pool as originally configured (`.211-.220`).
`07-istio-advanced` widens that pool to `.211-.225` rather than reusing `.216`, which the
legacy `05-istio` cross-cluster demo claims for its nginx proxy — double-booking it would
make the two demos mutually exclusive.

**Critical:** Never reassign the IPs above without updating the corresponding source file AND `sympozium-lb-setup.sh` AND `run-ui.sh`.

## Development Workflows

### Full Setup (from scratch)

```bash
make all                # prereqs + metallb + capi-init + target-cluster
make verify             # wait for VMs, then verify cluster health
make istio              # install Istio ambient on target cluster
make sympozium-install  # install cert-manager + Sympozium + agents on cluster2
make sympozium-lb       # expose Sympozium serving Services via MetalLB
make ui                 # launch web UI dev servers
```

### Cluster Lifecycle

```bash
make target-cluster            # DEFAULT — WARM fast-path (parallel boot, ~33.7s both nodes measured)
make target-cluster-warm       # same warm fast-path, explicit
make target-cluster-lite       # legacy lite, 2 CPU / 4Gi (sequential, :latest)
make target-cluster-full       # legacy full, 4 CPU / 8Gi (sequential, :latest)
make target-cluster-parallel   # full profile, worker boots in parallel (~61s, :latest)
make clean                     # delete target cluster (CAPI cleans up VMs)
```

**Warm fast-path** (`target-cluster-warm.yaml` / `.tmpl.yaml`) — **now the default** for
`make target-cluster`, `make all`, and the UI's Deploy Cluster (default image option).
It combines the parallel-boot worker (below) with a **warm-baked golden image** (`:warm`):
the image carries fixed CAs + token (`03-target-cluster/warm-ca/`) so `scripts/seed-cluster-secrets.sh`
can pre-seed matching CAPI secrets — KThrees adopts them, so first boot needs **no
`--cluster-reset` and no cert purge**. **Measured time-to-ready: median 34.5s**
(34.5 / 34.2 / 34.7, spread 0.5s) — the earlier "<40s target" in this file was an
aspiration never met by the image work, and the warm image itself contributes **no** speedup
over the plain parallel build (see `docs/sub-60s-cluster-strategy.md` §5-6; measured
warm 48.5s vs parallel 46.6-50.0s). What actually moved the number was
two one-line config fixes, neither of them an image change (50.0s -> 34.5s, -31%):

1. **`advertise-address` on the control plane** (50.0s -> 42.7s). A KubeVirt VM's pod IP
   is reachable from the node but **not from other pods**, so a worker whose agent is
   handed the CP's pod IP burns a full 10.000s dial timeout — kubelet start is gated on
   it — before falling back to the VIP. In all three target-cluster manifests.
2. **`supportContainerResources` on the KubeVirt CR** (42.7s -> 34.5s). KubeVirt gives
   the `volumesystemdisk` containerDisk container `cpu: 10m` and `guest-console-log`
   `cpu: 15m` by default; CFS throttling then makes the launcher pod take ~8s to reach
   qemu on every VM start. Raised to 1 core by `scripts/configure-kubevirt-perf.sh`
   (`make kubevirt-perf`, auto-run from `02-capi-init/init-management-cluster.sh`).
   **This is cluster state, not manifest state — re-run it after any cluster2 rebuild.**

**Re-measured 2026-09-17 after the Kubernetes/Istio/k3s upgrade** (Kind node image
1.35.0 -> 1.37.0, k3s v1.31.4+k3s1 -> v1.37.0+k3s1, cluster2 fully rebuilt): both nodes
Ready at median **33.7s** (30.4 / 35.1 / 33.7; control plane alone 24.7s) — no measurable
change from the pre-upgrade 34.5s. **An earlier 23.9s figure recorded here was wrong: it
was the control plane only.** Both `scripts/phase-timings.sh` and `scripts/time-to-ready.sh`
wait for "2 nodes Ready" *by count*, and the `:warm` image carries a ghost Node object
(`ubuntu-bake-vm-warm`, from the bake VM, still marked Ready) that the control plane's
async `warm-ghost-node-cleanup` step deletes only after startup. So the count reaches 2
(control plane + ghost) before the worker joins — observed in every run on 2026-09-17.
`phase-timings.sh` then also finds no worker Ready timestamp and treats it as zero, so its
"TOTAL (both nodes)" silently equals the control-plane time. Until the scripts wait for
nodes **by name**, measure by polling `-cp-` and `-workers-` Ready separately. The
worker's `[airgap-install]` output also never prints the "k3s started" line the
instrument looks for, so that worker row is always empty. Both Chapter 8 fixes above
still hold (re-applied fresh on the rebuilt cluster2).

Use `./scripts/phase-timings.sh` (`make phase-timings`; per-phase breakdown, read-only,
no SSH needed) rather than a single aggregate number when judging any change here — the
run-to-run spread is now 0.5s, so regressions are visible, but both fixes above were
invisible against the old 6.1s spread. The Make target runs
`ensure-warm-image` first (pre-pulls the `:warm` image, baking it if absent). Fixed CA + token
are committed — **demo use only**; safe only because exactly one `target-cluster` runs at a time.

**Parallel-boot variant** (`target-cluster-parallel.yaml`): normally CAPI serializes
worker creation behind the control plane (worker starts ~75s in → ~125s end-to-end).
This variant boots the worker VM alongside the CP (~61s end-to-end) via three changes:
a **pre-seeded `target-cluster-token` secret** (KThrees adopts it, so the join token is
known before the CP exists), a **static worker bootstrap** (`bootstrap.dataSecretName`,
bypassing the KThrees provider's wait-for-control-plane-initialized gate), and the
`machineset.cluster.x-k8s.io/skip-preflight-checks: All` annotation (skips CAPI's
`ControlPlaneIsStable` MachineSet gate). The worker's k3s-agent retries the static CP
VIP (`172.18.255.215`) until it answers. Token is static/committed — **demo use only**.

### Web UI Development

```bash
make ui       # runs run-ui.sh — starts Go backend + Vite frontend with hot reload
make ui-build # production build: frontend to ui/dist, backend binary to ui/dist/backend
```

The `run-ui.sh` script sets all Sympozium env vars and launches both servers:
- Frontend: `http://localhost:5173` (Vite dev server)
- Backend: `http://localhost:8080` (Go `go run .`)

Production deploy: `cd ui/k8s && ./build-and-deploy.sh` → available at `http://172.18.255.211`

### Golden Image

```bash
make bake-image   # ~5–8 min, bakes k3s binary + airgap images into a DataVolume
make pre-pull     # pre-pull the container disk on Kind nodes for faster deploys
make registry     # inspect images in local registry at 172.18.0.2:5000
```

## Web UI Architecture

### Frontend (`ui/frontend/`)

- **React 19** + **TypeScript** + **Vite** + **Tailwind CSS v4**
- Dependencies: `lucide-react`, `radix-ui`, `react-markdown` + `remark-gfm`, `react-router-dom v7`, `@xterm/xterm` + `@xterm/addon-fit` (web terminal)
- Path alias: `@/` → `ui/frontend/src/`

**Three operational modes** (toggled via header; state in `useMode` hook):
- `sre` — Cluster management dashboard
- `ai` — AI chat via Sympozium agents
- `visual` — Service mesh visualization

**Key files:**
- `src/App.tsx` — Root: renders `<AppShell />`
- `src/components/layout/AppShell.tsx` — Layout with mode-aware sidebar + main content
- `src/hooks/useMode.ts` — Mode state (`sre` → `ai` → `visual` → `sre` cycle)
- `src/lib/api.ts` — All API calls; streaming operations use `AsyncGenerator` over SSE
- `src/lib/types.ts` — Shared TypeScript types (`AppMode`, `DeployLogEntry`, etc.)

**Page components (`src/pages/`):**
- `SREDashboard.tsx` — Routes to SRE sub-views based on `activePath`
- `AIChat.tsx` — Renders `AgentConsole` (`src/components/ai/AgentConsole.tsx`): full-page iframe of the de-branded agent console at `<same-hostname>:8081` (the backend's console proxy — vendor top bar hidden, login/namespace pre-seeded; all dashboard side panes work)
- `VisualDashboard.tsx` — Routes to visual sub-views based on `visualPath`

**SRE components (`src/components/sre/`):**
- `Dashboard.tsx`, `NodeList.tsx`, `PodList.tsx`, `VMList.tsx`, `VMDetail.tsx`
- `EventList.tsx`, `NamespaceSelector.tsx`, `ImageRepo.tsx`, `Registry.tsx`
- `ClusterManager.tsx` — Target cluster deploy/delete with streaming log display
- `ResourceTable.tsx` — Reusable table component
- `TerminalView.tsx` — Multi-tab web terminal (xterm.js ↔ `/api/v1/terminal` WebSocket PTY); tabs and sessions persist across SRE sub-view switches (kept mounted in `SREDashboard.tsx`), but not across mode switches

**Visual components (`src/components/visual/`):**
- `TopologyView.tsx`, `TrafficManagement.tsx`, `ServiceManagement.tsx`
- `SecurityCenter.tsx`, `Observability.tsx`, `AmbientMesh.tsx`, `Diagnostics.tsx`
- `VisualSidebar.tsx`

### Backend (`ui/backend/`)

- **Go** with standard library `net/http` (no framework); non-stdlib deps: `k8s.io/client-go`, `gorilla/websocket` + `creack/pty` (web terminal)
- Module: `kubeui/backend` (Go 1.26)
- Kubernetes client: `k8s.io/client-go` v0.36.2

**API routes (`main.go`):**

| Method | Path | Handler |
|--------|------|---------|
| GET | `/api/v1/cluster/status` | Management cluster summary |
| POST | `/api/v1/cluster/deploy?profile=lite\|full` | Deploy target cluster (SSE stream) |
| GET | `/api/v1/cluster/deploy/logs` | Stream current deploy logs (SSE) |
| POST | `/api/v1/cluster/delete` | Delete target cluster (SSE stream) |
| GET | `/api/v1/cluster/target-status` | Target cluster CAPI status |
| DELETE | `/api/v1/cluster/target-delete` | Delete target cluster |
| POST | `/api/v1/cluster/istio` | Install Istio on target cluster (SSE stream) |
| GET | `/api/v1/images` | CDI DataVolumes |
| GET | `/api/v1/registry/images` | Local container registry catalog |
| DELETE | `/api/v1/registry/images/{name}:{tag}` | Delete registry image |
| GET | `/api/v1/registry/config` | Registry connection status |
| GET | `/api/v1/nodes` | Kubernetes nodes |
| GET | `/api/v1/pods?namespace=` | Pods (optional namespace filter) |
| GET | `/api/v1/namespaces` | Namespaces |
| GET | `/api/v1/events?namespace=` | Events |
| GET | `/api/v1/virtualmachines?namespace=` | All VMs |
| GET | `/api/v1/virtualmachines/{ns}/{name}` | Single VM |
| GET | `/api/v1/terminal` | Web terminal (WebSocket → PTY shell; one shell per connection/tab) |
| POST | `/api/ai/chat` | Proxy to Sympozium agent (SSE stream, OpenAI-compat) |
| GET | `/api/ai/agents` | List available Sympozium agents (SympoziumInstance CRs) |
| ANY | `:8081/*` (separate port) | De-branding reverse proxy to the Sympozium dashboard (`handlers/dashboard_proxy.go`): hides the vendor top bar via injected CSS, pre-seeds `sympozium_token`/`sympozium_namespace` in localStorage; separate port because the SPA's absolute `/assets` + `/api/v1` paths would collide with kubeui's routes |
| GET | `/healthz` | Health check |

**CORS:** Allows `localhost:5173`, `127.0.0.1:5173`, `172.18.255.211`

**Handler files:**
- `handlers/ai.go` — OpenAI-compatible chat-completions proxy to Sympozium; SSE streaming
- `handlers/dashboard_proxy.go` — Console proxy on `:8081` (AI tab embed); upstream from `SYMPOZIUM_DASHBOARD_URL`, port from `SYMPOZIUM_CONSOLE_PORT`
- `handlers/cluster_deploy.go` — CAPI cluster lifecycle; streaming log manager
- `handlers/resources.go` — Nodes, pods, VMs, events, namespaces
- `handlers/cdi.go` — CDI DataVolume listing
- `handlers/registry.go` — Container registry catalog + delete

### AI Integration (Sympozium / OpenAI Chat Completions)

The backend proxies AI chat to Sympozium agents via their OpenAI-compatible serving-mode endpoint:

- Endpoint: `POST <agent-base>/v1/chat/completions` with `stream: true`
- Auth: `Authorization: Bearer $SYMPOZIUM_API_TOKEN` (optional; token comes from the `sympozium-ui-token` Secret created by Sympozium)
- Agent URL resolution: env var `SYMPOZIUM_AGENT_URL_<NAME_UPPER>` → `SYMPOZIUM_AGENT_URL` (default agent) → in-cluster DNS `http://<name>-server.<namespace>.svc.cluster.local:8080/`
- Agent list: backend queries the Kubernetes API for `SympoziumInstance` CRs in `$SYMPOZIUM_NAMESPACE` (serving-enabled only) via the dynamic client
- Timeout: 120 seconds per request
- SSE buffer: 256KB scanner buffer for large lines

**Env vars for `run-ui.sh`:**
```
SYMPOZIUM_NAMESPACE=sympozium-system
SYMPOZIUM_DEFAULT_AGENT=cluster2-agent
SYMPOZIUM_AGENT_URL=http://172.18.255.213:8080/
SYMPOZIUM_AGENT_URL_TARGET_CLUSTER_AGENT=http://172.18.255.214:8080/
SYMPOZIUM_AGENT_URL_MESH_SRE_AGENT=http://172.18.255.222:8080/
SYMPOZIUM_API_TOKEN=<token from sympozium-ui-token Secret>
SYMPOZIUM_DASHBOARD_URL=http://172.18.255.212:8080   # console-proxy upstream (server-side; in prod the in-cluster svc DNS)
CLAUDE_DIR=<repo root>
```

### `mesh-sre-agent` — Istio ambient mesh observability

Third serving agent (`06-sympozium/mesh-sre-agent.yaml`, in the `kubectl apply -k
06-sympozium/` bundle), serving on `172.18.255.222`. Answers "is the mesh healthy /
what is degraded" for **cluster2's** Istio 1.31 ambient mesh: istiod / ztunnel /
istio-cni readiness, Gateway API `Programmed` status, waypoint enrollment,
AuthorizationPolicy coverage, east-west wiring. Model `qwen2.5:7b` (not llama3.2 —
it confabulates kubectl output when a tool call fails silently).

Skills: `web-endpoint` (declarative serving) + `sre-observability` (triage playbooks)
+ `k8s-ops` (the kubectl sidecar task/chat runs actually use). All three must be
listed on the Agent CR — see the 0.10.47 propagation note in `cluster2-agent.yaml`.

**`sre-observability`'s bundled clusterRBAC covers no Istio API groups** (core, apps,
autoscaling, metrics only), so every mesh query dead-ends on "permission issue" until
`06-sympozium/agent-istio-rbac.yaml` grants read on `networking.istio.io`,
`security.istio.io`, `telemetry.istio.io`, and `gateway.networking.k8s.io` to the
`sympozium-agent` SA. Same gap and same additive-ClusterRole fix as
`agent-kubevirt-rbac.yaml` for `kubevirt.io`.

**Scope is deliberately cluster2-only, and two things enforce that** — do not assume
the SkillPack's `prometheus-query` / `loki-and-logs` skills work here:
1. cluster1's Prometheus is a **ClusterIP** Service, unreachable from a cluster2 pod.
   Kiali is a LoadBalancer (`.204`) but shows cluster1's view.
2. Agent egress is denied by default — `sandbox-restricted`'s `networkPolicy`
   (`denyAll` + `allowedEgress` for Ollama and the K8s API), reinforced by the
   `sympozium-allow-ollama` NetworkPolicy. Neither Kiali nor Prometheus is allowlisted.

Opening the metrics path needs **both** the policy and the NetworkPolicy widened —
RBAC alone will not do it.

**Do not rely on the `systemPrompt` to enforce that.** The briefing opens with a
RULE 1 telling the agent to refuse metrics questions verbatim and never estimate.
Measured on qwen2.5:7b across repeated attempts, it does **not** reliably comply: it
ignores "do not run a command", and in one run answered "we can make an educated
guess based on the availability of related metrics". It never actually emitted a
fabricated number in testing — but the guard is a nudge, not a control. Moving the
rule to the top of the prompt did not fix it. If refusal must be guaranteed, enforce
it below the model (policy `toolGating`, or egress that fails closed and is surfaced
as an error), not in the prompt. Same class of limitation as the 7B tool-calling
ceiling already noted for llama3.2.

Verify reach deterministically, never by trusting the model:
`kubectl get gateways.gateway.networking.k8s.io -A --as=system:serviceaccount:sympozium-system:sympozium-agent`

**Demo:** `06-sympozium/demo-mesh-sre.sh` drives the real agent through 7 scenes and
(with `SCENE_DIR=<dir>`) writes a transcript; `scripts/render-demo-video.sh <dir>
<out.mp4>` renders it to `docs/demo/mesh-sre-agent-demo.mp4`. Frames are HTML rendered
by headless Chrome and stitched with ffmpeg — there is no asciinema/vhs/agg on this
host. The demo re-asks a question only when the reply is *structurally* broken (empty,
or a raw tool call leaked as text); a well-formed but unhelpful answer is kept, so the
video shows the agent's real hit rate rather than a curated one.

**Not in kubeui's agent dropdown.** `HandleListAgents` still queries the removed
`SympoziumInstance` CRD and falls back to listing only `SYMPOZIUM_DEFAULT_AGENT`, so
new `Agent` CRs never appear there. Reachable directly on `.222` or via
`SYMPOZIUM_AGENT_URL_MESH_SRE_AGENT`.

### Warm Pool (pre-deployed standby)

The backend keeps one `target-cluster` pre-built and labeled `pool.local/state=WARM`
(reconcile goroutine in `ui/backend/handlers/pool.go`). Deploy **claims** a warm standby
(relabel `CLAIMED`, synthetic SSE) in seconds instead of building (~50s). Delete tears
down and the controller rebuilds a standby in the background. Invariant: at most one
`target-cluster` at a time. Endpoint: `GET /api/v1/cluster/pool-status`
→ `{state: none|building|warm|claimed, clusterReady, lastError}`. **Opt-in**:
`run-ui.sh` defaults `POOL_ENABLED=false`; export `POOL_ENABLED=true` before
running it to have the backend auto-build/rebuild a standby in the background.
Standby builds via `target-cluster-parallel.yaml` (`:latest`);
the on-demand fallback uses the warm image. Demo-only (fixed CA/token, single cluster).

## Infrastructure Conventions

### CAPI / KubeVirt

- CAPI provider: `CAPK` (Cluster API Provider KubeVirt)
- Control plane: `KThreesControlPlane` (k3s)
- Target cluster network: pods `10.42.0.0/16`, services `10.43.0.0/16`
- VMs use bridge networking; each VM gets a unique pod IP

### Istio Ambient Mode

- No sidecar proxies — uses ztunnel (L4 mTLS) + istio-cni-node DaemonSet
- Cross-cluster traffic via `ServiceEntry` resources mapping hostnames to MetalLB IPs
- mTLS terminates at cluster edge; cross-MetalLB traffic is plain HTTP

### Container Registry

- Local registry: `172.18.0.2:5000` (insecure)
- Golden image: `172.18.0.2:5000/ubuntu-noble-k3s:latest`

## Streaming Pattern

All long-running operations (deploy, delete, Istio install) use **Server-Sent Events (SSE)**:

**Backend:** writes `data: <json>\n\n` lines; sends `data: [DONE]\n\n` when complete.

**Frontend (`api.ts`):** `AsyncGenerator` functions read the SSE stream and `yield` parsed `DeployLogEntry` objects. The UI consumes these with `for await...of`.

**Log entry types:** `step | info | success | error | warn | done`

## Kubernetes Deployment

The UI is deployed to the `kubeui` namespace on cluster2 (`ui/k8s/kubeui.yaml`):

- `ServiceAccount`: `kubeui-backend` with `ClusterRole` granting read access to nodes, pods, namespaces, events, VMs, DataVolumes, CAPI resources
- Backend deployment + frontend served as static files or separate container
- Frontend `LoadBalancer` at `172.18.255.211`

## Common Pitfalls

- **Kubernetes/Istio/k3s upgraded 2026-09-16**: cluster1 + cluster2 Kind node image
  1.35.0 -> **1.37.0** (`kind` CLI also bumped 0.31.0 -> 0.33.0 to get that image as
  its default), Istio 1.30.3 -> **1.31.0**, target-cluster k3s v1.31.4+k3s1 ->
  **v1.37.0+k3s1**, `ui/backend`'s `client-go`/`k8s.io/api`/`apimachinery` 0.36.2 ->
  **0.37.0**. Both Kind clusters were fully deleted and recreated (no in-place Kind
  upgrade path exists) — everything downstream was rebuilt from scratch: KubeVirt
  v1.9.0 + CDI v1.66.0 (**these, plus the `ubuntu-noble-dv` base-image DataVolume and
  the `cluster1`/`cluster2` Kind clusters themselves, are NOT created by any script in
  this repo** — they're manual prerequisites per the README, easy to forget when
  reproducing this), MetalLB, CAPI/CAPK/cluster-api-k3s providers, the `:warm` golden
  image, target-cluster, all of `07-istio-advanced` (prereqs through act5), and
  Sympozium 0.10.75. Full re-verification: `make istio-adv-verify` 22/22,
  `make verify` clean, live chat completions through all three Sympozium agents.
  **No Istio release supports Kubernetes 1.37 yet** (checked 1.31.0's own announcement
  — tested range is still 1.32-1.36) so this combination is running deliberately
  unsupported; it happened to pass every check here, but re-verify before trusting
  that a future patch bump keeps it that way. Findings from the rebuild:
  1. **`bake-common.sh`'s documented `K3S_VERSION` override never controlled the
     bake.** The version that actually reaches the VM is a second, independent
     hardcoded literal inside the embedded `/usr/local/bin/bake.sh` cloud-init
     content (that block is written to the VM verbatim; the outer bash variable
     never reaches it). Both literals are now `v1.37.0+k3s1` and commented to stay
     in sync by hand.
  2. **The registry survives cluster deletion** (it's a separate container, not
     inside either Kind cluster) but that means its `ubuntu-noble-k3s:latest`/
     `:warm`/`:preinit` tags keep whatever was baked into them — after this bump
     they still had the **old k3s v1.31.4 binary** until explicitly rebaked.
     `ensure-warm-image`/`pre-pull-*` only check tag *presence*, not version, so
     they would have silently reused the stale image. Rebaked `:warm` for the
     default path; **`:latest`/`:preinit` (the `target-cluster-lite/full/parallel`
     legacy paths) were NOT rebaked** and still carry the old k3s binary — rebake
     with `make bake-image build-containerdisk-preinit pre-pull-preinit` (and the
     plain non-preinit equivalent for `:latest`) before using those paths.
  3. **The k3s bootstrap/control-plane CAPI providers (still v0.3.0, unchanged)
     ship a dead `kube-rbac-proxy` image** (`gcr.io/kubebuilder/kube-rbac-proxy:v0.16.0`,
     `ImagePullBackOff: not found`). Both provider Deployments sit at 1/2 ready
     forever, `02-capi-init/init-management-cluster.sh`'s own readiness waits
     silently time out into their `|| true` fallback (so `make capi-init` reports
     success anyway), and the first `target-cluster` deploy fails opaquely on
     `failed calling webhook ...kthreescontrolplane: connection refused` (the
     mutating webhook that image would have served never started). Fixed by
     patching both Deployments' `kube-rbac-proxy` container to
     `quay.io/brancz/kube-rbac-proxy:v0.16.0` (confirmed pullable, and what the
     equivalent Deployments ran successfully before this rebuild) — now done
     automatically by `init-management-cluster.sh` right after `clusterctl init`.
  4. **`scripts/configure-kubevirt-perf.sh`'s auto-run from `init-management-cluster.sh`
     can silently no-op** (`|| true` swallows the failure) if it races KubeVirt's own
     operator still initializing. Verify `supportContainerResources` actually landed
     (`kubectl get kubevirt kubevirt -n kubevirt -o jsonpath='{.spec.configuration.supportContainerResources}'`)
     after any fresh `make capi-init`; re-run `make kubevirt-perf` by hand if empty.
  5. Fresh `AgentRun`s created concurrently by `fix-web-proxy-image.sh` can race
     into a terminal `Failed` phase (`"server Deployment not found"`) — same
     "controller does not retry a terminal Failed run" gap noted elsewhere in this
     file. `kubectl delete agentrun <name>-web-endpoint` and let the controller
     regenerate it.
- The backend uses `go run .` in development — no pre-compilation needed
- The `CLAUDE_DIR` env var is passed to the backend so it can find repo scripts (e.g., for `kubectl apply`)
- `target-cluster-kubeconfig` is a plain file in the repo root — used by `make istio` and verification scripts
- If Sympozium serving Services aren't reachable, run `make sympozium-lb` to (re)patch them to LoadBalancer
- `target-cluster-agent`'s tools reach the target cluster via the `sympozium-system/target-cluster-kubeconfig` Secret (mounted by the `target-k8s-ops` SkillPack). The target CA changes on every redeploy, so a stale Secret makes the agent's `kubectl` fail TLS (`x509: unknown authority`) **silently** — llama3.2 then confabulates plausible namespace output. `06-sympozium/refresh-target-kubeconfig.sh` re-syncs it and is auto-invoked by `04-verify/verify-cluster.sh` (`make verify`), the UI deploy (`cluster_deploy.go`), and the warm-pool claim (`pool.go`). Detect staleness by comparing the CA sha of the Secret vs `target-cluster-kubeconfig`; verify reach **deterministically** (never trust the model) with a pod in `sympozium-system` mounting the Secret and running `KUBECONFIG=/etc/target-kube/kubeconfig kubectl get ns`.
- **FIXED UPSTREAM in 0.10.47.** The web-proxy now runs correctly with
  `readOnlyRootFilesystem: true` (verified: both serving pods `1/1 Running`, 0 restarts,
  under exactly the condition that used to crash them). `fix-web-proxy-rootfs.sh` is
  retirable. Original 0.10.38 description follows.
- Sympozium's `web-proxy` image crashes forever under `readOnlyRootFilesystem: true` (exit 2, zero log output, every ~30s) — the CRD has no securityContext override, so `install-sympozium.sh`/`demo-sympozium.sh` patch each `<instance>-web-endpoint-server` Deployment via `06-sympozium/fix-web-proxy-rootfs.sh` after applying. If a `SympoziumInstance` you add manually shows `0/1` endpoints and endless restarts, run that script against its Deployment.
- The `sympozium-node-probe` DaemonSet (hostNetwork) checks `127.0.0.1:11434` to detect a local Ollama and populate `sympozium.ai/inference-*` node annotations (drives the Sympozium dashboard's Gateway/hardware view) — same host-vs-in-cluster reachability gap as the agent traffic path (see AI Integration section). Fix = an iptables OUTPUT DNAT rule (`127.0.0.1:11434` → `172.18.0.1:11434`) in the Kind node's netns, applied two ways: `06-sympozium/fix-node-probe-loopback.sh` (`make sympozium-fix-node-probe`, instant one-shot via `docker exec`) and `06-sympozium/node-probe-loopback-ds.yaml` (in the kustomize bundle; privileged hostPID DaemonSet that re-asserts the rule every 60s via `nsenter`, so it survives node-container/host restarts — the one-shot alone was lost on reboot and silently blanked the Gateway panel again).
- The `sympozium-llmfit-daemon` (hardware view / model-fit in the Sympozium dashboard) detects NVIDIA GPUs by shelling out to `nvidia-smi`, which doesn't exist in its container (and couldn't run: no NVML lib, no `/dev/nvidia*` in the pod) — so the NVIDIA entry gets `vram=null` and the AMD iGPU (read from sysfs `mem_info_vram_total`, ~0.5Gi carve-out) is reported as the primary GPU instead. `install-sympozium.sh` runs `06-sympozium/fix-llmfit-nvidia-smi.sh` (`make sympozium-fix-llmfit-gpu`): it captures real answers from the host's `nvidia-smi`, writes a replay shim into the Kind node at `/opt/llmfit-shim/`, and mounts it into the daemon at `/usr/local/sbin` (NOT `/usr/local/bin` — that holds the `llmfit` binary). Shim values are static; re-run after node recreation or GPU/driver changes.
- The Sympozium dashboard (172.18.255.212:8080) shows **no agents/runs/schedules/ensembles** until you switch its namespace picker (header dropdown) to `sympozium-system` — the frontend appends `?namespace=<localStorage sympozium_namespace>` to every API list call and defaults to `default`, where nothing lives. The choice persists in localStorage per browser; there is no server-side default-namespace knob (verified against the v0.10.38 apiserver binary). Console shortcut: `localStorage.setItem('sympozium_namespace','sympozium-system'); location.reload()`.
- **The `web-endpoint` SkillPack hard-codes `web-proxy:latest`, a MUTABLE tag.** The
  control plane is pinned but that tag is not, so it drifts to a newer web-proxy which
  then crash-loops with **exit code 2 and zero log output**, ~every 30s, forever — never
  binding :8080, with liveness/readiness reporting connection refused and nothing in
  events or logs naming the cause. Observed at ~5,680 restarts before diagnosis. Fix:
  `06-sympozium/fix-web-proxy-image.sh` (pins the sidecar to the installed chart
  version, purges the node's cached `:latest` since `pullPolicy: IfNotPresent` would
  reuse it, and deletes the serving AgentRuns so the controller regenerates the
  Deployments). Re-run `./sympozium-lb-setup.sh` afterwards — regenerated Services come
  back as ClusterIP and lose their MetalLB IPs.
- **`172.18.0.2:5000` is a registry ALIAS, and only containerd can resolve it.**
  docker gives `172.18.0.2` to whichever container joined the kind network first —
  here that is `cluster2-control-plane`, not the registry (which sits at `172.18.0.4`
  and publishes to the host on `localhost:5000`). Image pulls work because
  `scripts/fix-registry-hosts.sh` writes `/etc/containerd/certs.d/<alias>/hosts.toml`
  on each node, but **that mapping is containerd-only** — any plain HTTP client
  dialing the alias gets connection refused. This silently broke the UI's Registry
  tab (`{"error":"failed to connect to registry"}`, status `disconnected`) from both
  the host and in-cluster. `ui/backend/handlers/registry.go` now splits the two
  concepts: `REGISTRY_URL` is where it dials, `REGISTRY_ALIAS` is what it displays
  (the alias is what image references use, so it is what a user needs to see).
  `run-ui.sh` prefers `localhost:5000` (published port, stable across IP drift) and
  falls back to the container's live kind-network IP. `ui/k8s/kubeui.yaml` sets both
  explicitly — its `REGISTRY_URL` holds a real IP that **drifts**, so re-check it if
  the tab shows "disconnected" after a docker or host restart.
- **Upgrading the Sympozium chart:** apply the `sympozium-crds` chart first (Helm never
  upgrades CRDs in `crds/`), then `helm upgrade --server-side=true --force-conflicts`.
  Note `--server-side=true` **with a value** — Helm 4 made the flag take an argument, so
  a bare `--server-side --force-conflicts` fails with `invalid/unknown release
  server-side apply method: --force-conflicts`. `--force-conflicts` is required because
  the controller mutates its own built-in SkillPacks (`spec.sidecar.mountWorkspace`).
- **The "helm install silently drops built-in SkillPacks" pitfall is an admission race,
  not a drop.** Revision 1 of this release recorded: `failed calling webhook
  "vskillpack.sympozium.ai": dial tcp ...: connection refused` — the chart applies its
  SkillPacks before its own validating webhook is serving. On the 0.10.47 upgrade, with
  the webhook already running, all 11 SkillPacks landed with nothing missing.
- `make pre-pull` dramatically speeds up VM provisioning by pre-loading the container disk on Kind nodes
- `06-sympozium/labs/` — hands-on labs for each Sympozium capability (serving API, AgentRun, schedules, policies, MCP tools, ensembles, model fit); see `labs/README.md`. **Load-bearing finding from these labs:** `SympoziumInstance` has no controller reconciling it on this installed version (0.10.38) — only the separate `Agent` CRD is. `cluster2-agent`/`target-cluster-agent` work because they have both objects sharing a name; any new agent needs an `Agent` CR (not just a `SympoziumInstance`) or `AgentRun`/`SympoziumSchedule` reference to it fails admission. This affects how "Architecture Constraints" rule #4/#7 in `Sovereign_Cloud_Agentic_Strategy.md` (which assume `SympoziumInstance.spec.policyRef` binds policy) actually get satisfied in practice — `policyRef` must live on the `Agent` object to take effect. **Both agents are now committed as `Agent` CRs** (in `06-sympozium/{cluster2-agent,target-cluster-agent}.yaml`, alongside the kept `SympoziumInstance` which kubeui's dropdown lists), carrying model, `policyRef`, the environment briefing (`spec.memory.systemPrompt`), and `skills: [web-endpoint]`. **Declarative serving:** `web-endpoint` SkillPack has `sidecar.requiresServer: true`, so listing it in the Agent's `spec.skills` makes the AgentRun controller create the `mode: server` run + `<name>-web-endpoint-server` Deployment — no SympoziumInstance needed (on the migrated cluster a legacy Instance-owned serving run of the same name exists; the controller respects it, no duplicate). So `make sympozium-install` on a clean cluster now stands up briefed, serving agents unaided.
- **SUPERSEDED (was true on 0.10.38; the cluster now runs 0.10.47).** On 0.10.47 the
  apiserver DOES copy an Agent's `spec.skills` onto runs it creates, so the fix is to
  declare tool SkillPacks on the `Agent` CR — which `06-sympozium/{cluster2-agent,
  target-cluster-agent}.yaml` now do (`k8s-ops` / `target-k8s-ops` alongside
  `web-endpoint`). **The upgrade briefly made this worse before it made it better:**
  with only `web-endpoint` declared, chat runs got a non-empty `spec.skills` containing
  just a serving sidecar, which a `task`-mode run ignores — and non-empty meant the
  `skills-webhook`'s "inject only when `spec.skills` is empty" guard never fired, so
  pods came up with no tool sidecar at all. The `skills-webhook` is now redundant and
  can be retired; it is still deployed and harmlessly no-ops. Verified after the fix: a
  `POST /api/v1/runs` chat run returned the real node name `cluster2-control-plane`.
  Original 0.10.38 description follows.
- **Chat/dashboard AgentRuns get no skill sidecar, so tool calls silently go nowhere** ("It seems there might be an issue with the skill sidecar..." is the model giving up, not a real sidecar crash). Root cause: `POST /api/v1/runs` (what the dashboard chat and kubeui's `/api/ai/chat` proxy both create runs through) never sets `spec.skills` — nothing in 0.10.38 propagates an agent's tools into runs spawned on its behalf. A run without `spec.skills` gets only the `agent` + `ipc-bridge` containers, no `k8s-ops` sidecar to execute `kubectl`/`virtctl`. Fixed by `06-sympozium/skills-webhook/`: a mutating admission webhook (`MutatingWebhookConfiguration agentrun-skills-injector`, `failurePolicy: Ignore`) that patches `spec.skills` onto CREATEd AgentRuns when `spec.mode == "task"` and `spec.skills` is empty, choosing the SkillPack **per agent** via `INJECT_SKILL_MAP` (default `cluster2-agent=k8s-ops,target-cluster-agent=target-k8s-ops`). `cluster2-agent` gets `k8s-ops` (in-cluster SA → cluster2 API); `target-cluster-agent` gets `target-k8s-ops` (mounts the `target-cluster-kubeconfig` Secret → reaches *inside* the target k3s cluster at `172.18.255.215:6443`). Deploy: `06-sympozium/skills-webhook/build-and-deploy.sh` (builds the Go binary + image, `kind load docker-image` into cluster2, applies `deploy.yaml`; TLS via a self-signed cert-manager `Issuer`+`Certificate`, CA auto-injected via `cert-manager.io/inject-ca-from`). Verified: a `curl POST /api/v1/runs` with no `skills` field came back with real `kubectl get nodes` output instead of the sidecar-error text. Separately, weaker local models (llama3.2) can still emit a malformed tool call as raw text even with the sidecar present — that's the existing 7B tool-calling limitation, not this bug.
- **STALE as of a from-scratch rebuild against the currently-published chart**: `sympoziuminstances.sympozium.ai` is no longer shipped as a CRD at all (verified absent on chart versions 0.10.38 through 0.10.47 pulled fresh from `https://deploy.sympozium.ai/charts`) — applying one now fails admission outright (`no matches for kind "SympoziumInstance"`). The `SympoziumInstance` documents in `cluster2-agent.yaml`/`target-cluster-agent.yaml` referenced above have been removed; only the `Agent` CRs remain. kubeui's agent dropdown (`ui/backend/handlers/ai.go` `HandleListAgents`) still queries the (now-nonexistent) CRD, gets an error, and falls back to listing just `SYMPOZIUM_DEFAULT_AGENT` — functional but no longer auto-discovers other agents. `install-sympozium.sh` pinned `--version 0.10.38` at the time; that pin has since moved to 0.10.47 and then 0.10.57 (see the dated entry below) as each was verified against this file. Chart upgrades should be re-verified against this file before trusting it blind — do not assume `latest` is safe just because a past jump was.
- **Upgraded 0.10.47 -> 0.10.57 (2026-09-09, latest published at the time).** CRDs and
  `Agent`/`AgentRun.spec.skills`/`policyRef` schemas are unchanged (purely additive:
  `agentruntimes`, `agentrunturns`, `cellntools(submissions)`, `harnesssessions`,
  `workspacesessions` CRDs; `celln` stays `enabled: false` by default). Three things broke
  on the way, all now handled by `install-sympozium.sh`'s comment block and
  `06-sympozium/values.yaml`:
  1. A NEW chart-templated `sympozium-agent` ServiceAccount collided with the one the
     controller had already created out-of-band (no Helm ownership annotations) — `helm
     upgrade` refused to adopt it until labeled/annotated by hand. Confirmed harmless to
     adopt in place (no deletion, no UID change, no interruption to the 3 pods actively
     using it) and confirmed **all subjects across the chart's RoleBindings, and this
     repo's own additive `agent-kubevirt-rbac.yaml`/`agent-istio-rbac.yaml`, still target
     the literal name `sympozium-agent`** — the rbac.yaml comment about per-run
     `sympozium-run-*` accounts describes a template/annotation-copy pattern (for external
     cloud workload identity federation) that is not wired into normal AgentRun pod
     creation; verified via the actual `spec.serviceAccountName` on live pods before and
     after, and via the deterministic Istio/KubeVirt RBAC checks below.
     **SUPERSEDED by the 0.10.75 entry below: this stopped being true somewhere in
     0.10.58-0.10.75. Don't trust "all skill sidecars run as `sympozium-agent`" without
     re-checking `spec.serviceAccountName` on a live run pod.**
  2. `helm upgrade --wait` polls every `MCPServer`'s custom status, including `postgres`
     (an intentionally `Suspended`, never-configured example) — `--wait` times out on it
     regardless of chart version. Not a regression; drop `--wait` and verify manually.
  3. **0.10.57 defaults `nats.auth.enabled: true`** and wires `NATS_USERNAME`/`PASSWORD`
     into the controller and llmfit-daemonset — but the built-in `web-endpoint` SkillPack's
     sidecar env (`EVENT_BUS_URL` only) was never updated to match. Every served agent's
     web-proxy dialed NATS unauthenticated, was rejected (`nats` pod log: `authentication
     error`), and crash-looped with **exit 2, zero log output** — the same failure
     signature as the `:latest` tag-skew landmine and the old `readOnlyRootFilesystem` bug,
     a third distinct root cause behind the identical symptom. Fixed by setting
     `nats.auth.enabled: false` in `06-sympozium/values.yaml` (NATS is ClusterIP-only,
     never exposed via MetalLB — same demo-simplicity trade-off as the fixed CAs
     elsewhere). Re-evaluate if a future chart version wires the SkillPack correctly.
  As a byproduct, this also silently fixed `target-cluster-agent`'s **separate,
  unrelated, 11-day-old** crash loop (created 2026-08-29 under 0.10.47, before
  `nats.auth` existed at all — so a different cause, most likely the same
  `readOnlyRootFilesystem` class of bug already fixed upstream for the other two agents
  but never picked up because that pod was never recreated) — `fix-web-proxy-image.sh`'s
  mandatory post-upgrade re-pin regenerates all three serving AgentRuns, which recreated
  it too. Verified end-to-end: real `POST /v1/chat/completions` -> real model response
  through `cluster2-agent`, plus `kubectl get gateways.gateway.networking.k8s.io -A
  --as=system:serviceaccount:sympozium-system:sympozium-agent` and the KubeVirt
  equivalent both still resolve.

- **Upgraded 0.10.57 -> 0.10.75 (2026-09-16, latest published at the time).** `Agent`/
  `AgentRun.spec.skills`/`policyRef` schemas unchanged again. CRD/RBAC diff is purely
  additive (`cellnexecutionpolicies`, `cellnruntimeprofiles`, `clustercellntools`,
  `modelconnections` CRDs for the new Celln-fleet + model-gateway features, both
  `enabled: false` by default and untouched by `06-sympozium/values.yaml`) plus a real
  upstream bug fix (generated NATS `control-password`/`bridge-password` could start with
  a digit, which `nats.conf` parses as a number/duration and refuses — passwords are now
  letter-led). `helm upgrade --server-side=true --force-conflicts` (no `--wait`, per the
  0.10.57 entry above) went through clean with **zero SA-adoption conflict** — that part
  of the 0.10.57 upgrade doesn't recur once the SA carries Helm's ownership annotations.
  One real breaking change, found only by testing (not visible in the template/CRD diff —
  it's a controller runtime behavior change): **AgentRun task pods now run as a per-run
  ServiceAccount** (`sympozium-run-<agentrun-name>`, controller-created, owned by the
  `AgentRun`, deleted with it) **bound to a matching per-run `Role`+`RoleBinding` pair**
  (`sympozium-skill-<skillpack>-<agentrun-name>`, namespaced `Role` — not a `ClusterRole`,
  so aggregation-by-label isn't an option) — not the shared `sympozium-agent` SA every
  skill sidecar used to run as. Confirmed via `kubectl get sa
  sympozium-run-<real-run-name> -n sympozium-system -o yaml` (fresh SA,
  `ownerReferences` -> the `AgentRun`) and cross-checked against every `RoleBinding` in
  the namespace: runs created before this upgrade still show `sympozium-agent` as the
  subject, every run created after show `sympozium-run-<name>`. This silently broke
  `agent-istio-rbac.yaml` and `agent-kubevirt-rbac.yaml` (both bound only the literal
  `sympozium-agent` subject): `mesh-sre-agent`'s live tool calls degraded to `qwen2.5:7b`
  confabulating a "permission issue" writeup, while the old verification command —
  `kubectl get gateways.gateway.networking.k8s.io -A
  --as=system:serviceaccount:sympozium-system:sympozium-agent` — kept reporting success,
  because it was checking an identity the running pods no longer used. Caught only by
  driving a real `/v1/chat/completions` call end-to-end and getting a wrong answer, per
  the "verify deterministically, never trust the model" rule in the `mesh-sre-agent`
  section above — the deterministic-looking check was itself checking the wrong subject.
  Fixed by rebinding both `ClusterRoleBinding`s to the `system:serviceaccounts:
  sympozium-system` Group instead of the one SA name, so every current and future
  per-run SA inherits the grant with no per-run maintenance. Verified with
  `--as=system:serviceaccount:sympozium-system:sympozium-run-<real-run-name>` (the actual
  identity a live pod carries — get this from a live pod's `spec.serviceAccountName`, not
  assumed) and a live `mesh-sre-agent` chat completion whose reported gateway
  (`istio-eastwest`, `172.18.255.221`, `Programmed: True`) matched real cluster state.
  Also cleared two pre-existing, upgrade-unrelated `CrashLoopBackOff`s on
  `mesh-sre-agent-web-endpoint-server` and `target-cluster-agent-web-endpoint-server`
  (600+ restarts each, exit 2, zero log output — present before this upgrade started, same
  symptom class as the `:latest` tag-skew and `nats.auth` landmines but not diagnosed
  further since the mandatory post-upgrade `fix-web-proxy-image.sh` re-pin regenerates the
  serving `AgentRun`s regardless, same as it did for the 0.10.47 -> 0.10.57 jump).
  **Re-run the per-run-SA check after any future chart bump** — this identity model is new
  enough in this chart's history that it could change shape again, and the failure mode
  (a stale `--as=sympozium-agent` check reporting healthy while real traffic breaks) won't
  announce itself.

- **`helm install sympozium` silently drops some of the chart's built-in `SkillPack` resources** (kind: SkillPack, labeled `sympozium.ai/builtin: "true"` — observed: `web-endpoint`, `k8s-ops`, `code-review`, `incident-response`, `llmfit`, `memory`, `sre-observability`, `subagents`) even on a clean install with `--wait` reporting success and `helm get manifest` showing them as part of the release. Other kinds in the same chart (`SympoziumPolicy`, etc.) were not observed to be affected. Symptom: an `Agent` with `skills: [{skillPackRef: web-endpoint}]` gets an `AgentRun` stuck `Failed` with `"no sidecar with requiresServer=true found"` — no `<name>-web-endpoint-server` Deployment ever appears, so `sympozium-lb-setup.sh` has nothing to expose and the AI tab has no serving endpoint to call. Fixed by `06-sympozium/fix-missing-builtin-skillpacks.sh` (auto-run by `install-sympozium.sh` as step `2a/5`): diffs `helm get manifest`'s builtin-labeled SkillPacks against what's actually in-cluster and re-applies whatever is missing; idempotent. After it runs, delete any AgentRuns stuck `Failed` from before the SkillPack existed (`kubectl delete agentrun <name>-web-endpoint -n sympozium-system`) so the controller recreates them — it does not retry a terminal `Failed` run on its own.

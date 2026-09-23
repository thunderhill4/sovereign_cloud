# Appendices — The Owner's Reference

> *Every check asserts on a value read back from the cluster or an HTTP response code. No check passes on the absence of an error, and no screenshot counts as evidence.*
>
> — `07-istio-advanced/verify.sh`

Eight appendices, for readers who want to check the book rather than take its word. Each was checked against the repository and the running platform on 17 September 2026 (Appendix G, on 18 September). Where the repository's own documentation disagrees with what was running, both are shown.

- **A. The repo, stage by stage** — what each directory does, and the `make` targets that matter.
- **B. Addresses that must not move** — the network address plan, as reserved and as found.
- **C. The failure-mode catalog** — every documented failure, grouped by what you would see.
- **D. The measurement protocol** — the figures as last measured, and the rules for measuring them again.
- **E. The ten labs** — the agent-platform labs, with their original honesty markers and what this book re-checked.
- **F. A glossary for the boardroom** — one plain-English sentence per term.
- **G. Key files, quoted** — the load-bearing code and manifests, verbatim, for readers without the companion repository.
- **H. Sources** — what the book draws on, and which of it a reader can check.

## Appendix A — The repo, stage by stage

*Build log · 17 September 2026*

The numbered directories run in order. Four prerequisites are **not** created by any script, and are easy to forget when reproducing the platform: the two Kind clusters themselves (`cluster1` and `cluster2`); KubeVirt (v1.9.0) and CDI (v1.66.0) on `cluster2`; and the `ubuntu-noble-dv` base disk image the golden-image bake starts from.

### 00-prereqs

`install-clusterctl.sh` installs `clusterctl`, the Cluster API command-line tool. *Target:* `make prereqs`.

### 01-metallb

Installs MetalLB, which gives Services on the host's Docker network real IP addresses, and configures the address pool for `cluster2` (now `.211–.225`, widened by `07-istio-advanced`). **Do not run it against `cluster1`**: it tries to substitute a placeholder that does not exist in its configuration file, so it would advertise `cluster2`'s range on the shared network (Appendix B). *Target:* `make metallb`.

### 02-capi-init

`init-management-cluster.sh` turns `cluster2` into a cluster factory. It installs Cluster API v1.14.2, the KubeVirt infrastructure provider (CAPK) v0.11.2, and the k3s bootstrap and control-plane providers v0.3.0. It then replaces a dead proxy image those providers ship (Chapter 19), and runs `scripts/configure-kubevirt-perf.sh`, which applies the CPU setting that cut eight seconds from every VM start (Chapter 8). That auto-run can silently do nothing — not, as first diagnosed, because it races KubeVirt's start-up, but because the script's own readiness waits can outlast whatever time limit its caller imposes, and it is killed before reaching that last step (Chapter 19). Check afterwards that the setting landed. *Targets:* `make capi-init`, `make kubevirt-perf`.

### 03-target-cluster

The cluster declarations. `target-cluster-warm.yaml` is the default: seven objects in 258 lines, parallel worker boot, fixed demonstration certificate authorities in `warm-ca/` (Chapter 4). `target-cluster-lite.yaml`, `target-cluster.yaml` and `target-cluster-parallel.yaml` are older, slower paths on the `:latest` image. `generate-cluster.sh` and the `.tmpl.yaml` files render variants. The golden-image tooling sits at the top level (`bake-common.sh`, `bake-golden-image*.sh`, `build-containerdisk.sh`) and in `scripts/` (`seed-cluster-secrets.sh`, `gen-warm-ca.sh`, `fix-registry-hosts.sh`). *Targets:* `make target-cluster` (which runs `registry-fix`, then `ensure-warm-image`, then seeds the secrets and applies the warm manifest); `make clean`.

*Caution:* `ensure-warm-image` checks that the image *tag* exists, not what is inside it. After the September upgrade, the `:latest` and `:preinit` images still carried the old k3s binary (Chapter 19).

### 04-verify

`verify-cluster.sh` waits for the target cluster's virtual machines and checks its health. It also refreshes the agents' copy of the target cluster's credentials. *Target:* `make verify`.

### 05-istio

The "before" picture, kept on purpose (Chapter 10). `install-istio-ambient.sh` installs Istio 1.28 on the target cluster — outside Istio's tested range on Kubernetes 1.37. `cross-cluster-demo.sh` is the eight-step hand-plumbed demo, and `cross-cluster-mindmap.html` is its diagram. *Target:* `make istio`.

### 06-sympozium

The agent platform. `install-sympozium.sh` pins chart 0.10.75. The three serving agents are `cluster2-agent.yaml`, `target-cluster-agent.yaml` and `mesh-sre-agent.yaml`. Their extra permissions are `agent-kubevirt-rbac.yaml` and `agent-istio-rbac.yaml`, now bound to a group rather than one account (Chapter 15). Their network rule is `agent-egress-ollama.yaml`. The `fix-*.sh` scripts each answer a pitfall in Appendix C; `labs/` holds the ten labs (Appendix E); `demo-mesh-sre.sh` is the recorded agent demo. `cost-analyzer.yaml` and `incident-responder.yaml` cannot be applied on the current chart (Chapter 16). *Targets:* `make sympozium-install`, `make sympozium-lb`.

### 07-istio-advanced

The "after" picture: five acts on Istio 1.31 across `cluster1` and `cluster2` — Gateway API, waypoints, multicluster, the AI gateway, and observability — plus `verify.sh` (22 checks) and `clean.sh` (Chapters 11 and 12). *Targets:* `make istio-adv-prereqs`, `istio-adv-install`, `istio-adv-act1` to `act5`, `istio-adv-demo`, `istio-adv-verify`, `istio-adv-clean`.

### scripts/

The measurement instruments `phase-timings.sh` and `time-to-ready.sh` (Appendix D), along with `configure-kubevirt-perf.sh`, `fix-registry-hosts.sh`, `seed-cluster-secrets.sh`, `gen-warm-ca.sh`, `render-cluster.sh`, `render-demo-video.sh` and `import-base-images.sh`.

### ui/

The web console: a React frontend and a Go backend (port 8080), with SRE, AI and Visual modes (Chapters 12 and 13). `run-ui.sh` starts both for development. The production manifest in `ui/k8s/` was not deployed on 17 September: the `kubeui` namespace did not exist. *Targets:* `make ui`, `make ui-build`.

### docs/, and the top level

`docs/sub-60s-cluster-strategy.md` records the cluster-speed work and its dead ends (Chapters 7 and 8). `docs/warm-pool-strategy.md` covers the standby (Chapter 9), and `docs/demo/` holds the recorded agent demo. At the top level: `CLAUDE.md`, the platform's operating memory and pitfall list; `Sovereign_Cloud_Agentic_Strategy.md`, the agent strategy (Chapters 3, 14 and 16); `demo.sh` and `show-cluster.sh`, the original demo, whose narration has drifted (Chapter 5); and `target-cluster-kubeconfig`, the target cluster's credentials file.

### The make targets that matter

The Makefile defines 65 targets. These are the ones this book used.

| Target | What it does | Note, 17 September 2026 |
|---|---|---|
| `make all` | `prereqs` → `metallb` → `capi-init` → `target-cluster` | Assumes the four unscripted prerequisites above |
| `make target-cluster` | Builds the default warm cluster | 33.7 s to both nodes ready (Appendix D) |
| `make clean` | Deletes the target cluster | 45.7 s until every object is gone; the machines stop at 35.6 s (Chapter 17) |
| `make verify` | Checks the target cluster and refreshes agent credentials | |
| `make kubevirt-perf` | Applies the KubeVirt CPU setting | Cluster state, not manifest state: re-run after rebuilding `cluster2` |
| `make phase-timings` | Per-phase breakdown of a cluster build | Counts nodes, not names — see Appendix D |
| `make registry-fix` | Maps the registry alias for containerd | See Appendix C.5 |
| `make istio` | Istio 1.28 on the target cluster | The legacy path; unsupported on Kubernetes 1.37 |
| `make sympozium-install` / `sympozium-lb` | Installs the agent platform; gives its Services addresses | Re-run `sympozium-lb` after any serving Deployment is regenerated |
| `make sympozium-warm` | One-shot model warm-up | Its help text still names llama3.2 |
| `make sympozium-demo-agents` | Applies the April cost and incident personas | Fails on the current chart (Chapter 16) |
| `make ui` | Starts the web console | |
| `make istio-adv-demo` / `istio-adv-verify` | Runs the five acts / the 22 checks | 22 passed on 17 September; the Act 3 checks test configuration, not failover (Chapter 11) |

`make help` lists most targets, not all. It omits `kubevirt-perf`, `phase-timings` and every `istio-adv-*` target, and it still describes the default cluster as a *"~40s target."*

## Appendix B — Addresses that must not move

*Build log · 17 September 2026*

Both Kind clusters live on one Docker network, `172.18.0.0/16`, and both run MetalLB announcing addresses on that same network. If their pools overlapped, two clusters would answer for one address, and traffic would go to whichever answered first. So each cluster has its own pool, and every Service that needs a stable address pins one inside its own pool.

**Pools:** `cluster1` has `172.18.255.200–.210`, installed by `07-istio-advanced/00-prereqs/`. `cluster2` has `172.18.255.211–.225`, originally `.211–.220`, widened by `07-istio-advanced`.

| Address | Reserved for | Cluster | Allocated, 17 Sept | Files naming it |
|---|---|---|---|---|
| `.200` | `httpbin-lb` — legacy `05-istio` demo | cluster1 | No | 9 |
| `.201` | Act 1 edge gateway | cluster1 | Yes | 4 |
| `.202` | Act 3 east-west gateway | cluster1 | Yes | 2 |
| `.203` | Act 4 AI gateway | cluster1 | Yes | 3 |
| `.204` | Kiali | cluster1 | Yes | 3 |
| `.211` | Web console frontend | cluster2 | No — not deployed | 10 |
| `.212` | Sympozium API and dashboard | cluster2 | Yes | 4 |
| `.213` | `cluster2-agent` | cluster2 | Yes | 4 |
| `.214` | `target-cluster-agent` | cluster2 | Yes | 4 |
| `.215` | Target cluster API server | cluster2 | Yes | 18 |
| `.216` | Target-cluster nginx proxy — legacy `05-istio`, only while it runs | cluster2 | No | 5 |
| `.217` | Security agent | cluster2 | No | 4 |
| `.218` | `cost-analyzer` | cluster2 | No | 3 |
| `.219` | `incident-responder` | cluster2 | No | 3 |
| `.220` | Host Ollama load balancer (optional) | cluster2 | No | 4 |
| `.221` | Act 3 east-west gateway | cluster2 | Yes | 3 |
| `.222` | `mesh-sre-agent` | cluster2 | Yes | 3 |

"Files naming it" counts shell, YAML, Go, TypeScript, env and Makefile files outside `book/`. Seven of the seventeen reserved addresses were not in use on 17 September, and every one of them is still written into between three and ten files.

The node addresses on the same network matter too. `172.18.0.1` is the host, where the models run. `.2`, `.3` and `.4` were `cluster1-control-plane`, `cluster2-control-plane` and the image registry — in that order *on that day*. Docker hands those addresses out in the order containers join the network, which is why `172.18.0.2:5000`, the registry address baked into image references, is an alias that only containerd resolves (Appendix C.5).

### Pool discipline

- **Never overlap pools.** One cluster's installer must never advertise another's range.
- **Pin, don't request.** Every Service that anything else depends on names its address in an annotation. Both MetalLB annotation spellings appear in the repository, `metallb.universe.tf/loadBalancerIPs` and `metallb.io/loadBalancerIPs`; both worked on 17 September.
- **Change all three places.** The project's rule: never reassign an address without updating its source file, `sympozium-lb-setup.sh` and `run-ui.sh`.
- **A regenerated Service forgets its address.** When the agent platform regenerates its serving Deployments, their Services come back without MetalLB addresses; re-run `make sympozium-lb`.
- **An idle reservation is still a reservation.** `.216` stays free for the legacy demo; when `cluster2` needed another address, the pool was widened instead (Chapter 10).
- **Retire addresses on purpose.** A reservation nobody uses, named in ten files, is a future collision waiting for someone who reads only the table.

## Appendix C — The failure-mode catalog

*Build log · February–September 2026*

The platform's operating notes list 23 pitfalls, and the chapters of this book found more. They are grouped here by **what you would see**, because that is what an engineer has at 2 a.m. The most expensive group is not the crashes. It is the failures that look like success.

### C.1 A crash loop with exit code 2 and no log output

One sidecar — the agent platform's `web-proxy` — produced this identical symptom from four different causes.

| Cause | When | Fix or status |
|---|---|---|
| The image could not run with a read-only root filesystem | Chart 0.10.38 | Fixed upstream in 0.10.47; `fix-web-proxy-rootfs.sh` retired |
| The sidecar image was pinned to a moving `:latest` tag and drifted to an incompatible version (observed at about 5,680 restarts) | 0.10.47 era | `fix-web-proxy-image.sh` pins it to the chart version and purges the cached image |
| The chart turned on NATS authentication, but the serving sidecar was never given credentials | 0.10.57 | `nats.auth.enabled: false` in `06-sympozium/values.yaml` |
| Two long-running crash loops (600+ restarts), cause not diagnosed | Before the 0.10.75 upgrade | Cleared when the post-upgrade re-pin regenerated the Deployments |

### C.2 The script reported success; nothing happened

| Cause | Where | Fix or status |
|---|---|---|
| A provider shipped a proxy image that no longer exists; readiness waits timed out into an <code>&#124;&#124; true</code> fallback, so setup reported success | `make capi-init` (Chapter 19) | Image patched automatically in `init-management-cluster.sh` |
| The KubeVirt CPU setting's applying step never ran because the setup script itself was killed by an external time limit mid-readiness-wait (corrected 18 September; first misdiagnosed as a race with KubeVirt's own start-up) | `configure-kubevirt-perf.sh` auto-run (Chapter 19) | Check `supportContainerResources` after any `capi-init`; re-run `make kubevirt-perf` |
| A documented `K3S_VERSION` override never reached the image being baked | `bake-common.sh` (Chapter 19) | Both version literals updated and commented |
| Image checks tested whether a tag existed, not what it contained | `ensure-warm-image`, `pre-pull` | `:warm` rebaked; `:latest` and `:preinit` still stale |
| Built-in SkillPacks "silently dropped" at install — really the chart applying them before its own admission webhook was serving | `helm install` | `fix-missing-builtin-skillpacks.sh` |
| Agent runs created in quick succession raced into a terminal `Failed`, which the controller never retries | Serving re-pin | Delete the failed run and let the controller recreate it |
| A failover check counted any non-empty body as success, so a total outage printed `success=20 failed=0` | Act 3 `run.sh` (Chapter 11) | Not fixed; the console's failover action counts HTTP 200 |

### C.3 The status says healthy; it is not

| Cause | Where | Fix or status |
|---|---|---|
| The worker node was `Ready` but tainted as uninitialized, so it accepted no workloads | Target cluster (Chapter 5) | Not fixed |
| A ghost node baked into the golden image satisfied "2 nodes Ready" before the worker joined | Both timing scripts, and (found 18 September) the UI backend's own `targetNodesReady` — used by both the on-demand deploy path and the warm pool (Chapter 7, Chapter 9) | Measure/count by name (Appendix D); fixed in `pool.go` 18 September, standalone scripts still not fixed |
| A missing worker timestamp was treated as zero, so "both nodes" meant the control plane only | `phase-timings.sh` (Chapter 7) | Not fixed |
| The deterministic permission check tested an identity running pods no longer used | Agent RBAC after 0.10.75 (Chapter 15) | Bindings moved to a group; check the live pod's account |
| A topology view labelled `live` drew 14 nodes that did not exist, 13 of them healthy | Visual mode (Chapter 12) | Not fixed |
| A tenant's listener was refused, but the act printed its status instead of asserting it | Act 1 ListenerSet (Chapter 11) | Not fixed |
| "Locality-aware routing prefers in-cluster endpoints" was printed without comparing counts; measured traffic split 30/30 | Act 3 (Chapter 11) | Not fixed |

### C.4 The status says broken; it is not

| Cause | Where | Fix or status |
|---|---|---|
| Cluster API reported `Available=False` (`WaitingForKthreesServer`) for a working cluster | k3s control-plane provider v0.3.0 (Chapter 4) | Not investigated |

### C.5 Unreachable from here, reachable from there

| Cause | Where | Fix or status |
|---|---|---|
| `172.18.0.2:5000` is a registry alias only containerd can resolve; plain HTTP clients get connection refused | Web console Registry tab | Backend dials a real address, displays the alias |
| A host-network probe looked for Ollama on `127.0.0.1`; the redirect that fixed it was lost on reboot | Sympozium node probe | Redirect re-asserted every 60 s by a DaemonSet |
| A VM's pod address is reachable from its node but not from other pods, costing a 10 s timeout | Worker joining the control plane (Chapter 8) | `advertise-address` set on the control plane |
| Remote-cluster credentials embedded `127.0.0.1:<port>`, so the peer sat at `timeout` while all else looked healthy | `istioctl create-remote-secret` on Kind | Pass `--server https://<node-ip>:6443`; Act 3 asserts `synced` |
| `host.docker.internal` does not resolve inside Kind pods | Model endpoint | Use `172.18.0.1` through an in-cluster shim Service |
| Both Kind clusters number pods from `10.244.0.0/24`, so a pod address means different machines in each | Cross-cluster calls (Chapter 10) | Hand-plumbed proxy (before); east-west gateways (after) |

### C.6 The model gave a confident, wrong answer

| Cause | Where | Fix or status |
|---|---|---|
| Stale target-cluster credentials made the tool fail TLS quietly; llama3.2 invented plausible output | `target-cluster-agent` (Chapter 15) | `refresh-target-kubeconfig.sh`, auto-run by verify and deploy paths |
| Chat runs got no tool sidecar, so the model gave up or improvised | Sympozium 0.10.38 | Skills declared on the `Agent`; the webhook workaround is now redundant |
| A refusal rule in the system prompt was not reliably obeyed | `mesh-sre-agent`, qwen2.5:7b (Chapter 14) | Enforce below the model, not in the prompt |
| A tool denied by policy was used anyway | Lab 04 (Chapter 14) | Not fixed on 0.10.75 |
| After the per-run identity change, the agent reported a plausible "permission issue" | `mesh-sre-agent` after 0.10.75 | Group binding (C.3) |

### C.7 503 on every request, with every component healthy

| Cause | Where | Fix or status |
|---|---|---|
| An in-mesh route pinned traffic to non-global backends, overriding cross-cluster routing | Act 2's `echo-internal-split` during Act 3 | Act 3 deletes the route; still reproduces on Istio 1.31 (Chapter 11) |
| A waypoint resolved only local endpoints | Istio 1.30.3 | Did not reproduce on 1.31; the console's opt-out label is now harmless (Chapter 11) |

### C.8 It waits forever, though the thing is done

| Cause | Where | Fix or status |
|---|---|---|
| `kubectl wait --for=condition=Accepted` reads top-level conditions; a route's live under `status.parents` | HTTPRoute | `wait_route_accepted` in `07-istio-advanced/lib/common.sh` |
| `helm upgrade --wait` polls an intentionally suspended example MCPServer | Sympozium upgrades | Drop `--wait`; verify by hand |
| Gateway API CRDs exceed the annotation size limit of client-side apply | Gateway API install | Apply with `--server-side` |

### C.9 The screen is empty, or shows the wrong thing

| Cause | Where | Fix or status |
|---|---|---|
| The dashboard's namespace picker defaults to `default`, where nothing lives | Sympozium dashboard | Switch to `sympozium-system` (stored per browser) |
| The GPU detector shells out to `nvidia-smi`, which is not in its container, so it reports the integrated GPU | Model-fit hardware view | `fix-llmfit-nvidia-smi.sh` replay shim |
| The agent drop-down queries a resource type the chart no longer ships | Web console AI mode | Falls back to the default agent only; not fixed |
| Model calls through the AI gateway are absent from the mesh request metric and from Kiali | Observability (Chapter 12) | Not fixed |

### C.10 It changed its name or its shape

| What changed | Where | Consequence |
|---|---|---|
| `SympoziumInstance` stopped being reconciled, then stopped shipping | Sympozium 0.10.38 → 0.10.47 | Agents must be `Agent` resources |
| Skill sidecars moved from one shared account to one account per run | Sympozium 0.10.75 | Permissions bound to a single account silently stopped applying |
| `XListenerSet` graduated to `ListenerSet` | Gateway API v1.5 | Most write-ups use the old name, which fails |
| `istioctl verify-install` was removed | Istio 1.30 | Use `istioctl analyze` |
| `--server-side` started requiring a value | Helm 4 | A bare `--server-side --force-conflicts` fails |
| The three GatewayClasses name their generated Services differently; the Prometheus container is `prometheus-server` | Istio add-ons | Look Services up by label; name the container explicitly |
| VM networking moved from masquerade to bridge, in the same commit as the diagram describing masquerade | March 2026 (Chapter 10) | The mind map and README diagram still describe the old network |

## Appendix D — The measurement protocol

*Build log · June–September 2026*

### D.1 The figures, as last measured

| Measurement | Value | When, and how |
|---|---|---|
| Cluster bring-up, both nodes ready | **33.7 s** median (30.4 / 35.1 / 33.7) | 17 September, by node name, warm manifest, Kubernetes 1.37 |
| Control plane alone | **24.7 s** median | Same runs |
| Teardown | Delete command returns at **10.13 s**; machines stop at **35.6 s**; last object gone at **45.73 s** | 19 September, N=5 (spread 1.24 s). Supersedes an 18.9 s figure from a single run on 17 September |
| Standby claim | **467 ms** median (1.059 / 0.467 / 0.465) | 18 September, by name-based readiness check, warm-image standby; was 194 ms in June |
| Model load, cold vs warm | 2.3–2.6 s cold; 0.14–0.17 s warm | 17 September, GPU, qwen2.5:7b and llama3.2 (Chapter 9) |
| Local model energy | 1.19 J per generated token; 0.33 kWh per million tokens (GPU only) | 17 September, qwen2.5:7b (Chapter 20) |
| Cross-cluster failover | 30 of 30 HTTP 200, all from `cluster2` | 17 September, Istio 1.31 (Chapter 11) |

And the history behind the headline number:

| Stage | Median | Runs | Spread |
|---|---|---|---|
| Baseline (June) | 50.0 s | 52.7 / 50.0 / 46.6 | 6.1 s |
| + `advertise-address` | 42.7 s | 42.7 / 42.8 / 41.2 | 1.6 s |
| + `supportContainerResources` | 34.5 s | 34.5 / 34.2 / 34.7 | 0.5 s |
| After the Kubernetes 1.37 upgrade | 33.7 s | 30.4 / 35.1 / 33.7 | 4.7 s |

Notice the last column. The spread that had collapsed to half a second is back to 4.7 seconds after the rebuild. With three runs, a change smaller than about five seconds cannot be seen.

### D.2 The instrument

`scripts/phase-timings.sh` attributes a build's wall clock to its phases, read-only and without logging in to the guests. Its sources, in its own words:

- Cluster API and CAPK object creation timestamps — the controller chain.
- VM instance phase transitions — scheduling to running.
- The launcher container's start time — disk unpack and hypervisor start.
- The guest serial console (`kubectl logs -c guest-console-log`) — kernel, initramfs, systemd, cloud-init, and k3s start.
- Node `Ready` transitions — the metric itself.
- A host-side probe loop — the API server's first TCP accept, and its first healthy answer.

Guest console timestamps count seconds since boot, not time of day. The script anchors them to the wall clock using cloud-init's own log lines, which carry both: *"running 'init' at Wed, 09 Sep 2026 06:44:19 +0000. Up 6.81 seconds."* Subtract the uptime from the date, and every console line lands on the same clock as the Kubernetes objects.

`./scripts/phase-timings.sh --collect-only` rebuilds the table for whatever is running now, and changes nothing. `--runs N` deploys and measures `N` clean runs.

### D.3 The rules

1. **Instrument before you optimise.** Two dead ends each cost a full image bake to disprove before the phases were measured (Chapter 8).
2. **Name what you count.** Wait for the control-plane node and the worker node *by name*, never for "two nodes". A stale node object inside the golden image satisfied the count (Chapter 7).
3. **Missing is not zero.** An absent timestamp must fail the measurement, not add nothing to it (Chapter 7).
4. **Publish every run and the spread, not just the median.** With a six-second spread, three runs cannot resolve a lever smaller than about five seconds. Use seven or more runs, a quiet host, or both.
5. **When a number improves, check the instrument first.** The 23.9-second figure was the control plane alone (Chapter 7).
6. **Keep the raw data.** The raw files behind the 9 September phase table no longer exist to re-check.
7. **Judge a request by its status code, never its body.** An error page is a body (Chapter 11).
8. **Watch every check fail once, on purpose,** before it goes on a dashboard (Chapter 11).
9. **Measure use against reservation.** On owned hardware, reservations set capacity (Chapter 20).
10. **For models, separate cold from warm, and meter energy against tokens.** Read the model server's own `load_duration`, and sample GPU power every 200 ms alongside its `eval_count` (Chapters 9 and 20).
11. **Record what you changed, and put it back.** Every live experiment in this book ran with a restore step, and was checked afterwards.

### D.4 Measuring bring-up by name

The repository's two original timing scripts, `time-to-ready.sh` and `phase-timings.sh`, still wait for a node *count*, which a ghost node can satisfy before the worker joins. The procedure below is now committed as **`scripts/time-to-ready-by-name.sh`** (`make time-to-ready-by-name`), which waits by name, ignores and *names* any node matching neither pattern, and fails loudly on a missing timestamp rather than reporting it as zero. The 33.7-second figure was measured this way, three times:

1. Delete the cluster, and wait until no virtual machine or VM instance named `target-cluster` remains.
2. Seed the fixed certificate and token secrets (`scripts/seed-cluster-secrets.sh`).
3. Start the clock, and apply `03-target-cluster/target-cluster-warm.yaml`.
4. Every half-second, fetch the target kubeconfig if you don't have it yet, and list nodes with their `Ready` condition.
5. Record, separately, the first moment a node whose name contains `-cp-` is `Ready`, and the first moment a node whose name contains `-workers-` is `Ready`.
6. Cross-check both against each node's `Ready` `lastTransitionTime` from the API. *(The committed script does not do this step — it was performed by hand for the 17 September figures. Either do it by hand, or treat the polled values as the measurement, which is what Chapter 17's drill does.)*
7. Afterwards, refresh the agents' copy of the target credentials (`06-sympozium/refresh-target-kubeconfig.sh`).

Run it directly:

```
make time-to-ready-by-name RUNS=3
```

(The script previously required the manifest positionally before `--runs`, so that invocation failed with `manifest not found: --runs`. Fixed 19 September; either argument order now works.)

### D.5 The recovery drill

Chapter 17's cycle — destroy, rebuild, deploy, serve — measured on 19 September 2026, N=5 after one discarded rehearsal, with the boundary table committed to `measurements/README.md` **before the first measured run**. Raw data: `measurements/ch17-recovery-drill-20260919-163611.tsv`; script: `measurements/ch-recovery-drill.sh`.

| Figure | Median | Spread |
|---|---:|---:|
| Destroy to first served request | **81.96 s** | 3.63 s |
| Delete command returns | 10.13 s | 0.98 s |
| Teardown, last object gone | 45.73 s | 1.24 s |
| Build, both nodes Ready | 34.39 s | 4.24 s |
| Build, control plane alone | 25.60 s | 0.78 s |
| Workload to first HTTP 200 | 1.34 s | 0.58 s |

Pre-flight gates abort rather than adjust: the `:warm` image must already be cached (recording its image ID, not just its tag), `supportContainerResources` must be present at `cpu: "1"`, and no warm-pool controller may be running. The workload is `rancher/mirrored-library-busybox`, baked into the golden image — every run recorded zero `Pulling` events, so nothing was fetched from the internet inside the clock.

### D.6 The parallel-metering run

Chapter 20's cost comparison was measured on 18 September 2026 with `measurements/ch19-meter.sh`: an `nginx` deployment on the target cluster, sampled every 20 seconds for ten minutes, recording the host container's CPU and memory, the GPU's power draw, the target control plane's CPU and memory, and one HTTP probe of the workload per sample. All 30 samples returned `200`.

The raw samples are kept at `measurements/ch19-parallel-metering-20260918.tsv`, and `measurements/README.md` holds the full assumptions table and the dated price sources. Two things could not be measured on this host and are therefore assumptions, not figures: whole-host power (`/sys/class/powercap/intel-rapl:0/energy_uj` is root-only on this kernel, and the battery reads zero on AC) and the hardware's purchase price.

## Appendix E — The ten labs

*Build log · July–September 2026*

The labs in `06-sympozium/labs/` were written in July against Sympozium 0.10.38. Each one is standalone, cleans up after itself, and shows real captured output. Their index marks every lab ✅ or ⚠️, and explains every ⚠️ in the lab's own README. Those markers are preserved below. The last column is what this book re-checked on 0.10.75 and Kubernetes 1.37.

| # | Lab | What it shows | Marker (July, 0.10.38) | Re-checked in this book |
|---|---|---|---|---|
| 01 | `chat-serving` | An agent as an OpenAI-compatible chat endpoint | ✅ works | Serving endpoints answered on 17 September (`verify.sh`); live completions through all three agents on 16 September (Chapter 19) |
| 02 | `agentrun` | A one-shot task as a Kubernetes object | ✅ works — needs explicit `agentId`, `model`, `sessionKey`, `skills` | Re-run; required fields confirmed by a server dry run (Chapter 13) |
| 03 | `schedule` | Timed agent work | ⚠️ prompt-only tasks work; skill-using tasks don't | The platform's warm-up schedule was found stopped (Chapters 9 and 14) |
| 04 | `policy` | Tool gating, egress, sandbox | ⚠️ explicit override blocked; default deny not enforced; network policy inert | Override still refused; denied tool still used; network policy *is* now enforced, but the sandbox policy selects no pods (Chapter 14) |
| 05 | `mcpserver` | External tool servers | ✅ discovery works; ⚠️ tool calls flaky with the model | Not re-checked |
| 06 | `ensemble` | Agents handing work to each other | ✅ works — `spec.enabled: true` required | Re-run end to end and cleaned up (Chapter 16) |
| 07 | `model-fit` | Which models fit the hardware | ✅ works | Re-run: 1,365 candidate models scored against the host (Chapter 16) |
| 08 | `capstone` | A real task under policy | ✅ works, partially — an RBAC gap surfaces | Not re-checked |
| 09 | `metrics` | Tokens, tool calls and latency compared across models | ✅ works | Not re-run; figures predate three chart upgrades (Chapter 15) |
| 10 | `sre-triage` | A scored diagnosis of a broken workload | ✅ works — 3/3 root-caused an ImagePullBackOff | Not re-run (Chapter 15) |

Two statements in the labs index are now out of date. It describes `SympoziumInstance` objects in the agent manifests, and that resource type no longer ships with the chart. And its conventions describe *"observed 0.10.38 behavior"*, three upgrades ago.

## Appendix F — A glossary for the boardroom

- **Agent** (Sympozium) — an AI assistant declared as a Kubernetes object: its model, its tools, its policy, and its briefing, all in one reviewable file.
- **AgentRun** — one piece of work given to an agent, recorded as an object with its result, its token usage and its duration.
- **Agentic control plane** — software that lets AI agents observe and operate infrastructure through the same permissions, policies and audit trail as people.
- **Air-gapped** — able to run with no connection to the internet.
- **Ambient mesh** — Istio's service-mesh design with no proxy inside each application; a shared component on each machine handles encryption and identity.
- **CAPK** — the Cluster API provider for KubeVirt: the plug-in that lets the cluster factory build machines as virtual machines.
- **CDI** (Containerized Data Importer) — the KubeVirt companion that imports and prepares virtual-machine disk images.
- **Cluster API (CAPI)** — open-source software that uses one Kubernetes cluster to build and maintain others from written declarations.
- **Container** — a packaged application with everything it needs to run, lighter than a virtual machine because it shares the host's operating system.
- **Control plane** — the part of a Kubernetes cluster that decides what should run where; the cluster's brain.
- **Declaration and reconciliation** — writing down what should exist, and software continuously making reality match it.
- **Depreciation** — spreading the cost of hardware over its useful life; the landlords themselves disagree on whether a server lasts five years or six.
- **East-west gateway** — the door one cluster's mesh uses to reach another's, carrying encrypted, identity-bearing traffic between them.
- **Egress** — traffic leaving a system; also the fee some providers charge for data leaving them.
- **Gateway API** — the Kubernetes standard for declaring how traffic enters and moves between services, with ownership split between platform and application teams.
- **Golden image** — a pre-built machine image with everything a cluster node needs baked in, so nothing is downloaded at boot.
- **Istio** — the open-source service mesh this platform uses.
- **k3s** — a small, single-binary distribution of Kubernetes.
- **Kind** — "Kubernetes in Docker": complete Kubernetes clusters running as containers on one machine; used here as the host's data centre.
- **KubeVirt** — open-source software that runs virtual machines as Kubernetes objects, beside containers.
- **Kubernetes** — the open-source system that runs and manages containers across many machines.
- **Local-LLM-first** — the rule that AI models run on hardware the organization controls, with hosted models used only where explicitly allowed.
- **Management cluster / workload cluster** — the factory, and the clusters it builds.
- **MetalLB** — software that gives Kubernetes Services real network addresses on ordinary networks.
- **mTLS** (mutual TLS) — encryption in which *both* sides prove their identity, not just the server.
- **NetworkPolicy** — a Kubernetes rule stating which workloads may talk to which, enforced by the network layer.
- **Ollama** — the open-source server that runs AI models on the platform's own GPU.
- **Repatriation** — moving workloads from a rented cloud back onto infrastructure the organization owns.
- **ServiceEntry** — an Istio record that tells the mesh about a service outside it; the "before" picture of cross-cluster discovery.
- **SkillPack** — a bundle of tools an agent may use, with the permissions those tools need.
- **Sovereign cloud** — infrastructure whose data, technology and operations stay under the control of the organization and the jurisdiction it chooses.
- **SPIFFE identity** — a standard, cryptographically verifiable name for a workload, such as `spiffe://cluster.local/ns/demo-apps/sa/trusted`.
- **Switching charges** — fees for moving to another provider; banned in the EU from 12 January 2027 under the Data Act.
- **Token** — the unit AI models read and write — roughly a short word — and the unit most AI is metered and billed in.
- **Total cost of ownership (TCO)** — the full cost of an option over its life: purchase, operation, people, and exit.
- **Waypoint** — an optional proxy that adds request-level rules (paths, methods, identities) to an ambient mesh, one per namespace.
- **Warm standby** — a pre-built spare kept running so it can be handed over instantly; fast, but it holds capacity idle.
- **ztunnel** — the ambient mesh's per-machine component: it encrypts traffic and carries identities, but sees connections, not individual requests.

## Appendix G — Key files, quoted

*Read from the companion repository on 18 September 2026, branch `upgrade/k8s-1.37-istio-1.31-sympozium-0.10.75`.*

The chapters' **Open the repo** blocks point at files in the companion repository. Readers who do not have it should not have to take the book's word for what those files say, so the load-bearing ones are reproduced here, verbatim, each captioned with its path.

Two conventions. Where a file is long, an excerpt is shown and the elision is marked `# …`; nothing inside a quoted block is paraphrased or tidied. And where an SSH public key appears in a committed manifest, it is elided as `[public key elided]` — the key is real, and it is in the repository, but it is the author's and not the reader's business.

### G.1 The warm cluster manifest — seven objects

Chapter 5 describes a cluster as a written declaration; this is that declaration. Seven objects, applied together, produce two virtual machines running a Kubernetes cluster. Chapters 7, 8 and 9 all measure this file.

```yaml
# 03-target-cluster/target-cluster-warm.yaml (excerpt — header comment)
# WARM-boot variant of target-cluster.yaml — target time-to-ready <40s.
#
# Combines two ideas:
#   1. WARM IMAGE (BAKE_MODE=warm, image tag :warm). The golden image was baked
#      with a FIXED custom CA set (03-target-cluster/warm-ca/) and a fixed token,
#      with the API VIP in the serving-cert SAN. Its etcd DB, certs, and system
#      charts are fully initialized. First boot does NO `k3s server
#      --cluster-reset` and NO cert purge — it just starts k3s from warm state.
#   2. PARALLEL WORKER (from target-cluster-parallel.yaml). The worker boots
#      alongside the control plane via a static bootstrap secret + skip-preflight,
#      so it is not serialized behind the CP.
#
# ── REQUIRED ORDER ──────────────────────────────────────────────────────────
#   ./scripts/seed-cluster-secrets.sh        # seed ca/cca/etcd/token FIRST
#   kubectl apply -f 03-target-cluster/target-cluster-warm.yaml
#
# DEMO USE ONLY: fixed CA + token are committed (see warm-ca/).
```

The seven objects, in the order they appear in the file:

| # | Kind | Name | What it is |
|---|---|---|---|
| 1 | `Cluster` | `target-cluster` | The cluster itself: pod and service address ranges, and pointers to the two objects below |
| 2 | `KubevirtCluster` | `target-cluster` | The infrastructure side: pins the API server's address to `172.18.255.215` |
| 3 | `KThreesControlPlane` | `target-cluster-control-plane` | The control plane: k3s version, one replica, and the boot commands |
| 4 | `KubevirtMachineTemplate` | `target-cluster-cp` | The control-plane machine: 4 cores, 8 GiB, booting the `:warm` image |
| 5 | `Secret` | `target-cluster-workers-bootstrap` | The worker's static cloud-init, so it need not wait for the control plane |
| 6 | `MachineDeployment` | `target-cluster-workers` | One worker, with CAPI's stability preflight skipped |
| 7 | `KubevirtMachineTemplate` | `target-cluster-workers` | The worker machine: 4 cores, 6 GiB, same image |

```yaml
# 03-target-cluster/target-cluster-warm.yaml (excerpt — object 1 of 7)
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: target-cluster
  namespace: default
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
        - 10.42.0.0/16
    services:
      cidrBlocks:
        - 10.43.0.0/16
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta2
    kind: KThreesControlPlane
    name: target-cluster-control-plane
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: KubevirtCluster
    name: target-cluster
```

```yaml
# 03-target-cluster/target-cluster-warm.yaml (excerpt — object 4 of 7)
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: KubevirtMachineTemplate
metadata:
  name: target-cluster-cp
  namespace: default
spec:
  template:
    spec:
      virtualMachineBootstrapCheck:
        checkStrategy: none
      virtualMachineTemplate:
        metadata:
          namespace: default
        spec:
          runStrategy: Always
          template:
            spec:
              domain:
                cpu:
                  cores: 4
                memory:
                  guest: 8Gi
                ioThreadsPolicy: shared
                devices:
                  disks:
                    - disk:
                        bus: virtio
                      name: systemdisk
                  interfaces:
                    - bridge: {}
                      name: default
              networks:
                - name: default
                  pod: {}
              volumes:
                - containerDisk:
                    image: 172.18.0.2:5000/ubuntu-noble-k3s:warm
                  name: systemdisk
```

And the worker's static bootstrap — the object that lets the worker boot beside the control plane rather than behind it (Chapter 8):

```yaml
# 03-target-cluster/target-cluster-warm.yaml (excerpt — object 5 of 7)
# Static worker bootstrap (identical to target-cluster-parallel.yaml). Embeds the
# fixed WARM token so the agent can join the moment the CP API answers. The k3s
# agent retries the server VIP until it is reachable.
apiVersion: v1
kind: Secret
metadata:
  name: target-cluster-workers-bootstrap
  namespace: default
type: cluster.x-k8s.io/secret
stringData:
  value: |
    #cloud-config

    write_files:
    -   path: /etc/rancher/k3s/config.yaml
        owner: root:root
        permissions: '0640'
        content: |
          kubelet-arg:
          - cloud-provider=external
          server: https://172.18.255.215:6443
          token: f00dcafef00dcafef00dcafef00dcafe

    runcmd:
      - "mkdir -p /home/ubuntu/.ssh && echo '[public key elided]' > /home/ubuntu/.ssh/authorized_keys && chown -R ubuntu:ubuntu /home/ubuntu/.ssh && chmod 700 /home/ubuntu/.ssh && chmod 600 /home/ubuntu/.ssh/authorized_keys"
      # `agent` MUST be positional: the baked /opt/install.sh parses only positional
      # args and ignores INSTALL_K3S_EXEC, so without it the worker starts the k3s
      # SERVER unit, which fails ~0.8s later and — because install.sh runs under
      # `set -e` — aborts the script before writing the bootstrap-success sentinel.
      - INSTALL_K3S_SKIP_DOWNLOAD=true INSTALL_K3S_EXEC='agent' sh /opt/install.sh agent && mkdir -p /run/cluster-api && echo success > /run/cluster-api/bootstrap-success.complete
      - "systemctl daemon-reload && systemctl enable k3s-agent && systemctl start k3s-agent"
```

Note the `kubelet-arg: cloud-provider=external` in that worker configuration. It is the line behind Chapter 5's finding: the worker comes up `Ready` and carries a taint no controller ever clears, so it accepts no workloads.

The annotation that lets the worker start at all, rather than waiting:

```yaml
# 03-target-cluster/target-cluster-warm.yaml (excerpt — object 6 of 7)
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: target-cluster-workers
  namespace: default
  annotations:
    # Skip CAPI's "ControlPlaneIsStable" preflight so the worker MachineSet scales
    # up immediately and the worker VM boots in parallel with the control plane.
    machineset.cluster.x-k8s.io/skip-preflight-checks: All
```

### G.2 `advertise-address` — the first of the two one-line fixes

Chapter 8's headline: two configuration lines cut the build from 50.0 to 34.5 seconds. This is the first, with the comment that explains it, quoted whole because the comment is the evidence.

```yaml
# 03-target-cluster/target-cluster-warm.yaml (excerpt — inside KThreesControlPlane.preK3sCommands)
      # Advertise the API VIP, not this VM's pod IP. KubeVirt bridge binding gives
      # the VM the launcher pod's address, and that address is reachable from the
      # NODE but NOT from other pods (measured: node->VM:6443 = 200 in 0.3ms,
      # pod->VM:6443 = timeout). The worker's k3s agent is itself a pod, so when
      # the supervisor hands it the CP's pod IP it dials an address it can never
      # reach and eats the full 10.000s dial timeout — and the kubelet does not
      # start until that expires. Advertising the VIP, which every VM already
      # reaches, makes the first dial succeed: median time-to-ready 50.0s -> 42.7s,
      # run-to-run spread 6.1s -> 1.6s. Must run after KThrees writes config.yaml.
      - "echo 'advertise-address: 172.18.255.215' >> /etc/rancher/k3s/config.yaml"
```

The same file also carries the ghost node of Chapter 7 — the leftover `Node` object baked into the image, and the asynchronous cleanup whose lateness let a stopwatch stop early:

```yaml
# 03-target-cluster/target-cluster-warm.yaml (excerpt — KThreesControlPlane.postK3sCommands)
    postK3sCommands:
      # Reap the bake VM's leftover Node object (carried in the warm SQLite
      # state) once the API is up. Transient unit so it survives cloud-init.
      - systemd-run --unit=warm-ghost-node-cleanup --no-block /bin/bash -c 'for i in $(seq 1 60); do /usr/local/bin/k3s kubectl delete node ubuntu-bake-vm-warm --ignore-not-found && break; sleep 2; done'
```

### G.3 `supportContainerResources` — the second one-line fix

Chapter 8's second fix, worth 8.2 seconds on every cluster build, and the one that is cluster state rather than repository state — which is why Chapter 19 found it missing after the rebuild.

```bash
# scripts/configure-kubevirt-perf.sh (excerpt)
{"spec":{"configuration":{"supportContainerResources":[
  {"type":"container-disk",
   "resources":{"requests":{"cpu":"100m","memory":"40M"},"limits":{"cpu":"${CPU_LIMIT}","memory":"100M"}}},
  {"type":"guest-console-log",
   "resources":{"requests":{"cpu":"100m","memory":"60M"},"limits":{"cpu":"${CPU_LIMIT}","memory":"100M"}}}
]}}}
```

### G.4 Two ways to count a failover

Chapter 11's sharpest finding is a comparison, so both sides belong here. First, the demonstration script — which counts any non-empty response body as a success, and therefore prints `success=20 failed=0` when both clusters are down:

```bash
# 07-istio-advanced/act3-multicluster/run.sh (excerpt)
call() { $K1 exec -n "$NS_APPS" deploy/client-trusted -c curl -- \
         curl -s --max-time 10 http://echo/ 2>/dev/null || true; }

# …

  b=$(call)
  if [[ -n "$b" ]]; then okc=$((okc+1)); else failc=$((failc+1)); fi
done
detail "success=${okc}  failed=${failc}"
if (( okc >= 18 )); then
  ok "Traffic failed over to cluster2 with zero config change and no app restart"
else
  fail "Failover incomplete: ${failc}/20 requests failed"
fi
```

And the web console, checking the same thing correctly — on the status code:

```go
// ui/backend/handlers/cross_cluster.go (excerpt)
	ok := func(p ProbeResult) bool { return p.StatusCode >= 200 && p.StatusCode < 300 }

	httpbinAlive := ok(probes[0])
	nginxAlive := ok(probes[2])
	// Failover: a local service is down but the cross-cluster path is still up
	failoverActive := (!httpbinAlive && ok(probes[1])) || (!nginxAlive && ok(probes[3]))
```

Two checks of one behaviour, in one repository, written weeks apart. One cannot fail.

### G.5 The egress allowlist that allows more than it names

Chapter 14 grades the eight architecture constraints. This is the file behind the verdict on the egress rule: the last two rules carry ports but no destination, so they permit those ports to *anywhere*.

```yaml
# 06-sympozium/agent-egress-ollama.yaml (quoted whole)
---
# AgentRun pods are labeled app.kubernetes.io/part-of=sympozium, which makes
# them subject to the helm-installed `sympozium-allow-otel` NetworkPolicy
# (policyType: Egress, only ports 4317/4318 allowed). With kindnet enforcing,
# that implicitly blocks egress to Ollama on the docker bridge.
#
# This policy adds the missing egress paths (DNS + Ollama) for sympozium pods.
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: sympozium-allow-ollama
  namespace: sympozium-system
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/part-of: sympozium
  policyTypes:
  - Egress
  egress:
  # DNS
  - to:
    - namespaceSelector: {}
      podSelector:
        matchLabels:
          k8s-app: kube-dns
    ports:
    - port: 53
      protocol: UDP
    - port: 53
      protocol: TCP
  # Ollama via the host-ollama shim Service (Endpoints → 172.18.0.1:11434)
  - ports:
    - port: 11434
      protocol: TCP
  # Kubernetes API (skill-k8s-ops needs it)
  - ports:
    - port: 443
      protocol: TCP
    - port: 6443
      protocol: TCP
```

### G.6 An agent as a Kubernetes object

Chapter 13's central claim is that an agent is an object like any other. This is one, whole — model, endpoint, tools, policy, and the briefing it is given.

```yaml
# 06-sympozium/cluster2-agent.yaml (excerpt — the Agent object)
apiVersion: sympozium.ai/v1alpha1
kind: Agent
metadata:
  name: cluster2-agent
  namespace: sympozium-system
  annotations:
    sympozium.ai/description: "Cluster2 management-plane agent (KubeVirt VMs, CDI, CAPI, cluster nodes)"
spec:
  agents:
    default:
      model: qwen2.5:7b
      baseURL: http://host-ollama.sympozium-system.svc.cluster.local:11434/v1
      sandbox:
        enabled: false
  authRefs:
    - provider: ollama
      secret: llm-credentials
  skills:
    - skillPackRef: web-endpoint   # drives declarative serving (requiresServer=true)
    - skillPackRef: k8s-ops        # kubectl/virtctl sidecar for task (chat) runs
  policyRef: sandbox-restricted
  observability:
    enabled: false
  memory:
    enabled: false
    maxSizeKB: 256
    # Environment briefing — keep SHORT (7B model). Edit here; re-run the
    # installer (or kubectl apply -k 06-sympozium/) to reproduce.
    systemPrompt: |
      You operate "cluster2", the management cluster of a KubeVirt multi-cluster platform. Topology:
      - cluster2 (where your kubectl/virtctl run): a Kind cluster hosting Cluster API (CAPI), KubeVirt + CDI (VMs run as pods), MetalLB, and Sympozium. Your in-cluster tools talk to THIS cluster's API only.
      - target-cluster: a k3s cluster whose nodes are KubeVirt VirtualMachines hosted on cluster2, provisioned by CAPI. Inspect from cluster2 with: kubectl get clusters,machines -A (group cluster.x-k8s.io); kubectl get virtualmachines,virtualmachineinstances -A (group kubevirt.io); kubectl get datavolumes -A (CDI). target-cluster API VIP is 172.18.255.215.
      - cluster1: a SEPARATE Kind cluster (Istio ambient mesh, httpbin/sleep). You have NO kubectl access to it from here.
      Always use full resource names (virtualmachines, virtualmachineinstances, clusters, machines). Read-only unless the user explicitly approves a change.
```

Read that briefing against Chapter 19. It describes `cluster1` as running `httpbin/sleep` — which was true before the September rebuild and is not true now. The agent is told a fact about its own environment that has expired, and nothing in the platform notices.

### G.7 The documented knob that controlled nothing

Chapter 19's most quotable find. The outer variable, with the warning added after the discovery:

```bash
# bake-common.sh (excerpt — header comment)
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

K3S_VERSION="${K3S_VERSION:-v1.37.0+k3s1}"
```

And the literal that actually reaches the machine, a hundred lines below it. The `<< 'CLOUDINIT'` — note the quotes — is what makes the block above inert: a quoted heredoc expands nothing.

```bash
# bake-common.sh (excerpt — the embedded cloud-init content)
  cat << 'CLOUDINIT'
  # …
  - path: /usr/local/bin/bake.sh
    content: |
      #!/bin/bash
      set -e
      K3S_VERSION="v1.37.0+k3s1"
      # …
      curl -sfL https://get.k3s.io | \
        INSTALL_K3S_VERSION="$K3S_VERSION" \
        INSTALL_K3S_SKIP_START=true \
```

Two assignments of one name, in one file, a hundred lines apart. Setting the documented one changes nothing at all.

### G.8 The dead image, and the patch that survives a rebuild

Chapter 19's first real obstacle, and the fix that now runs automatically — quoted with its comment, because the comment records the uncertainty honestly rather than claiming a clean diagnosis.

```bash
# 02-capi-init/init-management-cluster.sh (excerpt)
# ship their kube-rbac-proxy sidecar as `gcr.io/kubebuilder/kube-rbac-proxy:v0.16.0`,
# a reference that no longer resolves (ImagePullBackOff, "not found") --
# apparently a dead/retired gcr.io path, though clusterctl fetches the release
# manifest fresh on every `init` so it's unclear whether this was ever pullable
# from this repo's install path. Whatever the history, the *deployment* that
# was actually running before this rebuild used
# `quay.io/brancz/kube-rbac-proxy:v0.16.0` (confirmed pullable), so that's the
# known-good pin here. Without this, both provider Deployments sit at 1/2
# ready forever and the `kubectl wait --for=condition=available` calls below
# silently time out into their `|| true` fallback -- so `make capi-init`
# reports success while KThreesControlPlane's mutating webhook is dead,
# and the first `target-cluster` deploy fails opaquely on
# "failed calling webhook ...kthreescontrolplane: connection refused".
for ns in capi-k3s-bootstrap-system capi-k3s-control-plane-system; do
  deploy="$(kubectl get deploy -n "$ns" -o name 2>/dev/null | head -1)"
  [[ -n "$deploy" ]] || continue
  if kubectl get "$deploy" -n "$ns" -o jsonpath='{.spec.template.spec.containers[?(@.name=="kube-rbac-proxy")].image}' 2>/dev/null \
      | grep -q '^gcr.io/kubebuilder/'; then
    echo "==> Patching $deploy ($ns): kube-rbac-proxy gcr.io image -> quay.io/brancz (known pullable)"
    kubectl patch "$deploy" -n "$ns" --type=strategic \
      -p='{"spec":{"template":{"spec":{"containers":[{"name":"kube-rbac-proxy","image":"quay.io/brancz/kube-rbac-proxy:v0.16.0"}]}}}}'
  fi
done
```

The `|| true` that the comment mentions is the pattern Appendix C.2 catalogues: a wait that fails, a fallback that swallows the failure, and a script that reports success.

## Appendix H — Sources

The book draws on three kinds of source, and they are not equally checkable. This appendix says which is which.

### H.1 The companion repository

The **Build log** sections come from the platform repository, `sovereign_cloud`. It is the only source a reader can run. Every build-log claim names the file or command behind it in that chapter's **Open the repo** block, and the load-bearing files are quoted in Appendix G.

One caveat, recorded in Chapter 19's draft notes: most of the material this book cites lives on the working branch `upgrade/k8s-1.37-istio-1.31-sympozium-0.10.75`, not on `main`. A reader who clones the default branch will not find much of it.

### H.2 Briefing material (not public, not in the repository)

Four items are the author's own briefing material. They are described here rather than linked, because no path or URL would resolve for a reader. Where the book quotes them, it quotes them verbatim and says so.

| Referred to in the book as | What it is |
|---|---|
| **The strategy deck** | *Reclaiming Margins with Sovereign Cloud*, a strategic briefing deck dated February 2026, in five chapters. Parts I and V adapt its argument; Chapter 20 takes its TCO slide apart. None of its figures carries a source, and the book says so each time it quotes one. |
| **The sovereign-cloud flyer** | A one-page campaign flyer, version 2, unbranded — six promises and a *"Save $Millions — Creating a Cloud Owning Culture"* band. The epilogue reads it claim by claim. A separate version of the same flyer carries a real company's branding; the book does not quote that one, and does not name the company. |
| **The recorded factory demonstration** | A screen recording, 8 minutes 56 seconds, of the cluster-provisioning path in Chapters 4 and 5. Used only as a source of stills. |
| **The recorded agent demonstration** | A screen recording, 2 minutes 50 seconds, of the agent console in Chapter 13. Used only as a source of stills. |

The cross-cluster mind map in Chapter 10 was previously cited from this material as well. It is in the repository, at `05-istio/cross-cluster-mindmap.html`, and the book now cites it there. The briefing copy is byte-identical, which is exactly Chapter 10's point: a diagram duplicated into two places goes stale in both.

### H.3 The public record

**Public record** sections cite published, checkable sources: the Andreessen Horowitz cloud-cost paper (2021), Dropbox's 2018 IPO filing, 37signals' published cloud-exit accounts, Gartner's public-cloud spending forecast, Flexera's annual cloud reports, the EU Data Act, the AT&T/Broadcom litigation coverage, Grand View Research's sovereign-cloud market sizing, and the hyperscalers' own server-depreciation disclosures. Each is named where it is used, with its date.

Their verification status, as re-checked on **18 September 2026**:

| Source | Status |
|---|---|
| Andreessen Horowitz, *The Cost of Cloud, a Trillion-Dollar Paradox* | **Verified against the article itself** — date (27 May 2021), authors, the 50%-of-cost-of-revenue benchmark, the one-third-to-one-half repatriation figure, and the $100B / >$500B market-value estimates all confirmed as quoted |
| 37signals, *Our cloud exit savings will now top ten million over five years* | **Verified against the post itself** — date (17 October 2024), $3.2M/year run rate, $1.3M, about $700,000 of servers, "well over ten million dollars over five years", and the "no hidden dragons" sentence all confirmed as quoted |
| EU Data Act, Regulation (EU) 2023/2854 | **Partly verified, and one error corrected.** In force 11 January 2024; obligations applicable 12 September 2025; charges withdrawn 12 January 2027. An earlier draft misread the in-force date as the date cost-only charging began (Chapter 2). Still to verify against the EUR-Lex text rather than commentary, and to have counsel review |
| Gartner, worldwide public cloud end-user spending forecast for 2025 | **Not verified** — the press release returns HTTP 403 to direct retrieval. The $723.4B figure comes from the citation as previously recorded. Confirm before publication |
| Dropbox 2018 IPO filing ($74.6M Infrastructure Optimization saving) | **Not re-verified** — currently cited via secondary coverage. Cite the S-1 directly |
| Flexera annual cloud reports (29% waste, 85%) | **Not re-verified this pass** — URLs recorded in the prologue's draft notes |
| AT&T / Broadcom litigation and settlement | **Not re-verified** — trade-press coverage only. Cite the court filing if possible |
| Grand View Research, sovereign-cloud market sizing | **Not verified** — the report page returned HTTP 403 when attempted; figures come from a search-result summary |
| Hyperscaler server-depreciation disclosures (Microsoft FY2022, Alphabet 2023, Amazon Feb 2025) | **Not verified** — secondary coverage summarising filings. Cite the 10-K/10-Q text |
| GDPR, Schrems II, CLOUD Act, India DPDP Act | **Not verified** — stated from general knowledge. Have counsel review (Chapter 3) |

A search summary or a news report is not the filing it describes, and every row above marked *not verified* is a row where the book is currently trusting one.

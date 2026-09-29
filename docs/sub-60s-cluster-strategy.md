# Sub-60s Target Cluster Provisioning — Strategy, Attempts, and What's Next

## TL;DR

- **Baseline (legacy `:latest` image):** 90–150s time-to-ready (p95).
- **Current (`:preinit` image, Phase 1 shipped):** ~136s in the one measurement we have.
- **Target:** <60s p95.
- **Gap:** we have **not** broken past the legacy band. The change that was supposed to deliver the speedup (preserving k3s state in the golden image) is neutralized by CAPK's bootstrap model — explained below.
- **Path to <60s:** requires one of the "Option B/C" strategies in §5, not more tuning of the current approach.

---

## 1. The provisioning pipeline (what takes time)

```
kubectl apply -f <cluster.yaml>
  → CAPI Cluster created                                             (~1s)
  → CAPK DataVolume clone kicks off                                  (0s if pre-pulled)
  → KubeVirt VMI scheduled → Pending → Running                       (~30–60s, dominated by clone if not pre-pulled)
  → cloud-init: write_files (CAs, config.yaml) + runcmd              (~5–10s)
  → /opt/install.sh: systemctl enable+start k3s                      (~1s)
  → k3s.service ExecStartPre = firstboot-regen.sh
      → k3s server --cluster-reset                                    (~20–30s)
  → k3s.service ExecStart = k3s server                                (~10–20s)
  → CAPK control-plane service + MetalLB IP binding                   (~5–10s)
  → KThreesControlPlane marks `Initialized`                           (~5s)
  → MachineDeployment creates worker VMI                              (starts AFTER CP is Ready)
  → worker VMI Running → k3s-agent joins                              (~30–60s)
──────────────────────────────────────────────────────────────────────
Total observed (preinit): ~136s                Target: <60s
```

Note the serialization: **the worker VM doesn't even start booting until the control-plane is `Ready`**. That's ~60s we can't reclaim without parallelizing or pre-booting.

---

## 2. What was tried (Phase 1 as planned)

The plan lived at [`/home/mahipal/.claude/plans/this-is-a-longterm-humming-moon.md`](the approved Phase 1–5 plan).

Phase 1 bake (`bake-common.sh`) preserved full k3s state in the golden image:
- `/var/lib/rancher/k3s/server/db/` — etcd DB with CRDs, RBAC, system charts
- `/var/lib/rancher/k3s/server/tls/` — CA + leaf certs
- `/var/lib/rancher/k3s/agent/images/` — airgap tarballs
- `firstboot-regen.sh` installed as k3s.service `ExecStartPre`

Hypothesis: first boot runs `k3s server --cluster-reset` (keeps DB, issues fresh etcd cluster ID), so each new cluster skips cold etcd bootstrap, cert gen, bootstrap-data write, system-chart apply. Estimated savings: 30–60s.

---

## 3. What failed, and how it was fixed

### 3.1 Clone DataVolume stuck at `WaitForFirstConsumer`

**Symptom:** `ubuntu-noble-k3s-preinit` DV sat at WFFC phase indefinitely; bake loop polled "DataVolume phase: …" with no progress.

**Root cause:** the default storage class uses WaitForFirstConsumer binding — CDI defers the import until a pod actually mounts the PVC. No pod = no mount = no progress.

**Fix:** added `cdi.kubevirt.io/storage.bind.immediate.requested: "true"` annotation on the import DV in `scripts/import-base-images.sh`.

### 3.2 Source DV `ubuntu-noble-dv` missing

**Symptom:** CDI reported `CloneWithoutSource: The source pvc ubuntu-noble-dv doesn't exist`. The repo assumes the base Ubuntu cloud image is imported but has no script to do it.

**Fix:** added `scripts/import-base-images.sh` that imports both `ubuntu-noble-dv` (~600 MB) and `ubuntu-minimal-noble-dv` (~300 MB) from the Ubuntu cloud-images HTTP URLs via CDI `http:` source.

### 3.3 CA-injection conflict → `k3s --cluster-reset` infinite loop

**Symptom (the big one):** cluster stuck at `KThreesControlPlane: WaitingForKthreesServer` for 20+ minutes. Inside the CP VM, k3s.service was `activating` (not `active`). The firstboot-regen log (`/var/log/k3s-firstboot-reset.log`) was spamming:

```
CA cert validation failed: Get "https://127.0.0.1:6444/cacerts":
  tls: failed to verify certificate: x509: certificate signed by unknown authority
```

**Root cause:** two-CA state on disk.
1. The bake generated k3s CAs (server-ca, client-ca, etcd/server-ca) during its warm-up init and signed every leaf cert (`client-admin.crt`, `serving-kube-apiserver.crt`, etc.) against them.
2. CAPK bootstrap writes **fresh per-cluster CAs** into the VM via cloud-init `write_files` — overwriting **only the `*-ca.crt` + `*-ca.key` files**, not the leaves.
3. So on first boot: CAs are CAPK's, but leaves + `dynamic-cert.json` + `cred/*.kubeconfig` are still bake-signed.
4. `k3s server --cluster-reset` starts its supervisor at `127.0.0.1:6444`, which presents a cert from `dynamic-cert.json` (bake-signed). The k3s client in the same process trusts `server-ca.crt` (CAPK's). Mismatch → retry forever.

This failure mode is invisible in design docs that only reason about etcd identity. It only showed up under real end-to-end boot.

**Fix:** in the CP's `preK3sCommands` (runs **after** CAPK `write_files` lands the fresh CAs, **before** k3s starts), purge everything in `/server/tls/` that isn't a CA:

```yaml
preK3sCommands:
  - rm -f /var/lib/rancher/k3s/server/token
  - find /var/lib/rancher/k3s/server/tls -type f
      ! -name '*-ca.crt' ! -name '*-ca.key' ! -name '*-ca.nochain.crt'
      -delete
```

k3s regenerates every leaf + `dynamic-cert.json` from the CAPK CAs on next start. `--cluster-reset` then validates cleanly, etcd comes up, API responds.

This fix is already in `03-target-cluster/target-cluster.tmpl.yaml` and `03-target-cluster/target-cluster-preinit-test.yaml`.

### 3.4 Why the fix neutralizes most of the speedup

The preservation-win plan assumed `/server/tls` survives to first boot. Since it doesn't (we purge it), k3s regenerates every leaf cert (~5–10s), and `--cluster-reset` (~20–30s) still runs to issue fresh etcd identity. The only real wins left:

- Pre-pulled airgap images → no CRD/system-chart download or apply.
- Preserved `/server/db` → RBAC and CRDs already in etcd (saves ~5–10s of initial apply).
- k3s binary + systemd unit baked in → no install-script download or first-boot compile.

Net: comparable to the tuned legacy `:latest` image. The sub-60s target needs a **different strategy**, not more tuning of this one.

---

## 4. What shipped (Phases 2–5, working)

Independent of the Phase 1 speedup result, the following are in and working:

- `bake-common.sh` — shared bake payload, parameterized by env.
- `bake-golden-image-minimal.sh` — Ubuntu Minimal variant (~300 MB base + k3s).
- `03-target-cluster/target-cluster.tmpl.yaml` — single manifest template driving both image variants.
- Backend renders + applies the template per deploy: `ui/backend/handlers/cluster_deploy.go` `runDeployment(profile, image)`.
- Make targets: `target-cluster-{lite,full}-{preinit,minimal}`, wired to `scripts/render-cluster.sh`.
- UI: profile + image variant toggles in `ClusterManager.tsx`, `api.deployCluster(profile, image)`.
- `scripts/import-base-images.sh`, `scripts/render-cluster.sh`, `scripts/time-to-ready.sh`.

So the multi-option UX — pick Noble or Minimal, pick lite or full — works end-to-end. What **doesn't** work is "sub-60s".

---

## 5. How to actually get under 60s

Ranked by likely impact and invasiveness.

### Option A (cheap, ~10–20s win) — Parallelize CP and worker boot

CAPK defaults to starting the worker only after CP is `Ready`. That's ~60s of serialized wait.

**Approach:** remove the MachineDeployment's implicit dependency by pre-creating the worker VMI before CP is Ready, then gate k3s-agent startup (not VM boot) on CP API availability.

- Worker's `kthreesConfigSpec.postK3sCommands` already waits for the supervisor URL. If the VM boots in parallel with CP, k3s-agent retries its join until CP serves — adds maybe 5–10s of idle retry, saves ~40s of serialized boot.
- Tradeoff: slight risk that worker k3s-agent retries burn log noise. Acceptable.

Projected: ~100s → **~75s**. Still not <60s.

### Option B (high impact, ~40–60s win) — Warm VM pool

The original plan flagged this as "out of scope, revisit if Phase 1 misses <60s" — which is where we are.

**Approach:** keep N pre-booted target-cluster VMs in a `Paused` state. On deploy, resume the pool, re-identify them as the new cluster (re-run firstboot-regen logic), and hand over.

- CP comes up in <10s (VM already booted, k3s already warm).
- Worker in parallel, same pool.
- Main cost shifts to pool maintenance, not user-visible deploy.

Projected: **20–40s** per deploy, once pool is warm.

Tradeoffs:
- Maintenance load: need to refresh pool after deletes.
- Memory: pool VMs consume RAM while idle.
- Identity re-issue at resume time is tricky — each cluster needs distinct etcd cluster ID, kubelet identity, node name. The `--cluster-reset` path we already have is the right primitive, but it has to run on resume without blocking.

### Option C (invasive, 30–50s win) — Skip `--cluster-reset` by renaming etcd member in place

The approved plan **rejected** this approach (etcd raft WAL has the cluster ID, renaming dirs won't reissue a cluster ID). But if `--cluster-reset` is the dominant cost and we can't eliminate it, forking the k3s-side logic to do an in-place metadata rewrite might be the only way.

**Approach:** stop k3s → `etcdutl` surgery to write new cluster ID into WAL + snapshot → restart. Risky (WAL format is internal) but avoids the 20–30s of `--cluster-reset`.

This is a research project, not a week of work.

### Option D (invasive, hard to estimate) — Custom bootstrap provider

CAPK writes CAs because it wants to manage cluster identity from the management cluster. A custom bootstrap provider (or a CAPK patch) could be told "the target cluster already has a CA — fetch it via the kubeconfig secret instead of injecting."

**Approach:** fork `KThreesConfig` logic, or sit a mutating admission webhook in front of it to strip the `write_files` CA entries. Then the bake-signed leaves + dynamic-cert.json stay valid — no purge, no regen, maybe no `--cluster-reset`.

Projected: **50–70s**, if combined with the other wins.

Tradeoff: upstream-drift risk, ongoing maintenance.

---

## 6. Recommendation

1. **Ship what we have now** — Phases 1–5 deliver the multi-image UX and a working pipeline. 136s is not the target, but it's not worse than legacy.
2. **If <60s becomes business-critical:** go straight to **Option B (warm VM pool)**. Everything else is smaller wins with similar effort.
3. **Don't** keep tuning the Phase 1 preserved-state approach. It's CA-injection-bound and will not break through the ~100s floor with CAPK in the picture.

---

## 7. Reproducing today's result

```bash
# one-time
./scripts/import-base-images.sh                    # ~3 min, ~900 MB download
make bake-image-preinit                            # ~7 min
make build-containerdisk-preinit                   # ~2 min
make pre-pull-preinit                              # seconds

# per run
kubectl delete cluster target-cluster --ignore-not-found --wait=true
./scripts/time-to-ready.sh                         # apply → 2/2 Ready
```

Expected: ~130–150s total to `RESULT:` line. p95 over 5 runs will take ~15 min of wall time.

---

## 6. Phase-level measurement (2026-09-09) — instrumentation, and three dead ends

Everything above was judged on a single aggregate number. `scripts/phase-timings.sh`
(new) attributes the wall clock per phase, read-only and without SSH (the
`v1.subresources.kubevirt.io` APIService is `Available=False` on this host, so
`virtctl ssh/console` is unusable). Guest uptime is anchored to wall clock from
cloud-init's own log lines, which carry an absolute date and the uptime together.

### The real budget (warm path, median of 3 clean runs)

```
apply → VMI created            6.6s   14%   CAPI/CAPK/KThrees controller chain
launcher pod → qemu running   10.0s   21%   8s of it is containerDisk unpack
firmware + kernel + initramfs  3.7s    8%
systemd + cloud-init → k3s     5.5s   12%
CP k3s → first 200 /readyz     7.0s   15%
worker join after API ready   11.8s   25%
──────────────────────────────────────
median total                  50.0s          (52.7 / 50.0 / 46.6)
```

**36% of the wall clock (16.6s) elapses before the guest kernel starts.** No
image-side change can reach it. This is a third, independent reason the `:preinit`
and `:warm` experiments could not have delivered what they promised.

### Dead end 3 — gating the worker on CP readiness

Hypothesis: the k3s agent starts at t+26.7s, the API first answers at t+34.8s, and the
agent retries on a ~5s cycle, so it lands on the t+36.7s attempt — 1.9s of pure
quantization loss. Fix: spin on `/readyz` (which answers unauthenticated) before
starting the agent.

**Result: 53.1s median, a 3.1s REGRESSION.** The premise was wrong. The journal shows
the agent starts containerd *before* it fetches config, so the retry window is
productive work overlapping CP boot. Delaying the agent serialized it. Reverted.

### Dead end 4 — the `no route to host` dials

KubeVirt bridge binding gives the guest the launcher pod's address *and* its /24, so
the guest treats every pod IP as on-link and ARPs for it — but pod-to-pod on kindnet
is L3-routed, not one L2 segment, so the ARP cannot resolve. The agent learns the CP's
**pod** IP from the supervisor and dials it directly, taking `connect: no route to
host` three times at ~3.1s each before falling back to the VIP. It also received a
stale `10.0.2.2:6443` (the QEMU slirp gateway) baked into the warm datastore.

Routing the pod CIDR via the gateway fixes it completely: 6 errors → 0, one clean dial,
one apiserver address. **Timing benefit: none (54.3s median).** The journal shows why —
between `"Running kubelet --address=0.0.0.0"` and the kubelet's first output there is a
**10s silent gap**, and the failed dials were running *concurrently inside* it, not
adding to it. A real bug, worth keeping for correctness; not a speedup.

### Fixed: workers were starting the k3s SERVER unit

The worker bootstrap passed the mode as `INSTALL_K3S_EXEC='agent'`, but the baked
`/opt/install.sh` parses only positional args and ignores the env var, so it defaulted
to server. `k3s server` failed ~0.8s later and, because install.sh runs under `set -e`,
aborted the script before writing the `bootstrap-success.complete` sentinel (harmless
only because `virtualMachineBootstrapCheck.checkStrategy: none` is set). Fixed by
passing `agent` positionally. Verified: `Mode: k3s-agent`, zero `k3s.service` failures.

### FIXED: the 10s kubelet gap — 50.0s -> 42.7s

Debug logging on the agent named it exactly:

```
Dial error from server 10.244.0.186:6443@UNCHECKED after 10.000162522s: i/o timeout
```

A **10.000s TCP dial timeout**, with the kubelet's first log line landing 7ms after it
expires — kubelet start is gated on the agent's tunnel dial. The address being dialled
is the **control plane's pod IP**, handed to the worker by the supervisor's apiserver
endpoint list. And that address cannot work:

```
node -> CP VM pod IP:6443   =  200, connect 0.0003s
pod  -> CP VM pod IP:6443   =  timeout
```

**A KubeVirt VM's pod address is reachable from the node but not from other pods.** The
worker's k3s agent runs inside a pod, so it can never reach it; it waits out the full
timeout before falling back to the VIP it already had configured as `default`.

Fix: `advertise-address: <API_LB_IP>` appended to the CP's `config.yaml` in
`preK3sCommands`, so the supervisor advertises the VIP every VM already reaches.
Applied to `target-cluster-warm.yaml`, `target-cluster-warm.tmpl.yaml`, and
`target-cluster-parallel.yaml`.

| | baseline | fixed |
|---|---|---|
| median time-to-ready | 50.0s | **42.7s** |
| runs | 52.7 / 50.0 / 46.6 | 42.7 / 42.8 / 41.2 |
| run-to-run spread | 6.1s | **1.6s** |
| dial timeouts per run | 4-6 | **0** |
| worker join after API ready | 11.8s | 5.1s |

The spread collapsing from 6.1s to 1.6s matters as much as the median: the 10s timeout
was also the dominant source of run-to-run variance. Verified safe — the `kubernetes`
service endpoint becomes the VIP and a pod in the target cluster still reaches
`kubernetes.default` (200).

Two false starts on the way, both worth recording. The dials were *first* seen failing
with `no route to host`, because bridge binding also hands the guest the pod CIDR as an
on-link /24 so it ARPs for a pod IP that is L3-routed. Routing that subnet via the
gateway removed the ARP failure — and bought nothing, because the dial then simply
blackholed for the same 10s. And the gap measured 10.019s / 10.017s across runs with
completely different dial behaviour, which looked like proof of a fixed timer unrelated
to the dials; it was actually the dial timeout itself, reached by two different routes.

### FIXED: the 8s containerDisk phase — 42.7s -> 34.5s

KubeVirt gives its per-VM support containers microscopic CPU limits by default:

| container | image | cpu limit | start |
|---|---|---|---|
| `guest-console-log` (virt-tail) | virt-launcher | **15m** | +3.0s |
| `volumesystemdisk` (containerDisk) | noble-k3s 1.17GB | **10m** | +9.0s |
| `compute` (virt-launcher) | virt-launcher | *(none)* | +9.0s |

`10m` is 1% of a core: the CFS quota grants ~1ms per 100ms period, so even
`container-disk --no-op` needs dozens of periods just to dynamically link and fault
in its pages. Deterministic 8.0s on every VM, every start.

Isolated by elimination — each of these was measured and ruled out first:

| hypothesis | test | result |
|---|---|---|
| image size (1.17GB) | mount same image as an *image volume* | container started **+0.0s** |
| concurrency/contention | solo VMI, nothing else starting | **9.0s** — worse than 8.0s |
| node CRI latency | trivial 2-container pod | **1.0s** total |
| image-volume mounting | as above | fast |

The tell was that the cost is *deterministic* (8.0s every run) — the signature of a
quota, not of load. Fix: `supportContainerResources` on the KubeVirt CR, applied by
`scripts/configure-kubevirt-perf.sh` (`make kubevirt-perf`, and auto-run from
`02-capi-init/init-management-cluster.sh` so it survives a rebuild). The limits are a
ceiling, not a reservation — these containers are idle after startup.

| | before | after |
|---|---|---|
| containerDisk phase | 8.0s | **1.0s** |
| pod → qemu running | 10.0s | 4.0s |
| median time-to-ready | 42.7s | **34.5s** |
| runs | 42.7 / 42.8 / 41.2 | 34.5 / 34.2 / 34.7 |
| spread | 1.6s | **0.5s** |

### Where the two fixes leave things

```
baseline                 50.0s   (52.7 / 50.0 / 46.6, spread 6.1s)
+ advertise-address      42.7s   (42.7 / 42.8 / 41.2, spread 1.6s)
+ supportContainerRes    34.5s   (34.5 / 34.2 / 34.7, spread 0.5s)
                       ────────
                        -15.5s   (-31%), gate was <40s
```

Both fixes are one-line configuration, not architecture. Neither is an image change —
which is the running theme of this document: every attempt to make the *image* smarter
(`:preinit`, `:warm`) failed, while the two things that worked were a bad advertised
address and a CPU quota. The spread collapsing 6.1s → 0.5s is worth as much as the
median: the pipeline is now predictable enough that a 1s regression is visible.

### What is actually left, by measured size

0. ~~10s kubelet gap~~ and ~~8s containerDisk~~ — **both fixed above.**
1. **~6s CAPI controller chain before the guest boots** (apply → VMI created).
   Untouched by any image or guest work.
2. **~10s silent gap inside kubelet startup on the worker** — the single largest
   in-guest item, and completely opaque so far.
3. **7s CP k3s → API ready.**
4. **5.5s systemd + cloud-init.** Note this is the phase a systemd-less image would
   attack; a perfect result there is worth ~3s, so it is the *fourth* lever, not the
   first.

### Measurement caveat that limits all of the above

Run-to-run spread on this host is **~6s** (baseline 46.6–52.7). With N=3 that swamps
any lever smaller than ~5s. Anything finer needs N≥7, or a quiet host, or both.

---

## 7. The CAPI chain after the September upgrade (2026-09-23) — a CAPK watch bug

Same-session baseline (`scripts/time-to-ready-by-name.sh --runs 3`, warm manifest,
upstream CAPK v0.11.2): **31.3 / 33.2 / 62.8s**. The 62.8s run was not guest-side:
the control-plane VM was created 30s after its bootstrap data was ready, while the
worker VM (static bootstrap) appeared at +5s.

Two stacked CAPK bugs, both from CAPK v0.11.2 being built against CAPI v1.11 while
still watching the **v1beta1** API types:

1. **All three CAPK watches are dead.** `KubevirtMachine<-Machine`,
   `KubevirtMachine<-Cluster`, `KubevirtCluster<-Cluster` watch v1beta1 objects, but
   CAPI v1.11's `util.MachineToInfrastructureMapFunc`, `util.ClusterToInfrastructureMapFunc`
   and the cluster predicates type-assert `*v1beta2` objects. The assertion fails on
   every event, so nothing is ever enqueued. CAPK learns that
   `Machine.Spec.Bootstrap.DataSecretName` is set only when an unrelated event or a
   10-20s `RequeueAfter` happens to fire (observed: 20.5s in one run).
2. **The "waiting for bootstrap data" path dials the target API.** `reconcileNormal`
   returns a zero result while waiting, which falls through to `updateNodeProviderID`
   and a workload-API call to the VIP — before any VM exists. With no backend, the dial
   fails in anything from 2s (`connection refused`) to the full 30s client-go dial
   timeout (`i/o timeout`), and the reconcile is blocked for all of it.

Across the three baseline runs, "bootstrap data ready -> CP VM created" was
**2.0 / 11.2 / 30.0s** — the entire run-to-run spread.

**Fix:** `02-capi-init/patch-capk.sh` (`make capk-patch`, auto-run from
`init-management-cluster.sh`) builds CAPK from `capk-patches/<tag>.patch` — watch the
v1beta2 types, skip the providerID lookup while no bootstrap data exists, bound
workload-API calls to 5s — and layers only the binary onto the official image.
Keyed to the installed CAPK tag; on any other tag it warns and does nothing.
**Cluster state, not manifest state: re-run after any cluster2 rebuild.**

| CAPK | both nodes Ready (s) | median | spread |
|---|---|---|---|
| upstream v0.11.2 | 31.3 / 33.2 / 62.8 | 33.2 | 31.5 |
| patched | 34.4 / 32.8 / 33.4 | 33.4 | **1.6** |

**This is a tail fix, not a median win.** A lucky upstream run was already ~33s; the
patch makes every run that. Chain after the fix: VIP assigned / KubevirtCluster Ready
+3s, Machines +4-5s, CP bootstrap secret +6s, both VMs +6-7s.

**Measurement fix found on the way:** `time-to-ready-by-name.sh` polled with
`kubectl` and no request timeout, so its own dial to the backend-less VIP could hang
and overstate the result (seen: node Ready timestamps +24s/+31s, script 34.8s for both).
It now uses `--request-timeout=2s`. The A/B above was measured after that fix.

---

## 8. The worker join (2026-09-23) — k3s's jittered config retry, and a taint nobody removed

After §7 the budget was: CP Ready ~25.5s, worker Ready 6-9s later. The worker
journal (read via SSH through cluster2's node netns) showed where the gap went:

```
06:25:05.1  agent starts; CA fetch fails (CP API not up), retries every 2s
06:25:10.25 "Waiting to retrieve agent configuration; server is not ready:
             serving-kubelet.crt: 503"                         <- 0.27s too early
06:25:10.52 CP: "Kube API server is now running"
06:25:17.4  agent retries, gets config, starts containerd -> kubelet -> registered 19.3
```

k3s v1.37's agent-config loop is `wait.JitterUntilWithContext(..., 5*time.Second, 1.0, ...)`
— a **random 5-10s** between attempts. A first attempt that lands even a fraction of a
second before the supervisor's runtime core is ready costs 5-10s of idle time, and
that jitter was also most of the remaining run-to-run spread.

### Dead end 3 revisited — gating the agent now works

§6 recorded gating the agent on CP readiness as a 3.1s *regression* on k3s v1.31,
because the agent then started containerd *before* fetching config, so the wait
overlapped useful work. **On v1.37 the order is reversed**: containerd starts only
after the config arrives (`Running containerd` right after the config fetch), so
holding the agent back costs nothing.

**Probe choice matters.** First attempt used `/v1-k3s/config` (token auth): it answered
200 ~2s *before* the CP API was up, so the gate opened early and the agent hit the same
503 — no gain (32.8s). The 503 comes from the node-password check (`controller == nil`
-> `ErrCoreNotReady`); probing that directly would create a node-password secret for
the probe's name. The right probe is the supervisor's own **`/v1-k3s/readyz`** (token
auth, no side effects), which 503s until `Runtime.Core` is set.

The static worker bootstrap now spins on it every 0.2s (bounded ~5 min) before
`install.sh agent`. Agent timeline after: CP API up 03.00 -> gate open, agent starts
03.57 -> containerd 04.98 (no retry) -> registered 06.77.

### The worker was never schedulable

The same journal showed `k3s-agent` failing after exactly 15 minutes with
`network policy controller failed to wait for node.cloudprovider.kubernetes.io/uninitialized
taint to be removed`, then restarting forever — and cloud-init blocked on it for ~30 min.
The static bootstrap passed `kubelet-arg: cloud-provider=external`, but the CP runs
`disable-cloud-controller: true` with no CCM, so nothing ever removed the taint: the
worker went Ready but `NoSchedule`, and every pod in the cluster ran on the CP. Removed
the flag (warm, warm.tmpl, parallel). Worker now untainted, agent `active`,
cloud-init `done`. No timing effect on its own (median 33.0s).

| change (cumulative, patched CAPK) | both nodes Ready (s) | median |
|---|---|---|
| before | 34.4 / 32.8 / 33.4 | 33.4 |
| drop `cloud-provider=external` | 30.6 / 34.6 / 33.0 | 33.0 |
| + gate on `/v1-k3s/config` (wrong probe) | 33.9 / 32.8 / 31.6 | 32.8 |
| + gate on `/v1-k3s/readyz` | 28.9 / 29.0 / 30.8 | **29.0** |

Worker now joins 3.4-4s after the CP (was 6-9s).

### Exposed: cross-node pod networking does not work

With the taint gone, pods schedule on the worker — and it turns out **VM-to-VM traffic
has never worked**: worker VM -> CP VM IP is 100% loss both ways, the on-link ARP for the
other VM is `FAILED`, and routing it via the gateway (10.244.0.1) still drops — the same
blackhole as §6 "dead end 4" (node -> VM works, forwarded pod -> VM does not). So flannel
VXLAN between the nodes has no path. metrics-server has always failed to scrape the worker
(`no route to host`), and now CoreDNS can land on the worker where CP pods cannot reach it.
Before this change the taint hid it by keeping everything on the CP. **Open; not a timing
issue.**

---

## 9. VM-to-VM networking fixed (2026-09-23) — and why §6 "dead end 4" looked like a blackhole

**Root cause.** KubeVirt bridge binding gives the guest its launcher pod's address *with
the Kind node's pod prefix* (`10.244.0.211/24`), so the guest treats every other pod on
that node — including the other target-cluster VM — as on-link and ARPs for it. kindnet
routes pods at L3 (point-to-point veths, no proxy ARP), so the ARP never resolves.

**Why routing via the gateway "still blackholed" in §6.** A capture on the Kind node
showed the node forwarding the worker's SYN correctly out of the CP's veth — and the CP
never answering, because *its reply* to the worker was still on-link and ARPed into the
void. §6 routed only one side. With the gateway route on **both** VMs: TCP 6443 -> 200,
ICMP 0% loss.

**Fix.** `/usr/local/sbin/pod-subnet-via-gateway.sh`, shipped in both bootstraps
(`kthreesConfigSpec.files` + first `preK3sCommands` on the CP; `write_files` + first
`runcmd` on the worker) in warm, warm.tmpl and parallel. It adds the two half-prefixes of
each on-link subnet via the default gateway — more specific than the kernel's connected
/24, which DHCP keeps owning — computed from the live route table, since the prefix
depends on which Kind node the VM lands on. No rebake.

**Verified on a cluster built only from the manifests:** both VMs carry the /25 routes;
VM<->VM 0% loss; a pod on the CP node resolves DNS through CoreDNS on the worker and pings
it across nodes (flannel VXLAN, 0% loss, ~0.5ms); `kubectl top nodes` reports both nodes
(metrics-server on the worker scraping the CP); `make verify` passes. Timing unaffected:
**30.0 / 28.8 / 28.7s, median 28.8s**.

**Not covered:** the legacy KThrees-generated paths (`target-cluster.yaml`, `-lite`,
`target-cluster.tmpl.yaml`) don't carry the script, and their workers also can't reach the
CP VM. Port it to their `preK3sCommands` if those paths are kept.

**Caveat:** the routes are runtime state. They survive DHCP renewals, but a
`systemd-networkd` restart drops foreign routes (`ManageForeignRoutes=yes` by default).
A VM restart is fine — a containerDisk VM re-runs cloud-init from scratch. Baking the
script into the image as a networkd-dispatcher hook (lever 3's rebake) would make it
durable.

---

## 10. Guest boot path (lever 3, 2026-09-23) — ~4s off every VM start

Method: edit a live target VM, `systemctl reboot` it (a KubeVirt guest reboot keeps the
ephemeral disk), then read `systemd-analyze` plus the console log's own timestamps
(`reboot: Restarting system` -> first kernel line = SeaBIOS + GRUB + kernel/initrd load).
Two reboots per step. Per-VM-start savings, worker VM:

| change | phase | before | after |
|---|---|---|---|
| smaller initrd (`MODULES=dep`, 30.7MB -> 17.5MB) | firmware + GRUB | 1.73s | 1.09-1.34s |
| initrd-less boot (`GRUB_FORCE_PARTUUID`) | firmware + GRUB | | **0.78-0.83s** |
| `modprobe.blacklist=ahci,libahci,psmouse`, drop btrfs-progs/mdadm hooks | kernel + initramfs | 3.14-3.26s | 2.79-2.82s |
| + initrd-less boot | kernel | | **1.32-1.53s** |
| `LinkLocalAddressing=no` on enp1s0 (+ wait-online `--ipv4`) | wait-online | 1.31-1.89s | **11-12ms** |

Findings along the way:
- **wait-online was waiting for IPv6**, not DHCP: DHCPv4 landed 11ms after it started,
  and it finished at "Gained IPv6LL" ~1.3s later. `--ipv4` alone did NOT fix it:
  networkd keeps the link `configuring` until IPv6LL finishes, so it has to be switched
  off in the `.network` file (which the bake writes: `LinkLocalAddressing=ipv6` -> `no`).
- **A wait-online drop-in named `10-*` is silently overridden** by netplan's generated
  `/run/.../10-netplan.conf` (drop-ins apply in filename order). Use `99-*`.
- The kernel probe gaps were the q35 AHCI controller with no disks (0.31s), the PS/2 mouse
  probe (0.72s) and the raid6 benchmark from the btrfs/mdadm initramfs hooks (0.44s).
- `virtio_blk`, `virtio_pci` and `ext4` are built into Ubuntu's generic kernel, so
  Ubuntu's initrd-less boot (`GRUB_FORCE_PARTUUID`; GRUB falls back to the initrd if the
  boot fails, which is why `panic=-1` appears on the cmdline) works on these VMs.

Baked into `bake-common.sh` (step 4e + the `10-enp1s0.network` change). Side effect:
removing `btrfs-progs` also removes the `ubuntu-server` metapackage (a metapackage only; it
uninstalls nothing else).

**End to end** (patched CAPK, worker readyz gate, VM routes; `time-to-ready-by-name.sh`):

| image | both nodes Ready (s) | median | CP Ready median |
|---|---|---|---|
| before lever 3 | 30.0 / 28.8 / 28.7 | 28.8 | 25.1 |
| lever 3, GitHub-downloaded bake | 24.0 / 25.0 / 24.8 | 24.8 | 21.6 |
| lever 3, mirror bake | 25.1 / 25.7 / 24.8 | 25.1 | 21.4 |
| lever 3, final (marker removed) | 25.8 | — | 21.0 |

Same build three ways, so the spread is noise: **median of all 7 samples 25.0s** (-3.8s). That
sits *on* the 25s line, not reliably under it (3 of 7 runs above 25.0s). Verified on the final
image: initrd-less boot, blacklist applied, wait-online 4-9ms, no IPv6, VM routes, no failed
units, cross-node DNS/ping, `kubectl top` on both nodes, `make verify` passes.

### The bake no longer downloads from GitHub

GitHub was degraded from this host on 2026-09-23 (every connect ~15s, as low as 11B/s), and
the bake VM's k3s download (Step 1) took ~19 min — past the script's 900s wait, so the script
gave up while the VM carried on. Fix: `scripts/fetch-k3s-artifacts.sh` fetches the k3s binary,
airgap tarball, checksum file and install script ONCE into `.cache/k3s/<version>/`
(gitignored; resumable, retried, checksum-verified, no-op when complete).
`bake-common.sh`'s `start_k3s_mirror` runs it, serves the cache over the Kind network gateway
(`http://172.18.0.1:18080`) for the life of the bake, and passes the URL to the VM through a
`write_files` marker; `bake.sh` downloads from it and re-checks the checksums. Falls back to
GitHub on any failure; `K3S_MIRROR=off` forces the old path. The mirror is refused if the host
`K3S_VERSION` differs from the literal `bake.sh` installs. The marker is removed in Step 10.

Measured: VM <- host, 81MB binary in 0.03s. **Full clean pipeline (fresh clone, bake,
package, pre-pull): 206s and 205s**, vs ~19 min for Step 1 alone before.

**Rebake trap:** with the `ubuntu-noble-k3s-warm` DV still present, a rebake **bakes on top
of the previous bake** (the bake ends with `cloud-init clean`, so the old disk re-runs
`bake.sh`) instead of cloning the clean base. Seen once today — a 4-minute "rebake" whose
image was discarded. The script now warns; delete the DV for a clean rebake.

Rollback tags in the local registry: `ubuntu-noble-k3s:warm-pre-lever3` (Sept 16) and
`:warm-lever3-gh` (lever 3, GitHub-downloaded bake).

---

## 11. Under 20 seconds (2026-09-30) — a CAPI rate limiter, an early k3s start, and a skewed instrument

**Result: 24.4s -> 18.1s median, both nodes Ready** (`time-to-ready-by-name.sh --runs 5`,
clean teardown, same session). Every run under 20s (17.7-18.3).

| configuration | control plane Ready (s) | both nodes Ready (s) | median |
|---|---|---|---|
| baseline (`:warm-b0`, CAPI rate limiting on) | 21.1 / 20.6 / 20.5 / 20.7 / 20.7 | 25.6 / 24.2 / 24.4 / 24.8 / 24.2 | **24.4** |
| + early k3s start (`:warm-b1`) | 19.7 / 20.1 / 19.5 / 20.1 / 20.5 | 24.2 / 23.8 / 23.4 / 23.9 / 24.1 | **23.9** |
| + `ReconcilerRateLimiting=false` | 14.4 / 13.6 / 14.1 / 14.5 / 14.3 | 18.1 / 17.7 / 18.3 / 18.1 / 18.1 | **18.1** |

### The instrument was skewed — fixed first

`time-to-ready-by-name.sh` applied the next cluster as soon as `kubectl delete cluster`
returned, while the previous run's virt-launcher pods were still terminating and holding
8Gi + 6Gi. The new control-plane pod then sat in `FailedScheduling: Insufficient memory` for
up to 12s. **Every run on 2026-09-30 before the fix was affected**, including the baseline
(events at 20:36-20:39); the 6-7s CAPI chain only partly hid it. It surfaced when the chain
shrank to 1s: that configuration first measured *28.5s* (worse), while `phase-timings.sh`,
which already waits for VMs to be gone, measured 17.6s for the same build. The script now
waits for the cluster's VMs, VMIs and launcher pods to be gone before applying (7-12s per
run, outside the clock). A real deploy onto a clean node never pays this; the warm pool's
rebuild-after-delete might, and is worth checking.

### Lever 1 — CAPI `ReconcilerRateLimiting` (-5.8s)

Per-hop controller logs showed `capi-controller-manager` acting ~2.0s after every new object
it owns (Cluster/MachineDeployment -> MachineSet 2.6s, MachineSet -> Machine 2.0s, CP Machine
-> first reconcile 2.0s), while CAPK and the k3s providers reacted within 10-60ms of an event.
Cause: CAPI v1.14's `ReconcilerRateLimiting` feature gate (beta since v1.13, **on by
default**) limits every reconciler to one reconcile per object per second
(`util/controller/controller.go`); a build re-reconciles the same Cluster and Machines
several times in quick succession. Off, the whole chain — apply to both VMs created — takes
~1s instead of 6-7s. Set in `02-capi-init/init-management-cluster.sh`
(`EXP_RECONCILER_RATE_LIMITING=false`). **Cluster state, not manifest state:** a running
cluster2 needs the `capi-controller-manager` Deployment's `--feature-gates` arg patched
(done live today) or `make capi-init` re-run. The gate exists to protect management clusters
with many workload clusters; this one runs one at a time.

### Lever 2 — start k3s at `basic.target` (-0.5s)

The CP started k3s from KThrees' `runcmd` in cloud-init's **final** stage. On this image
cloud-init's network stage (which runs `write_files`) is ordered `Before=sysinit.target`, so
by `basic.target` every CA and `config.yaml` is already on disk (measured: network stage done
4.51s, `basic.target` 4.56s, k3s started 5.61s). `bake-common.sh` Step 6d (warm only) adds
`k3s-early.service` (`After=basic.target cloud-init.service`): it makes the same edits as the
manifest's `preK3sCommands` (VM routes, drop `cluster-init`, remove `etcd-proxy.yaml`,
`advertise-address` from the first `tls-san`) and starts k3s. It does nothing on a worker
(`server:` in config.yaml) or when the CA was not written this boot. The manifests'
`advertise-address` append is now idempotent — a duplicate key would break the next k3s start.
Measured on a live CP: k3s start moved from 1.05s to 0.17s after `basic.target`; end to end
-0.5s (not the full 0.9s: the worker still joins on its own timeline).

Rejected on the way: a bolder variant starting k3s *before* cloud-init with a baked copy of
the config (~0.85s more) would have to skip `sysinit.target`, where `br_netfilter`/`overlay`
and the k3s sysctls load.

### Where the 18.1s goes now (one clean run, from apply)

```
apply -> both VMs created          ~1s    (was 6-7s)
CP: launcher -> qemu running        3s
CP: kernel -> systemd -> k3s start  ~6s    (k3s now starts at basic.target)
CP: k3s start -> Ready              ~3.5s
worker: gate open -> Ready          ~3.5-4s
```

What is left, by size: k3s server startup (~3.3-4s), the worker join after the CP API is up
(~3.5-4s), and systemd before cloud-init (~2s).

Rollback tags: `ubuntu-noble-k3s:warm-b0` (image before Step 6d), `:warm-b1` (= current `:warm`).

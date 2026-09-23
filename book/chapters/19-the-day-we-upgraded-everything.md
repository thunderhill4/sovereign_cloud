# Chapter 19 — The Day We Upgraded Everything

> *Do not assume `latest` is safe just because a past jump was.*
>
> — `CLAUDE.md`, the repository's operating notes

*Meridian · a Monday in September · the fourth-floor boardroom*

The memo from the board's risk committee was one paragraph long, and Anita Rao had underlined its last sentence twice.

*If the platform were lost, how long would it take to rebuild it, and from what?*

She slid it across the table. "I couldn't answer that. Can you?"

"From the repository," Vikram said. "In theory."

"When did we last do it?"

"End to end? Never."

Anita waited.

"There's a second thing," he said. "The tools on our laptops are already two Kubernetes versions ahead of the clusters, which is outside what Kubernetes supports. And the clusters can't be upgraded in place. To move them forward, we delete them and build them again from nothing."

"So the upgrade is the rebuild."

"The upgrade is the rebuild."

"Then it's also the answer to the committee," Anita said. "Do both. Time it. And write down everything you needed that wasn't in the repository."

"One decision is yours first. The newest Kubernetes is 1.37. The newest release of the service mesh is only tested up to 1.36. We can stop at 1.36 and stay supported, or go to 1.37 and run the mesh outside its tested range."

"What does *outside its tested range* mean, in practice?"

"It means nobody at Istio ran their test suite on it. It doesn't mean it breaks. It means that if it breaks, we're the ones who find out."

She thought about it longer than he expected. "This is the pilot. Go to 1.37. Put the decision in writing, next to the version number, where the next person will trip over it."

* * *

*Build log · 16 September 2026*

## Three things called Kubernetes

The instruction that started the day was four words long: *upgrade Kubernetes too.* The first finding was that this platform contains three different things called Kubernetes, at three different versions, with three very different upgrade costs.

- **The two host clusters**, `cluster1` and `cluster2`, ran Kubernetes **1.35.0** in Kind containers. Kind has no in-place upgrade: a new version means deleting the cluster and creating it again.
- **The target cluster** — the k3s cluster that runs as virtual machines — was pinned to k3s **v1.31.4**, six minor versions behind.
- **The web UI's backend** was built against the Kubernetes client library at **v0.36.2**.

The workstation's own `kubectl` was already at **1.37.0**, and every command printed the same warning: *"version difference between client (1.37) and server (1.35) exceeds the supported minor version skew of +/-1."* The tools had moved on without the clusters.

The upgrade was carried out by an AI coding agent working in the repository under human direction — the same arrangement as the investigations in Chapters 8 and 15. Its first act was not to upgrade anything. It asked which of the three the instruction meant. The answer was all three.

That morning, the agent platform itself had already been upgraded, from Sympozium 0.10.57 to 0.10.75 — and Chapter 15 tells what that broke. By early afternoon, the upgrade had become something the risk committee in the Meridian story would recognize: a full rebuild of the platform from its repository, the test most platforms never run until they have to.

## Two decisions that belonged to a human

The newest versions were easy to find. Kind 0.33.0 shipped node images up to Kubernetes **1.37.0**; upstream's stable release was 1.37.0; k3s **v1.37.0+k3s1** had been published two days earlier. The conflict was one layer up.

The mesh ran Istio **1.30.3**, and the repository's own README recorded its tested range: *"1.30 supports 1.32–1.36."* Before recommending anything, the agent checked whether a newer Istio would clear the way. Istio **1.31.0** was the newest release, and its announcement was unambiguous: *"Istio 1.31.0 is officially supported on Kubernetes versions 1.32 to 1.36."* No Istio release anywhere supported Kubernetes 1.37.

That made the version a decision rather than a lookup, and it went to a human, twice. The first time, the recommendation was to stop at **1.36.4** — the newest Kubernetes with a supported mesh — and the human chose 1.37.0. The second time, with the Istio finding spelled out, the options were to cap at 1.36.4 or proceed with the mesh *best-effort*. The human chose 1.37.0 again, knowingly.

That choice is now written where the next person will find it, in the mesh's README:

*"Running here unsupported/best-effort, by deliberate choice, not because it was verified compatible."*

KubeVirt and its storage importer, CDI, were left exactly where they were — KubeVirt 1.9.0 was already its latest release, and nobody had asked for either to change. Deleting two clusters is not the moment to pick up extra upgrades.

The agent stopped once more before the irreversible step, laid out what deletion would destroy and roughly how long the rebuild would take, and waited for an explicit *yes*.

## Before anything was deleted

The reversible work came first.

The k3s version was pinned in **nine** files: seven cluster manifests and templates, a template-variables file, and the golden-image bake script. All nine moved to v1.37.0+k3s1. The ninth held a surprise. `bake-common.sh` documented a `K3S_VERSION` variable as the way to choose which k3s to bake, and set it at the top of the file. But the script that actually runs inside the bake VM is written out through a *quoted* heredoc — `cat << 'CLOUDINIT'` — which copies its contents verbatim and expands no variables at all. Inside it sat a second, hard-coded `K3S_VERSION`. The documented knob had never controlled anything. Anyone who had used it would have baked the old version and been told they had baked the new one.

The client library moved to v0.37.0; the backend built, passed `go vet`, and passed its tests. The Istio version default moved to 1.31.0. The `kind` binary on the workstation moved to 0.33.0.

Then came the inventory that matters most in this chapter: what exists in the running clusters that the repository does not create. It was checked against the scripts, the Makefile, and the README, and the answer was uncomfortable.

- **The Kind clusters themselves.** No configuration file, no `kind create` command anywhere in the repository. They had been created by hand, with defaults.
- **KubeVirt 1.9.0** and its configuration — the operator, the custom resource, and the tuned CPU limits from Chapter 8.
- **CDI 1.66.0**, the component that imports disk images into the cluster.
- **The base Ubuntu image**, a DataVolume called `ubuntu-noble-dv`, which every golden-image bake starts from.

Each one's specification was read out of the live cluster and saved before anything was deleted. Without that step, the rebuild would have begun by reverse-engineering the platform from memory.

One item on that list was wrong, and the error is part of the story. It comes back at the end of the chapter.

## Rebuilding from nothing

The destructive part took two commands.

```
kind delete cluster --name cluster1
kind delete cluster --name cluster2
```

The local container registry survived, because it is a separate container rather than part of either cluster, and it was still attached to the Kind network when the new clusters came up. Both clusters were recreated from the `kindest/node:v1.37.0` image and reported Ready at v1.37.0 within minutes. KubeVirt and CDI went back on at their previous versions and reported `Deployed`. MetalLB went back on `cluster2`. Then Cluster API was initialized, which is where the day stopped being mechanical.

### The dead image that reported success

Cluster API's initialization finished and reported success. The first attempt to create the target cluster failed immediately:

```
failed calling webhook "default.kthreescontrolplane.controlplane.cluster.x-k8s.io":
  ... connect: connection refused
```

The k3s provider's admission webhook was not running. Its pod had been stuck for eighteen minutes with one of its two containers failing to start, on an image that no longer exists:

```
Failed to pull image "gcr.io/kubebuilder/kube-rbac-proxy:v0.16.0": ... not found
```

The provider's release manifest — still version 0.3.0, unchanged — named a proxy image at an address that no longer serves it. And the setup script had not noticed, because every readiness wait in it ends in `|| true`: wait five minutes, fail, fall back to a second five-minute wait, fail again, and carry on as though nothing had happened.

The clue to the fix was in the cluster that had just been deleted. Before deletion, its providers had been running a different image, `quay.io/brancz/kube-rbac-proxy:v0.16.0`, which still pulls. Nothing in the repository, the Cluster API configuration, or any script recorded that change. It had been made once, in the running cluster, and it existed nowhere else. It was re-applied to both provider deployments, the target cluster was created, and the fix now lives in `02-capi-init/init-management-cluster.sh`, which patches the dead image right after initialization.

### The setting that went missing

Chapter 8 ended with a warning: the CPU ceiling worth eight seconds on every VM start lives in the running cluster's KubeVirt configuration, not in Git. The script that applies it is called at the very end of the Cluster API setup script, so that a rebuild restores it automatically.

After the rebuild, it was not there. It was applied by hand before any VM was built, which is why the measurements later in this chapter include it.

The reason it was missing is worth telling precisely, because the first explanation written down was wrong. The setup script's output stopped at a single line — `Waiting for k3s bootstrap controller...` — and nothing after it ran. The agent had launched the script under a 590-second time limit. The script's readiness waits for that provider, stuck on the dead proxy image, could take up to 600 seconds on their own. The time limit killed the script partway through its waiting, before it ever reached the step that restores the CPU ceiling.

And the run still reported success. The command's output had been piped through `tail` to shorten it, so the exit code that came back belonged to `tail`, which had succeeded, and not to the script, which had been killed. The same trap caught a failed installation later the same day.

The agent noticed the missing setting and restored it. Then, in the repository's notes, it recorded the cause as a race with KubeVirt's startup. The log does not support that; the evidence points to the time limit instead. The correction is in this chapter's draft notes, and it belongs in `CLAUDE.md` before anyone relies on the wrong one.

### The image that still had the old version inside

The golden images were next, and here the surviving registry turned from an asset into a trap. It still held all three of the platform's k3s images — `:warm`, `:preinit`, and `:latest` — and all three had k3s **v1.31.4** baked inside them.

The Makefile's `ensure-warm-image` step checks whether a `:warm` image is *present*. It does not check what is inside. Left alone, it would have found the old image, reported it present, and booted every new cluster on the version the upgrade was meant to replace.

The `:warm` image — the default path — was rebaked from scratch: a bake VM ran for 160 seconds and downloaded k3s v1.37.0; the result was packaged as a 1.35 GB container image and pushed over the old tag. The `:preinit` and `:latest` images, used only by older deployment paths, were **not** rebaked, and still carry v1.31.4.

### The status that disagreed with the nodes

The target cluster came up, and its control-plane object promptly reported a problem: `ControlPlaneComponentsUnhealthy`, *"Following machines are reporting control plane errors,"* with its ready flag never set.

The machines themselves reported every condition true. The cluster's own API, reached through its kubeconfig, showed both nodes `Ready` on `v1.37.0+k3s1`, Ubuntu 24.04.5, containerd 2.3.4. The repository's verification script passed. The status field was stale, and a wait loop polling it was stopped rather than left to time out. It is the smaller cousin of Chapter 15's lesson: a status field is a claim too.

## Twenty-two green checks

The mesh was the part running outside its tested range, so it was the part that most needed evidence. The repository's mesh demonstration is five acts, and each act ends in assertions rather than screenshots.

Istio 1.31.0 installed cleanly on both clusters, and both control planes adopted the same root certificate — identical fingerprints, the precondition for joining them into one mesh. Then the five acts ran, on Kubernetes 1.37:

- **Gateway API.** HTTPS served with a locally rooted certificate; a 90/10 canary split measured 93/7; header-based routing sent 10 of 10 internal requests to v2; request mirroring and the HTTP-to-HTTPS redirect worked; and a rogue tenant's attempt to claim the hostname was refused with `NotAllowedByListeners`.
- **Waypoints.** Identity-based authorization: the trusted identity got `200`, the untrusted one `403`; a `POST` was denied while a `GET` was allowed; swapping a workload's identity revoked its access with no policy change; the client's identity arrived in a forwarded-certificate header; an in-mesh traffic split measured 24/26.
- **Multicluster.** Each cluster's control plane synced with its peer. Then every local replica of a service was scaled to zero, and calls kept being served from the other cluster, with no configuration change. (The act's own tally, `success=20 failed=0`, proves less than it appears to — it prints the same line when both clusters are down; see Chapter 11. Re-measured by HTTP status on 17 September: 30 of 30 calls returned `200`, every one served by cluster2.)
- **AI gateway.** Model traffic routed through the mesh to the host's Ollama; a completion came back `"ok"`.
- **Observability.** Prometheus held the mesh's request metrics.

The verification script's final table, once the agent platform was reinstalled: **22 passed, 0 failed, 0 skipped.**

It is worth being exact about what that proves. *Supported* means the Istio project ran its test suite on this combination. *Passing* means this repository's twenty-two assertions held. Those are different sets of checks, and the gap between them is not empty.

Act 1's output contained one line that no assertion examines. The act lets a tenant contribute its own listener to the shared gateway through a `ListenerSet`, prints the result, and moves on. The result was:

```
Accepted=False (ListenersNotValid)
Programmed=False (ListenersNotValid)
```

— *"None of the ListenerSet's listeners are valid."* Yet the gateway's Service exposes that tenant listener's port, `8443`, exactly as it did before the rebuild. The status says the listener is invalid; the deployed Service says it exists. Whether traffic actually flows on port 8443 was never tested, and nobody kept the pre-upgrade status to compare. It may be a regression, a long-standing quirk, or a status bug. It is a question, not a finding — and it would have stayed invisible behind a clean 22 of 22.

## Five known traps, one attempt each

Reinstalling the agent platform on the new `cluster2` produced five errors in a row. None needed an investigation, because every one was already written down in the repository's notes, left there by earlier upgrades.

1. **The admission race.** The first install failed with `failed calling webhook "vskillpack.sympozium.ai" … connection refused`: the chart applies its built-in skill packs before its own validating webhook is serving. Documented. Wait for the webhook, retry.
2. **The wait that never ends.** The retry, run with Helm's `--wait`, timed out on an example `postgres` tool server that is deliberately left suspended and never becomes ready. Documented, with the instruction to drop `--wait` and verify by hand. (This was the second failure of the day to report success through a pipe to `tail`.)
3. **The field-ownership conflict.** Without `--wait`, the install stopped on a conflict over `.spec.sidecar.mountWorkspace`, a field the platform's own controller rewrites after install. Documented, with the exact flags: `--server-side=true --force-conflicts`. The next attempt deployed.
4. **Runs that never retry.** Recreating the serving agents left two of their runs in a terminal `Failed` state — *"server Deployment not found"* — because they had been recreated at the same moment their deployments were being replaced. Documented: the controller does not retry a failed run; delete it and let it be regenerated.
5. **The silent crash loop.** One serving pod crash-looped with no log output. Documented, with three distinct past causes and the same first move for all of them: regenerate the run. It came up healthy.

The per-run service-account fix from Chapter 15 had already been written into the manifests that morning, so the reinstall applied the corrected version rather than rediscovering the problem.

Five errors, five single attempts. The difference between a known failure and an unknown one is the difference between a minute and an afternoon, and the only thing that separated them was a text file.

## No faster than before

With everything rebuilt, the target cluster's time-to-ready was measured with the same instrument as Chapter 8: three clean runs, host load recorded before each. It reported **23.9, 24.1, and 23.7 seconds** — ten seconds faster than the 34.5 measured a week earlier, and steadier.

That result was wrong, and finding out why is the second half of Chapter 7. In short: the warm golden image carries a ghost Node left over from the machine it was baked on, still marked `Ready`. For the first seconds of every new cluster, the ghost and the control plane together satisfy any stopwatch that waits for "two nodes Ready" — so the instrument stopped its clock before the worker had joined, and reported the control plane's time as the total.

Measured again the next day, naming each node rather than counting them, three clean runs:

| | Run 1 | Run 2 | Run 3 | Median |
|---|---:|---:|---:|---:|
| Control plane Ready | 23.4 s | 25.1 s | 24.7 s | 24.7 s |
| Worker Ready — both nodes | 30.4 s | 35.1 s | 33.7 s | **33.7 s** |

**33.7 seconds**, against 34.5 before the upgrade. Within the noise of three runs, the upgrade changed nothing. Six minor versions of k3s and two of Kubernetes, and a cluster that builds in the same time — which is a perfectly good result, as long as nobody claims it is a better one.

Tearing the cluster down was measured separately, once: the delete command returned after **10.15 seconds** while Cluster API worked through its finalizers, and every VM and machine object was gone at **18.9 seconds**. That was a single run, and it did not survive being measured again: five runs in Chapter 17 put it at **45.73 seconds**, with the machines themselves stopping at 35.6 s. The figure below is left as it was recorded on the day.

## The balance sheet of memory

Here is the day's accounting in the terms the risk committee asked for.

**Rebuilt from the repository:** both host clusters, the golden image, the target cluster, the service mesh with all five acts, and the agent platform with its three agents. **Recovered from the old cluster, because nothing else held it:** the Kind cluster setup, the KubeVirt and CDI installation, and a working proxy image for the k3s providers. **Found wrong in the repository:** a version variable that controlled nothing, an image check that tested presence rather than contents, and a service-account binding that had quietly stopped applying. **Already known, and therefore cheap:** five errors that cost one attempt each.

And one item from the inventory was simply wrong. The base image was listed as something no script creates, and the repository's notes were updated to say so. While this chapter was being researched, the script turned up: `scripts/import-base-images.sh`, which imports the base image *and* a second, minimal Ubuntu image that the rebuild never recreated. It exists. It is mentioned in exactly one document — a post-mortem about something else. Nothing on the setup path — not the Makefile, not the README, not the operating notes — points to it. A script nobody can find is not institutional memory. It is an attic.

That is the real lesson of the day, and it is the one this book's opening chapters were building toward. Sovereignty is usually argued as a question of *where*: whose data center, whose jurisdiction, whose model. But owning a platform also means being able to rebuild it without asking anyone. By that measure, everything that lives only in a running cluster — a hand-applied image patch, a tuned CPU limit, a cluster created with defaults — is not owned. It is on loan from the cluster's continued existence.

The notes paid for themselves in minutes, on a single afternoon. Written knowledge compounds, and unwritten knowledge is deleted along with the cluster that held it.

* * *

*Meridian · the following Monday*

Vikram brought one page to answer the committee's memo.

*Rebuilt from the repository, on current versions, in one afternoon. Four things were not in the repository. One is now automated. The other three are written down but not yet scripted — including a setup script that existed all along and that nobody could find. One component runs outside its vendor's tested range, by decision, recorded next to the version. Cluster build time: 33.7 seconds for both nodes, unchanged by the upgrade — after our own stopwatch briefly said otherwise.*

Anita read it and went back to one line. "One afternoon. Because we'd never done it."

"The first time is the expensive one."

"Then it doesn't get to be the only one." She wrote on the memo before handing it back: *Rebuild from nothing — quarterly.*

"A platform we have never rebuilt," she said, "is a platform we only think we own."

## The ledger

- **Upgraded:** Kubernetes 1.35.0 → 1.37.0 on both host clusters; k3s v1.31.4 → v1.37.0 in the target cluster; Istio 1.30.3 → 1.31.0; Sympozium 0.10.57 → 0.10.75; `kind` 0.31.0 → 0.33.0; the client library 0.36.2 → 0.37.0.
- **Upgraded without anyone choosing it:** Cluster API 1.14.0 → 1.14.2, installed at whatever version `clusterctl` fetched that day; and cert-manager on `cluster2` at 1.21.1, brought in by `clusterctl`, while the agent platform's install script pins 1.16.2 — it found cert-manager already present and skipped its own.
- **Rebuilt and verified:** both host clusters, the golden image, the target cluster, all five mesh acts (22 of 22 checks), and the agent platform.
- **Measured:** bring-up 33.7 s median for both nodes (control plane alone, 24.7 s) — no measurable change from 34.5 s; teardown 18.9 s — superseded by Chapter 17's 45.73 s across five runs. An earlier reading of 23.9 s was the control plane only (Chapter 7).
- **Written back into the repository:** the dead-image patch, the bake-script warning, the unsupported-mesh decision, and an upgrade entry in the operating notes.
- **Still owed:** the Kind clusters, KubeVirt, and CDI documented as prerequisites but still not scripted; a mesh outside its tested range; a ListenerSet whose status and Service disagree; `:latest` and `:preinit` images still carrying k3s v1.31.4; a minimal base image never re-imported; Kind node images pinned by tag rather than digest; timing scripts that count a ghost node as a worker; and a recorded cause for the missing CPU setting that is wrong.

## Ask your team

1. **If we lost the platform tonight, could we rebuild it from what is written down — and when did we last prove it?** Not believe it. Prove it.
2. **What in our running systems exists nowhere else?** Compare what is deployed with what the repository would deploy, and treat every difference as a liability.
3. **Where are we running outside a vendor's supported range, who decided that, and where is the decision written?** An unrecorded exception looks exactly like an accident.

## Open the repo

- `CLAUDE.md` — the entry *"Kubernetes/Istio/k3s upgraded 2026-09-16"*, with the rebuild's findings.
- `02-capi-init/init-management-cluster.sh` — the dead-image patch, and the comment explaining it.
- `bake-common.sh` — the header comment on the `K3S_VERSION` variable that controlled nothing.
- `07-istio-advanced/README.md` — the note recording the unsupported combination; `make istio-adv-verify` reruns the 22 checks.
- `./scripts/phase-timings.sh --runs 3` — the bring-up measurement protocol.
- `scripts/import-base-images.sh` — the script the rebuild could not find.
- **Without the repo:** Appendix G.7 quotes both `K3S_VERSION` assignments and the quoted heredoc between them; G.8 quotes the dead-image patch.

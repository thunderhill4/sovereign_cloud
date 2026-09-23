# Chapter 9 — Warmth Is Mandatory

> *Warmth is mandatory. No persona may enter the fleet without a warm-path guarantee.*
>
> — Architecture Constraint 5, `Sovereign_Cloud_Agentic_Strategy.md`

*Meridian · a Thursday in September · the CTO's office*

The prospect had asked the AI console a simple question, and for thirty seconds the answer did not come. Then the request timed out, on the big screen, in front of their head of infrastructure. Vikram asked again. The second answer arrived in four seconds and was correct.

"Why was the first one slow and the second one fast?" Anita asked afterwards.

"The first question had to wake the model up. It loads into memory on first use. After that it stays loaded, and everything is quick."

"Like the clusters," she said. "Build one from nothing, it takes half a minute. Hand over one that's already running, it's instant."

"Same idea."

"Then it has the same economics." She pulled a pad toward her. "Keeping things warm costs hardware that sits idle. Being cold costs moments like that one. I want both prices, for both things, on one page. And I want to know who is keeping them warm right now."

* * *

*Build log · June–September 2026*

## Two kinds of cold

This platform has two things that are slow to start and fast once running, and the whole of this chapter is about the difference.

The first is a **cluster**. Chapter 7 measured a cold build — from `kubectl apply` to both nodes `Ready` — at 33.7 seconds after the September upgrade, and Chapter 8 tells how it came down from fifty. Every one of those seconds is intrinsic to building a cluster from nothing: controllers creating virtual machines, the machines booting, k3s starting, a worker joining. Chapter 8's lesson was that a smarter golden image cannot remove them. Only one thing can: doing the work *before anyone asks*.

The second is a **model**. An agent's first request loads its language model into memory. The strategy document wrote the cost into its fifth architecture constraint, as measured when the platform's model ran on the CPU: *"Cold-start for llama3.2 on CPU is ~28–30s; unwarmed agents time out the first real request."*

Different machinery, same trade: capacity held in reserve against time lost to a cold start.

## A standby, not a pool

The design for warm clusters is in `docs/warm-pool-strategy.md`, and it begins by rejecting the obvious answer. A classic pool keeps several pre-built machines and hands them out. This platform runs exactly one target cluster at a time, so the document asks for something smaller: a *size-1 pre-deployed standby*. Keep one cluster built and idle. "Deploy" means *claim* it. "Delete" means tear it down and build the next standby in the background, off the user's clock.

The lifecycle is a label on the cluster object, `pool.local/state`:

```
   (none) ──build in background──▶ WARM ──user clicks Deploy──▶ CLAIMED
      ▲                                                             │
      └──────────── rebuild in background ◀── user clicks Delete ◀──┘
```

One constraint makes it tractable, and it is easy to miss. A KubeVirt VM's pod address is assigned when its launcher pod starts, and the document records that a changing address was what broke the datastore in the warm-image experiment. *"A pooled VM that stays alive keeps its pod IP"* — so its datastore identity and its certificates stay valid. That is also why the document rejects the other obvious answer, snapshot and restore: a restore *"re-provisions the PVC and **reboots**, so you pay the 17s boot again and get a new IP."*

A second level is described and deliberately deferred: a pool of *paused* VMs, frozen by KubeVirt with their memory and addresses intact, resumable in under a second. It would allow more than one warm cluster, at the price of a real controller and, in the document's words, *"ongoing maintenance."*

## One hundred ninety-four milliseconds

*Build log · June 2026*

The standby was built on 27 June as a background loop inside the UI's Go backend. Every few seconds it looks for a target cluster. If none exists, it builds one through the same Cluster API path as any other deploy and labels it `WARM` once it is ready. When a user clicks Deploy and a warm standby exists, the handler relabels it `CLAIMED`, refreshes the credentials the agents use to reach it, and streams a short progress report.

Measured end to end, with a warm standby in place: **194 to 195 milliseconds** from the click to done, with both nodes already `Ready`. The background rebuild after a delete took about 60 seconds, and the cycle — claim, delete, rebuild, claim — repeated cleanly. The acceptance gate had been under forty seconds; the claim beat it by a factor of about two hundred.

One bug is worth recording, because it is Chapter 7's lesson in miniature. The first version reset the progress log from a background thread while the user's browser was already subscribing to it. The browser received the *previous* build's log and an immediate "done": a fast, confident, false success. Only an end-to-end test caught it.

Then came the price. The standby uses the full profile: four cores and 8 GiB of memory for the control plane, four cores and 6 GiB for the worker. That is **eight cores and fourteen gigabytes held idle, all day, on purpose**, on a host that had about 21 GiB free. The document says it plainly: *"Idle resource cost. One cluster always running."*

Within a week, that cost produced a decision. The pool had started out enabled, which meant a standby cluster was built every time the UI started and rebuilt after every cleanup — whether or not anyone intended to deploy. That surprised its own user. On 1 July it became opt-in, and the start script now says so: *"Opt-in: export POOL_ENABLED=true before running this script to have the backend auto-build/rebuild a standby cluster in the background."* Warmth that nobody asked for is not a feature. It is a machine working when no one is watching.

## The standby today

*Build log · 17 September 2026*

Checked against the running platform on 17 September, the fastest path in the platform turns out to be the least exercised one.

The UI backend reports no standby: `{"state":"none"}`. The pool is opt-in, and the script that starts the UI leaves it off by default. It also lives only as long as that backend process runs, so the command-line deploy path has never benefited from it.

Its standby manifest still points at the image tagged `ubuntu-noble-k3s:latest`, and that image was built on **3 March**. It was not rebuilt during the September upgrade in Chapter 19. Turning the pool on today would build its standby from a six-month-old image, under a manifest that now declares a much newer k3s. Nobody has tried it.

Its readiness check counts *two Ready nodes*, the same pattern that fooled the stopwatch in Chapter 7. A standby built from the warm image would be declared ready while its worker was still booting, because that image carries the ghost node. Whether the March image carries anything similar has not been checked.

And the 194-millisecond claim has not been measured since June.

None of this means the design is wrong. It means a warm path is only as good as the last time someone used it. The image aged, the platform moved underneath it, and a number from June stayed on the page.

## Warming the model

*Build log · April–September 2026*

The model's warmth has a history of its own, told in three tools.

The first was a script, `06-sympozium/ollama-warm.sh`: send a one-token request and ask the model server to keep the model loaded for twenty-four hours. The second was the Phase 0 deliverable from Chapter 16: a heartbeat schedule that pings `cluster2-agent` every four minutes. Its comment explains why: *"Cold-start on CPU is ~28-30s; this schedule removes that latency from the first real KubeUI chat request."* Chapter 14 found that schedule had fired twice and stopped. The third tool is the one actually doing the job: the host's Ollama service runs with `OLLAMA_KEEP_ALIVE=-1`, which keeps a loaded model loaded indefinitely. That setting lives in the host's service configuration, not in the repository.

## The GPU changed the price

*Build log · 17 September 2026*

The constraint's thirty seconds was a CPU number. The host now has an NVIDIA RTX 4050 laptop GPU with 6 GB of memory, and Ollama places each of the fleet's models entirely on it. So the cold start was measured again on 17 September, using the load time Ollama reports for each request — first with a model unloaded, then with it loaded:

| Model | Cold load | Warm load |
|---|---:|---:|
| `qwen2.5:7b` | 2.57 s, and 2.38 s on a second cold start | 0.14 s, 0.17 s |
| `llama3.2` | 2.33 s | 0.17 s |

**About two and a half seconds cold, under two-tenths of a second warm.** The difference is still roughly fifteen-fold, and still visible in an interactive chat. But a two-and-a-half-second load no longer times out a request by itself. The hardware changed under the rule, and the rule's number went stale.

The same measurement showed something the rule never anticipated. **The GPU holds one of these models at a time.** Every cold load evicted the other model. `OLLAMA_KEEP_ALIVE=-1` does not mean *forever*. It means *until something else needs the room*.

That matters because the fleet uses both. `cluster2-agent` and `mesh-sre-agent` run on `qwen2.5:7b`; `target-cluster-agent` runs on `llama3.2`. And it had already happened that morning, unnoticed. At 00:55 UTC, `qwen2.5:7b` was the loaded model. At 01:06, the two-agent pipeline lab from Chapter 16 ran its reviewer on `llama3.2`. By the time anyone checked, `llama3.2` was loaded and `qwen2.5:7b` — the model two of the three agents depend on — was not. A test run had quietly cooled the production agents. (It was restored for this chapter.)

Warmth on shared hardware is not a property of a model. It is a queue.

## Two inventories, both one deep

Put the two kinds of warmth side by side and they turn out to have the same shape:

| | A cluster | A model |
|---|---|---|
| Cold | 33.7 s, both nodes | 2.3–2.6 s on the GPU (28–30 s on CPU) |
| Warm | 467 ms median (18 Sept, 3 runs; was 194 ms in June) | 0.14–0.17 s |
| What warmth holds | 8 cores and 14 GiB of memory, idle | most of a 6 GB GPU |
| How deep | one standby | one model at a time |
| What keeps it warm today | a background loop that is off by default | a host setting outside the repository |

For a CxO, warmth is inventory. It trades capacity you pay for up front against time you lose later, and it has the same three failure modes as any inventory.

**It can be the wrong size.** One standby covers the first request in a minute, not the second. One model slot covers one model, not a fleet that uses two.

**It can spoil.** A standby built from a March image, and a heartbeat that stopped in September, look like warmth on paper and are not.

**It can be taken by someone else.** A test run that loads a different model empties the shelf the production agents were counting on.

* * *

*Meridian · the following Monday*

Vikram's page had two columns, one for clusters and one for models, and a price at the bottom of each.

Anita read it twice. "The model costs two and a half seconds cold now, not thirty."

"On this GPU. And only if it's the model that was already loaded. The GPU holds one at a time, and our agents use two."

"Then we use one," she said. "The one that uses its tools properly. One model for the whole fleet, until the hardware says otherwise." She moved to the other column. "And the cluster standby?"

"Built from a March image, and switched off."

"Then it isn't warm. It's a cold path with a design document." She wrote across the bottom of the page. "Rebuild the image. Turn it on. And every week, someone claims it, deletes it, and times the rebuild — naming both nodes."

"Every week?"

"Anything we keep warm, we exercise," Anita said. "Otherwise we are paying for idle hardware to hold a promise nobody has checked."

## Turned on, and timed

*Build log · 18 September 2026*

The mandate was carried out the next day, and it found the same ghost that Chapter 7 found in the timing scripts, waiting in the same shape.

`targetNodesReady`, the function both the on-demand deploy path and the pool controller use to decide "are both nodes up," counted Ready nodes without checking their names. A standby built from the `:warm` image would have been declared ready on the control plane plus the leftover bake-VM ghost node, before the real worker ever joined — exactly the bug that made Chapter 7's stopwatch stop early, now confirmed live in `ui/backend/handlers/pool.go`, not just in the standalone scripts. Fixed the same way Chapter 7 recommends: count only nodes named `<cluster>-cp-*` or `<cluster>-workers-*`.

With that fixed, `POOL_STANDBY_MANIFEST` moved from `target-cluster-parallel.yaml` (the March image, k3s v1.31.4) to the warm manifest — and the pool's build function gained the one step it was missing to use it safely: seeding the fixed CA and token secrets before applying, the same order the on-demand deploy path already used, so KThrees adopts the baked material instead of falling back to a slower path.

Then the cycle Anita asked for — claim, delete, rebuild, claim — ran three times, timed from the `POST /api/v1/cluster/deploy` request to the stream's closing `done` event:

| Run | Time |
|---|---:|
| 1 | 1.059 s |
| 2 | 0.467 s |
| 3 | 0.465 s |

**Median: 467 milliseconds.** Not 194. The first run is reported rather than discarded — an unexplained outlier, kept on the same principle this book applies to every other number in it. Runs 2 and 3 agree closely, and the time is not free: of the roughly 465 milliseconds, about 265 is `clusterctl get kubeconfig` alone; relabeling the cluster and syncing the in-cluster kubeconfig Secret split most of the rest.

Two and a half times the June figure, and still far faster than building from nothing: the claim now beats the 33.7-second cold build by a factor of about 72 — not the factor of two hundred the June number implied, but still the same conclusion Anita drew from the wrong number: handing over something already running beats building it, by nearly two orders of magnitude. The design held. The number on the page had simply gone stale, in exactly the way this chapter's own middle section warned that warmth does.

## The ledger

- **Built (June):** a one-deep cluster standby, claimed in 194–195 ms instead of built from scratch, rebuilt in about 60 seconds in the background.
- **Cost of that warmth:** one full cluster idle — eight cores and 14 GiB of memory.
- **Today (17 September):** no standby running; the pool is opt-in and off by default; its manifest points at an image built on 3 March; the claim time has not been re-measured since June.
- **Fixed and re-measured (18 September):** the pool's node-readiness check now counts by name, not by count (the same ghost-node bug Chapter 7 found, confirmed live in `pool.go`); its standby manifest switched to the warm image, with CA/token seeding added to the build path. Three claim-delete-rebuild-claim cycles: 1.059 s, 0.467 s, 0.465 s — **median 467 ms**, not 194. Still about 72x faster than a cold build. The pool remains opt-in — this fixes what happens when it's on, not the default.
- **Model warmth, measured on the GPU (17 September):** 2.3–2.6 s cold, 0.14–0.17 s warm — against the 28–30 s CPU figure in the rule.
- **Found:** the GPU holds one model at a time, and a lab run evicted the model two of the three agents use.
- **Held by:** a host setting outside the repository, not the heartbeat the rule names, which stopped after two runs.

## Ask your team

1. **What do we keep warm, what does that capacity cost us, and what would being cold cost instead?**
2. **When did we last actually use our fast path — not measure it once, but use it?**
3. **What shared resource do our warm things compete for, and whose work gets evicted first?**

## Open the repo

- `docs/warm-pool-strategy.md` — the three design levels, and the stable-IP constraint.
- `ui/backend/handlers/pool.go` — the standby controller; `POOL_ENABLED`, `POOL_STANDBY_MANIFEST`, and `GET /api/v1/cluster/pool-status`.
- `docs/superpowers/specs/2026-06-26-warm-pool-standby-design.md` — the acceptance gate.
- `06-sympozium/ollama-warm.sh` and `06-sympozium/schedules/ollama-warm.yaml` — the first two ways the model was kept warm.
- `curl -s localhost:11434/api/ps` on the host — which model is warm right now; the `load_duration` field of any `/api/generate` response shows whether a request paid for a cold start.
- `POOL_ENABLED=true POOL_STANDBY_MANIFEST=03-target-cluster/target-cluster-warm.yaml` — the corrected pool config (18 September); `git log --oneline -- ui/backend/handlers/pool.go run-ui.sh` for the fix commits.

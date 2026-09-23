# Chapter 7 — Thirty-Six Percent Before the Kernel

> *`scripts/time-to-ready.sh` prints ONE aggregate number, which is why two optimisation dead ends … each cost a full bake cycle to disprove.*
>
> — the header of `scripts/phase-timings.sh`

*Meridian · a Monday in September · the platform team's room*

The number on the wall display was `48.5 s`, in type large enough to read from the door. Anita Rao stood under it for a while before anyone noticed she was there.

"That's the cluster build?" she asked.

"Apply to ready," Vikram said. "Every time we change something, we run it and watch that number."

"And when it doesn't move?"

"Then the change didn't work."

"Or it worked, and something else got slower." She turned to him. "If finance sent me our cloud bill as one number — no line items, no accounts, just a total — what would I do with it?"

"Send it back."

"So why is this one allowed to be a single number?"

Vikram looked at the display. "Because nobody has built the itemized version."

"Then build that first," Anita said. "Before anyone tries to make it smaller."

* * *

*Build log · September 2026*

## One number

For most of 2026 the platform's cluster-build time was measured by `scripts/time-to-ready.sh`: seventy lines of shell that note the time, apply the cluster manifest, poll until two nodes report `Ready`, and print one line.

```
RESULT: 48.520s (2 nodes Ready)
```

It is a perfectly reasonable script, and it shaped a summer of decisions. On its word, two golden-image experiments were built, baked, measured, and abandoned — the dead ends Chapter 8 tells in full. Each took a full bake cycle to disprove, because a single total cannot say *which part* of a cluster build a change affected, only whether the whole moved. It also could not separate a real improvement from a quiet afternoon: an early run of the warm image recorded 35 seconds that turned out to be *"quiet-host luck rather than a real improvement."*

In September, before anyone was allowed to optimize anything, the brief described in Chapter 8 made one step mandatory: *"Step 1 (mandatory, before any optimisation): instrument the phases."* It listed the phases it wanted — VM creation to running, kernel start to cloud-init finished, k3s start to the API's first healthy answer, worker join, each node Ready — and it distrusted its own starting estimate: *"The 17/17/18 split above is a coarse estimate — confirm or correct it before acting on it."*

## A cluster you cannot log into

The brief also suggested how to see inside the virtual machines: log in over SSH and run `systemd-analyze`, since *"SSH key and user `ubuntu` are baked in."*

That was not possible. On this cluster, the KubeVirt API service that `virtctl console` and `virtctl ssh` depend on reported `Available=False`. The obvious way into the guests was closed.

So the instrument was built from what the platform already writes down, without logging in to anything. Its header lists every source:

- **Creation timestamps** on the Cluster API and KubeVirt objects — how long the controllers took to turn a manifest into a virtual machine.
- **Phase transition timestamps** on each VM instance — scheduled, running.
- **Container start times** inside each VM's launcher pod — the disk container, the console container, the one that runs QEMU.
- **The guest's serial console**, which KubeVirt already copies into a pod log — kernel, initramfs, systemd, cloud-init, and k3s milestones, as the guest printed them.
- **Each node's `Ready` transition** — the metric itself.
- **A probe on the host** that tries the control plane's virtual IP every quarter-second, recording the first TCP connection and the first healthy `/readyz` answer.

The result, `scripts/phase-timings.sh`, is read-only and needs nothing installed inside the guest. It also has a `--collect-only` mode that changes nothing at all and reconstructs the table for whatever VMs are already running.

## Anchoring a clock that starts at zero

One problem had an elegant answer, and it is worth a moment.

A guest kernel does not timestamp its log lines with the time of day. It uses seconds since boot — `[   15.738316]` — so a serial console full of milestones says how long each took after the machine started, but not *when* the machine started. Without that, the guest's timeline cannot be placed on the same clock as the Kubernetes objects around it.

The answer was already in the log. Cloud-init prints lines that carry *both* clocks at once:

```
running 'init' at Wed, 09 Sep 2026 06:44:19 +0000. Up 6.81 seconds.
```

Subtract the uptime from the wall-clock time, and you have the moment the kernel started, to the precision of the log. Every other console timestamp can then be placed on the same timeline as the objects, the pods, and the host probe. No clock synchronization, no agent, no login.

## Small lies in the plumbing

Most of the instrument's 320 lines are not measurement. They are defenses against its own data, and each one is recorded in a comment because each one produced a wrong answer first.

- **The login prompt interleaves with the kernel log.** A console line can read `target-cluster-cp login: [   15.738316] cloud-init[976]: ...`. A search anchored to the start of the line *"silently drops exactly the late lines we care about."* The fix was to search anywhere in the line, and to require a decimal timestamp so that a process ID like `cloud-init[976]` is not mistaken for one.
- **Two clocks disagree by a second.** VM phase timestamps and container start times are both rounded to the second, and are written by different components. Mixing them *"yields negative durations."* The launcher pod's own creation time became the anchor for that phase instead — and even so, a one-second rounding can still produce a small negative number in a table.
- **The disk container is gone by the time you look.** It runs once and exits, so its start time lives in a different status field from the console container's, which keeps running.
- **A background probe can freeze its own caller.** The host probe is started in the background and its process ID is read back through command substitution, which waits until every process holding the output pipe has exited. Without redirecting the probe's output, the script *"deadlocks the caller before it can even apply the manifest."*

None of this is glamorous, and all of it is the job. An instrument is mostly the code that stops it from lying.

## The first reading

On 9 September the instrument ran three clean builds of the warm path, re-measuring the baseline in the same session as the brief required. The baseline was not 48.5 seconds but 50.0 — runs of 52.7, 50.0, and 46.6. Here is where those seconds went, median of three:

| Phase | Seconds | Share |
|---|---:|---:|
| `kubectl apply` → VM instance created (Cluster API controllers) | 6.6 | 14% |
| Launcher pod created → QEMU running | 10.0 | 21% |
| Firmware, kernel, initramfs | 3.7 | 8% |
| systemd and cloud-init → k3s start | 5.5 | 12% |
| Control-plane k3s start → first `200` from `/readyz` | 7.0 | 15% |
| Worker joins, after the API is already answering | 11.8 | 25% |
| **Median time-to-ready** | **50.0** | 52.7 / 50.0 / 46.6 |

Two findings came out of the first reading, and together they redirected the whole effort.

The coarse estimate had given the control plane's k3s startup seventeen seconds. The instrument measured seven. Both image experiments had been built on the theory that k3s startup was the slow part — and that theory had bet on a phase worth fifteen percent of the total.

And the top two rows: **thirty-six percent of the wall clock — 16.6 seconds — elapsed before the guest kernel printed its first character.** No change to the golden image could touch any of it, because none of it happens inside the image. Of those seconds, the table labelled eight as *"containerDisk unpack"* — a label that Chapter 8 shows was a guess, and wrong.

What the team did with that table is the next chapter. This chapter has one more thing to say about the instrument itself, because it was discovered while this chapter was being written.

* * *

*Build log · 17 September 2026*

## The stopwatch that stopped early

After the platform upgrade in Chapter 19, the same instrument reported a cluster build of **23.9 seconds** — 23.9, 24.1, and 23.7 across three runs, a spread of only 0.4 seconds. Ten seconds faster than before the upgrade, and steadier. The number went into the platform's operating notes, into two chapters of this book, and into an answer about how the platform got under forty seconds.

It was wrong, and the instrument's own output said so to anyone who looked closely.

In all three tables, and in a fourth run later that day, one row was empty:

```
  [WORKER]
  ...
  E node Ready                                        —      —
```

And the total at the bottom of each table — labelled *"TOTAL time-to-ready (both nodes)"* — was identical, to the tenth of a second, to the **control plane's** Ready time. Reading the code explained why. The total is the later of the two nodes' Ready times, and when one of them is missing, it is silently treated as zero. With the worker's timestamp absent, the "both nodes" total was the control plane alone.

That raised a second question: why was the worker's timestamp missing? Queried by hand the next morning, the same worker node reported its Ready time without trouble — seven seconds after the control plane's. So the instrument had looked for it *before the worker was Ready*. And it had looked then because its wait loop — like the older one-number script — stops when *two nodes* report Ready.

On 17 September, the platform's canonical timer, `time-to-ready.sh`, was run three times and reported 26.1, 24.9, and 25.6 seconds, each with *"(2 nodes Ready)"*. Immediately afterwards, the cluster's node list was read by name. It did not contain a worker at all. It contained the control plane and this:

```
node ubuntu-bake-vm-warm    Ready at 2026-09-16T07:10:37Z
```

`ubuntu-bake-vm-warm` is the virtual machine the warm golden image was *baked* on. Chapter 6's warm image carries a copy of k3s's datastore from bake time, and that datastore still contains the bake machine's Node object — with a `Ready` status stamped at 07:10 on 16 September, while the image was being baked. Every new cluster booted from that image begins life believing it already has a second, healthy node.

The repository knew about the ghost. Since June, the control plane has run a cleanup job at startup that deletes it: a background unit that tries every two seconds, for up to two minutes, and does not block anything while it waits. That keeps the cluster tidy. But it runs *concurrently* with every stopwatch that counts Ready nodes. In every run measured on 17 September, when the control plane turned Ready, the ghost was still there. The count reached two, and the clock stopped — with the real worker still booting.

Measured again on 17 September, naming each node rather than counting them, three clean runs from `kubectl apply`:

| | Run 1 | Run 2 | Run 3 | Median |
|---|---:|---:|---:|---:|
| Control plane Ready | 23.4 s | 25.1 s | 24.7 s | 24.7 s |
| Worker Ready — both nodes | 30.4 s | 35.1 s | 33.7 s | **33.7 s** |

In every run, at the moment the control plane first reported Ready, the node list read: control plane `Ready`, `ubuntu-bake-vm-warm` `Ready`. No worker.

The upgraded platform builds a two-node cluster in **33.7 seconds**, median of three. Before the upgrade the figure was 34.5. The upgrade did not make the cluster ten seconds faster. Within the noise of three runs, it changed nothing.

The tight 0.4-second spread that made 23.9 look so trustworthy was not a sign of a stable pipeline. Measured by name, the spread is 4.7 seconds. The instrument had simply been timing the steadier half of the cluster.

Two more things turned up in the same hour. The worker's install path never prints the *"k3s started"* line the instrument searches for — the control plane prints it at 12.8 seconds, the worker never does — so that row could never have been filled. And the KubeVirt API that was unavailable on 9 September, and that forced the instrument's no-login design, reports `Available=True` on the rebuilt cluster. The constraint that shaped the instrument did not survive the rebuild. The instrument's virtues did: nothing on the guests, nothing changed by measuring.

What this does not overturn deserves the same precision. The 9 September tables report a worker-join phase, which means the worker's Ready time was captured in those runs — so, as far as the surviving tables show, the chain from 50.0 to 34.5 seconds in Chapter 8 was a both-nodes measurement. But the raw files behind those tables were not kept, so that cannot be re-checked. The 16 September files were kept, in a temporary directory — which is the only reason this chapter could find its own mistake.

## What an instrument owes you

The lessons are unglamorous, and they apply to any number a business steers by.

- **Name what you count.** "Two nodes Ready" was satisfied by a node that did not exist. "The control plane and the worker are Ready" could not have been.
- **A missing value is not zero.** A dashboard that renders a gap as a zero will one day report an improvement that never happened.
- **Keep the raw data behind the headline.** The correction was possible only because one run's files survived in a temporary directory.
- **Re-verify the instrument whenever the thing it measures changes.** A new image, a new version, a new cluster — each can quietly change what a count means.
- **Be most suspicious of good news.** A number that improves by thirty percent with no change anyone can point to is not a result. It is a question about the stopwatch.

* * *

*Meridian · the following Monday*

The wall display had changed. It no longer showed one number. It showed two, each with a node name beside it, and a date underneath.

"Thirty-three point seven," Anita read. "Last week it said twenty-four."

"Last week it was timing half the cluster," Vikram said. "There's a machine baked into our golden image that doesn't exist, and for the first few seconds of every build it counts as a node. Our stopwatch counted it."

"And nobody noticed because the number looked better."

"Better, and steadier. That's what made it convincing."

Anita looked at the display for a moment. "When a number improves and nobody can say why," she said, "it gets audited before it gets celebrated. Put that on the wall too."

"Under the numbers?"

"Above them," she said. "An instrument is a claim too."

![Stacked timeline of a fifty-second cluster build in six phases, with a bracket marking 16.6 seconds before the guest kernel starts.](../figures/fig-07-1-boot-timeline.svg)

*Figure 7.1 — Where fifty seconds went, and where the golden image could not help.*

## The ledger

- **Built:** a read-only, 320-line instrument that reconstructs a cluster build from records the platform already keeps — object timestamps, pod status, the guest's serial console, node transitions, and a host-side probe.
- **First reading (9 September):** 36% of the wall clock passed before the guest kernel started; the "slow k3s startup" theory behind two failed experiments had bet on 15% of it.
- **Enabled:** the two fixes in Chapter 8.
- **Found wrong (17 September):** a node count satisfied by a ghost Node baked into the golden image; a "both nodes" total that treated a missing worker timestamp as zero; a worker milestone the worker never logs.
- **Corrected:** the upgraded platform builds both nodes in 33.7 seconds (median of three; control plane alone, 24.7) — not 23.9, and no measurable change from before the upgrade.
- **Still owed:** both timing scripts still count nodes rather than naming them; the 9 September raw files no longer exist to re-check.

## Ask your team

1. **When a metric improves, what do we check first — the change, or the instrument?**
2. **What do our dashboards do with missing data: show a gap, or show a zero?**
3. **Do we keep the raw data behind our headline numbers long enough for someone to prove them wrong?**

## Open the repo

- `scripts/phase-timings.sh` — the instrument; its header lists every data source, and `--collect-only` changes nothing.
- `scripts/time-to-ready.sh` — the one-number timer it replaced, and whose "two nodes Ready" condition it shares.
- `docs/fable-cluster-speed-prompt.md` — the brief's mandatory Step 1.
- `docs/sub-60s-cluster-strategy.md` — "Phase-level measurement", with the 9 September budget.
- `03-target-cluster/target-cluster-warm.yaml` — the `warm-ghost-node-cleanup` step, and why it must run first.
- The ghost, live: list the target cluster's nodes the moment the control plane turns `Ready`, and look for `ubuntu-bake-vm-warm`.
- **Without the repo:** Appendix G.2 quotes the `warm-ghost-node-cleanup` step and the ordering comment above it.

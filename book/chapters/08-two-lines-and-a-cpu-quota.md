# Chapter 8 — Two Lines and a CPU Quota

> *Do not claim anything is faster until you have pasted the timing output that shows it.*
>
> — from the brief, `docs/fable-cluster-speed-prompt.md`

*Meridian · a Tuesday in September · the fourth-floor boardroom*

The slide said **< 40 s**.

It had said so since June: in the platform team's quarterly review, on the internal wiki, and in the one-page summary Anita Rao had sent the board when she asked them to fund the sovereign-cloud pilot instead of another three-year committed-spend agreement. Forty seconds from *I need a cluster* to a cluster. It was the number that made the pilot sound like a product.

Vikram Iyer put a single sheet of paper on the table. It was not a slide. It was the raw output of a script, and its last line read `48.520s`.

"That's the cold build," he said. "Clean host, both nodes ready."

"When?"

"June. Nothing has moved it since."

Anita read the sheet twice. "Then where did forty come from?"

"Forty was the target. Somewhere between the plan and the wiki, somebody wrote the target down in the column for results."

She did not ask who. She asked the more expensive question. "The standby pool hands a team a cluster in two hundred milliseconds. You showed me. Why do I care what the cold build does?"

"Because the pool is one deep." He had rehearsed this part. "The first team that asks gets the standby. The second team that asks in the same minute gets the cold build. And the standby is eight cores and fourteen gigabytes of memory sitting idle, on purpose, all day. Every second I take off the cold build is idle capacity I don't have to buy to hide it."

"So the cold build is an inventory problem."

"It's an inventory problem."

Anita turned the sheet over, as if there might be a better number on the back. "We spent June making the golden image smarter. Twice. What did that buy us?"

"Two things we now know don't work," Vikram said. "And a rule. Nobody changes a line until we can say where the forty-eight seconds go."

* * *

*Build log · June 2026*

## Making the image smarter

Chapter 6 described the golden image: a virtual-machine disk with k3s already installed, baked once and copied for every cluster, so that a new node does not spend its first minutes downloading software. By early summer the default build — control plane first, worker only once the control plane reported ready — took between 90 and 150 seconds. The obvious next move was to put even more into the image.

The theory was attractive and, on paper, sound. The first time k3s starts, it does a great deal of work: it creates its datastore, generates certificate authorities and signs certificates with them, registers its built-in resource types, and installs its system components. All of that is identical on every cluster. Do it once, at bake time, and every cluster afterwards inherits it for free.

Two experiments followed. Both failed, for different reasons, and both failures are worth the space.

### Dead end one: `:preinit`

The first attempt preserved k3s's entire state inside the image — the datastore with its resource definitions and access rules, the certificate directory, the air-gapped container images — and added a first-boot step that ran `k3s server --cluster-reset` to give each new cluster a fresh datastore identity. The estimated saving was 30 to 60 seconds.

What actually happened was a cluster stuck at `WaitingForKthreesServer` for more than twenty minutes, while a log inside the control-plane VM repeated one line:

```
CA cert validation failed: Get "https://127.0.0.1:6444/cacerts":
  tls: failed to verify certificate: x509: certificate signed by unknown authority
```

The cause was a disagreement between two systems, each doing exactly what it was built to do. The bake had generated certificate authorities and signed every certificate on disk with them. Then the Cluster API provider for KubeVirt booted the VM and wrote *fresh* per-cluster certificate authorities into it — because managing cluster identity from the management cluster is its job. It overwrote the authority files and left in place every certificate the old authorities had signed. k3s started, presented a certificate signed by the old authority, validated it against the new one, and retried forever.

The repository's post-mortem puts it plainly: *"This failure mode is invisible in design docs that only reason about etcd identity. It only showed up under real end-to-end boot."*

The fix was to delete every certificate that was not an authority before k3s started, and let k3s regenerate them. That worked. It also threw away most of what the image had preserved. The reset still ran, the certificates were still generated at boot, and the one measurement taken came in at about 136 seconds — back inside the 90-to-150-second band the whole exercise had been meant to escape.

### The change that did work

The same week, a different kind of change was tried. Cluster API normally holds the worker back until the control plane reports ready, so the two VMs boot one after the other. Three settings removed that wait: a join token created in advance, a static bootstrap configuration for the worker, and an annotation telling Cluster API to skip its readiness gate. The worker's agent simply retries the control plane's virtual IP until it answers.

Three runs: 46.6, 49.7, and 50.0 seconds. It was the largest single improvement of the summer, and it was not an image change. Nobody drew that conclusion yet.

### Dead end two: `:warm`

The second image experiment attacked the certificate conflict directly. If the provider insists on writing certificate authorities into the VM, make sure it writes the *same* ones the image was baked with. The bake used a fixed set of authorities and a fixed join token; before each deploy, a script pre-loaded those same authorities into the management cluster's secrets; the provider found them there, adopted them, and wrote back identical files. No conflict, no purge, no reset.

It worked exactly as designed. k3s kept the pre-placed authority byte for byte. And the first few runs came in at around 35 seconds.

That was the dangerous moment. Thirty-five seconds is under forty. It is the number on the slide.

The team ran it again, head to head, on the same host, against the plain parallel build it was supposed to beat. The warm image landed between 46 and 56 seconds. The plain build had landed at 46.6, 49.7, and 50.0. A clean cold run of the warm path, measured a few days later, took 48.520 seconds. The two were statistically indistinguishable. The 35-second runs had happened on a quiet host, and this host is rarely quiet.

In hindsight the reason is simple. Everything the warm image pre-computed — certificates, resource definitions, system components — sat *off the critical path*. A node reports ready well before k3s has finished installing its optional components, which apply in the background anyway. The image had made fast a set of steps nobody was waiting for.

The warm image became the default regardless. It was no slower, it was more reliable than the preinit path, and it was what the parallel build now ran on. What went onto the wiki next to it was a line describing it as the path that *"targets time-to-ready < 40s."*

> Two capable attempts, each built on a plausible theory, each costing a full bake cycle to disprove. Neither theory had been measured before it was built.

This is one of the most common ways engineering budgets disappear, and it rarely looks like failure while it is happening. The team optimized the part of the system it could reason about most easily — k3s's own startup. Nobody had yet asked where the fifty seconds actually went.

* * *

*Build log · September 2026*

## The brief

Before anyone changed a line in September, someone wrote a brief. It was addressed to an AI coding agent — the same kind of worker Part IV of this book is about — and it lives in the repository as `docs/fable-cluster-speed-prompt.md`. It is worth reading in full, because nearly everything that went right afterwards is already in it.

It named **one number** and defined it precisely: the wall time from `kubectl apply` of the cluster manifest until both nodes report `Ready`. *"It is the only metric that counts."*

It listed what had already failed, under a heading that leaves no room for interpretation: **"Do NOT retry these — they are settled negative results."** Baking more state into the image. The warm image. Parallelizing the worker's boot, which had already been done.

It called out its own documentation. The wiki's under-forty claim was *"a stale aspiration, never achieved. Where CLAUDE.md and the table above disagree, the table wins."*

It made instrumentation mandatory before any optimization: *"Every optimisation attempt so far has been evaluated against a single aggregate number, which is why two dead ends took a full bake cycle each to disprove."* It even distrusted its own starting estimate — a rough split of seventeen seconds for the VMs to boot, seventeen for the control plane to start, eighteen for the worker to join — and asked for it to be checked: *"If it is wrong, say so; that finding alone is worth more than a speculative patch."*

It set a measurement protocol and called it non-negotiable. Tear the old cluster down and confirm it is gone. Record host load before every run. Three runs minimum; report the median *and the full spread*. Re-measure the baseline in the same session as the candidate, never against a number from another day. Change one variable at a time. And it said why: the host was shared with two Kind clusters and three unrelated containers, and *"an earlier run of this work recorded 35s times that turned out to be quiet-host luck rather than a real improvement."*

It defined success — a median under 40 seconds over three clean runs, with the cluster verification passing afterwards — and then it defined an acceptable failure: *"'No lever found' is an acceptable and valuable outcome."*

Its last line is this chapter's epigraph.

Three things in that brief cost nothing and are almost never written down. A single metric, defined precisely enough that two people measuring it get the same answer. A list of what has already been proven not to work. And explicit permission to come back empty-handed. The third matters most. A team that cannot report *no lever found* will, eventually, report a lever.

## Where the fifty seconds went

Chapter 7 tells the story of the instrument itself: a read-only script, `scripts/phase-timings.sh`, that reconstructs a cluster's boot from timestamps the platform already records — when each Kubernetes object was created, when each VM changed phase, when the guest kernel printed its first line, when k3s announced it had started — without logging in to anything.

Its first job was the baseline. Re-measured in the same session, as the brief required, it was not 48.5 seconds but 50.0: runs of 52.7, 50.0, and 46.6. Here is where those seconds went, median of three clean runs:

| Phase | Seconds | Share |
|---|---:|---:|
| `kubectl apply` → VM instance created (Cluster API controllers) | 6.6 | 14% |
| Launcher pod created → QEMU running | 10.0 | 21% |
| Firmware, kernel, initramfs | 3.7 | 8% |
| systemd and cloud-init → k3s start | 5.5 | 12% |
| Control-plane k3s start → first `200` from `/readyz` | 7.0 | 15% |
| Worker joins, after the API is already answering | 11.8 | 25% |
| **Median time-to-ready** | **50.0** | 52.7 / 50.0 / 46.6 |

Two things in that table changed the plan immediately.

The first was how wrong the estimate had been. The rough split gave the control plane's k3s startup seventeen seconds. The instrument measured seven. The theory both image experiments had been built on — *the slow part is k3s starting up* — had bet on a row worth fifteen percent of the total.

The second was the top two rows. **Thirty-six percent of the wall clock — 16.6 seconds — elapsed before the guest kernel printed its first character.** No change to the image could touch any of it, because none of it happens inside the image.

Two rows looked suspicious. Ten seconds between a launcher pod being created and QEMU running, eight of which the first version of the table labelled *"containerDisk unpack."* And nearly twelve seconds for a worker to join a control plane that was already answering. The team went after the worker first. It took two wrong turns.

## Dead end three: the retry that was doing work

The worker's k3s agent does not wait politely for the control plane. It starts, tries to reach the API, fails, waits, and tries again. The phase data showed the agent starting at 26.7 seconds, the API first answering at 34.8, and the agent's next attempt landing at 36.7 — because it retries on a cycle of roughly five seconds. That looked like 1.9 seconds of pure waste: the API was ready, and the agent was asleep.

The fix looked obvious. Have the worker wait until the API's unauthenticated `/readyz` endpoint answers, *then* start the agent, so that its first attempt succeeds.

It made the cluster slower. The median went to **53.1 seconds — a 3.1-second regression.** In the repository's words: *"The premise was wrong."*

The worker's own log explained why. During those "wasted" retry cycles, the agent was starting its container runtime and preparing the node — real work, overlapping with the control plane's boot. The retry window was not idle; it was parallel. Holding the agent back until the API answered turned two things happening at once into two things happening in sequence. The change was reverted.

> From the outside, a retry loop looks like waste. Often it is the only part of the system working in parallel.

## Dead end four: no route to host

The worker's log held something else: a connection error, three times, each attempt taking about 3.1 seconds, before the agent fell back to the control plane's stable virtual IP, `172.18.255.215`, and succeeded.

```
connect: no route to host
```

The explanation is a genuinely strange property of running VMs inside Kubernetes. When KubeVirt connects a VM to the network in bridge mode, the guest receives its pod's IP address *and that address's /24 subnet*. So the guest believes every other pod in the range is on the same local wire, and tries to find them the way machines on a shared wire do: by broadcasting an ARP request. But pods on this cluster's network are not on a shared wire. Traffic between them is routed. The broadcast finds nothing, and the connection fails with *no route to host*.

Why was the worker trying to reach a pod address at all? Because when the agent first contacts the control plane, the control plane hands back a list of addresses where its API can be reached, and on that list was the control-plane VM's own pod IP. There was also a stale `10.0.2.2:6443` — the address of QEMU's internal gateway — left over from bake time in the warm image's datastore.

Routing the pod range through the gateway fixed the errors completely. Six logged errors per run became zero, and the agent made one clean connection.

The timing benefit was nothing. **The median was 54.3 seconds.**

The log showed why. Between the line where k3s announced that it was starting the kubelet — `Running kubelet --address=0.0.0.0` — and the kubelet's first output, there was a **ten-second silence**. The failed dials had been happening *inside* that silence, not adding to it. Remove them, and the silence was still ten seconds long.

It was a real bug, and the fix was kept for correctness. It was not the slow thing.

> A real bug is not necessarily the expensive one. The instrument is what tells you which is which.

### Found on the way

The same investigation turned up a third bug, one that cost nothing today and could have cost a great deal later. The worker's bootstrap asked for k3s's *agent* mode through an environment variable, which the installer baked into the image silently ignored: it read only positional arguments. So every worker started the k3s *server*, which failed about 0.8 seconds later and aborted the install script before it wrote the file that marks a successful bootstrap. The cluster came up anyway — but only because the provider had been configured not to check for that file. Moving one word to the right place on the command line fixed it.

## Ten point zero zero zero

The ten-second silence had one more trick. Across runs where the dials behaved completely differently, it measured 10.019 seconds and 10.017 seconds. A gap that stable, whatever the network did, looked like a fixed timer somewhere inside k3s that had nothing to do with the dials.

Turning on the agent's debug logging named it exactly:

```
Dial error from server 10.244.0.186:6443@UNCHECKED after 10.000162522s: i/o timeout
```

The kubelet's first log line landed seven milliseconds after that timeout expired.

It was not a timer unrelated to the dials. It *was* the dial: a ten-second TCP connection timeout, reached by two different routes. Before the routing fix, the agent spent the window failing repeatedly. After it, a single connection hung silently until the timeout ran out. Either way, the kubelet did not start until the agent's connection attempt gave up.

And the address it was trying to reach, `10.244.0.186`, was the control-plane VM's pod IP. A two-line test made the problem undeniable:

```
node -> CP VM pod IP:6443   =  200, connect 0.0003s
pod  -> CP VM pod IP:6443   =  timeout
```

**A KubeVirt VM's pod address is reachable from the node it runs on, but not from other pods.** And the worker VM, as far as the network is concerned, is just another pod. It could never reach that address. It waited out the full ten seconds, then fell back to the virtual IP it had been configured with all along.

The fix is one line, added to the commands the control plane runs before k3s starts:

```yaml
- "echo 'advertise-address: 172.18.255.215' >> /etc/rancher/k3s/config.yaml"
```

It tells the control plane to advertise the virtual IP — an address every VM can already reach — instead of its own pod address.

| | Before | After |
|---|---:|---:|
| Median time-to-ready | 50.0 s | **42.7 s** |
| Runs | 52.7 / 50.0 / 46.6 | 42.7 / 42.8 / 41.2 |
| Run-to-run spread | 6.1 s | **1.6 s** |
| Dial timeouts per run | 4–6 | **0** |
| Worker join after API ready | 11.8 s | 5.1 s |

Before trusting it, the team checked that nothing had broken. Inside the target cluster, the built-in `kubernetes` service now pointed at the virtual IP, and a pod could still reach `kubernetes.default` with a `200`.

The spread row matters as much as the median. Run-to-run variation fell from 6.1 seconds to 1.6, because the ten-second timeout had been its largest source. What had looked like a noisy host was, mostly, the bug.

> The noise was the bug.

## One percent of a core

That left the other suspicious row: ten seconds from a VM's launcher pod being created to QEMU actually running, eight of them attributed in the first table to *"containerDisk unpack."*

That label was a guess, and it was wrong.

Every VM in KubeVirt runs inside a launcher pod, and that pod carries small helper containers alongside the one that runs the VM. The instrument showed when each of them started:

| Container | Job | CPU limit | Started at |
|---|---|---:|---:|
| `guest-console-log` | copies the VM's serial console to the pod log | **15m** | +3.0 s |
| `volumesystemdisk` | holds the VM's disk image for the launcher | **10m** | +9.0 s |
| `compute` | runs QEMU | none | +9.0 s |

The limits are the clue. In Kubernetes CPU units, `10m` means ten millicores: one percent of one core. Linux enforces that as a quota — roughly one millisecond of CPU time in every hundred. Even a program that does almost nothing (this container's process literally runs with a `--no-op` flag) needs CPU time to load its libraries and page itself into memory. At one millisecond per hundred, that takes dozens of scheduling periods. The disk container could not start until it had them, and the VM could not start until the disk container had.

Nobody guessed that. It was found by elimination, and every rival explanation was measured and discarded first:

| Hypothesis | Test | Result |
|---|---|---|
| The 1.17 GB image is slow to unpack | Mount the same image as an image volume | Container started in **+0.0 s** |
| Two VMs starting at once contend | Start one VM on its own | **9.0 s** — *slower* than 8.0 s |
| The node's container runtime is slow | Start a trivial two-container pod | **1.0 s** in total |
| Mounting image volumes is slow | As above | Fast |

What gave it away was that the eight seconds was **deterministic**: 8.0 seconds on every VM, on every run, whether the host was busy or not. Load produces variation. A quota produces a constant.

Nobody on the team had chosen ten millicores. It came with the platform. Changing it meant raising a ceiling in KubeVirt's own configuration — `supportContainerResources` — from one percent of a core to a full core:

```json
{"type": "container-disk",
 "resources": {"requests": {"cpu": "100m", "memory": "40M"},
               "limits":   {"cpu": "1",    "memory": "100M"}}}
```

The limit is the part that removed the eight seconds, and a limit is a ceiling, not a reservation. These containers do their work at startup and then sit idle, so raising the ceiling costs nothing once the VM is running. (The same change sets a modest request of a tenth of a core per helper container.)

| | Before | After |
|---|---:|---:|
| containerDisk phase | 8.0 s | **1.0 s** |
| Pod → QEMU running | 10.0 s | 4.0 s |
| Median time-to-ready | 42.7 s | **34.5 s** |
| Runs | 42.7 / 42.8 / 41.2 | 34.5 / 34.2 / 34.7 |
| Run-to-run spread | 1.6 s | **0.5 s** |

There is one catch, and it belongs in the ledger. This setting lives in the running cluster's KubeVirt configuration, not in a manifest in Git. Rebuild the management cluster and it is gone. The fix was wrapped in a script, `scripts/configure-kubevirt-perf.sh`, and wired into the management cluster's setup so that a rebuild would reapply it. A week later, after a rebuild from nothing, it was found missing anyway and had to be applied by hand. That story is Chapter 19.

## Where it landed

```
baseline                 50.0 s   (52.7 / 50.0 / 46.6, spread 6.1 s)
+ advertise-address      42.7 s   (42.7 / 42.8 / 41.2, spread 1.6 s)
+ supportContainerRes    34.5 s   (34.5 / 34.2 / 34.7, spread 0.5 s)
                        ────────
                        −15.5 s   (−31%); the gate was < 40 s
```

Fifteen and a half seconds off every cold build, and a pipeline that now finishes within half a second of itself.

Neither fix touched the image. The repository's own summary is the sentence this chapter is named for: *"Every attempt to make the image smarter (`:preinit`, `:warm`) failed, while the two things that worked were a bad advertised address and a CPU quota."* Add the parallel boot from June, and the pattern holds across the whole summer: every change that moved the number was about how the pieces were orchestrated and configured, not about what was baked into them.

There is a rhyme between June and September worth pausing on. In June a stopwatch said 35 seconds, and it was luck. In September it said 34.5, and it was true. The difference was not the number. It was a spread of half a second across three runs, and two causes you can point to in a log.

The instrument also said what is left, by measured size: about six seconds of Cluster API controllers doing their work before any VM exists, which no change inside the guest can reach; seven seconds for the control plane's k3s to answer; five and a half seconds of systemd and cloud-init. That last phase is the one a leaner, systemd-free image would attack — and a perfect result there is worth about three seconds. It is the fourth lever, not the first.

## The caveat that limits all of this

Every number above comes with a condition, and the repository records it under a heading that does not soften it: *"Measurement caveat that limits all of the above."*

At the start of this work, run-to-run spread on this host was about six seconds. With three runs per candidate, any change smaller than roughly five seconds disappears into that noise. The two fixes that worked were each worth seven to eight seconds — just large enough to see. A three-second improvement would have been invisible, and a three-second regression could have shipped unnoticed. Anything finer needs seven or more runs, a quiet host, or both.

The fixes changed that condition too. With the spread down to half a second, a one-second regression is now visible in three runs. Removing the bug did not only make the cluster faster. It made the cluster *measurable*.

## Postscript: the ten seconds that weren't

On 16 September, a week later, everything underneath the platform was upgraded at once — Kubernetes on the two host clusters from 1.35 to 1.37, k3s inside the VMs from 1.31 to 1.37 — and both host clusters were destroyed and rebuilt from nothing. Afterwards, the same instrument, three clean runs, reported **23.9, 24.1, and 23.7 seconds.** Another ten seconds, and steadier.

It was wrong.

As Chapter 7 tells, the instrument had been stopping its clock when the control plane turned `Ready` — fooled by a ghost node carried inside the golden image — and reporting the control plane's time as the total for both nodes. Measured again on 17 September, naming each node instead of counting them, the upgraded platform builds both nodes in **33.7 seconds** (30.4, 35.1, 33.7). Within the noise of three runs, that is the same as the 34.5 before the upgrade.

This chapter's rule — a faster number is a claim until the phase table says why — turned out to apply to the phase table too. This time the explanation for the good news was the stopwatch.

* * *

*Meridian · the following Tuesday*

Vikram put another sheet on the table. Three lines this time: `34.5`, `34.2`, `34.7`.

Anita looked at the spread before she looked at the median. "Half a second."

"Half a second."

"What did it cost?"

"A script to measure it. Two lines of configuration."

"Hardware?"

"None. The second line raises a ceiling. It doesn't reserve anything."

"And the standby?"

"Still one deep. But the second team in the minute now waits thirty-four seconds instead of fifty."

"And June?"

"June is written down, with its numbers. Nobody here pays for it twice."

She opened the board summary on her laptop, found the line that said *< 40 s*, and deleted it. Then she typed the replacement herself: *34.5 s median, three runs, spread 0.5 s — measured 9 September.*

"Put a date on every number," she said. "A number without a date and a spread is a slogan."

![Dot plot of three runs at each of three stages; the median bar falls and the spread line shortens.](../figures/fig-08-1-median-and-spread.svg)

*Figure 8.1 — Two one-line changes: the median falls from 50.0 s to 34.5 s, and the spread from 6.1 s to 0.5 s.*

## The ledger

- **Spent:** two golden-image experiments, each a full bake cycle to disprove; one instrumentation script; two further hypotheses — gating the worker, rerouting the pod network — that did not survive measurement.
- **Changed:** one line in the control plane's startup commands; one CPU ceiling on KubeVirt's helper containers.
- **Hardware added:** none.
- **Saved:** 15.5 seconds on every cold cluster build — 50.0 s to 34.5 s, −31%. Run-to-run spread from 6.1 s to 0.5 s.
- **Proved:** four dead ends, recorded with their numbers so that nobody pays for them twice.
- **Still owed:** a setting that lives in cluster state rather than in Git, and two timing scripts that can stop the clock on a ghost node (Chapter 7).

## Ask your team

1. **What is our one number, exactly how is it measured, and when was it last measured on this hardware?** Ask to see the timing output, not the wiki page.
2. **What have we already proven does not work, and where is it written down?** If the answer is "people remember," you will pay for it again.
3. **Which of our performance-critical settings did nobody here choose?** Ten millicores came with the platform. Find yours.

## Open the repo

- `make phase-timings ARGS="--runs 3"` — runs `scripts/phase-timings.sh` and prints the per-phase table, using this chapter's three-run protocol.
- `docs/fable-cluster-speed-prompt.md` — the brief, verbatim.
- `docs/sub-60s-cluster-strategy.md` — "What failed, and how it was fixed" for `:preinit`; "Phase-level measurement" for the budget, both dead ends, and both fixes.
- `03-target-cluster/target-cluster-warm.yaml` — the `advertise-address` line in the control plane's `preK3sCommands`.
- `make kubevirt-perf` — applies the CPU ceiling; the header of `scripts/configure-kubevirt-perf.sh` explains it.
- `git show c7ad792` — the commit that landed both fixes, with the measurements in its message.
- **Without the repo:** Appendix G.2 quotes the `advertise-address` line with its measurement comment; G.3 quotes the `supportContainerResources` patch.

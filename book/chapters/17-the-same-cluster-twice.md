# Chapter 17 — The Same Cluster Twice

> *Safe only because exactly ONE target-cluster runs at a time.*
>
> — `03-target-cluster/target-cluster-warm.yaml`, the comment above the fast path

*Meridian · a Tuesday in October · the operations floor*

The runbook for a sick cluster ran to forty-one pages. Vikram had written most of it, and he was not proud of it.

"Page fourteen," Anita said. "*If etcd reports a lost quorum, identify the surviving member.*" She turned the page. "And then eleven more pages of that."

"Every line of it was written after an incident."

"I believe you. That's what worries me." She put the binder down. "How long does it take to build a new one?"

"Thirty-four seconds."

"Then why is any of this forty-one pages?"

Vikram had been waiting for that question for about a year, and he still did not have a clean answer. "Because the cluster isn't the thing we're protecting. It's what's *in* it. And because—" he stopped, and started again. "Because the answer changes depending on which thirty-four seconds you mean."

"Explain that."

"Building one is thirty-four seconds. Getting rid of the old one is longer, and I don't actually know how much longer, because nobody has ever timed it properly. And rebuilding the whole platform underneath it took an afternoon." He shrugged. "One of those is on a slide. The other two aren't."

Anita picked up the binder again and weighed it in her hand.

"Time the whole thing," she said. "Not the flattering part. Destroy one and put it back, and start the clock when you type the destroy command, not when you start building. And do it five times, because the last time we published a number from one run I had to explain it twice."

* * *

*Build log · 19 September 2026*

## The cycle, timed

The book has measured a cluster being built — 33.7 seconds to both nodes `Ready` — and it has measured one being destroyed, once. It has never measured the thing an operator actually cares about: **how long from deciding a cluster is beyond saving to having a working one serving traffic again.**

So the boundaries were written down and committed *before* the first measured run. That ordering matters, because the estimate landed near a minute, and a boundary that can be moved after the data arrives is not a boundary. The clock starts when the destroy command is issued and stops on the first HTTP 200 served through the replacement cluster's Service — not on "pod Running," which is a claim, but on a request that was answered.

Five runs, after one discarded rehearsal, on a quiet host:

| | Median | All five runs | Spread |
|---|---:|---|---:|
| **Destroy to first served request** | **81.96 s** | 80.40 / 81.20 / 81.96 / 82.59 / 84.03 | 3.63 s |
| Delete command returns | 10.13 s | 9.16 / 10.13 / 10.13 / 10.13 / 10.14 | 0.98 s |
| Teardown — last object gone | 45.73 s | 44.76 / 45.58 / 45.73 / 45.97 / 46.00 | 1.24 s |
| Build — both nodes `Ready` | 34.39 s | 32.15 / 33.89 / 34.39 / 34.49 / 36.39 | 4.24 s |
| Build — control plane alone | 25.60 s | 25.30 / 25.46 / 25.60 / 25.67 / 26.08 | 0.78 s |
| Workload to first HTTP 200 | 1.34 s | 1.07 / 1.08 / 1.34 / 1.41 / 1.65 | 0.58 s |

The build reproduced. **34.39 seconds against the 33.7 published in Chapter 7** is comfortably inside the spread of either measurement — a year of this book's arguments rests on that number, and it survived being measured again by a different instrument on a different day.

The teardown did not reproduce, and it is not close.

## The number nobody had checked

Chapter 7 publishes teardown as **18.9 seconds**, and Appendix D is careful to label it *a single run*. Measured five times, it is **45.73 seconds**, with a spread of 1.24 seconds. Not a noisy figure that happened to land low once: a figure that is wrong by a factor of two and a half.

Because the published number counted API objects, and there are two defensible definitions of *gone*, one more run separated them:

| Boundary | Time |
|---|---:|
| The delete command returns | 9.11 s |
| The qemu processes stop — the machines are actually off | 35.60 s |
| The last API object disappears | 45.33 s |

Neither definition is 18.9 seconds. And the ordering is the interesting part: **the objects outlive the machines by about ten seconds.** Cluster API is still reconciling finalizers on things that have already stopped existing in any way that matters. If you want the honest headline, it is the second row — the machines are gone at 35.6 seconds — and the platform spends another ten seconds tidying up records of them.

This is the third figure in this book to be corrected, and the pattern is worth naming, because it is not about arithmetic. A control-plane-only time was published as a both-nodes time because the stopwatch counted a ghost. A 194-millisecond claim stood for three months because nobody re-ran it. Now a teardown measured once turns out to be less than half the real figure. Every one of them was a number that nobody had reason to doubt, in a book whose entire method is doubting numbers.

So: **destroy to serving, 82 seconds.** Bring-up is under a minute. The cycle is not. The slow half is the half nobody had put on a slide.

## What the number buys

Four things follow from this, and this platform can show all four.

**Recovery stops being a procedure and becomes a command.** The industry's reference point is DORA's, which puts elite performers at restoring service in **under one hour**. Against that, 82 seconds of infrastructure is not a line item; it is a rounding error. Which is precisely the point, and the uncomfortable half of it: if replacing the infrastructure costs eighty seconds, then **everything left in your recovery time is data**, and that is the part this platform has never once tested. Chapter 18 is blunt about it — *"a restore that returns a running cluster with an empty database has restored nothing a business cares about."* The forty-one-page runbook does not disappear. It gets shorter and much more specific: it stops being about etcd quorum and starts being about the data.

**Debugging gets a control group.** This book already works this way without having named it. Chapter 8 records four hypotheses tested and discarded against rebuilt clusters. Chapter 11 records a failure that stopped reproducing after an upgrade, which is only a sentence you can write if you can rebuild the thing and look again. When a rebuild is cheap, the question *"is this my change or is this accumulated state?"* has an experiment instead of an argument. Palantir make the same case for their own infrastructure: *"Immutable nodes derived from instance templates simplify the debugging of any node issues encountered in production."*

**Upgrades stop being events.** Chapter 19 is this chapter's case study and it did not set out to be one. Kind has no in-place upgrade, so moving Kubernetes forward meant deleting both clusters and building them again — and the chapter's own line is *"the upgrade is the rebuild."* What was experienced as a limitation of a development tool turns out to be the operating model. The caveat is the one Chapter 19 also records: it deleted first and rebuilt after. A true blue/green — new beside old, traffic shifted, old deleted — needs two clusters alive at once, which is exactly what this platform cannot do, for reasons the next section is about.

**Drills become a calendar entry.** Chapter 9 already prescribed one: *"every week, someone claims it, deletes it, and times the rebuild — naming both nodes."* It was never run. This chapter is that drill, finally executed, and it found a published number was wrong by 2.4×. A rehearsal that costs a day gets scheduled once a year and quietly skipped. A rehearsal that costs eighty seconds has no excuse.

## The trade nobody put on the slide

Now the other four — cluster-per-pull-request, many small clusters instead of a few large ones, cluster-per-tenant, and rotating clusters on a schedule to deny an attacker persistence. All four are good ideas. All four need the same thing: **several clusters, at once, with separate identities.** This platform can produce neither, and the reason is not an oversight. It is the same decision that bought the thirty-four seconds.

The fast path works by baking a fixed certificate authority set, a fixed join token and a fixed API address into the golden image, and pre-seeding matching secrets so the control plane adopts them. That removes the certificate work from the boot path — no `--cluster-reset`, no cert purge. The manifest says what it costs, in a comment sitting directly above the thing it enables:

> *Safe only because exactly ONE target-cluster runs at a time.*

The design document is blunter still: *"If you ever allow concurrent clusters, switch to per-build random identity."* And per-build identity is what the slower path does — the `:preinit` build, whose floor is around a hundred seconds. **The speed and the multiplicity are the same decision, taken in opposite directions.** You may have a thirty-four-second cluster, or you may have many clusters. Not both, on this design.

Chapter 5 noticed the consequence by accident: a kubeconfig found in `/tmp`, written for a cluster that had since been rebuilt six times, still worked. This chapter tested it on purpose. Run 1's administrator credential was kept. Five destroy-rebuild cycles later, it was presented to a cluster thirty-two seconds old — a different cluster, different node names, built from a different `kubectl apply`:

```
$ kubectl --kubeconfig /tmp/drill-kubeconfig-run1 get nodes
target-cluster-cp-lmrt2              Ready   control-plane   32s   v1.37.0+k3s1
target-cluster-workers-xq7mb-vbjr2   Ready   <none>          23s   v1.37.0+k3s1

$ kubectl --kubeconfig /tmp/drill-kubeconfig-run1 auth can-i '*' '*'
yes
```

It authenticated, as full cluster administrator.

Read that against the four ideas. **Cluster-per-tenant** is not merely unbuilt here; a credential issued to one tenant would open every other tenant's cluster, which is worse than the namespace-based isolation Chapter 14's scorecard already grades *not held*. **Rotation as a defence** returns a cryptographically identical cluster — the attacker's credential survives the rotation, so the rotation defends nothing. Palantir's 48-hour node lifetime works because *"compromising a single node is insufficient for an attacker to gain persistent access"* — but that is a claim about **nodes inside a long-lived cluster**, where the cluster's identity outlives the node and can revoke it. Rotating whole clusters that share one identity is a different operation that happens to look the same from the outside.

And even if the identity were fixed, the arithmetic would not budge. Chapter 20 measured one target cluster reserving **14.9 GiB** — roughly half this host. A second one does not fit. Many small clusters is not blocked by build time. It is blocked by memory, and by an address plan where every service address is pinned by hand in a table.

So the honest scorecard for the paradigm: four consequences demonstrated, four unavailable — and the four unavailable ones are unavailable *because of* the decision that produced the four demonstrated ones.

## What replacement destroys

There is one more cost, and this book has paid it three times without writing it down as a cost.

When the cluster was rebuilt for the upgrade, **fifty-three agent run records went with it** — every record that existed that morning, the oldest nearly four weeks old (Chapter 13). Chapter 14's audit rule is graded *not held* with the observation that the trail *"vanished entirely in the rebuild."* And when Chapter 11 tried to re-examine a finding from July, it could not: *"the July cluster no longer exists."*

That last one is the sharpest, because it is the debugging argument running backwards. Cheap replacement lets you bisect against a known-good present. It also destroys the past you might have wanted to bisect *against*. A cluster you can rebuild in eighty seconds is a cluster whose evidence has a half-life of eighty seconds.

Which resolves the question of what backup means here, and it resolves it in two directions rather than one.

**For the infrastructure, backup genuinely does become archival.** If the cluster is reconstructible from a repository and an image, there is little point snapshotting its control plane. Chapter 18 makes the stronger version of this argument: if the repository is the source of truth, rebuilding from it *"is the only version of a disaster-recovery plan that is tested every day."* Keep the artefacts cold, for the auditor and for the forensic case, and stop maintaining a restore path for a thing you can regenerate.

**For everything else, the opposite.** The run records, the logs, the audit trail, the application data — the cluster was never their archive, and in a replaceable world it very obviously is not. They need somewhere to go *before* the replacement, and a restore that someone has actually performed. Chapter 18's rule survives intact and gets sharper: *until the restore has happened, the backup is a hypothesis.* What changes is where the rule points. It stops pointing at etcd and starts pointing at the export.

That is the whole discipline in one sentence: **when the cluster stops being durable, the export becomes the archive.**

## What eighty-two seconds does not mean

One caution before the ledger, because the number is quotable and a quotable number travels further than its caveats.

This is not a production recovery time. There is no detection in it, no decision, no data restore, no high availability, no storage to re-attach, no DNS to reprogram, no other tenants to consider, and no human deciding whether replacement is the right call. It is a lower bound on mechanical rebuild.

Nor is it provisioning from scratch. Every run recorded one line that settles this: the rebuilt cluster's `kube-system` namespace reports a creation timestamp of **16 September** — the day the image was baked — while its nodes were created that morning. The golden image carries a pre-initialised datastore. This is **restore-from-image**, and the image took five to eight minutes to bake. Without it already cached on the node, the honest number is minutes.

And *both nodes Ready* is still not two nodes of capacity. In all five runs the workload landed on the control plane, because the worker carries the taint Chapter 5 documented and nothing on this platform removes it. One correction to that chapter, though, from a test run for this one: a pod that explicitly tolerates the taint and pins to the worker **runs there perfectly well.** The worker is not broken. It is excluded. That is a scheduling decision that nobody made on purpose, which is a different and more fixable problem than the one Chapter 5 described.

The next chapter is the right place to take all of this apart, and it is deliberately unsentimental about exactly this kind of claim: *a reader who skips that list will mistake thirty-four seconds for readiness.*

* * *

*Meridian · the following week*

Vikram brought the binder back, with a bookmark in it.

"Pages one to fourteen can go," he said. "That's everything about repairing a control plane. We don't repair control planes any more; we replace them, and I can show you the eighty-two seconds."

"And the rest?"

"The rest is pages about data, and those pages are now the whole runbook. They used to be an appendix." He put it down. "There's a second thing. Every cluster we rebuild comes back with the same certificates as the one it replaced. That's what makes it fast. It also means our rebuild isn't a rotation, and if we ever want one cluster per customer we have to give the speed back."

Anita thought about that for a while.

"Write that on the same page as the eighty-two seconds," she said. "Not in a footnote. Somebody is going to read that number and ask for a cluster per customer by Friday, and the answer is that we can have it, and here is the price."

## The ledger

- **Measured (19 September, N=5, boundaries pre-registered before the first run):** destroy to first served request **81.96 s** median (80.40 / 81.20 / 81.96 / 82.59 / 84.03, spread 3.63 s). Build to both nodes **34.39 s**, reproducing Chapter 7's 33.7 s. Workload to first HTTP 200, **1.34 s**. Zero image pulls in every run.
- **Corrected:** teardown, published at 18.9 s from a single run, is **45.73 s** (N=5, spread 1.24 s). Split once: the machines stop at 35.60 s, the last API object goes at 45.33 s. The objects outlive the machines by about ten seconds.
- **Proved:** an administrator credential from run 1 opened a different cluster, thirty-two seconds old, five rebuild cycles later, with full cluster-admin.
- **Also corrected:** Chapter 5's worker is excluded by a taint, not broken — a pod that tolerates it runs there normally.
- **Carried from other chapters:** 33.7 s build and 18.9 s teardown (Chapter 7, the latter now superseded); 467 ms warm claim (Chapter 9); 14.9 GiB reserved per cluster (Chapter 20); one afternoon for a full platform rebuild (Chapter 19).
- **Demonstrated consequences:** recovery as a command, debugging with a control group, upgrades as rebuilds, drills as routine.
- **Unavailable on this platform, and why:** cluster-per-PR, many small clusters, cluster-per-tenant, rotation-as-defence — all four need concurrent clusters with separate identities, and the fixed identity that makes the 34-second build possible forbids exactly that. Per-build identity costs roughly a hundred seconds.
- **Not claimed:** any production recovery time; any data restore; that this generalises beyond one cluster on one host.

## Ask your team

1. **How long does it take us to destroy something, and when did we last measure it?** Every organisation can quote its build time. The teardown in this book was wrong by a factor of two and a half because it had been measured once.
2. **If replacing the infrastructure took ninety seconds, what would be left in our recovery time?** That remainder is your real recovery plan. Everything else is a procedure for a machine you could have thrown away.
3. **When we rebuild something, does it come back with a new identity or the old one?** If it is the old one, the rebuild is not a rotation, and anything you were relying on it to revoke has not been revoked.

## Open the repo

- `measurements/ch-recovery-drill.sh` — the drill, with its pre-flight gates; `measurements/README.md` carries the pre-registered boundary table, written before the first measured run.
- `measurements/ch17-recovery-drill-20260919-163611.tsv` — all five runs, raw.
- `measurements/drill-workload.yaml` and `drill-workload-worker.yaml` — the served workload, and the worker-pinned variant that tolerates the taint.
- `03-target-cluster/target-cluster-warm.yaml` — the header comment: *"Safe only because exactly ONE target-cluster runs at a time."* Quoted in Appendix G.1.
- `docs/warm-pool-strategy.md` — *"If you ever allow concurrent clusters, switch to per-build random identity."*
- **Without the repo:** Appendix G.1 quotes the manifest that makes the trade; Appendix D carries the measurement protocol.

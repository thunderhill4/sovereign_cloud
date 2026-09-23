# Chapter 18 — From the Lab to the Data Centre

> *That is not a data centre, and the book never pretends it is. What transfers from a host to a data centre is not scale. It is the pattern, the measurements, and — more often than the vendor brochures would suggest — the failures.*
>
> — the prologue of this book

*Meridian · a Wednesday in October · the fourth-floor boardroom*

The pilot had done what a pilot is for. Anita Rao put one slide up, and it had two columns rather than a number.

"Everything on the left, we measured," she said. "Everything on the right, we would have to build. I am not asking the board to approve the right-hand column today. I am asking it to stop confusing the two."

The chief financial officer read it twice. "The left column is a host."

"One host. Twenty-nine gigabytes of memory, one consumer graphics card, one power supply, one of everything. It builds a Kubernetes cluster in thirty-four seconds and hands over a spare in under half a second, and both of those numbers are real."

"And in a data centre?"

"The same thirty-four seconds, probably. That part is the software, and the software does not care where it runs." She let that sit. "What changes is everything the host lets us skip. One of everything is fine until the one fails."

"Give me the list."

"That is the right-hand column. High availability. Storage that survives a disk. Backups we have restored from, not just taken. Addresses that work on a real network. Secrets that are not in Git." She paused. "Some of those we already know we are missing, because this year we went looking, and the looking is the part I would actually defend to you."

The CFO turned to Vikram. "How much of the right-hand column is new engineering?"

"Less than you would think. Most of it is choosing well-known components and wiring them in. The part that is genuinely new is the discipline, and we already have that. It is the only thing we built that I would call finished."

"Then write it down as a design," Anita said, "and mark every line of it clearly as something we have not done. The moment this chapter reads like a status report, it becomes the thing we spent the year arguing against."

* * *

*Design guidance · October 2026*

**A note on what this chapter is.** Every other chapter in this book reports something that happened, with a date. This one does not. It is a design: an honest mapping of what the platform in the preceding sixteen chapters would need in order to run a production workload in a data centre. **Nothing in it was built, measured, or tested.** Where it draws on a real number, that number comes from an earlier chapter, and the chapter is named. Where it names a piece of software, that is a category illustrated by a well-known example, not a recommendation and not a verified fit.

The reason for the chapter is the question that follows every demonstration in this book: *yes, but would it work for real?* The useful answer is not "yes." It is a list.

## What the host was allowed to skip

Start with what makes the lab a lab. Each of these is a real property of the platform described in this book, recorded in the chapter named.

- **One of everything.** One machine. The management cluster runs as a single Kind node; the target cluster has one control-plane machine and one worker. There is no second copy of anything, anywhere.
- **One failure domain.** A power cut, a kernel panic, or a full disk takes the entire platform — management cluster, workload cluster, models, agents, registry and console — at once.
- **Storage that is a directory.** Virtual machine disks live on the host's filesystem. There is no replication, no snapshot policy, and no backup. The platform has never restored anything.
- **A network with no network in it.** Everything shares one Docker bridge. Addresses are handed out by MetalLB in layer-2 mode from a hardcoded range, `172.18.255.200–225` (Appendix B). There is no router, no VLAN, no firewall, no external DNS.
- **Trust that was generated once and committed.** The warm path's certificate authorities are fixed, and their private keys are in version control, labelled demo-only (Chapter 3). The mesh's root of trust was generated on the host.
- **No identity.** The web console has no login. Neither does the agent console embedded in it. The one authenticated surface is the agent platform's own API, which does require a bearer token.
- **Capacity that is one deep.** One warm standby cluster, one model resident on the GPU at a time (Chapter 9).

None of this is a criticism of the platform. A lab that paid for all of it would have taught less, more slowly. But every item is a thing the data centre would have to supply, and a reader who skips this list will mistake thirty-four seconds for readiness.

## High availability, and what it is actually for

The platform's control planes are single. Cluster API's own model is the first thing that changes: `KThreesControlPlane` takes a `replicas` field, and production practice is three, across three failure domains, so that the loss of one machine leaves a quorum. The same declaration that makes one machine makes three; the cost is not complexity in the manifest, it is having three real places to put them.

Three points are worth a board's attention, because they are where the money and the mistakes are.

**A failure domain is a physical claim, not a label.** Three control-plane machines spread across three hypervisors in one rack, on one power feed, behind one top-of-rack switch, are three copies of one failure. The declaration will look correct. Chapter 5's lesson applies exactly: a status that says `Ready` is a claim, and the claim needs checking against what is actually true.

**The management cluster becomes production infrastructure.** In the lab it is scaffolding. In a data centre it manufactures and repairs workload clusters, so its own availability is now a dependency of everything it made. It needs the same treatment as the clusters it produces — which is the argument for keeping it small, boring, and rebuildable, rather than letting it accumulate.

**Availability has to be tested by breaking things.** This is the book's own rule, from Chapter 11: a check that has never failed is not a check. A high-availability design that has never lost a node on purpose, in front of someone, during working hours, is a diagram.

## Storage, and the backup nobody has restored

Virtual machine disks are the platform's real state. In the lab they are files on one disk. In a data centre they need replicated block storage with snapshots — Rook/Ceph is the usual open-source answer for a Kubernetes-native deployment, and an existing enterprise SAN reached through its CSI driver is the usual answer for an organization that already has one. Either satisfies the requirement: a disk can fail without losing the machine that was using it.

Backup is a separate requirement, and the distinction matters to a board because the two are frequently conflated. Replication protects against a disk dying. It does not protect against a deletion, a bad upgrade, or a corrupted image being faithfully replicated three times. Velero is the common tool for backing up Kubernetes objects and, with the right driver, the volumes behind them.

The honest framing is the one the platform's own history supplies. Chapter 19 describes deleting both clusters and rebuilding them from the repository in an afternoon. That is a real disaster-recovery rehearsal, and most organizations have never run one. But note what it proved and what it did not: it proved the *platform* could be rebuilt. No workload data was in it. A restore that returns a running cluster with an empty database has restored nothing a business cares about.

So the design guidance is a sequence, not a product: choose replicated storage; back up objects and volumes; and then restore from that backup into a different cluster, on a schedule, as a test that is allowed to fail. Until the restore has happened, the backup is a hypothesis.

## Networking that leaves the bridge

MetalLB transfers; the way this platform uses it does not. In the lab it runs in layer-2 mode on a Docker bridge with a hardcoded range, and the book's own address table is a discipline invented to stop two clusters advertising the same address (Appendix B). In a data centre the same component runs in BGP mode, peering with the real routers, announcing service addresses into the network the rest of the organization already uses. That is a conversation with the network team, and it is usually the longest lead-time item in the whole build.

Four more differences, each of which the lab simply does not have:

- **Segmentation.** Management traffic, workload traffic, storage replication and out-of-band console access belong on separate VLANs. The lab has one flat network, which is why nothing in this book has ever had to think about it.
- **DNS.** The platform resolves nothing externally; the mesh's cross-cluster names work because Istio was told about them. Production clusters need real names in the organization's DNS, with a plan for who creates records and how fast they change.
- **Certificates.** The mesh's root of trust was generated locally, and the warm path's certificate authorities are committed to the repository (Chapter 3). In a data centre both belong in the organization's own public-key infrastructure, with a defined lifetime and a rotation procedure that has been performed at least once. Automated issuance — cert-manager is already present on the management cluster — covers the leaf certificates but does not answer the question of who owns the root.
- **Firewalling.** Chapter 14 found the agent egress rule allows ports rather than destinations, so anything the agents can address on port 443 is reachable. On one host that is a finding. On a network with production systems on it, it is the finding.

## Capacity planning, from the one measurement that transfers

This is the section where the lab genuinely earns something, because Chapter 20 measured the thing most capacity plans get wrong.

On the management cluster, 59 running pods had **reserved 22.42 GiB** of memory while the cluster was **using 7.45 GiB** — about a third. The target cluster's control-plane machine had been given 4 cores and 8 GiB, and was using **59 millicores, 1% of its processor allowance**, and 1,114 MiB of memory. Those are measurements, from 17 September 2026, on this platform.

The transferable lesson is not the numbers; it is which number governs. **Capacity is consumed by what is reserved, not by what is used.** The lab demonstrated the consequence directly: a second target cluster would not fit, because its machines could not reserve the memory, even though two-thirds of the memory already reserved sat idle.

For a data centre that produces this sizing method:

1. Size hosts from the **sum of reservations** of everything you intend to place on them, not from observed utilization.
2. Then treat the gap between reservation and use as the thing to manage down, because that gap is the owned-hardware form of the cloud waste in the prologue. Right-sizing requests is the work; it is unglamorous and it is where the margin is.
3. Reserve headroom deliberately: enough to lose a failure domain and still place everything that was running on it. That is an availability cost that appears on the capacity plan, and it is routinely forgotten.
4. Price warmth explicitly. Chapter 9's standby holds eight cores and 14 GiB idle, permanently, to make a claim take under half a second instead of thirty-four. That may be an excellent trade. It is never a free one, and it belongs on the plan as a line.

## Secrets, identity, and the two things the lab has no answer for

**Secrets.** The platform commits certificate authority private keys and a fixed join token, marked demo-only (Chapter 3), and the agent platform's tokens live in cluster Secrets, which are encoded rather than encrypted. Neither transfers. The two common patterns are SOPS, which encrypts values inside the Git repository so the repository stays the source of truth, and External Secrets, which keeps material in a dedicated store — HashiCorp Vault, or a hardware security module — and syncs references into the cluster. The choice matters less than the property both provide and the platform currently lacks: *no plaintext secret in version control, and a rotation procedure that someone has actually run.*

**Identity.** Chapter 12 records that the model gateway shipped without a lock and Kiali runs with anonymous access; the web console has no login at all. In a data centre, every human-facing surface authenticates against the organization's identity provider through OIDC, and Kubernetes role-based access control is bound to real groups rather than to a shared credential. The agent platform makes this sharper rather than softer: Chapter 15 found that an upgrade silently changed which service account agent runs execute as, and a permission check written against the old identity kept reporting success. Identity is not a one-time configuration. It is something to re-verify after every upgrade, against the identity a running pod actually carries.

## GitOps — the answer this book already argued for

Chapter 19 is the reason this section exists, and it makes the case better than any vendor's slide.

When both clusters were deleted and rebuilt, four things turned out to exist nowhere but in the running clusters: the Kind clusters themselves, KubeVirt and CDI, a hand-applied container image patch that no script recorded, and the KubeVirt CPU setting worth eight seconds on every machine start. One of those, the image patch, had been made once, by hand, months earlier, and was recovered only because someone thought to read the old cluster before deleting it. The chapter's conclusion is this book's central claim about sovereignty: *everything that lives only in a running cluster is not owned; it is on loan from that cluster's continued existence.*

GitOps is the mechanism that makes that conclusion enforceable rather than aspirational. A controller in the cluster — Flux and Argo CD are the two common implementations — continuously reconciles the cluster against a Git repository. The properties that matter here are three:

- **Drift becomes visible.** A hand-applied patch shows up as a difference between the cluster and the repository, rather than as institutional memory in one person's head.
- **The rebuild is the normal path, not an emergency one.** If the repository is the source of truth, rebuilding from it is what the system does continuously, which is the only version of a disaster-recovery plan that is tested every day.
- **Change has a review and an audit trail** — which is also what Constraint 6 of the platform's own agent strategy asks for, and what an agent that proposes changes will eventually need in order to be allowed to make them.

The design guidance is therefore blunt: **put the four unscripted things in Git first, before adding anything new.** The cluster definitions, the platform components and their versions, the tuning that is currently cluster state, and the base images. Chapter 19 already produced that list at no extra cost, by the expensive method of losing everything on purpose.

## Observability that remembers

The lab's observability is real but has no memory. Prometheus holds the mesh's metrics with the sample add-on's retention and no durable storage; Kiali is anonymous; the recorded agent demonstrations are files on a host. Chapter 12 found the model traffic — the traffic the board asks about most — carried no model name and no token count, and was displayed nowhere.

For a data centre, three requirements follow, in this order of usefulness:

1. **Retention long enough to answer a question about last quarter**, which means durable storage behind the metrics system and a stated retention period, chosen against the audit obligations rather than the disk price.
2. **Logs and traces with the same lifetime**, so that an incident review can reach the evidence rather than the dashboard's memory of it.
3. **The AI-specific meter the platform still lacks:** tokens by model, by tenant. Chapter 20 measured energy per token on the GPU and found the electricity is a rounding error, which makes the meter a capacity and chargeback instrument rather than a power one — but it remains the closest thing to an invoice line that local AI has.

And one rule carried forward unchanged from Part III: a dashboard is not evidence. Every panel a decision rests on should be traceable to a check that can fail, and every such check should have been watched failing at least once.

## Running outside the vendor's tested range, on purpose

Chapter 19 records a deliberate decision to run Istio on a Kubernetes version outside its tested range, written down next to the version number. That decision was correct for a pilot and is a live risk in a data centre, so it needs a policy rather than a habit.

Design guidance, and this is the part most organizations skip until an incident forces it:

- **Decide who carries the risk.** Either an internal team is willing to debug a mesh nobody upstream has tested this combination of, or the organization buys support and then lives inside the supported matrix. Both are defensible. Running unsupported while assuming someone else will help is not.
- **Record every exception where the next person will trip over it** — which the platform already does, in the mesh's README, and which is the cheapest control in this chapter.
- **Distinguish passing from supported.** The platform's own 22 of 22 checks passed on an unsupported combination. That proves the assertions held, not that the combination is tested. Chapter 19 says so explicitly, and the distinction is exactly what an auditor will ask about.
- **Budget for the upgrade treadmill.** A supported posture means tracking upstream release cadences and their support windows, and that is a standing cost in people, not a project.

## Transfers, and does not

The two-column slide Anita put on the wall, filled in from the book's own chapters.

| Transfers as-is | Does not transfer |
|---|---|
| The declarative model: a cluster is a written declaration, applied (Chapters 4, 5; Appendix G.1) | Single control planes and a single failure domain — everything in this book runs one of each |
| Build and teardown speed: 34.4 s and 45.7 s, which are properties of the software, not the hardware (Chapters 7 and 17) | The storage under it: local files, never replicated, never restored from |
| Cluster API and KubeVirt as the factory, including VMs as first-class objects (Chapter 4) | A flat Docker bridge with a hardcoded address range (Appendix B) |
| The service mesh's model: one label for cross-cluster failover, identity-based authorization with no sidecars (Chapter 11) | MetalLB in layer-2 mode; production wants BGP peering with real routers |
| Golden images and air-gapped boot — nothing fetched from the internet at start-up (Chapter 6) | Fixed certificate authorities and a join token committed to the repository (Chapter 3) |
| The warm-standby pattern, and its explicit price in idle capacity (Chapter 9) | One standby, one resident model — a depth of one is a lab constraint, not a design |
| Agents as Kubernetes objects, with policy, identity and tools declared (Chapters 13, 14) | Unauthenticated consoles, an unlocked model gateway, anonymous Kiali (Chapter 12) |
| Local models on owned hardware, and the measured energy per token (Chapters 3, 19) | One consumer GPU holding one model at a time (Chapter 9) |
| The capacity lesson: reservations govern, and the reservation-to-use gap is the margin (Chapter 20) | The absolute figures — 22.42 GiB reserved, 7.45 GiB used — which are one host's numbers, not a sizing model |
| **The discipline**: measure by name, assert on values, keep the before picture, record the dead ends (Chapters 7, 10, 11, 15, 18) | The assumption that anything here has been operated — no on-call, no SLA, no users, no auditor |

Read the right-hand column as a work plan and it is long but ordinary: well-understood components, wired in by people who have done it before. Read the left-hand column as what a year of a single engineer and an AI assistant produced, and the interesting claim of this book survives the move to a data centre — not *we built a cloud on a host*, but *we found out what is true, cheaply, before spending the money.*

That is the last thing that transfers, and the only one that cannot be bought.

* * *

*Meridian · the same afternoon*

The CFO had been writing while Vikram talked, and what he read back was a question rather than a note.

"If we approve the right-hand column, how long before someone shows me a screen that says everything is fine?"

"About a week," Anita said.

"And how long before I should believe it?"

She thought about the year: a worker marked `Ready` that could not run anything; a failover check that printed success during a total outage; a map labelled *live* drawing fourteen things that did not exist; a permission check that passed while the thing it checked had quietly moved.

"When someone has broken it in front of you," she said, "and the screen went red."

## The ledger

- **Measured, and carried into this chapter:** build 33.7 s (Chapter 7, reproduced at 34.4 s in Chapter 17) and teardown 45.7 s (Chapter 17, superseding an 18.9 s single run); a warm claim in 467 ms (Chapter 9); 22.42 GiB reserved against 7.45 GiB used, and a control-plane machine using 1% of its processor allowance (Chapter 20); energy per generated token on the GPU (Chapter 20).
- **Designed, not built:** every requirement in this chapter — high availability, replicated storage, tested restores, BGP and segmentation, external DNS and PKI, secrets management, OIDC identity, GitOps reconciliation, observability retention, and a support policy. **None of it exists on the platform.**
- **Already known to be missing, from the book's own findings:** replicated storage, any restore test, real authentication on the consoles, destination-scoped egress, secrets outside version control, and four pieces of platform state that live only in a running cluster (Chapters 3, 12, 14, 18).
- **The cheapest item on the list:** writing the exception down next to the version number. The platform already does it.
- **Not claimed:** that any of this was tested, costed, or scheduled. This chapter is a list of requirements, and a list is not a plan.

## Ask your team

1. **Which parts of our platform exist only in a running system — and what would we have to reverse-engineer if we lost it tonight?** Chapter 19 produced that list by accident; produce yours on purpose.
2. **When did we last restore from a backup into a different cluster, and who watched it?** A backup that has never been restored is a hypothesis with a budget line.
3. **Where are we running outside a vendor's tested range, who decided it, and who carries the risk if it breaks at three in the morning?**

## Open the repo

There is nothing in the repository that implements this chapter, which is the point. What the repository does hold is the evidence behind each requirement:

- `03-target-cluster/target-cluster-warm.yaml` — `replicas: 1` on the `KThreesControlPlane`, and the fixed certificate authorities it depends on (quoted in Appendix G.1).
- `03-target-cluster/warm-ca/` and `scripts/gen-warm-ca.sh` — the committed demonstration keys, and the script that regenerates them.
- `06-sympozium/agent-egress-ollama.yaml` — the port-only egress rule (quoted in Appendix G.5).
- `01-metallb/metallb-config.yaml` — layer-2 mode and the hardcoded address range.
- `CLAUDE.md`, the 2026-09-16 entries — the four things that existed only in a running cluster.
- **Without the repo:** Appendix G quotes the manifests above; Appendix B holds the address plan; Appendix C.3 catalogues the statuses that were wrong in both directions.

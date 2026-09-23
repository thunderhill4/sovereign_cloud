# Chapter 2 — The Landlord Problem

> *You're playing in the vendor's sandbox, and they set the rules on how high you can build.*
>
> — *Reclaiming Margins with Sovereign Cloud*, strategic briefing, February 2026

*Meridian · a Tuesday in November 2025 · the CTO's office*

Two letters arrived in the same week, and Anita put them side by side on her desk.

The first was from Meridian's internal audit team. A new control required a particular setting on the Kubernetes clusters that ran the payments platform: a rule that the clusters' control planes check every request against, before it is accepted. The managed Kubernetes service Meridian rented did not offer that setting at the level they used. The supplier's account manager had been helpful and apologetic. The setting might be possible on a higher tier. It might arrive in a future release. It was not on the roadmap anyone could share.

The second letter was the renewal quote for Meridian's virtualization estate — fifteen years of it, running the systems that had never moved to the cloud. The supplier had been acquired. The licensing model had changed. The number at the bottom was one nobody had budgeted for.

Vikram read both. "One's a cloud and one's our own data centre," he said.

"And they're the same letter," Anita said. "In both, somebody else decides what we can change, and what it costs to stay. We don't own either one. We just thought we owned the second."

* * *

## The analogy, taken seriously

*Briefing · February 2026*

The strategy deck makes the argument with an analogy most board members will have lived through. *"Managed Cloud = Renting an Apartment"*: convenient, occupied immediately, but *"you cannot renovate the walls, upgrade the infrastructure, or optimize the layout for your specific needs. The landlord controls everything."* Against it: *"Sovereign Cloud = Owning Your Property"*.

It then names the three things that make a landlord a problem:

1. **No architectural control** — *"Cannot customize control planes, networking, or security configurations."*
2. **Rent increases.**
3. **No equity building** — *"Years of payments build no lasting infrastructure value."*

The first is the one Meridian's auditors ran into. The deck puts it most sharply about Kubernetes: *"With managed services like AKS, you cannot customize the control plane — the brain of your cluster. Need specific RBAC configurations for compliance? You might hit a brick wall because vendors abstract that layer to keep things simple."*

That is fair, with one precision worth keeping. Managed Kubernetes services do let customers change some control-plane settings. The problem is that the *supplier* chooses which ones, on the supplier's timetable, at the supplier's price tiers. The wall is not that nothing can be changed. It is that the list of things you are allowed to change belongs to somebody else.

## What the analogy leaves out

Analogies persuade by what they leave out, and this one leaves out the boiler.

A tenant who wakes to a broken boiler calls the landlord. An owner fixes it. Owning the brain of your cluster means owning its mistakes too, and this book records plenty of them:

- In September 2026, Kubernetes and everything around it were upgraded on the platform this book describes. It took a week of rebuilding. The rebuild turned up a dead container image that never started, behind a setup script that reported success anyway, a version setting that had never controlled what it claimed to, and a mesh release running outside its tested range. That is Chapter 19.
- For months, the dashboard reported a worker machine as `Ready` while a single configuration line prevented it from running any workload at all. That is Chapter 5.
- A software component pinned to a *moving* version tag quietly updated itself one day and then crashed about every thirty seconds, with no log output, for thousands of restarts before anyone diagnosed it. That is recorded in the repository's own list of pitfalls.

None of these would have been Meridian's problem in a rented managed service. All of them were the platform team's problem here. The landlord problem is real, but so is the owner's problem, and a CxO should be sold both.

## Rent increases come from every landlord

*Public record · 2023–2024*

Meridian's second letter is not fiction in its shape. In November 2023, Broadcom completed its $61 billion acquisition of VMware, the virtualization software under a large share of the world's corporate data centres. Broadcom then ended the sale of perpetual licences for VMware's products in favour of subscriptions.

The dispute that made this public was AT&T's. In a lawsuit filed in August 2024, AT&T said Broadcom had proposed a price increase of **1,050%** for continued support under its existing arrangement. The two companies later settled.

The lesson for a board is not about any one supplier. It is that "on-premises" and "owned" are not the same thing. An organization can own every server in its data centre and still rent the layer that makes those servers useful. The landlord problem belongs to any single supplier controlling a layer you cannot easily leave, whether that layer runs in someone else's building or your own.

## The exit fee is being legislated away. The lock-in isn't.

*Public record · 2024–2027*

The deck's third chapter prices the cost of leaving: an *"exit cost multiplier"* of 3–5x, *"18–24 months to migrate"*, and an *"average migration cost"* of $2–5 million. It gives no source for any of the three, so they are best read as illustrations.

One part of the exit cost is falling, and falling by law. The European Union's Data Act — Regulation (EU) 2023/2854 — entered into force on 11 January 2024, and its obligations became applicable on **12 September 2025**, from which point a provider may recover only its costs for a switch. From **12 January 2027** switching charges are to be withdrawn entirely. The largest cloud providers moved ahead of the law rather than behind it: Google Cloud stopped charging data-transfer fees to departing customers in January 2024, and Amazon Web Services followed in March 2024, with Microsoft following suit — each of them a commercial decision taken before any obligation had bitten.

So the most visible exit fee — paying to take your own data out — is going away, at least for customers covered by the Data Act. What regulation does not remove is the part of the exit cost the deck gets exactly right. It calls this *"The Dependency Spiral"*: basic infrastructure has a *"low exit cost"*; add platform services and there is *"moderate lock-in"*; add software services and there is *"high entanglement"*; adopt the full ecosystem and the exit becomes *"prohibitive"*. Every proprietary database, queue, identity service or AI platform a workload depends on is another thing that has to be rebuilt somewhere else before it can move.

That is where Meridian's real lease lives. Not in the egress price, but in the list of services nobody else sells.

## What owning the brain looks like

*Build log · June 2026*

The strongest argument for owning a layer is being able to change it the day you need to.

Chapter 8 tells how the platform in this book brought the time to build a cluster down from 50.0 seconds to 34.5 seconds. Every attempt to make the golden image smarter failed. The two changes that worked were one line each, and both were in layers a tenant usually cannot touch:

- **A control-plane startup setting** — telling the cluster's control plane which address to advertise to its workers. That one line removed a ten-second network timeout and took the build from 50.0 to 42.7 seconds.
- **A virtualization-layer resource setting** — raising the processor allowance that KubeVirt gives the small helper containers beside each virtual machine. With a tiny default allowance, the operating system throttled them while each VM started. That took the build from 42.7 to 34.5 seconds.

Neither is dramatic. Both are exactly the kind of change the deck's apartment analogy is about: knocking down a wall. On rented infrastructure, the first would have been a feature request. The second would not have been possible at all.

## The owner's ledger

The deck is honest about what ownership demands, and this chapter should be too. Its own list of *"Infrastructure Requirements"* for a self-managed platform is: hardware; data-centre facilities; a platform engineering team of *"Kubernetes experts, DevOps, SRE specialists"*; automation and tooling; and operational processes for incidents, changes and security. It estimates *"12–18 months to full production readiness"* and *"$2–5M for mid-size enterprise deployment"* — again without a source.

This book adds three items from experience:

- **Upgrades are yours.** On a managed service, a Kubernetes upgrade is a button and a maintenance window. Here it was a week (Chapter 19).
- **Measurement is yours.** Nobody tells an owner how long things take, or whether a dashboard is telling the truth. Chapters 5, 7 and 11 are about building that for yourself.
- **Pinning is yours.** An owner who does not fix software versions in place has quietly handed the landlord's role to whoever publishes the latest release.

Owning does not remove the landlord's work. It moves the work in-house, where it can be done on your schedule and to your standard — or not done at all.

* * *

*Meridian · the following week*

Vikram's answer to both letters was a table. One row for every layer of Meridian's technology, from physical buildings up to the AI assistant the service desk was piloting. Four columns: *who supplies it*, *what we cannot change that we need to*, *what leaving would take*, and *who at Meridian could run it if we owned it*.

Anita read the fourth column first. Most of it was blank.

"That's the real finding," Vikram said. "It's not the lock-in. We know about the lock-in. It's that for most of these layers, if the landlord went away tomorrow, we wouldn't have anyone who could fix the boiler."

"Then that's our lease," Anita said. "The second column is what we pay the landlord in control. The fourth is what we'd pay ourselves in people." She circled two rows: the Kubernetes clusters, and the virtualization estate. "Start there. Those are the two where somebody else owns the brain, and where learning to own it ourselves would be worth something even if we never left."

## The ledger

- **The landlord problem (briefing):** no architectural control, rising rent, no equity. On managed Kubernetes, which control-plane settings a customer may change is decided by the supplier.
- **Every landlord (public record):** Broadcom completed its $61 billion VMware acquisition in November 2023 and ended perpetual licences; AT&T alleged a proposed 1,050% increase in a 2024 lawsuit, later settled.
- **The exit fee (public record):** EU Data Act in force 11 January 2024, applicable from 12 September 2025 (cost-only switching charges), with charges withdrawn entirely from 12 January 2027; Google Cloud and AWS dropped exit data-transfer fees in 2024, ahead of the obligation.
- **The lock-in that stays (briefing):** the dependency spiral of proprietary services.
- **Owning the brain (build log):** two one-line changes, in layers a tenant rarely reaches, cut cluster build time from 50.0 s to 34.5 s.
- **The owner's cost (build log):** a week-long upgrade; a dashboard wrong for months; a moving version tag that crashed a service thousands of times.
- **Not supported:** the deck's 3–5x exit multiplier, 18–24 months, and $2–5 million figures have no source.

## Ask your team

1. **For each layer of our technology, which settings do we need to change but aren't allowed to — and who decides?**
2. **Which of our suppliers could raise our price tenfold, and what would we do the next morning?**
3. **If a supplier disappeared tomorrow, which layers could we run ourselves, and which could we not even diagnose?**

## Open the repo

- `docs/sub-60s-cluster-strategy.md` — the two one-line changes, and the four that failed.
- `03-target-cluster/target-cluster-warm.yaml` — the control-plane setting, in the manifest (`advertise-address`).
- `scripts/configure-kubevirt-perf.sh` — the virtualization-layer setting (`supportContainerResources`).
- `CLAUDE.md`, "Common Pitfalls" — the owner's side of the ledger, including the upgrade record and the moving-tag crash.
- The strategy deck, slides 5, 8, 9 and 16. (Not a repository file; see Appendix H — Sources.)

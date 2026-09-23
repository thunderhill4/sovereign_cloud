# Chapter 20 — Reclaiming the Margin

> *The question is no longer whether you can afford to become your own cloud provider, but whether you can afford not to.*
>
> — *Reclaiming Margins with Sovereign Cloud*, strategic briefing, February 2026

*Meridian · a Friday in September 2026 · the CFO's office*

The chief financial officer had a single slide on his screen, and he had clearly been waiting all week to show it to someone.

Two columns. *Managed Cloud (5-Year): $12.5M — escalating costs, no equity.* *Sovereign Cloud (5-Year): $8.2M — predictable costs, asset value.* And across the bottom, in green: *Net Savings: $4.3M (34% reduction) + owned infrastructure assets.*

"A consultant sent it," he said. "It's the number I asked you for a year ago. Which parts of the bill are rent. I want it in the board pack."

Anita read it twice. "Whose twelve and a half million?"

"An enterprise like ours, apparently."

"Then it isn't ours." She sat down. "I'm not saying it's wrong. I'm saying I can't defend it, because I can't find any of it on our invoice. Can we take it apart before the board does?"

The CFO looked at the green line for a moment, then closed the slide and opened a blank spreadsheet. "Line by line," he said. "Go on, then."

* * *

## The slide

*Briefing · February 2026*

The slide is the fifth and last chapter of the strategy deck behind this book, and it is the most quoted. It compares five-year total cost of ownership — the full cost of an option over its life, not just its first invoice — for a managed cloud and a self-managed sovereign cloud:

| | Five-year cost | The deck's description |
|---|---|---|
| Managed cloud | **$12.5M** | *"Escalating costs, no equity"* |
| Sovereign cloud | **$8.2M** | *"Predictable costs, asset value"* |
| Net saving | **$4.3M (34%)** | *"+ owned infrastructure assets"* |

It explains the saving with four *"Cost Elimination Factors"*:

| Factor | Saving | The deck's description |
|---|---|---|
| Success Tax elimination | −$1.8M | *"No more exponential scaling fees"* |
| Innovation Tax removal | −$1.2M | *"Stop paying premium for stagnation"* |
| Vendor licensing savings | −$900K | *"Open-source alternatives"* |
| Infrastructure optimization | −$400K | *"Right-sized for actual workloads"* |

Elsewhere, the deck adds the numbers a board will ask about next: *"20–35% Cost Reduction"*, a *"5–7yr Payback Period"*, *"12–18 months to full production readiness"*, an *"Initial Investment: $2–5M for mid-size enterprise deployment"*, and a *"3–5 Year Timeline"* for the whole transformation.

None of these figures has a source. They describe an illustrative enterprise, not a measured one. That doesn't make the slide useless. It means the slide has to be read the way a CFO reads any model: by asking whether it holds together.

## Three questions for any TCO slide

### Do the parts add up?

They do, exactly. $1.8M + $1.2M + $0.9M + $0.4M is $4.3M, which is $12.5M minus $8.2M, which is a 34.4% reduction.

That exactness is the problem. All four lines are *savings*. There is no line for anything owning *adds*. Yet the deck's own previous chapter lists what a self-managed platform requires: hardware; data-centre facilities; a platform engineering team of *"Kubernetes experts, DevOps, SRE specialists"*; automation and tooling; operational processes. The migration itself costs money, and so does the period of paying for both the old estate and the new one. If the four savings add up exactly to the net figure, every one of those costs is either hidden inside the four lines or missing from the model.

### Can you find the lines on an invoice?

Two of the four, yes. Licences are an invoice line, and so is infrastructure you can right-size.

The other two are not. *"Success Tax elimination"* and *"Innovation Tax removal"* are the deck's names for two ideas from Chapter 1: that costs grow with success, and that money spent standing still is money not spent building. Both are real ideas. Neither is an account. Together they are **$3.0M of the $4.3M — about 70% of the saving** — sitting in lines no finance system records.

### Do the slides agree with each other?

A 34% saving over five years, and a payback of five to seven years, are hard to hold at the same time. Payback is how long it takes the savings to repay the initial investment. If the model saves money within five years, it has paid back within five years.

The two agree only under assumptions the deck doesn't state. Take the deck's own initial investment of $2–5 million, and suppose the $8.2 million *leaves it out*:

| Initial investment | Five-year cost of owning | Five-year saving |
|---|---|---|
| $2.0M | $10.2M | **+$2.3M (18%)** |
| $3.5M | $11.7M | **+$0.8M (6%)** |
| $5.0M | $13.2M | **−$0.7M (−6%)** |

Or suppose the $4.3 million is a steady saving of $0.86 million a year: then a $2 million investment pays back in **2.3 years**, and a $5 million one in **5.8 years**. The deck's five-to-seven-year payback fits only at the expensive end of its own range — where the five-year saving shrinks toward nothing, or turns negative.

None of this means the slide is wrong about the direction. It means the slide is a *shape*, not a number. Its real value is the list of lines. The amounts have to come from somewhere else.

## An asset that wears out

*Public record · 2022–2025*

The slide's right-hand column claims *"asset value"*, and an earlier slide lists *"Asset value appreciation"* and gives the asset value as *"∞"*. Servers do not appreciate. They wear out, and they become obsolete, and accounting says how fast.

The clearest evidence comes from the landlords themselves. Microsoft, in its 2022 fiscal year, and Alphabet, in 2023, lengthened the accounting life of their servers and network equipment from four years to six, which reduced their reported depreciation by billions. In February 2025, Amazon went the other way for part of its fleet, shortening it from six years to five from the start of that year, and citing *"the increased pace of technology development, particularly in the area of artificial intelligence and machine learning."*

The three largest owners of servers in the world disagree about how long a server lasts. A TCO for a sovereign platform should not assume they last forever. Plan a hardware refresh inside, or just after, a five-year model — and expect AI hardware to age faster than the rest.

## What the platform can put on the worksheet

*Build log · 17 September 2026*

The platform in this book cannot supply Meridian's numbers. It can supply something the slide lacks: inputs that were measured. Four of them matter for a TCO.

**Speed.** A two-machine Kubernetes cluster is built from its declaration to both machines ready in a median of 33.7 seconds, and torn down in 45.7 (Chapters 7 and 17). For steady workloads, "we rent because renting is faster" is no longer a line that has to be paid for.

**Reservation versus use.** Waste is not only a cloud problem. On the management cluster that afternoon:

- Its 59 running pods had **reserved 22.42 GiB** of memory. Of that, 14.57 GiB was held by the two virtual machines of the target cluster.
- The whole management cluster was **actually using 7.45 GiB** — about a third of what it had reserved.
- The target cluster's control-plane machine had been given 4 cores and 8 GiB. It was using **59 millicores — 1% of its processor allowance — and 1,114 MiB of memory.**
- The node had reserved 4.59 processor cores. At the moment it was sampled, it was using well under half of one.

On owned hardware, capacity is set by what is **reserved**, not what is used. On this host, a second target cluster would not fit — its machines could not reserve the memory — even though two-thirds of the memory already reserved sat idle. That is the deck's *"overprovisioning"*, on hardware you own. It is the owner's margin to reclaim or waste, and right-sizing reservations is where it lives.

**The energy cost of AI.** The model most of the platform's agents use, `qwen2.5:7b`, was measured on the host's GPU. With the model loaded and idle, the GPU drew **6.4 W**. While generating text it drew **38–42 W**, at **34–37 tokens per second**. That comes to **1.19 joules per generated token — 0.33 kWh per million tokens**, for the GPU alone.

Multiply that by any electricity price between 10 and 50 cents a kilowatt-hour, and the power for a million generated tokens costs roughly **3 to 17 cents**. The lesson is not that local AI is nearly free. It is that for local AI, the electricity is a rounding error. The cost is the hardware — which, on this GPU, holds one model at a time (Chapter 9) — and the people who run and check it.

**The people.** The repository behind this book holds **88 commits, on 18 distinct days**, from 11 February to 18 September 2026, from **a single committer, Mahipal**. **70 of those 88 commits** carry an AI coding assistant as co-author. (The count moved while this chapter was being written, because writing the book added commits to the platform — which is its own small lesson about measuring a thing you are standing on.) The platform's operating notes run to 612 lines, including 23 entries in a list of pitfalls — most of them a failure that looked like success.

Read that carefully, because it is the easiest number in this book to misuse. It describes a lab: no on-call rota, no service-level agreement, no production users, no auditors. It says nothing about the cost of *operating* a sovereign platform at Meridian's scale. What it does suggest is narrower, and still worth a line in the model: in 2026, the effort to *build* such a platform is not the effort a 2021 estimate would assume. And the cost that never appears on a TCO slide is the one those 23 pitfalls describe: the continuous work of knowing what is true.

One more fact belongs on the worksheet. The platform's cost-analysis agent — the persona written in April to find idle virtual machines and propose stopping them — cannot currently be installed; it was written for a resource type the agent platform no longer ships (Chapter 16). The tool meant to measure the owner's waste is itself waiting to be fixed.

## The worksheet

Put the slide's shape and the platform's measurements together, and the chapter's real deliverable is a worksheet: the lines a sovereign-cloud TCO must contain, and what this book can say about each. The amounts come from your own invoice.

| Line | Renting | Owning | What this book measured |
|---|---|---|---|
| Capacity | On the invoice | Hardware, amortized over its refresh cycle (4–6 years, per the public record) | 22.42 GiB reserved against 7.45 GiB used |
| Speed to capacity | Minutes | The factory's build time | 33.7 s to a two-machine cluster; 45.7 s to remove one |
| Software | Managed-service fees and licences | Open source, plus optional support | The whole stack is open source (Chapters 3–4) |
| Leaving | Switching fees (banned in the EU from 2027) and rebuilding proprietary services | Migration project, and paying for both during it | No migration tooling in the repository (Chapter 4) |
| Power and facilities | Inside the rent | Power, cooling, space | 0.33 kWh per million generated tokens, GPU only |
| AI inference | Per token, hosted | Hardware sized for the models | 34–37 tokens/s; one model at a time on a 6 GB GPU |
| People | The supplier's staff | Platform team, on-call, training | A lab: one committer (Mahipal), 70 of 88 commits AI-co-authored, over 18 days |
| Upgrades | A button and a maintenance window | Yours | A week, for one major upgrade (Chapter 19) |
| Knowing what's true | The supplier's status page | Your own checks | Gauges found wrong in Chapters 4, 5, 11 and 12 |

## One workload, metered both ways

*Build log · 18 September 2026*

A worksheet with no amounts in it is still a slide. So the test this chapter proposes — meter one workload both ways, in parallel — was run at the only scale available: one workload, on the host.

**The workload.** An `nginx` deployment on the target cluster, served continuously and probed every twenty seconds for ten minutes. It answered **30 of 30 probes with `200`**. It is a small, steady, unglamorous workload, which is exactly the kind the a16z paper says is cheapest to own.

**What it consumed, measured.** The platform as a whole ran at a **median 0.549 of the host's 24 threads — 2.29% of its processor — and 8.71 GiB of 29.96 GiB**. The target cluster's control-plane node used 67 millicores and 1,142 MiB.

But use is not what capacity costs, and this is where the lab produced its sharpest number. The two virtual machines **reserve 14,920 MiB — 48.6% of the host's memory** — while reserving only 400 millicores each, 3.3% of its processor. The `cores: 4` in each machine's declaration is a guest topology setting, not a claim on the host's CPU at all. Memory is the binding constraint, and one cluster of this shape holds down **roughly half the machine**, whatever it is doing.

**What could not be measured, and therefore is not claimed.** Whole-host power. The CPU energy counter at `/sys/class/powercap/intel-rapl:0/energy_uj` is present but not readable without root on this kernel, and the battery reads zero on AC. So host power below is an assumption with a range, not a measurement. The GPU was measured — a steady 5.22 W — and is idle: this workload never touches it.

**The rented side**, priced from published list prices, accessed 18 September 2026, with no committed-use discount:

| Item | Price |
|---|---|
| DigitalOcean CPU-Optimized Droplet, 4 vCPU / 8 GB × 2 | $0.125 / hour each |
| DigitalOcean Kubernetes standard control plane | free |
| *(alternative)* high-availability control plane | $40 / month ≈ $0.055 / hour |
| *(for comparison)* Amazon EKS control plane, standard support | $0.10 / cluster-hour |

**Rented: $0.250 per workload-hour** for the same two-machine shape with a free control plane — $0.305 with a highly available one. A managed control plane is not always free, which is what the EKS line is doing there.

**The owned side** needs four inputs. One is measured, three are assumptions, and the assumptions are stated so a reader can substitute their own:

| Input | Value | Measured? |
|---|---|---|
| Share of the host this workload holds | 48.6% | **measured** (memory reservation) |
| Host capital cost | $1,500–$3,000 | assumption |
| Amortization | 4 years — the conservative end of the public record above | assumption |
| Host power under this load | 25–65 W, at $0.10–$0.50/kWh | assumption; RAPL unreadable |

That gives hardware at **$0.021–$0.042** per workload-hour and power at **$0.001–$0.016** — call it **$0.02–$0.06 per workload-hour** for the metal.

Against $0.250 rented, that looks like a rout: four to eleven times cheaper. It is also wrong, because it leaves out the only line that actually decided this.

### The line that decides it

The repository carries commits on **18 distinct days** between 11 February and 18 September 2026. That is a measured lower bound on the human effort — real work happens on days without commits, and a commit day is not a full day — but it is the only effort figure this book has that is not a guess.

Put that on the same worksheet. At an assumed $400–$1,200 per engineer-day, spread over four years:

| Steady workloads sharing the build effort | Build effort, per workload-hour | Total owned, per workload-hour | Against $0.250 rented |
|---|---|---|---|
| 1 | $0.21–$0.62 | **$0.23–$0.68** | loses, or breaks even at best |
| 2 | $0.10–$0.31 | **$0.12–$0.37** | roughly a wash |
| 3 | $0.07–$0.21 | **$0.09–$0.27** | wins, except at the pessimistic end |
| 5 | $0.04–$0.12 | **$0.06–$0.18** | wins |
| 10 | $0.02–$0.06 | **$0.04–$0.12** | wins comfortably |

**The crossover is at roughly two to three steady workloads.** Below it, the build effort swamps everything and the cloud is cheaper. Above it, the hardware arithmetic starts to look like the a16z paper's "one-third to one-half," and keeps improving.

That is the first line in this book that is true about money, and it is worth being precise about how small it is. It is one workload, on one host, for ten minutes, against one provider's list price, with three of its four cost inputs assumed rather than measured. It does not capture high availability, replicated storage, backup, support contracts, on-call, an SLA, facilities, networking hardware, or any of the right-hand column of Chapter 18 — every one of which the rented price *includes* and the owned price does not.

So the honest statement of the result is narrow, and it is still the most useful sentence the platform has produced about cost:

> **On steady workloads, owning beats renting on the metal by roughly an order of magnitude — and the build effort is so much larger than the metal that it, not the hardware, decides whether owning pays. At this lab's effort level, the answer turns positive somewhere around the second or third workload.**

Which is the same conclusion the deck reached, by a different route, and with one difference that matters: the deck put 70% of its saving in lines no invoice records. This puts the deciding line — people — in the open, where a CFO can argue with it.

## The journey

*Briefing · February 2026*

The deck's last recommendations slide is a seven-step *"Sovereign Cloud Journey"*. It holds up better than its TCO slide, because each step is an action, not a number. Here are the steps, with where each one shows up in this book:

1. **Assess current state** — *"Audit cloud spend, identify lock-in points, catalog proprietary service dependencies."* Meridian's workload spreadsheet, and its dependency table (Chapters 1 and 2).
2. **Evaluate workload portability** — which workloads move easily, and which need rework. The steady ones first (Chapter 1).
3. **Start with non-critical workloads** — *"Build expertise on development, testing, or low-risk production systems."* The pilot, and Parts II to IV.
4. **Invest in CAPI & Kubernetes** — the factory (Chapter 4 and Part II).
5. **Consider hybrid approaches** — *"Maintain some managed services during transition to reduce risk."* Meridian is still renting, on purpose.
6. **Build platform engineering** — the blank fourth column of Vikram's table in Chapter 2: who could fix the boiler.
7. **Plan a 3–5 year timeline** — *"Full transformation requires patience but delivers lasting value."*

The deck lists four *"Key Success Factors"*: executive sponsorship, adequate budget, skills development and hiring, and incremental delivery. This book would add a fifth, ahead of the others: **measurement before savings.** Nothing in steps 1 to 7 produces a saving until someone has metered the same workload both ways.

## A cloud owning culture

The campaign flyer promises *"Save $Millions"* and names the means: *"Creating a Cloud Owning Culture."* The book agrees with the second half, and has spent eighteen chapters finding out what it means.

Owning a cloud is not owning servers. It is owning the work a landlord used to do out of sight: knowing what is true about the platform, and being able to show someone else how you know. At Meridian that took the form of rules, each written after a measurement proved an assumption wrong:

- Don't say *ready*; say *working*, and only once something has worked (Chapter 5).
- Anything we keep warm, we exercise (Chapter 9).
- Keep the before picture, and label it (Chapter 10).
- No check goes on a dashboard until someone has watched it fail on purpose (Chapter 11).
- A model gets what every service gets: an identity, a lock, and a meter (Chapter 12).
- An agent may advise; it may not act — until it has earned it, in writing (Chapters 15 and 16).

None of those rules appears on a TCO slide. All of them are part of the cost of owning, and all of them are where the margin is actually kept.

## What to ask your vendors — including the one you become

When a sovereign-cloud programme starts, the organization becomes a supplier to itself. The same questions apply to both.

1. **Price.** How much notice do we get before a price change, and is there a cap?
2. **Leaving.** What does it cost to leave, in fees and in rebuilding, and in what format do we get our data back?
3. **Control.** Which settings we need can we *not* change, and who decides when that list changes?
4. **Jurisdiction.** Where are our data, our logs and our AI prompts processed — and under whose law?
5. **Metering.** Can we see tokens used, by model and by team, every month?
6. **Truth.** When your status page says *healthy*, what did you check, and how would we check it ourselves?

A platform team that cannot answer those six for its own platform has rebuilt the landlord, and moved it indoors.

* * *

*Meridian · the same Friday, two hours later*

The spreadsheet had nine rows, the same nine as the worksheet, and three columns: *renting today*, *owning, estimated*, and *owning, measured*. The first column was nearly full; the finance team had the invoices. The second was full of ranges. The third was empty except for three cells Vikram had sent over: a build time, a reservation figure, and an energy number.

"So what goes in the board pack?" the CFO said. "They asked for a saving."

"They'll get one," Anita said. "Not from a consultant. From us." She pointed at the first row of the workload spreadsheet from January — the steadiest system Meridian had. "We run that on the pilot platform for a quarter, in parallel. We meter both sides — the invoice on one, and hardware, power and hours on the other. People's time goes in the model, not in a footnote. At the end of the quarter we have one number we can defend, for one workload."

"One workload isn't four point three million."

"No. It's the first line of it that's true." She closed the consultant's slide. "The board has been shown a lot of green this year. I'd like the first number we give them about the money to be one we measured."

The CFO typed the board-pack line himself. *Sovereign cloud: no savings claimed. One workload metered in parallel, Q4; first measured result in January.*

He read it back. "It's a very boring sentence."

"Good," Anita said. "Boring sentences are the ones that turn out to be true."

## The ledger

- **The deck's model (briefing):** $12.5M rented against $8.2M owned over five years; a $4.3M (34.4%) saving that is exactly the sum of four savings lines, with no line for anything owning adds. About 70% of the saving ($3.0M) is in two lines no invoice records. The 5–7 year payback agrees with the five-year saving only at the expensive end of the deck's own $2–5M investment range.
- **Depreciation (public record):** Microsoft (FY2022) and Alphabet (2023) moved servers from four- to six-year lives; Amazon moved part of its fleet from six to five years from January 2025, citing AI.
- **Owned waste (build log, 17 September 2026):** 22.42 GiB reserved against 7.45 GiB used on the management cluster; a control-plane VM using 1% of its CPU and 14% of its memory.
- **AI energy (build log):** `qwen2.5:7b` on an RTX 4050 laptop GPU at 6.4 W idle and 38–42 W generating; 34–37 tokens/s; 1.19 J per token; 0.33 kWh per million generated tokens, GPU only.
- **Effort (build log, 18 September 2026):** 88 commits from one committer, Mahipal, over seven months, 70 of them co-authored with an AI coding assistant, on 18 distinct days — a lab, not an operation. (The figure moved while this book was being written, because writing it added commits.)
- **Metered, one workload both ways (build log, 18 September 2026):** an nginx deployment on the target cluster, 30 of 30 probes `200` over ten minutes. It holds **48.6% of the host's memory by reservation** while using 2.29% of its processor. Rented equivalent, at published list prices: **$0.250 per workload-hour**. Owned metal, on stated assumptions: **$0.02–$0.06**. Owned including the build effort: **$0.23–$0.68 at one workload**, falling below the rented price at **roughly two to three steady workloads**.
- **Not claimed:** any saving for Meridian; any figure for high availability, backup, support, on-call or an SLA, none of which the owned side includes and all of which the rented price does.

## Ask your team

1. **For our biggest workload, what would it cost to meter it both ways, in parallel, for one quarter?**
2. **In any savings model we have been shown, which lines appear on an invoice — and which lines for new costs are missing?**
3. **What do we reserve against what we actually use — in the cloud, and on the hardware we already own?**

## Open the repo

- `kubectl --context kind-cluster2 describe node cluster2-control-plane` — reserved against allocatable, under *Allocated resources*.
- `docker stats --no-stream` — what the clusters actually use.
- `kubectl --kubeconfig target-cluster-kubeconfig top nodes` — what a target-cluster machine uses of what it was given.
- `nvidia-smi --query-gpu=power.draw --format=csv -lms 200`, alongside `ollama`'s `eval_count` and `eval_duration` — energy per token.
- `git log --format='%B' | grep -c 'Co-Authored-By'`, `git shortlog -sn`, and `git log --format=%ad --date=short | sort -u | wc -l` — how the platform was built, and on how many distinct days.
- `measurements/ch19-parallel-metering-20260918.tsv` and `measurements/ch19-meter.sh` — the metering run's raw samples and the script that produced them; `measurements/README.md` holds the full assumptions table and the dated price sources.
- `06-sympozium/cost-analyzer.yaml` — the FinOps agent, and why it does not install (Chapter 16).
- The strategy deck, slides 16, 18 and 19; the unbranded sovereign-cloud flyer, version 2. (Neither is a repository file; see Appendix H — Sources.)

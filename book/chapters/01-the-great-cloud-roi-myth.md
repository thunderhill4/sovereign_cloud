# Chapter 1 — The Great Cloud ROI Myth

> *You're crazy if you don't start in the cloud; you're crazy if you stay on it.*
>
> — Sarah Wang and Martin Casado, Andreessen Horowitz, "The Cost of Cloud, a Trillion-Dollar Paradox", 2021

*Meridian · a Thursday in October 2025 · the finance committee*

The chief financial officer had brought the original business case for the cloud migration. It was seven years old, and one line was highlighted in yellow: *lower total cost of infrastructure within three years.*

"I'm not here to blame anyone," he said. "The migration was right. I signed it. But this line was the reason, and it hasn't been true for a while. So what are we paying for now?"

Anita had expected the question. "Speed," she said. "The migration did everything else it promised. Teams get infrastructure in minutes, not months. We launched in two new markets without building anything. That was real, and we'd make the same choice again."

"And the cost?"

"Nobody promised it would stay cheap. We assumed it. The price of an hour of computing didn't go up much. We bought many more hours — because it was easy, and because the business grew. The bill grows with our success."

The CFO looked at the yellow line. "Then I want to know which parts of the bill are buying speed, and which parts are just the rent."

"So do I," Anita said. "Give me a quarter."

* * *

## The promise was real

*Briefing · February 2026*

The strategy deck behind this book opens its first chapter fairly. It sets out *"The Promise"* before it gets to the problem: speed and agility, *"rapid deployment, instant scalability"*; cost efficiency through *"pay-as-you-go models, reduced capital expenditure"*; and *"access to cutting-edge services, global infrastructure, continuous updates."*

None of that was marketing. A team that needs a database for a two-week experiment should not wait a quarter for hardware. A company that does not know whether a product will have a hundred users or a million should not buy for a million. A business entering a country where it has no data centre should not build one first. For uncertain, spiky or brand-new demand, renting is not merely convenient. It is the correct economics.

So the myth this chapter is about is not "the cloud is expensive." It is narrower, and more costly to believe: *that renting stays cheaper as you grow.*

## The paradox

*Public record · May 2021*

The clearest statement of the problem came, perhaps surprisingly, from a venture capital firm whose portfolio companies are among the cloud's biggest customers. In "The Cost of Cloud, a Trillion-Dollar Paradox", Sarah Wang and Martin Casado of Andreessen Horowitz put it in one sentence — the epigraph above — and then tried to measure it.

Among the public software companies they benchmarked, committed cloud spend averaged **about 50% of cost of revenue**. Practitioners who had moved workloads back to their own infrastructure told them that doing so cost **one-third to one-half** as much as running the same workloads in the cloud. The authors then estimated what cloud costs were doing to company valuations, through margins: **around $100 billion** of market value lost across fifty top public software companies, and **more than $500 billion** across a wider universe.

Treat the valuation figures as what they are: a model, built from margins and market multiples, and contested at the time. The operational claim is the one that matters for a CTO, and it has a simple shape. *At scale, for steady workloads, the rent can cost two to three times the cost of owning.*

## Two companies that left

*Public record · 2018 and 2024*

Two companies made the case with their own accounts rather than a model.

**Dropbox** began moving user files off Amazon's storage service onto its own infrastructure in 2015. Its IPO filing in 2018 disclosed that the project, which it called *Infrastructure Optimization*, had reduced operating costs by **$74.6 million over two years**.

**37signals**, the company behind Basecamp and HEY, announced in 2022 that it would leave the cloud. In October 2024 its co-founder David Heinemeier Hansson reported the result. The cloud bill had fallen from an original run rate of **$3.2 million a year to $1.3 million**, a saving of almost two million dollars a year, for new servers costing about $700,000. With storage to follow, the company projected savings of *"well over ten million dollars over five years."* He also addressed the objection every CFO raises: *"There were no hidden dragons of additional workload associated with the exit that required us to balloon the team."*

Read both cases carefully, because they are easy to over-sell. Both companies run large, steady, predictable workloads — file storage and web applications with well-understood demand. Both have unusually strong engineering teams. Neither is a bank with a fifteen-year virtualization estate and a regulator. They prove that the paradox is real for *some* workloads. They do not prove it for yours.

## Two taxes

*Briefing · February 2026*

The strategy deck gives the problem two names, and both are useful in a boardroom.

The **Innovation Tax** is *"money spent maintaining old environments instead of building new capabilities."* The deck states the paradox bluntly: *"You're paying a premium for the privilege of being stuck in an ecosystem that limits your ability to innovate."*

The **Success Tax** is the Meridian CFO's complaint, put as a principle: *"The better you perform, the more you pay — disproportionately so."*

The names are worth keeping. Several numbers the deck attaches to them are not, at least not without evidence. It claims a *"2–3x cost multiplier"*, *"15–25% annual increases"*, and that *"80% of IT budgets"* go to *"cloud maintenance vs. innovation."* None of these has a source in the deck. Its headline market figure has a problem too. The deck puts global cloud spend at *"$1.3T … in 2025"*. Gartner's published forecast for worldwide public cloud end-user spending in 2025 was **$723.4 billion**, up 21.5% on the year. The deck may be counting a wider market. Without a source, it cannot be checked.

That growth figure carries the chapter's most practical distinction. A bill can rise because prices rise, or because consumption rises. They call for opposite responses. Rising prices are a supplier problem, answered by negotiation or by leaving. Rising consumption is a management problem, and changing suppliers doesn't fix it. The Success Tax is real only for the share of growth that cost more than the business value it bought.

## Waste doesn't care who owns the hardware

The published waste figure from the prologue — 29% of cloud spend, by respondents' own estimate — is often used as an argument for leaving. It is really an argument for managing.

Owned hardware can be wasted too. Chapter 9 of this book measured a *warm standby*: a spare cluster kept running so that a user could claim one in under half a second instead of waiting for one to be built. It is a good design. It also holds eight processor cores and 14 GiB of memory idle, around the clock, waiting. On the platform this book describes, that standby is switched off by default, for exactly that reason.

Idle capacity costs money wherever it runs. The difference is who sends the bill, and whether anyone reads it.

## The promise that is worth testing

*Build log · June–September 2026*

If owning is going to compete with renting, it has to match the cloud's signature promise: capacity in seconds, not months. That is testable, and the rest of this book tests it.

On the platform in this book, a complete Kubernetes cluster — a control plane and a worker, each running as its own virtual machine — is built from a written declaration to both machines ready in a median of **33.7 seconds**, measured on 17 September 2026 (Chapter 7). Tearing one down takes **45.7 seconds** (Chapter 17; an earlier single-run figure of 18.9 s did not survive being measured five times). Claiming a pre-built standby took **467 milliseconds**, median of three runs measured 18 September 2026 — about 72 times faster than the cold build (Chapter 9).

Speed, it turns out, is not something only a landlord can offer. But the same chapters also show what it cost to get there. The first measured build took 50 seconds. Four attempts to make it faster failed. The two changes that worked took months of instrumentation to find (Chapter 8). And the dashboard reported the worker as ready for months while it was unable to run anything (Chapter 5).

That is the honest shape of the ROI question. Owning can match renting on speed. What it costs in people and attention is the part the rest of this book counts.

* * *

*Meridian · the following January*

Anita's quarter produced a spreadsheet, not a recommendation. It had one row for each major workload, and four columns.

*Shape of demand*: steady, seasonal, spiky, or unknown. *Cost trend against the revenue it supports*: flat, rising, or rising faster. *What leaving would take*: the proprietary services each workload depends on. And *what we could test*.

The CFO ran his finger down the first column. "Most of it is steady."

"Most of it is steady," Anita agreed. "Which means most of it is exactly where renting stops being the obvious answer. It doesn't mean owning is the answer. It means we're allowed to ask."

"So what do you recommend?"

"Nothing yet. I'm not going to move a production system on the strength of a venture capital essay and two companies' blog posts." She turned to the last column. "I recommend a test. Small, on hardware we own, measured the way we'd measure a supplier. If it can't build infrastructure as fast as we rent it, we stop."

## The ledger

- **The promise:** speed, elasticity and managed services are real, and are the right economics for uncertain or spiky demand.
- **The paradox (public record):** committed cloud spend averaging about 50% of cost of revenue among benchmarked public software companies; repatriation reported at one-third to one-half of the cloud cost for equivalent workloads (Andreessen Horowitz, 2021).
- **The evidence (public record):** Dropbox, $74.6 million saved over two years; 37signals, a cloud bill cut from $3.2 million to $1.3 million a year, with more than $10 million projected over five years.
- **Not supported:** the deck's 2–3x multiplier, 15–25% annual increases, and "80% of IT budgets" have no source. Its $1.3 trillion 2025 market figure does not match Gartner's $723.4 billion forecast.
- **Measured on owned hardware (build log):** a two-VM Kubernetes cluster ready in 33.7 s median; torn down in 45.7 s.

## Ask your team

1. **For each major workload, is its demand steady, seasonal, spiky or unknown — and who decided?**
2. **How much of last year's cloud growth came from higher prices, and how much from using more — and did the extra use pay for itself?**
3. **What would it cost, in services and in months, to move our steadiest workload off its current supplier?**

## Open the repo

- `docs/sub-60s-cluster-strategy.md` — how cluster build time was measured and brought down, including the dead ends.
- `scripts/phase-timings.sh` (`make phase-timings`) — the per-phase measurement behind the 33.7-second figure.
- `docs/warm-pool-strategy.md` and `ui/backend/handlers/pool.go` — the standby, and what it costs to keep warm.
- The strategy deck, slides 3–9 — its first two chapters. (Not a repository file; see Appendix H — Sources.)

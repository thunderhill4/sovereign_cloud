# Chapter 16 — The Fleet

> *Execute Phase 0–1 well and you have a credible demo. Execute Phase 2–3 well and you have a product.*
>
> — `Sovereign_Cloud_Agentic_Strategy.md`, §7

*Meridian · a Tuesday in September · the CTO's office*

The catalog ran to eleven pages, and Anita Rao had read it the way she read a hiring plan: job titles down the left margin, and a question mark next to most of them.

*VM Lifecycle Warden. Golden Image Curator. MetalLB IP-Space Accountant. Workload Onboarder. Chaos & Resilience Tester. RBAC Auditor. Idle VM Reaper. Right-Sizer. Incident Postmortem Drafter.*

"Twenty-nine," she said. "Twenty-nine agents."

"Grouped by where they work," Vikram said. "Three planes — the management cluster, the tenant clusters, the mesh between them — and four shared teams: security, cost, developer experience, observability. The roadmap says most of them are a day or two of configuration and prompt writing each."

"Nobody I hire is productive in a day or two." She turned to the last page. "How many of these work today?"

"Three. Sort of."

"Sort of?"

"They answer questions. None of them acts on anything."

Anita capped her pen. "Then before we talk about twenty-nine, tell me what it would take for one of them to be allowed to act. Not what it does. What it has to prove before it gets the job."

* * *

*Build log · April–September 2026*

## Twenty-nine job titles

The strategy document organizes its agents the way the platform is organized. An *infrastructure plane* on the management cluster: virtual machines, disk images, cluster provisioning, load-balancer addresses. A *tenant plane* inside the target clusters: workloads and their health. A *mesh plane* across clusters: service discovery, mutual TLS, traffic. Over all three sits a *control tower* — the UI's AI mode, drawn as *"Orchestrator + chat surface + approval gate"* — and beneath them a shared layer for security and compliance, cost, developer experience, and observability.

Counted section by section, the catalog lists twenty-nine personas: five infrastructure agents, four tenant agents, three mesh agents, five for security, four for cost, five for developer experience, and three for observability. Each entry reads like a job description — what it does, which tools it needs, what triggers it, a sample task, and the value it delivers. The first one sets the tone:

> *"Find all VMIs not in `Running` phase. For each, check: DataVolume ready? CPU/memory pressure on node? virt-launcher pod scheduled? Report root cause and, if safe and idle > 10 min, restart the VMI."*

Read the catalog for its verbs and a pattern appears. The VM warden *restarts*. The cluster provisioner *drives the rollout*. The idle-VM reaper *proposes a stop*. The onboarder deploys; the ServiceEntry generator generates; the CVE watcher *opens issues*. The value the catalog promises comes overwhelmingly from agents that change things.

Two sentences set the catalog's technical assumptions. The first: *"Every agent is one `SympoziumInstance`; every persona pack bundles the instances that belong together."* The second, from the build-versus-reuse section: *"build three custom skill packs (`kubevirt-ops`, `capi-ops`, `istio-ops`) and maybe 15–20 bespoke prompts. Everything else you can reuse."*

Chapter 13 has already told what happened to the first noun: `SympoziumInstance` was never reconciled, and the chart no longer ships it. The second never existed; the document's own April status note records that *"there is no `PersonaPack` CRD."* The three custom tool packs were never built.

## What exists on 17 September

Checked against the running platform on 17 September:

| Closest catalog persona | What exists | State on 17 September |
|---|---|---|
| Infrastructure-plane agents (VM warden, disk-image custodian, and others) | `cluster2-agent`, a generalist | Serving; read-only outside its namespace |
| Target-Cluster SRE | `target-cluster-agent` | Serving; its tools use the target cluster's administrator credential — read-only only by instruction |
| Service-Mesh Diagnostician | `mesh-sre-agent` (August) | Serving; read-only; no path to metrics |
| Idle VM Reaper | `cost-analyzer` (April demo) | Cannot be applied: written for a resource type the chart no longer ships |
| Incident responder | `incident-responder` (April demo) | Cannot be applied: same reason |
| Security and policy agents | An IaC security agent (March) — a separate Go service with policy rules and a local model, not a Sympozium agent | Not redeployed after the September rebuild |
| The warm-up "keep-alive agent" | The `ollama-warm` schedule | Stopped firing (Chapter 14) |

Asked to validate the two April personas without creating anything, the cluster refused both outright: *"no matches for kind 'SympoziumInstance'."* They were the first agents on the platform designed to *act* — one to stop an idle virtual machine on request, the other to diagnose and remediate a stopped worker — and they are now documents describing something that cannot be installed.

One more row belongs in the table, because the strategy document's reuse plan depends on it. The document recommends starting from Sympozium's `platform-team` bundle for the SRE and security personas. That bundle is installed. So are seven other example teams that ship with the chart, one of them seven personas strong. All eight are listed as ensembles in the agents' namespace, and all eight are switched off.

Three agents are running. Each is closest to one entry in the catalog but broader than it, and none is designed to act. Two are held to reading outside their own namespace by the permissions Chapter 14 examined. The third deserves a sentence of its own: `target-cluster-agent` reaches its cluster through the kubeconfig stored for it, and that kubeconfig belongs to `target-cluster-admin`. On the tenant cluster, the only thing keeping that agent read-only is its briefing.

## The roadmap against the calendar

The roadmap was written with a status snapshot dated 24 April, and it gave its phases durations: one week for Phase 0, two to three for Phase 1, two for Phase 2, two to three for Phase 3. Phases 0 through 3 — the credible demo and the product, in the epigraph's terms — came to seven to nine weeks.

On 17 September, twenty-one weeks later:

- **Phase 0 — tighten the foundation.** Delivered on paper. The manifests are bundled into one install. The two network policies ship — and, as Chapter 14 found, the sandbox policy selects no pods. The warm-up schedule ships, and has stopped.
- **Phase 1 — infrastructure-plane personas.** Not built as personas; `cluster2-agent` covers the plane as a read-only generalist.
- **Phase 2 — mesh plane.** The Diagnostician exists, since August. The ServiceEntry generator does not, and neither does the planned hook that would run a diagnosis when someone clicks an edge in the mesh view.
- **Phase 3 — tenant plane and cost.** `target-cluster-agent` exists. The onboarder, the generalized reaper, the right-sizer, and the per-persona model selector do not.
- **Phase 4 — security and developer experience.** Not built as agents.
- **Phase 5 — the orchestrator.** Not built. The mechanism it would stand on works, as the next section shows.

It would be easy to call this a slipped roadmap and move on. The repository's own history says something more useful about where the weeks went. The Sympozium labs in July discovered that the persona object had no controller, that tools did not reach runs, and that policy guardrails were weaker than their schema implied. The agent platform was upgraded three times — to 0.10.47, 0.10.57, and 0.10.75 — and each upgrade broke something that had to be diagnosed. The cluster speed investigation ran in June and again in September. The five-act service-mesh demonstration was built in August. And in September the whole platform was rebuilt on new Kubernetes versions.

None of that was on the roadmap. All of it was underneath it. A persona that takes *"~1–2 days of YAML + prompt engineering"* takes one or two days only on a platform that behaves the way its documentation says — and on this platform, finding out how it actually behaved was most of the work.

## A pipeline of two

Phase 5 imagines *"a 'planner' persona whose tools are the other agents"*: one chat window, a whole fleet behind it. Sympozium's closest building block is the `Ensemble`, which lets agents hand work to one another. Lab 06 builds the smallest useful version: an analyst answers a question, and a reviewer receives the analyst's answer with instructions to *"verify claims, flag anything unsupported, and produce the final improved answer."* The analyst runs on `qwen2.5:7b`, the reviewer on `llama3.2`.

It was run again on 17 September, on the current release, and the mechanics worked exactly as designed. Applying one object created two new `Agent` objects, each with its own memory Deployment. The kickoff prompt was delivered to the analyst. When the analyst's run finished, the reviewer's run started on its own, with the analyst's output passed along. About a hundred seconds from start to finish, with nothing triggered by hand.

The work did not go as well.

The analyst ran for 25.5 seconds, used 3,091 input and 859 output tokens, and called no tools. It produced a tidy, sectioned comparison of k3s clusters in KubeVirt VMs against bare Kind clusters — and one of its central claims was wrong. It described Kind clusters as *"leveraging the hypervisor's direct hardware access for containers."* Kind runs Kubernetes nodes as ordinary containers on the host. There is no hypervisor involved.

That is exactly what a reviewer is for. The reviewer ran for 6.8 seconds, used 5,877 input and 126 output tokens, and made one tool call. Its entire output was a heading, *"Tool Usage,"* a sentence saying it would search the workflow's memory, and a short block of Python calling a memory-search function. No verdict. No correction. No final answer.

The lab had recorded the same outcome in July, on an earlier release: the reviewer *"ended with 'Please proceed with breaking down the key trade-offs...' — a hand-back instead of a refined final answer."* Its conclusion holds word for word two releases later: *"the orchestration is deterministic, the output quality is not … don't assume the last persona in a pipeline produces the polished deliverable every time."*

Chapter 15 argued that an agent's answer is a claim, not evidence. A reviewer agent does not change that. Its output is another claim, from another small model, and a pipeline of agents adds each model's unreliability to the chain rather than a check on it. The planner in Phase 5 would be a longer chain.

## A model for each job, on hardware you own

A sovereign fleet runs on the hardware the organization owns, so the size of the fleet is bounded by that hardware rather than by an invoice. Lab 07 asks the platform's model-fit service what fits.

On 17 September it described the host as an AMD Ryzen AI 9 HX 370 with an NVIDIA RTX 4050 laptop GPU — 6 GB of video memory — and about 30 GB of system memory, and it scored 1,365 candidate models against that hardware. Its top picks were 30- to 36-billion-parameter mixture-of-experts models, rated a *"Good"* fit at an estimated 26 to 32 tokens per second.

The two models the fleet actually uses are more revealing:

| Model family, as rated | Fit on this host | Estimated speed |
|---|---|---|
| Llama 3.2, 3B Instruct (llama.cpp) | Perfect | ~60 tokens/s |
| Qwen 2.5, 7B Instruct (llama.cpp) | Marginal | ~36 tokens/s |

The model this hardware suits best is `llama3.2`: the one that invented a list of nodes in Chapter 15, and the reviewer that handed back instead of reviewing above. The model that used its tools more reliably in lab 09, `qwen2.5:7b`, is the one rated a marginal fit. On this platform, the best fit for the hardware is not the best fit for the job, and a fleet chosen by a fit score alone would get worse.

The host has fourteen models downloaded. The fleet uses two. The per-persona model selector the roadmap planned for Phase 3 was never built, and every persona that needs a different model will compete for the same six gigabytes.

## The auditors' job descriptions

Two entries in the catalog deserve a closer reading, because they are the agents a CxO would reasonably want first: the ones that watch the others.

The **RBAC Auditor** *"scans every `ClusterRoleBinding` + `RoleBinding` across both clusters. Flags wildcards, flags service accounts bound to `cluster-admin`, proposes least-privilege replacements."* The **Secret Hygiene Bot** *"flags in-cluster secrets older than 90 days, secrets mounted by more pods than necessary, and literal credentials in ConfigMaps."*

Now hold them against the most serious finding in this part of the book. Chapter 14 found that every run of `cluster2-agent` holds permission to read every Secret in its namespace — including an administrator credential for the target cluster. The role that grants it lists its resources and verbs explicitly: no wildcards. No agent is bound to `cluster-admin`. The Secret is not old, and it is mounted only by the agent it belongs to.

As written, the auditor's flagging rules would not have raised it, and neither would the hygiene bot's. Both job descriptions came from general best practice — sensible, and blind to the platform's actual risk.

The lesson generalizes. Write an auditor's job description against the incidents you have already had, then score it the way Chapter 15 scored the triage agent: plant a copy of the real finding, fix a strict pass rule, run it several times, and record what it costs. An auditor that cannot find the problem you already know about will not find the one you don't.

## What a persona needs before it joins

Anita's question — what an agent has to prove before it gets the job — has an answer, and it is the rest of Part IV compressed into a checklist. The catalog's *one or two days of configuration and prompt writing* is only the first line.

1. **A job narrow enough to score.** A planted fault, a strict pass rule, several runs (Chapter 15).
2. **A model chosen for the job's behavior on this hardware** — not by fit score alone, and not by reputation (lab 07, lab 09).
3. **Tools delivered as named sidecars, with permissions checked against a live run's identity** — not against the documentation (Chapters 13 and 14).
4. **Every "must not" enforced below the model**, or explicitly declared unenforceable and verified instead (Chapter 14).
5. **Evidence attached to every claim that drives a decision** (Chapter 15).
6. **A known cost per task**: tokens, seconds, and the model it keeps loaded (Chapters 14 and 15).
7. **A record that outlives the cluster** (Chapter 13).
8. **For any agent that acts: an approval step between its decision and the change.** The strategy document draws one in its control tower. The platform's only working version today is a one-entry allowlist in the UI's backend.

Twenty-nine personas times eight requirements is a program, not a sprint — and it is the honest size of the fleet the catalog describes.

* * *

*Meridian · the following Monday*

Anita's decision fit on one line of the catalog's cover page, written across the title.

*Three, not twenty-nine.*

Underneath, smaller: *Keep the three. Next two are the auditors — after their job descriptions include the finding from Chapter 14, and after each of them catches a planted copy of it, three times out of three. Nothing that acts until the approval step exists.*

Vikram read it. "The auditors first. Not the ones that save money?"

"The ones that save money need permission to change things," she said. "Before anything gets that permission, I want something watching the permissions." She handed the catalog back. "Hire the auditors first. Then check that they'd have caught what we already know."

## The ledger

- **Designed:** 29 personas across three planes and four shared groups; a six-phase roadmap; three custom tool packs.
- **Running on 17 September:** three agents — generalists, one per plane, none designed to act. Two are read-only outside their namespace by permission; the third holds an administrator credential for the tenant cluster and is read-only only by instruction.
- **Stranded:** two April personas written for a resource type the chart no longer ships; a March security service not redeployed after the rebuild; eight example ensembles installed and switched off.
- **Built of the three custom tool packs:** none.
- **Measured on 17 September:** a two-agent pipeline that orchestrated flawlessly in about 100 seconds (analyst 3,950 tokens, reviewer 6,003) and whose reviewer produced no review; hardware-fit ratings that favor the less reliable of the fleet's two models.
- **Roadmap:** Phases 0–3 estimated at seven to nine weeks from 24 April; at twenty-one weeks, Phase 0 delivered on paper and one persona from Phase 2 running.

## Ask your team

1. **For each AI agent we plan to add, what fault would we plant to score it, and what counts as a pass?**
2. **Which of our planned agents act rather than report — and what stands between an agent's decision and the change?**
3. **Would our planned auditors have caught the last real incident we had?**

## Open the repo

- `Sovereign_Cloud_Agentic_Strategy.md` — §2 (the three planes), §3 (the catalog), §5 (the roadmap), §6 (build versus reuse).
- `06-sympozium/labs/06-ensemble/` — the two-agent pipeline, and its note on run-to-run variance.
- `06-sympozium/labs/07-model-fit/run.sh` — what fits this hardware.
- `06-sympozium/cost-analyzer.yaml`, `06-sympozium/incident-responder.yaml`, and `docs/superpowers/specs/2026-04-21-sympozium-sample-agents-design.md` — the April personas.
- `docs/superpowers/specs/2026-03-29-iac-security-agent-design.md` and `make security-deploy` — the security service.
- `kubectl get ensembles -n sympozium-system` — the chart's example teams, installed and switched off.

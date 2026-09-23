# Own Your Cloud
### From Cloud Tenant to Infrastructure Owner — a sovereign-Kubernetes story told in chapters

*Working title. Alternates: **The Thirty-Four-Second Cluster** · **Reclaiming the Margin**.*

---

## Who this book is for

Two readers, one story:

- **The CxO who signs the cloud bill.** Parts I and V, plus every chapter's closing blocks, can be read on their own. No commands required.
- **The engineer who would build the alternative.** Parts II–IV are a build-along: every chapter ends in a state that can be reproduced from this repository, with the numbers that were actually measured.

## How the chapters are told

- **A composite enterprise, "Meridian"** — fifteen years of VMware estate, a cloud bill rising faster than revenue. A CTO and a platform lead recur throughout. Each Part opens in the boardroom (the stake, in money and risk) and cuts to the build log (what actually happened).
- **Every chapter closes with three short blocks:**
  - **The ledger** — what it cost, what it saved, what it proved. Numbers only where they were measured.
  - **Ask your team** — three questions a CxO can put to their own organization the next morning.
  - **Open the repo** — the path, `make` target, or lab that reproduces the chapter.
- **Honesty is the voice.** Dead ends are recorded with their numbers. Unsupported combinations are labeled as such. Nothing is scripted output. This is the book's difference from a vendor whitepaper.

---

## Prologue — The Invoice

Meridian's CTO reads the monthly bill for its slope, not its total. The briefing deck's waste figures — 35% over-provisioned, 28% idle, 18% orphaned, a 40% markup — meet the published one (29%, self-estimated), and the book explains how it keeps score. The prologue ends on the question the book answers: what would it take to **own** this instead of rent it?

> Source: the strategy deck *Reclaiming Margins with Sovereign Cloud* (February 2026), ch. 1–2. See Appendix H.

---

## Part I — The Bill
*Why. The CxO arc, adapted from the strategic briefing deck. Business first; the machinery arrives only at the end of the Part.*

**1. The Great Cloud ROI Myth**
The promise versus the P&L. The **Innovation Tax** — *"paying a premium for the privilege of being stuck"* — and the **Success Tax** — *"the better you perform, the more you pay."*
> Source: deck ch. 1.

**2. The Landlord Problem**
Renting versus owning. *"You're playing in the vendor's sandbox, and they set the rules on how high you can build."* Control-plane opacity: *"you cannot customize the brain of your cluster."* The apartment-rental analogy, taken seriously.
> Source: deck ch. 2.

**3. The Sovereign Imperative**
Data residency and the buyers who actually ask for it — *"enterprise Europe / India / public sector."* Sovereignty as a hard constraint, not a feature. **Local-LLM-first** introduced as a business decision before it becomes a technical one.
> Source: deck ch. 3; `Sovereign_Cloud_Agentic_Strategy.md` §4 "Non-Negotiables", §7.

**4. Kubernetes That Runs Kubernetes**
Cluster API explained for a board: a management cluster that manufactures workload clusters. The managed-Kubernetes spectrum (Tanzu vs. CAPZ vs. Kubermatic). Why an organization with a VMware estate should care that **VMs are first-class citizens** here (KubeVirt). The bridge from money to machinery.
> Source: deck ch. 4, slides 14–16; `README.md` architecture diagram; the recorded factory demonstration for figures.

---

## Part II — The Factory
*The cluster-provisioning story. Every number in this Part was measured on this repository.*

**5. The First Cluster**
From `make clean` to a k3s cluster whose control plane and worker *are* virtual machines — CAPI, CAPK, CDI, and KubeVirt explained as each step happens on screen. `show-cluster.sh` as the victory lap.
> Source: `demo.sh`, `show-cluster.sh`, stages `00-prereqs` → `04-verify`.

**6. Ten Minutes Becomes Two**
Golden images: bake, containerDisk, a local registry. And the trap that *`172.18.0.2:5000` is an alias only containerd can resolve* — the address that drifts, and the script that keeps it honest.
> Source: `bake-common.sh`, `build-containerdisk.sh`, `scripts/fix-registry-hosts.sh`, `CLAUDE.md` (registry alias pitfall).

**7. Thirty-Six Percent Before the Kernel**
Instrumentation first. Why `phase-timings.sh` exists — *"two optimisation dead ends each cost a full bake cycle to disprove."* Measuring a cluster you cannot log into. The measured phase budget, and the sentence the whole speed story hangs on: *"36% of the wall clock (16.6 s) elapses before the guest kernel starts."* And the day the stopwatch itself was caught stopping early — fooled by a ghost node baked into the golden image.
> Source: `docs/sub-60s-cluster-strategy.md` (phase-level measurement), `scripts/phase-timings.sh`.

**8. Two Lines and a CPU Quota**
Four documented dead ends — `:preinit`, `:warm`, gating the worker on control-plane readiness (*"53.1 s median, a 3.1 s REGRESSION. The premise was wrong"*), the no-route-to-host dials — and the two one-line fixes that delivered −31%: `advertise-address` (50.0 → 42.7 s) and `supportContainerResources` (42.7 → 34.5 s). Thesis: *"every attempt to make the image smarter failed, while the two things that worked were a bad advertised address and a CPU quota."* With the measurement caveat that limits every claim (N=3 against a 6 s spread; N≥7 or a quiet host).
> Source: `docs/sub-60s-cluster-strategy.md` §3, §6; `scripts/configure-kubevirt-perf.sh`.

**9. Warmth Is Mandatory**
The warm pool: a standby cluster **claimed in 467 ms** (18 September; was 194 ms in June, unmeasured since — re-measured after fixing the pool's own ghost-node readiness bug) instead of built in ~34 s, and the stable-pod-IP constraint that makes it tractable. Then the same rule applied to models — llama3.2's **28–30 s CPU cold start** is why *"Warmth is mandatory"* is an architecture constraint shared by VMs and LLMs alike.
> Source: `docs/warm-pool-strategy.md`, `ui/backend/handlers/pool.go`, `Sovereign_Cloud_Agentic_Strategy.md` constraint #5.

---

## Part III — The Fabric
*The mesh. Before/after storytelling: `05-istio/` is the confession, `07-istio-advanced/` is the fix.*

**10. Hardcoded IPs and Other Confessions**
The "before" picture: cross-cluster discovery hand-plumbed with a `ServiceEntry`, a hardcoded MetalLB IP, and a NodePort proxy. It worked. It was also a lie waiting to be found out. The eight-step demo from the cross-cluster mind map is the chapter's spine; the mind map is its figure.
> Source: `05-istio/cross-cluster-demo.sh`, `05-istio/cross-cluster-mindmap.html`.

**11. One Label**
Acts 1–3. The Gateway API ownership split: a `rogue-tenant` route the API server *accepts* and the Gateway *refuses* — and a second refusal nobody checked, of the demo's own tenant. Waypoints enforcing L7 authorization on SPIFFE identity with no sidecar in any pod. Multicluster failover where `istio.io/global: "true"` replaces everything in chapter 10: measured by status code, it works — but the act's own tally prints *"success=20 failed=0"* with both clusters down, and healthy traffic splits half-and-half across sites.
> Source: `07-istio-advanced/act1-gateway`, `act2-waypoint`, `act3-multicluster`; README "Gotchas".

**12. The Model Is Just Another Service**
Act 4: agentgateway puts one control point in front of the host GPU's models — at no measurable latency cost, but with no lock, no token meter, and none of the fleet's agents routed through it. The repo's own *"HONEST LIMITATION, worth saying out loud on stage"* describes a tier that never ran. Act 5: Kiali, Prometheus, and the UI's Visual mode — where a topology view labelled *live* draws fourteen things that no longer exist. Seeing the mesh is a board-level deliverable; so is knowing what a screen actually checked.
> Source: `07-istio-advanced/act4-ai-gateway`, `act5-observability`; `ui/frontend/src/components/visual/`.

---

## Part IV — The Brain
*Agentic operations. Authored fresh — the strategy document is the backbone.*

**13. An Agent as a First-Class Object**
Sympozium: an `Agent` is a Kubernetes object, an `AgentRun` is a one-shot job, a serving agent is a drop-in OpenAI-compatible endpoint. The three UI modes — SRE, AI, Visual. The thesis: *"You are not building 'Claude for Kubernetes.' You are building a sovereign, air-gap-capable, VM-native, mesh-aware agentic control plane — a category that barely exists."*
> Source: labs 01–02, `06-sympozium/cluster2-agent.yaml`, `ui/backend/handlers/ai.go`, strategy doc §1.

**14. Rules the Model Cannot Break**
The eight architecture constraints as a story: local-LLM-first, a pinned model endpoint, MetalLB discipline, an egress allowlist, mandatory warmth, an append-only audit, namespace-as-tenant isolation, sandboxed chat. Policy and tool gating and their *real* limits — default-deny that is not enforced, network policy that is inert on this CNI. The hard lesson: *"Do not rely on the systemPrompt to enforce that… enforce it below the model."*
> Source: `Sovereign_Cloud_Agentic_Strategy.md` "Architecture Constraints"; lab 04; `CLAUDE.md` (mesh-sre-agent).

**15. Trust, but Verify**
Seven scenes with a live agent, and the ethic behind them: *"Nothing here is scripted output. If a call fails, the failure is what gets recorded."* The day the deterministic check was checking the wrong identity. A model that confidently reported nodes that did not exist, and the pod that mounted the real kubeconfig and returned the truth. A scored triage — **3 of 3 correct root causes, 6–10k tokens, 14–29 s** — and a real model comparison in which one run marked *Succeeded* never ran a single command (qwen2.5:7b at 11.0 tokens/s against llama3.2 at 4.0).
> Source: `06-sympozium/demo-mesh-sre.sh`, `docs/demo/mesh-sre-agent-demo.mp4`, labs 09–10, `CLAUDE.md` (0.10.75 entry).

**16. The Fleet**
The three-plane agent fleet — Infrastructure, Tenant, Mesh — plus Security, FinOps, DevEx, and Observability personas. Ensembles, model fit, the cost-analyzer and incident-responder, the IaC security scanner. The six-phase roadmap and its promise: *"Execute Phase 0–1 well and you have a credible demo. Execute Phase 2–3 well and you have a product."*
> Source: strategy doc §2, §3, §5, §6; labs 06–07; `docs/superpowers/specs/`.

---

## Part V — The Dividend
*Back to the boardroom.*

**17. The Same Cluster Twice**
What sub-minute bring-up actually changes, measured: **81.96 s** from the destroy command to a served request (N=5, boundaries pre-registered before the first run). The build reproduces at 34.39 s; the teardown, published from a single run at 18.9 s, turns out to be **45.73 s**, and the API objects outlive the machines. Four consequences the platform can demonstrate — recovery as a command, debugging with a control group, upgrades as rebuilds, drills as routine — and four it cannot, because the fixed identity that buys the 34-second build forbids concurrent clusters. Proved on purpose: an admin credential from five rebuilds ago opens a 32-second-old cluster with cluster-admin.
> Source: `measurements/ch-recovery-drill.sh` and its raw data; DORA; Palantir on ephemeral compute.

**18. From the Lab to the Data Centre**
The honest mapping: what a single-host platform transfers to a production data centre, and what it does not. High availability and failure domains; storage and backup; networking beyond one Docker bridge; capacity planning built on Chapter 20's measured reservation-versus-use gap; secrets, identity and GitOps — the last presented as the direct answer to Chapter 19's "state that lives only in a running cluster"; observability retention; and a support strategy for the deliberate decision to run outside a vendor's tested range. Every claim is labelled **design guidance**, not measurement. Nothing in this chapter was built.
> Source: the book's own chapters, re-read as requirements. No new measurements.

**19. The Day We Upgraded Everything**
Kubernetes 1.35 → 1.37, Istio 1.30 → 1.31, k3s 1.31 → 1.37; both clusters destroyed and rebuilt from nothing. What nobody had scripted (the clusters themselves, KubeVirt, CDI) — and the base-image script that existed all along but that nobody could find. The dead image that reported success. The version variable that had never controlled anything. The Istio release rated for "up to 1.36" that passed 22 of 22 checks anyway. And a cluster that seemed to come back ten seconds faster, until the stopwatch was checked: 33.7 s for both nodes, unchanged. The moral: *reproducibility is the real sovereignty*, and institutional memory is a balance-sheet asset.
> Source: `CLAUDE.md` (2026-09-16 entries), `02-capi-init/init-management-cluster.sh`, `07-istio-advanced/README.md`.

**20. Reclaiming the Margin**
The briefing deck's $12.5M → $8.2M five-year TCO, taken apart line by line: savings that add up exactly with no line for what owning costs, 70% of it in lines no invoice records, and a 5–7 year payback that fits only at the expensive end. Then what the platform measured — reservations against use, 0.33 kWh per million generated tokens, the effort in the repo — a TCO worksheet, the seven-step Sovereign Cloud Journey, a **"Cloud Owning Culture"**, and what to ask your vendors, including the one you become.
> Source: deck ch. 5, slide 19; the unbranded sovereign-cloud flyer, v2 (Appendix H).

## Epilogue — Own Your Cloud
Meridian's CTO, twelve months later. Short. It ends on the flyer's line.

---

## Appendices

- **A. The repo, stage by stage** — `00-prereqs` through `07-istio-advanced`, one paragraph each, and the `make` targets that matter.
- **B. Addresses that must not move** — the MetalLB IP allocation table and pool discipline.
- **C. The failure-mode catalog** — the pitfalls and the chapters' findings, reorganized by *symptom*: four causes behind one identical "exit 2, zero log output" crash; scripts that reported success; statuses wrong in both directions; the registry alias; the admission race; per-run identities.
- **D. The measurement protocol** — `phase-timings.sh`, why N≥7 on a noisy host, counting nodes by name rather than by number, and the figures as last measured: teardown 45.7 s, bring-up 33.7 s for both nodes.
- **E. The ten labs** — quick reference, with the ✅ / ⚠️ honesty markers preserved.
- **F. A glossary for the boardroom** — CAPI, KubeVirt, ambient mesh, waypoint, SPIFFE, agentic control plane, sovereign cloud — one plain-English sentence each.
- **G. Key files, quoted** — the load-bearing manifests and code, verbatim with their repo paths, so the book can be checked without the companion repository: the warm manifest's seven objects, the two one-line speed fixes, the two ways to count a failover, the egress allowlist, an agent as an object, the `K3S_VERSION` that controlled nothing, and the dead-image patch.
- **H. Sources** — the companion repository; the author's briefing material, described rather than linked because no path would resolve for a reader; and the public record, with each citation's verification status.

---

## Production notes

**Must be written from scratch (no existing source):**
- All of Part IV in CxO-readable prose — the repo holds the technical truth, nothing holds the narrative.
- The **TCO bridge** in ch. 20: connecting 24-second clusters, warm pools, and local models to dollars. The deck's figures are industry-generic; Meridian needs real cost data to land.
- The Meridian framing device and the recurring chapter closers.
- A boardroom-level explanation of KubeVirt and "why VMs still matter" (ch. 4) — absent from both the deck and the repo.

**Figures to source:**
- Deck slides 14–16 → ch. 4.
- `05-istio/cross-cluster-mindmap.html` (render to SVG) → ch. 10.
- The recorded factory demonstration (8m56s) and agent demonstration (2m50s) → stills for ch. 5 and ch. 13 (Appendix H).
- `docs/demo/mesh-sre-agent-demo.mp4` → frames for ch. 15.
- `phase-timings.sh` tables → charts for ch. 7–8 and Appendix D.
- `README.md` ASCII architecture → redraw once, reuse across Parts.

**Suggested writing order:**
1. Part II and Part IV first — the richest, best-documented material; writing them fixes the voice.
2. Part III — the acts are already scripted stories that end in assertions.
3. Part I — adapt the deck; keep it short so engineers don't bail.
4. Part V, prologue, epilogue last — they need real cost numbers and finished Parts to point back to.

**If it needs to be shorter:** ~10 chapters by merging 1+2, 5+6, 7+8, 11+12, 15+16, 17+18. The closers work with or without Meridian.

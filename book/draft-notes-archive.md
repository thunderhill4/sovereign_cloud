# Draft notes — archive

Collected from the manuscript on 18 September 2026, immediately before the
`Draft notes` sections were stripped for publication (Task 8).

The manuscript itself no longer carries them. They are preserved here, and in
full context at the git tag `with-draft-notes`:

```
git show with-draft-notes:chapters/<file>.md
```

Every unresolved item the book knows about is in this file. It is the punch
list for publication.

---


## 00-prologue-the-invoice

- **The deck's four numbers** are on slide 9 of the strategy deck (February 2026; Appendix H). Neither the slide text nor the notes carry a source. If a source exists, add it and soften the paragraph; if not, keep the check as written.
- **Flexera figures.** The 2026 press release (18 March 2026) gives 29% waste, "the first time in five years" it rose, and 85% naming cost management a top challenge, from 750+ respondents: https://www.flexera.com/about-us/press-center/flexera-finds-cloud-value-is-rising-while-ai-waste-grows. The 2025 figures (27%, 84%) are from https://www.flexera.com/about-us/press-center/new-flexera-report-finds-84-percent-of-organizations-struggle-to-manage-cloud-spend. The 2025 release called cost management the top challenge for "the third year in a row". The 2026 release says only that it "remains a top challenge", so the chapter says that.
- **Meridian's financials are deliberately unquantified.** Every earlier chapter promises that figures in Meridian scenes were measured, and nothing here was. Keep the invoice qualitative, or clearly mark any invented figure.
- **Timeline.** The prologue is September 2025, so the epilogue ("twelve months later") lands in September 2026, the book's present. This is consistent with Chapter 5 (pilot funded in February 2026) and the repository's first commit (11 February 2026).
- **Hardware** was read on 17 September (`lscpu`, `free -g`, `nvidia-smi`); it matches Chapter 16.
- **Flyers.** One of the three flyers in the author's briefing material carries a real company's branding. The book quotes only the unbranded one, and Appendix H describes it without naming the folder it came from.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 01-the-great-cloud-roi-myth

- **Sources.** Andreessen Horowitz, 27 May 2021: https://a16z.com/the-cost-of-cloud-a-trillion-dollar-paradox/ (paradox sentence; "averaged 50% of COR"; "one-third to one-half"; "$100B" across 50 companies; "over $500B"). Dropbox $74.6 million from its 2018 S-1, as reported by GeekWire: https://www.geekwire.com/2018/dropbox-saved-almost-75-million-two-years-building-tech-infrastructure/ — cite the S-1 directly before publication. 37signals, 17 October 2024: https://world.hey.com/dhh/our-cloud-exit-savings-will-now-top-ten-million-over-five-years-c7d9b5bd. Gartner, 19 November 2024: https://www.gartner.com/en/newsroom/press-releases/2024-11-19-gartner-forecasts-worldwide-public-cloud-end-user-spending-to-total-723-billion-dollars-in-2025.
- **Verified 18 September 2026 against the primary sources:** the a16z article (date, authors, the 50%-of-COR benchmark, one-third-to-one-half, $100B/>$500B) and the 37signals post (17 October 2024, $3.2M → $1.3M, ~$700,000 of servers, "well over ten million dollars over five years", the "no hidden dragons" sentence). Both are quoted as published. See Appendix H for the full status table.
- **Still unverified:** the Gartner 2025 forecast (its press release returns HTTP 403 to direct retrieval) and the Dropbox figure (cited via GeekWire, not the S-1 itself).
- **37signals' 2022 announcement date** is from general knowledge; confirm it against the original post before publication.
- **Counter-arguments exist.** The a16z paper drew public rebuttals (for example a VentureBeat opinion piece). The chapter's caveats cover the substance; decide whether to name one.
- **Deck figures not repeated as fact.** From slides 4 and 7: $3.86 ROI per $1, 94% of companies using cloud, 67% delaying deployments, 46% revenue loss, 30% fines. None has a source.
- **"The price of an hour of computing didn't go up much"** is Anita's claim, in fiction. It was not verified, and it should stay in the character's voice unless it is sourced.
- **The standby's cost** (8 cores, 14 GiB) is from Chapter 9. **Resolved 18 September** — the claim time was re-measured (was 194 ms in June, unmeasured since; now 467 ms median, three runs); this chapter's figure is updated.
- **The Meridian scenes are fiction.** No Meridian figure is quantified.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 02-the-landlord-problem

- **Sources.** AT&T/Broadcom: Channel Futures, https://www.channelfutures.com/channel-business/att-vmware-fight-impasse-over-1-050-price-increase; settlement, CIO Dive, https://www.ciodive.com/news/broadcom-att-vmware-settlement-licensing-support-lawsuit/733763/. Cite the court filing directly if possible. The $61 billion figure and the end of perpetual licences are as reported in that coverage. The November 2023 close date is from general knowledge; confirm it.
- **EU Data Act — corrected 18 September 2026.** An earlier draft said cost-only charges applied "from January 2024". That is the date the Regulation *entered into force* (11 January 2024); its obligations became **applicable on 12 September 2025**, and several law-firm commentaries agree on that split. The 12 January 2027 full-withdrawal date is corroborated across the same commentaries. **Still to do before publication: verify both dates against the text of Regulation (EU) 2023/2854 itself on EUR-Lex, not against commentary,** and have counsel confirm the characterisation of "only at cost". Original note follows. The dates were cross-checked in several legal commentaries, e.g. https://kempitlaw.com/insights/the-end-of-switching-charges-commercial-impact-and-compliance-priorities/. The provider waivers (Google January 2024, AWS March 2024, Microsoft afterwards) come from secondary coverage; confirm each against the provider's own announcement, and check the exact scope and conditions.
- **Managed Kubernetes wording.** The deck names AKS. The chapter deliberately generalizes ("the supplier chooses which settings"). Do not claim any specific managed service cannot set any specific flag without checking current documentation.
- **The moving-tag crash** ("~5,680 restarts") is recorded in `CLAUDE.md`; the chapter says "thousands" to avoid citing a figure it did not re-measure.
- **Meridian's letters are fiction,** modelled on public events. Keep the virtualization supplier unnamed in Meridian scenes.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 03-the-sovereign-imperative

- **Legal facts** (GDPR 2018; Schrems II, July 2020; replacement framework July 2023; CLOUD Act 2018; India DPDP Act 2023) are stated from general knowledge. Have counsel review them, and cite primary sources before publication. Data Act dates: see Chapter 2's notes.
- **Market figures.** The deck's $96.77 billion (2024) / 23.8% appear to come from an earlier edition of a market report. Grand View Research's current figures ($117.53 billion 2025, $648.87 billion 2033, 24.1% CAGR 2026–2033) come from a search-result summary; the report page returned HTTP 403 to direct fetch. Verify before publication: https://www.grandviewresearch.com/industry-analysis/sovereign-cloud-market-report.
- **Committed keys.** On branch `perf/target-cluster-sub-40s` (17 September), `git ls-files 03-target-cluster/warm-ca` lists private keys. A project memory note says the keys were removed from tracking in a PR; that is either on another branch or out of date. Check `main` before publishing the sentence.
- **Provenance** was checked only by searching the agent manifests in `06-sympozium/` for "provenance" and "digest". The platform may record the model name on runs elsewhere (Sympozium records `spec.model` on the Agent). The weights digest is the missing piece. Soften to "not recorded per run" if run records carry the model name.
- **"Fourteen models"** is the host's `ollama list` on 17 September (Chapter 12).
- **Model sizes.** Ollama reports `llama3.2` at 3.2B parameters and `qwen2.5:7b` at 7.6B (both Q4_K_M), read with `/api/show` on 17 September. The chapter rounds them to 3 and 7 billion. The confabulation is llama3.2's (Chapter 15); the refusal failure and the 3-of-3 triage are qwen2.5:7b's (Chapters 14 and 15).
- **Meridian's tender is fiction.** Section eleven's wording is invented, but modelled on real public-sector requirements. Do not quote it as a real tender.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 04-kubernetes-that-runs-kubernetes

- **Versions and objects** were read live on 17 September from cluster2's controller images, the KubeVirt/CDI status, and `kubectl get` on the Cluster API resources.
- **`Available=False`** was read at about 09:00 on 17 September, about 143 minutes after the cluster was built. The target cluster served Chapter 10's measurement at 08:10; Chapter 11's measurements ran on cluster1 and cluster2, not on the target cluster. The cause is not investigated. The KThreesControlPlane also showed `Ready=False`, `ControlPlaneComponentsHealthy=False`, and `initialized=true`. It is plausibly a health check that expects separate control-plane component pods, which k3s does not run. Check the cluster-api-k3s issue tracker before implying a cause.
- **"No VMware migration tooling"** was checked by searching the repository (excluding `book/` and `.worktrees/`) for vsphere, forklift, virt-v2v, vmdk and ovf. The only match was an unrelated substring in `scripts/render-demo-video.sh`.
- **Product descriptions** (Tanzu, CAPZ, KKP) are quoted from the deck, not independently verified. KKP is a Kubermatic product. Keep the chapter from endorsing or disparaging any vendor beyond the deck's quoted labels.
- **The alternatives scan ("What the slide leaves off") is entirely unverified,** and says so in the body. Harvester, OpenStack, Talos, RKE2, OpenShift, Anthos, EKS Anywhere, OpenTofu/Terraform, Forklift and virt-v2v are described from each project's public self-description, from general knowledge, as of September 2026. Nothing was installed or tested. Before publication: re-check each description against current documentation, confirm Harvester's current vendor relationship (SUSE) and that it still ships KubeVirt, and confirm Forklift's current name and scope. If any description cannot be confirmed, cut the line rather than soften it.
- **Deliberately not ranked.** The section names no winner and gives no comparison table, because a table implies criteria that were never applied. If a reviewer asks for one, the honest version would require actually running them.
- **The README diagram** shows `mc-demo`, the `sample` namespace, VM addresses 10.244.0.60/.61 and the `.216` proxy. Redraw it for the book's figure 4.1 rather than reproducing it.
- **Figures.** 4.1: management cluster → declaration → workload cluster on VMs. 4.2: the deck's spectrum, relabelled neutrally. 4.3: the two gauges side by side — `Available=False` here and `Ready` in Chapter 5.
- **The Meridian board scene is fiction.** It sets up Chapter 5's "demo that won the pilot its funding in February".
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 05-the-first-cluster

- **The worker taint is the most important finding in this chapter,** and it reaches beyond it. Verified on the cluster running on 17 September only. The setting is in the static worker bootstrap Secret of `03-target-cluster/target-cluster-warm.yaml` (and in its template); the control plane sets `disableCloudController: true`. The fix is untested. Also check `target-cluster-parallel.yaml` and the legacy sequential manifest, whose worker bootstrap is generated by the k3s provider rather than written by hand.
- **Consequences for other chapters.** Chapters 7, 8, 9, and 18 measure the time until both nodes report `Ready`. Those measurements stand as measurements of `Ready`, but "both nodes" does not mean "both usable." Add a sentence to Chapter 7 (and to the time-to-ready definition in the brief) once the fix is tested.
- **`demo.sh`** fails at the registry check because it dials the alias over plain HTTP; use the registry's real address (as `ui/backend/handlers/registry.go` does) and refresh its narration — VM sizes, the `:latest` image, containerDisks.
- **`show-cluster.sh`** narrates *"curl ${SVC_IP}"* but runs `wget http://localhost` inside a pod; its DataVolume section is obsolete; it trusts any kubeconfig already in `/tmp`.
- **Identity across rebuilds.** The `/tmp` kubeconfig (written 16 September) and the current one share a CA fingerprint; the warm image's fixed identity is demo-only by design. Worth one line in Chapter 14's scorecard.
- **Object count.** "About eighteen" counts, on 17 September: 1 KThreesConfig, 1 MachineSet, 2 Machines, 2 KubevirtMachines, 2 VirtualMachines, 2 VMIs, 2 launcher pods, 1 Service, and 5 Secrets (control-plane bootstrap data and its user data, worker user data, kubeconfig, SSH keys).
- **Meridian scene** is fiction; the demo output it shows is real, from 17 September.
- **Figures.** 5.1: the ownership chain, with the missing link drawn dashed. 5.2: the six layers as nested boxes.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 06-ten-minutes-becomes-two

- **"Ten minutes becomes two."** The chapter's title is the team's stated estimate, repeated in `README.md`, `docs/DEVELOPER.md`, `docs/USER_GUIDE.md` (which says 2–5 minutes), and the strategy document. No measurement is recorded. Find the original numbers, or keep the chapter's framing of it as an estimate from before the instrument existed.
- **containerDisk header claim.** `build-containerdisk.sh` promises 30–60 s spin-up; `docs/sub-60s-cluster-strategy.md` records a June baseline of 90–150 s (sequential worker). Update or remove the header claim.
- **Image dates** were read from each tag's image configuration in the registry: `:latest` 2026-03-03, `:preinit` 2026-04-18, `:warm` 2026-09-16. The `:preinit` date predates the June pre-init experiments in Chapter 8; confirm which bake produced it.
- **Checked live on 17 September:** address ownership on the Kind network; a plain HTTP request to the alias (connection failed) versus the registry's real address (`200`); the translation file's contents; a cold pull of `ubuntu-noble-k3s:preinit` (1.29 GB, 4.2 s) and a cached pull (0.1 s), after which the image was removed from the node; which tags were cached (only `:warm`).
- **`CLAUDE.md` is stale on the alias.** It says `172.18.0.2` belongs to `cluster2-control-plane`; after the rebuild it belongs to `cluster1-control-plane`.
- **Pull policy.** No manifest sets `imagePullPolicy`; the June notes say the `:latest` containerDisk is pulled on every start. That follows Kubernetes' usual default for `:latest` tags, but was not re-verified for KubeVirt containerDisks here.
- **The Meridian regulator's questionnaire** is fiction. The answer to it — nothing downloaded at boot, one bake per image — is the platform's real design.
- **Figures.** 6.1: the pipeline — base image → bake VM → baked volume → `qcow2` → container image → registry → node cache → VM. 6.2: the alias — three containers on one network, and the translation file that makes one address mean another.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 07-thirty-six-percent-before-the-kernel

- **This discovery corrected earlier drafts in this pass.** Chapter 8's postscript and Chapter 19's measurement section, ledger, and Meridian memo previously reported 23.9 s as a both-nodes time; the table of contents did too. All now carry the corrected 33.7 s. The 23.9 s figure in `CLAUDE.md`, added earlier at the author's request, is corrected as well.
- **The scripts are not fixed yet.** Both `scripts/phase-timings.sh` and `scripts/time-to-ready.sh` wait for "two nodes Ready" by count. Fix by naming the nodes (or excluding `ubuntu-bake-vm-warm`), by failing loudly when a node's timestamp is missing instead of treating it as zero, and by giving the worker's install path a "started" marker the instrument can find. Alternatively, remove the ghost from the image at bake time.
- **Measurement method for the corrected numbers:** from just before `kubectl apply`, poll the target cluster's nodes every 0.5 s by name (`-cp-`, `-workers-`), then read each node's `Ready` `lastTransitionTime` from the API (1-second granularity). The table uses the API transition times; the polled observations agree within about a second (control plane 25.1 / 25.2 / 25.6 s, worker 31.0 / 36.3 / 34.5 s). Host load was 1.2–2.3. The measuring script lives outside the repository; consider adding it.
- **Cannot be re-checked:** the 9 September runs (artifacts not kept) and June's 48.520 s from `time-to-ready.sh`, which may have included the ghost. The ghost cleanup dates from 23 June.
- **Chapter 8 repeats the 9 September phase table.** Now that this chapter introduces it, trim Chapter 8's copy to a back-reference.
- **Figures.** 7.1: the boot timeline with its data sources — object timestamps, pod status, console, probe — on one clock. 7.2: the ghost — a node list at the moment the control plane turns Ready.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 08-two-lines-and-a-cpu-quota

- **Character names.** Anita Rao (CTO) and Vikram Iyer (platform lead) are placeholders. Settle them in the Prologue and carry them through.
- **Figure 8.1.** Per-run dots for the three stages (52.7 / 50.0 / 46.6 → 42.7 / 42.8 / 41.2 → 34.5 / 34.2 / 34.7), so the median falling and the spread collapsing read as one picture.
- **Figure 8.2.** Reachability sketch: node → control-plane VM pod IP (`200`, 0.3 ms); worker VM → control-plane VM pod IP (timeout); worker VM → VIP `172.18.255.215` (reachable).
- **Source conflict — resolve before print.** In `docs/sub-60s-cluster-strategy.md`, "What is actually left" strikes the 10 s kubelet gap as fixed, then lists "~10s silent gap inside kubelet startup on the worker" as still open. Worker join after API ready fell to 5.1 s after the fix, which suggests the second entry is stale. This draft omits it.
- **Source conflict.** `:preinit` at ~136 s is "not worse than legacy" in the strategy doc and "slower than legacy" in the brief. The draft says "back inside the 90-to-150-second band," which is true under both readings.
- **Source conflict.** `CLAUDE.md` gives the parallel-boot build as "~61s"; the brief's measured table gives 46.6 / 49.7 / 50.0. The draft uses the brief (it says its table wins).
- **Section numbers.** The strategy doc has two sections numbered 6, so the draft cites section titles. Renumber the doc if you want to cite by number.
- **The agent.** The September investigation was run by an AI coding agent working from the brief (commit `c7ad792` is agent co-authored). This draft says so once; decide whether to foreground it here or save it for Part IV.
- **Postscript corrected 17 September.** The earlier draft reported 23.9 s as a both-nodes time; it was the control plane only (see Chapter 7). The corrected both-nodes median is 33.7 s. The 9 September runs cannot be re-checked — their raw files were not kept — though their tables report a worker-join phase, so the worker was captured.
- **Trim the phase table.** Chapter 7 now introduces the 9 September budget; replace this chapter's copy with a back-reference.
- **Optional.** CxO readers will compare against managed-Kubernetes cluster-creation times. If you add that comparison, source it; the draft deliberately quotes no external number.


## 09-warmth-is-mandatory

- **Resolved 18 September — the 194 ms claim has been re-measured.** Was from June (k3s 1.31, recorded in the warm-pool design notes and the project's memory), unmeasured since. Both prerequisite fixes this note asked for were made: `POOL_STANDBY_MANIFEST` now points at the warm manifest, and the pool's node-count readiness check (`targetNodesReady` in `pool.go`) now counts by name. Re-measured median: 467 ms, three runs (1.059 / 0.467 / 0.465 s). See "Turned on, and timed" above.
- **Model measurements (17 September):** `POST /api/generate` with a one-token limit, reading Ollama's own `load_duration`; RTX 4050 laptop GPU with 6 GB; every loaded model reported 100% on GPU. After measuring, `qwen2.5:7b` was left as the loaded model, matching the state before the Chapter 16 lab evicted it.
- **The 28–30 s figure** is the CPU-era number from the architecture constraint and the repository's Ollama notes. It was not re-measured on CPU.
- **Pool status wording.** The backend reported `state: none`. That status alone does not distinguish "disabled" from "enabled but idle beside an operator-managed cluster"; the chapter relies on the opt-in default in `run-ui.sh`.
- **Superseded estimates.** `docs/warm-pool-strategy.md` uses the coarse 17/17/18-second phase split that Chapter 7 corrected. Its "~50 s" cold build is the June figure; today's is 33.7 s.
- **The Meridian demo scene is fiction,** but it mirrors a real, recorded symptom: agents' first requests timing out on a cold model.
- **Figures.** 9.1: the standby lifecycle. 9.2: the two-inventories table as a diagram — shelves one item deep.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 10-hardcoded-ips-and-other-confessions

- **The 17 September re-plumb.** Target cluster: namespace `ch10-before`, `nginx:alpine` (already present in the image, so no pull), NodePort 30080. cluster2: `default/target-cluster-nginx` Service (`.216`) plus hand-written Endpoints, using the script's own selection loop and YAML. cluster1: namespace `ch10-before`, labelled ambient (redirection confirmed `enabled`), with a curl pod and the ServiceEntry applied verbatim. Everything was deleted afterwards and verified gone; `.216` was released. Script and log: scratchpad `ch10-before-path.sh` / `.log`.
- **This was not a faithful re-run of the March demo.** The target cluster had no Istio: `install-istio-ambient.sh` pins 1.28, which is outside its tested range on Kubernetes 1.37, and it was not run. cluster1 runs Istio 1.31 from `07-istio-advanced`, not whatever was installed by hand in March. The `mc-demo` httpbin/sleep were not recreated. The reverse direction was tested against `.201`, the Act 1 gateway (HTTP 404, so reachable), because `.200` is not allocated today.
- **Name resolution after the ServiceEntry** was measured on Istio 1.31 ambient. Do not claim it would have worked on the March install without checking.
- **The DestinationRule claim was not tested live.** It rests on the 07 README's verdict and ambient's documented L4/L7 split. A test that would settle it: make nginx return 5xx responses and see whether its endpoint gets ejected.
- **The first request from the host failed** (`000`, after 3.1 s), sent within a second of MetalLB assigning `.216`. The next four succeeded in 0.6–1.0 ms. Not investigated; most likely the L2 announcement had not yet reached the host. The pod-side measurements started about 8 s later, and every one succeeded.
- **A suspicion that did not hold.** The demo reads `/tmp/target-cluster-kubeconfig`, which was a day old. It still authenticated, because the warm image's CAs are fixed. It is not a confession, and not in the chapter. It does mean CLAUDE.md's *"The target CA changes on every redeploy"* is outdated for the warm path.
- **CLAUDE.md wording.** It says cluster1's httpbin/sleep are *"not recreated by any script"*. In fact `cross-cluster-demo.sh` does create them; what no script did before `07-istio-advanced` was install Istio on cluster1.
- **The March commit carries an AI coding assistant as co-author** (commit trailer). Decide whether to say so in the text. It sharpens Part IV's "trust, but verify" theme, but the chapter stands without it.
- **The Meridian briefing scene is fiction.** The badge logic it depicts is real and unchanged.
- **Figures.** 10.1: the mind map as drawn, with the masquerade concept annotated. 10.2: the path as measured (bridge binding, no NAT hop). 10.3: the two ztunnel log lines side by side, identity fields highlighted.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 11-one-label

- **Three measurements on 17 September**, each with a restore-on-exit trap (scratchpad `ch11-failover.sh`, `ch11-outage.sh`, `ch11-route.sh`, with `.log` files). Temporary changes were limited to `demo-apps`: echo replicas scaled to zero (cluster1 three times, cluster2 once), the `istio.io/use-waypoint=none` label on Service `echo`, and a re-applied `echo-internal-split` route. Afterwards: 2/1 replicas on both clusters, the original two routes, the original labels. `verify.sh` ran between the second and third scripts: 22/22.
- **The August morning inference is not proven.** It rests on the order in which the acts ran, the evening commit's 503 measurement, and a demonstration that the counter prints the same line for a total outage. Say "cannot be recovered", not "was 503".
- **Why the waypoint gotcha stopped reproducing is not established.** 1.30.3 is no longer installed to compare against. Check the Istio 1.31 release notes before naming a cause.
- **The even split.** 60 of 60 calls were `200`; per pod: three cluster1 pods at 10 each, and all 30 remote calls on one cluster2 pod (`x8dph`). Hypotheses, not verified: the east-west endpoint is weighted by the remote pod count (3 + 3), and one pooled tunnel connection pins remote calls to one pod. Whether `spec.trafficDistribution: PreferClose` changes the split was not tested.
- **Latency numbers** are curl `time_total` inside the client pod, on one host: local means little about a real WAN.
- **Not re-run on 17 September:** the identity-swap revocation beat, request mirroring, the 301 redirect, and XFCC after a waypoint restart (the header was present, but the restart itself wasn't repeated).
- **Canary figures differ across runs:** 89/11 (August, 100 requests), 93/7 (16 September, from Chapter 19), 8% (17 September, 60 requests). All are within the act's accepted band.
- **ListenerSet.** Whether it ever worked in August is unknown; the August commit does not list it among verified results. There is no `ReferenceGrant` anywhere in the repository.
- **Unsupported combination.** Istio 1.31 on Kubernetes 1.37 is outside Istio's tested range (Chapter 19).
- **Repo fixes not made:** count status codes in `run.sh`; assert (or remove) the locality claim; add a `ReferenceGrant` or drop the ListenerSet beat; update README gotcha 2 and the console's handler comment for 1.31; add a behavioural failover check to `verify.sh`.
- **Figures.** 11.1: the Chapter 10 plumbing next to the one label. 11.2: the two refusal status lines. 11.3: the two identical `success=20 failed=0` screens — failover and total outage. 11.4: the 60-call distribution.
- **The Meridian scenes are fiction.** The outage test and its output are real.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 12-the-model-is-just-another-service

- **Measurements (17 September).** Scratchpad `ch12-ai-path.sh` / `.log`, plus ad hoc read-only queries. No Kubernetes objects were created or changed. The only state change was Act 4's own completion swapping the GPU model; a trap reloaded `qwen2.5:7b`, and `/api/ps` confirmed it afterwards.
- **Latency.** Host-side curl: direct is `localhost:11434`; the gateway path goes through the MetalLB address and the Kind node. Seven interleaved pairs after a warm-up, `max_tokens` 5, temperature 0. The difference is noise; do not say the gateway is faster.
- **Request counts.** `agentgateway_requests_total` showed POST 12 / GET 28 when queried. That does not exactly match the requests this session sent (scrape timing, the console backend's own GETs). Cite the metric's labels, not its values.
- **Host exposure.** `*:11434` is the bind address. Reachability from another machine on the LAN was not tested and depends on the host firewall.
- **Tier 2 validity** was checked against the CRD schemas in the downloaded v1.0.1 `manifests.yaml`, not by applying to a cluster (the CRDs are not installed).
- **Routing agents through the gateway** would also need their egress widened: per Chapter 14's reading of `agent-egress-ollama.yaml`, the agents may use ports 443, 6443 and 11434, and the gateway listens on 80. Not tested.
- **Stale agent briefings.** `cluster2-agent`'s system prompt describes cluster1 as running `httpbin/sleep`; `mesh-sre-agent`'s says Istio 1.30. Both are pre-September facts.
- **Kiali anonymous** is the sample add-on's default; Istio labels those add-ons as not intended for production. Say so if a reader might think it was a deliberate choice.
- **Topology** is judged by the API response, not a screenshot of the rendered graph. The frontend may add decoration; the node list and statuses are what the backend sent.
- **Figures.** 12.1: two model paths — agents → cluster2 shim → host (used), and the AI gateway on cluster1 (unused). 12.2: the topology view with the 14 non-existent nodes annotated. 12.3: Kiali's empty `ai-gateway` graph beside `demo-apps`.
- **The Meridian scenes are fiction.** Every figure quoted in them was measured.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 13-an-agent-as-a-first-class-object

- **Re-verified on 17 September** (Sympozium 0.10.75, Kubernetes 1.37): the three required `AgentRun` fields (server dry run, nothing created); lab 02's run and every figure in its table; the models endpoint. **Not re-verified:** lab 01's streaming behavior and the count-to-five file-writing anecdote, which date from 0.10.38.
- **The strategy document is stale in three places this chapter quotes around.** Its object table lists `SympoziumInstance` as the persona; Constraint 1 names llama3.2 at `172.18.0.1`, while the agents run `qwen2.5:7b` through the in-cluster `host-ollama` Service; Constraints 4 and 7 bind policy through `SympoziumInstance.spec.policyRef`, which must live on the `Agent`. Update it, or quote it explicitly as a historical document.
- **Dates.** The kagent-to-Sympozium timeline comes from commit history: kagent in February, Sympozium personas in April, the chat proxy's switch landing in a late-June checkpoint commit, and the embedded console in August. The exact switch date is not recorded.
- **Fifty-three records.** Counted from the run list captured on the morning of 16 September, before the rebuild; the oldest was 26 days old.
- **Known defects named in the chapter:** `HandleListAgents` in `ui/backend/handlers/ai.go` still queries `sympoziuminstances`; the console proxy on port 8081 is unauthenticated; the skills webhook is still deployed although it has done nothing since 0.10.47.
- **Chapter 15 overlap.** This chapter now owns the sidecar explanation; trim the opening of Chapter 15's "Where the truth goes missing" to a back-reference.
- **Meridian's vendor** is deliberately generic. Keep it from resembling any real product.
- **Figures.** 13.1: the object map — `Agent` → `AgentRun` → pod (`agent`, `ipc-bridge`, skill sidecar) → per-run identity and permissions → model endpoint. 13.2: the three UI modes as windows onto one set of objects.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 14-rules-the-model-cannot-break

- **Tracked, not fixed.** This chapter describes, at design level, a path from an agent run to a target-cluster administrator credential. Rather than fixing the live platform as part of writing this book, it is tracked as `TODO.md` §3 ("agent runs can write in their namespace and read every secret in it") in the `sovereign_cloud` repository. The chapter describes the hole as open; it is not confirmed fixed as of this draft.
- **Probe method.** Two `curlimages/curl` pods in `sympozium-system`, one labeled `sympozium.ai/role=agent,app.kubernetes.io/part-of=sympozium` and one unlabeled; `curl` with a 5-second connect timeout; "no connection" is curl status `000`. Both pods were deleted. The labels were checked against the pod of lab 04's real run the same morning. Kind 0.33.0, `kindnetd` image `v20260820`.
- **Lab 04's README is wrong about the network plugin**, at least today. Update it. Note that `06-sympozium/host-ollama-service.yaml` already says policy enforcement is what made the shim necessary.
- **Re-verified on 17 September:** the admission refusal (dry run); `fetch_url` used despite the deny (one tool call, 3.8 s of model time, 6,956 input and 121 output tokens — July's run used 4,010 and 95); webhook failure policies (`vagentpod` and `vskillpack` fail closed; the repository's own skills webhook fails open). **Not re-verified:** the RULE 1 refusal measurements, which date from August on `qwen2.5:7b`.
- **The strategy document is stale in several places this chapter grades:** Constraint 1 names llama3.2 at `172.18.0.1`; Constraint 3's pool is now `.211–.225`; Constraints 4 and 7 bind policy through `SympoziumInstance.spec.policyRef`; Constraint 8 calls the policy `sympozium-sandbox-restricted`, while the repository's policy is named `sandbox-restricted` (the generated network policy carries the longer name).
- **Warm-up schedule.** The cause of it stopping is not established. The pre-rebuild record also shows far fewer runs than a four-minute schedule implies (18 over about 26 days). The heartbeat's own comment says it keeps llama3.2 warm, but it runs against `cluster2-agent`, whose model is `qwen2.5:7b`.
- **Verdict counts.** The Meridian close says one held, four partly, three not — matching the scorecard, with rule 2 counted as held.
- **Figures.** 14.1: the seven-layer ladder, with each of the eight rules placed where it actually lives. 14.2: the probe results as a two-column reachability diagram.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 15-trust-but-verify

- **Correction made to the table of contents.** It gave the lab 09 comparison as "llama3.2 at 22.4 tok/s against qwen2.5:7b at 15.1." Those are average run durations in seconds. Throughput was 4.0 against 11.0 tokens per second. Fixed in `book/TABLE_OF_CONTENTS.md`.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders, as in Chapter 8.
- **Evidence not yet in the repo.** The 16 September answers (`ronics-01`, `target-node-1`/`target-node-2`, and the per-run-account permission error) come from the session record of the upgrade, not from a committed file. Commit a verification log before print, or the chapter's opening exhibit fails its own "Open the repo" promise.
- **Demo script is out of date.** Scenes 3 and 6 of `demo-mesh-sre.sh` verify as `sympozium-agent`; since chart 0.10.75 the agent's runs use per-run accounts. Derive the account from a live run pod instead. Separately, scene 6's narration points to "pod-name suffixes and the gateway address"; in the August recording, the only unguessable detail the agent actually produced was the age.
- **`CLAUDE.md` needs two corrections.** Its `mesh-sre-agent` section still recommends verifying as `sympozium-agent`. Its 0.10.75 entry says the agent was "confabulating a 'permission issue' writeup"; the recorded answer was accurate and named the real per-run account.
- **The group binding is broader than the agent.** It grants read on mesh and VM configuration to every service account in `sympozium-system`. Decide whether the book should recommend a narrower design, or present this as the demo-grade trade-off it is.
- **Lab 10's scorer.** Its README treats two or more tool calls as a real investigation, but run 1 passed with one. Consider requiring the literal missing tag in the answer.
- **Lab figures predate later upgrades.** Labs 09 and 10 were captured on the 0.10.38 release, before the 0.10.47, 0.10.57, and 0.10.75 upgrades and the Kubernetes 1.37 rebuild. Re-run both before print.
- **Prompt-phrasing anecdotes conflict.** The demo script's comments say asking for a judgment gets more real content than asking for a relay; on 16 September, asking for exact output beat asking for a sentence. Both are single observations — keep either from hardening into advice.
- **Dependencies.** The sidecar explanation overlaps Chapter 13, and the "nudge, not a control" refusal finding is Chapter 14's; trim whichever repeats once those are written.
- **Figures.** 15.1: the two node lists beside the probe pod's output. 15.2: the ladder of evidence. 15.3: stills of scenes 5 and 6 side by side (extractable from the video with `ffmpeg`).


## 16-the-fleet

- **Re-verified on 17 September** (Sympozium 0.10.75, Kubernetes 1.37): the ensemble lab, run end to end and cleaned up (both runs and all created objects deleted); the model-fit service; the April personas (server dry run, nothing created); the installed ensembles and skill packs; the security service's absence.
- **Persona count.** 29 is the number of persona headings under §3.1–3.7 of the strategy document.
- **The analyst's error.** The chapter states that Kind runs Kubernetes nodes as containers with no hypervisor involved; have a technical reviewer confirm the wording.
- **Model-fit caveat.** The ratings are for model variants in the fit service's catalog, matched by family name — not the exact quantized builds Ollama serves (`qwen2.5:7b`, `llama3.2`). Lab 07 already warns to read each entry's notes rather than its label.
- **Model names omitted on purpose.** The fit service's top picks are community fine-tunes whose names include other vendors' product names; the chapter describes them by size and architecture instead.
- **"Where the weeks went"** is reconstructed from commit history and the repository's operating notes, not from time tracking.
- **The strategy document is stale** in the places this chapter quotes: "every agent is one `SympoziumInstance`", the persona-pack assumption, and the April status snapshot.
- **Figures.** 16.1: the catalog as 29 marks across the planes, with three lit. 16.2: the two-agent pipeline with its token counts, and the analyst's wrong claim passing through the reviewer unchanged.
- **Target-cluster agent's credential.** Verified on 17 September: the `target-k8s-ops` tool pack sets `KUBECONFIG` to the mounted `target-cluster-kubeconfig` Secret, whose user is `target-cluster-admin`. Add this to Chapter 14's scorecard (rule 7) and to the fixes list — a read-only service account for the target cluster would move that rule below the model.
- **No `cluster-admin` bindings** for any identity in `sympozium-system` (checked 17 September), which is why the RBAC Auditor as specified would not have flagged Chapter 14's finding.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 17-from-the-lab-to-the-data-centre

- **This chapter contains no new measurements, by design.** Every figure in it is carried from Chapter 7, 9 or 19 and attributed there. If a reviewer finds an unattributed number in this chapter, it is a defect — the chapter's whole claim to belong in this book is that it does not pretend.
- **Product names are categories, not recommendations.** Rook/Ceph, Velero, SOPS, External Secrets, Vault, Flux, Argo CD, cert-manager. None was evaluated, benchmarked, or run against this platform. Consider adding one sentence to that effect in the chapter body if a reviewer reads the current framing as endorsement — the "design guidance" label at the top is doing a lot of work.
- **Unverified technical assertions to check before publication:** that `KThreesControlPlane` accepts `replicas: 3` and behaves as described (the platform has only ever run `replicas: 1`); that MetalLB's BGP mode is a configuration change rather than a redeployment; and that Velero's volume backup works with the CSI driver a reader would actually use under KubeVirt. All three are stated as well-known practice and none was tested here.
- **The HA claim about quorum** is generic etcd/Raft practice. k3s with an embedded datastore behaves differently from k3s with SQLite, and this platform's warm path deliberately forces SQLite (Appendix G.1's header comment). A three-replica k3s control plane is therefore a *different* configuration from the one this book measured, not a scaled-up one. Worth a sentence in the body if a technical reviewer agrees.
- **Chapter numbering.** This chapter was inserted as 17; the upgrade chapter moved to 18 and the TCO chapter to 19. Cross-references throughout the book were shifted accordingly on 18 September. Re-grep before print.
- **Overlap to watch.** The capacity section and Chapter 20's worksheet cover adjacent ground. If Chapter 20 gains real cost figures, trim this chapter's capacity section to the sizing *method* and let 19 own the money.
- **The Meridian scenes are fiction.** No Meridian figure is quantified here either; the CFO's questions are the chapter's argument in dialogue form.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 18-the-day-we-upgraded-everything

- **Resolved 18 September — the specific files are committed.** `06-sympozium/demo-mesh-sre.sh`, `06-sympozium/agent-istio-rbac.yaml`, and `docs/demo/mesh-sre-agent-demo.mp4` all landed in commit `84669ad` ("feat(sympozium): add mesh-sre-agent for Istio ambient mesh health", 2026-09-17), which is pushed to `origin`.
- **A bigger version of the same problem remains: the branch isn't merged to `main`.** All of this — the mesh-sre-agent work, the Kubernetes 1.37/Istio 1.31/Sympozium 0.10.75 upgrade this chapter describes, and in fact most of Parts II–V's technical material — lives on `upgrade/k8s-1.37-istio-1.31-sympozium-0.10.75`, 40 commits ahead of `main` with none of `main`'s own commits missing from it. `main` has 161 tracked files against this branch's 312. A reader who clones `main` following any "Open the repo" pointer into `06-sympozium/`, `07-istio-advanced/`, or most of the upgrade-era scripts will not find them. Decide before print: merge the branch (open a PR), or have "Open the repo" name the branch explicitly.
- **`CLAUDE.md` had two errors from this rebuild; both corrected 18 September.** Its upgrade entry said `scripts/configure-kubevirt-perf.sh` "can silently no-op if it races KubeVirt's operator" — corrected to the evidence: the setup script was killed by a 590-second time limit during its readiness waits. It also said the base image is "NOT created by any script in this repo" — corrected: `scripts/import-base-images.sh` creates it, but isn't wired into the Makefile/README setup path.
- **Table of contents corrected.** Its Chapter 19 entry listed the base image among the things nobody had scripted. Fixed.
- **Minimal image still missing, not yet fixed.** `ubuntu-minimal-noble-dv` was not re-imported on the rebuilt `cluster2` (confirmed still absent 18 September — `kubectl -n default get dv` on cluster2 lists only `ubuntu-noble-dv`), so `make bake-image-minimal-preinit` has no source image. `scripts/import-base-images.sh` fixes it; running it is a live-cluster change, deliberately not done as part of this documentation pass.
- **ListenerSet.** Resolve before print: send a request to port 8443 with the `tenant-a.demo.istio.local` hostname, and either report a real regression or explain the status/Service mismatch. Act 1 also prints the gateway's own listeners under the caption "attached ListenerSets", which overstates what it shows.
- **Digest pinning.** The Kind 0.33.0 release notes say node images must be referenced by `@sha256` digest to guarantee an image built for that release. The rebuild used the plain tag.
- **Unmeasured.** The rebuild's total wall-clock time was not recorded, so the Meridian close says "one afternoon" rather than a number. The teardown timing came from a one-off command, not a script.
- **Measurement corrected 17 September.** The earlier draft reported 23.9 s and a phase comparison implying the worker no longer lagged; both came from runs that stopped the clock at the control plane (see Chapter 7). Replaced with the by-name measurement. The phase comparison was removed rather than corrected: its worker row was the part the bug invalidated.
- **The agent's role.** This chapter says plainly that an AI coding agent performed the rebuild, including the two mistakes it made (the time-limited run and the wrong recorded cause). Decide whether that belongs here or in Part IV.
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 19-reclaiming-the-margin

- **This chapter still has no *Meridian* figures,** and should not acquire any. What it now has is a lab-scale metering result of its own ("One workload, metered both ways", 18 September). If real Meridian figures become available, the worksheet table is where they go — the metered section stays as the method.
- **The metering experiment's honesty boundary.** One workload, one host, ten minutes, one provider's list price. Three of the four owned-side inputs are assumptions (host cost, amortization, host power) and are labelled as such in the body and tabulated in `measurements/README.md`. Only the 48.6% host share and the 18 commit-days are measured. **Host power could not be measured at all** — `/sys/class/powercap/intel-rapl:0/energy_uj` is root-only on this kernel and the battery reads zero on AC — so no wattage for the host is asserted anywhere.
- **Rented-side prices** are DigitalOcean list prices accessed 18 September 2026, with no committed-use discount, plus the EKS control-plane fee for contrast. **AWS EC2 instance prices are deliberately absent:** the on-demand tables are script-rendered and could not be read, so no EC2 figure is quoted. Before publication, re-check every price and re-date it; cloud list prices move.
- **The crossover claim (two to three workloads) is arithmetic, not an experiment.** A second workload was never actually placed on this host — and by the memory reservation it measures, roughly two of these clusters is all the host would hold. Chapter 20's own earlier finding ("a second target cluster would not fit") and this crossover are in tension at exactly the interesting point; say so rather than resolving it by assertion.
- **Deck arithmetic** was computed in a scratch script: 4.3/12.5 = 34.4%; (1.8+1.2)/4.3 = 69.8%; the investment and payback scenarios are as tabled. The deck does not say whether $8.2M includes the initial investment. The chapter tests both readings rather than choosing one.
- **Depreciation sources** are secondary (search results summarizing filings): Microsoft Q4 FY2022; Alphabet January 2023; Amazon's 7 February 2025 disclosure, effective 1 January 2025, quoting "increased pace of technology development, particularly in the area of artificial intelligence and machine learning." Cite the 10-K/10-Q text directly before publication, and confirm the "billions" wording (the coverage cites about $3.7B for Microsoft's FY2023).
- **Reservation figures.** Memory requests were summed over Running/Pending pods from `kubectl get pods -A -o json`. That gives 22.42 GiB; `describe node` reported 24,192,159,232 bytes (≈22.5 GiB). "Used" is `docker stats` for the `cluster2-control-plane` container (7.453 GiB), a single sample. CPU use is the same sample (41.46% of one core). The target VM figures are `kubectl top nodes` inside the target cluster; the worker reported `<unknown>`, consistent with Chapter 5's tainted worker.
- **"A second target cluster would not fit"** is arithmetic on memory requests (about 7.5 GiB unreserved, against 14.57 GiB for a target cluster's VMs). It was not tested by scheduling one. The platform also enforces one target cluster at a time by design.
- **Energy.** Scratchpad `ch18-energy.sh` / `.log`:
  - Five runs of 400 generated tokens, temperature 0, with the model already loaded (0.13–0.17 s load time); `nvidia-smi` sampled every 200 ms.
  - The first run's mean was lower (38.3 W), likely ramp-up. Throughput drifted from 37.1 to 33.9 tokens/s across runs, possibly thermal.
  - GPU-board power only: not CPU, memory, the rest of the host, or cooling. The host was on AC power.
  - Chapter 15's "11.0 tokens per second" is a different metric: whole agent runs, including prompt processing and tool calls.
  - The 3–17 cents range is illustrative arithmetic at 10–50 cents per kWh, not a measured price.
- **The effort figures** are `git rev-list --count HEAD` (88 as of 18 September; 81 on 9 September), `git shortlog -sne` (1 author), a count of commit messages containing "Co-Authored-By: Claude" (70; 63 on 9 September), and distinct commit days (18). The count moved because writing this book added commits to the platform repository — worth one line in the body if a reviewer finds the change confusing rather than charming. 28 working-tree changes (including this book and the September upgrade) were uncommitted at the time of writing. **Decided:** the committer is named (Mahipal) in the text above; how much further to describe the AI assistance is still open.
- **The Meridian scenes are fiction.** The board-pack sentence sets up the epilogue, where savings are still "not yet measured".
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## 99-epilogue-own-your-cloud

- **Flyer text** is from the sovereign-cloud flyer, version 2, unbranded (Appendix H; the "Save $Millions" and "Creating a Cloud Owning Culture" band appears only in v2). The six pillar lines are quoted verbatim from its extracted text. Do not quote the separately branded flyer.
- **Every measured claim in the scene** traces to a drafted chapter:
  - 33.7 s / 18.9 s — Chapter 7.
  - Standby claim "under half a second" — 467 ms median, three runs, 18 September, Chapter 9 (was 194 ms in June, resolved 18 September).
  - Worker `Ready` that couldn't work — Chapter 5.
  - Before picture kept and labelled — Chapter 10.
  - Failover 30/30, and the check that couldn't fail — Chapter 11.
  - Map with 14 of 21 non-existent nodes; gateway without a lock; model traffic without model or token labels — Chapter 12.
  - Agents may triage, not act — Chapter 15.
  - 3-of-3 triage and llama3.2 confabulation — Chapter 15.
  - `Available=False` — Chapter 4.
  - Week-long upgrade — Chapter 19.
  - Moving-tag crash — Chapter 2.
  - Committed demo keys — Chapter 3.
- **"A single rack in the basement"** is Meridian fiction. The real platform in the build logs runs on one host, and the prologue says so. Keep the two distinct.
- **Chapter 20 now exists and carries a metered result** (18 September), so the savings line has been rewritten: the scene reports the lab's one-workload comparison and keeps "not yet measured" for *Meridian's* own invoice, which is the distinction the whole epilogue turns on. Re-read the scene against Chapter 20 before print — if that section's figures move, this scene moves with them.
- **"Anybody can buy servers… It's most of what the rent pays for"** is an opinion in Anita's voice. Keep it as dialogue, not narration.
- **The Meridian scene is fiction.**
- **Character names.** Anita Rao and Vikram Iyer remain placeholders.


## appendices

- **Checked live on 17 September:**
  - MetalLB pools and allocated LoadBalancer addresses on both clusters.
  - The `kubeui` namespace (absent).
  - Makefile target count (65) and the `make help` output.
  - Controller image versions.
  - The labs index text.
  - "Files naming it" counts, from `grep -rl` over `*.sh *.yaml *.go *.ts *.tsx *.env Makefile`, excluding `book/`, `.worktrees/` and `node_modules`.
- **Node addresses** (`.2`/`.3`/`.4`) are from `docker network inspect kind` on 17 September. `CLAUDE.md` still records `.2` as `cluster2-control-plane`.
- **The by-name procedure** (D.4) is from a measurement script kept outside the repository during the writing of this book. Consider committing it as `scripts/time-to-ready-by-name.sh`, or fixing the two repository scripts, so Appendix D can point at the repo.
- **The spread of 4.7 s** after the upgrade is computed from the three recorded runs (30.4 / 35.1 / 33.7). No chapter states it; consider adding it to Chapter 7's ledger.
- **Catalog rows marked "Not fixed"** reflect the state on 17 September; revisit before publication. Rows sourced only from `CLAUDE.md` (C.1, C.8, C.9, most of C.10) were not re-reproduced for this appendix.
- **MetalLB annotations.** Both spellings "worked on 17 September": `metallb.io/…` via the live Act 1/3/4 gateways, `metallb.universe.tf/…` via Chapter 10's re-plumb. Check whether the older spelling is deprecated in the installed MetalLB version.
- **Glossary** definitions are deliberately non-technical. Have an engineer check that each is still true enough, and a non-engineer check that each is understandable.
- **Appendix G is the priority subset, not a complete audit.** The nine load-bearing excerpts are quoted (read 18 September from branch `upgrade/k8s-1.37-istio-1.31-sympozium-0.10.75`). Every other file named in an "Open the repo" block is still a bare pointer. Before publication, walk each remaining block and decide, reference by reference, whether the argument depends on the file's contents — if it does, quote it here; if it does not, leave the pointer.
- **SSH key elided** in G.1. The committed manifest carries a real public key; the book prints `[public key elided]`. Anyone diffing the book against the repository will see the difference — say so here rather than let it look like a transcription error.
- **G.6 quotes a stale briefing on purpose.** `cluster2-agent`'s `systemPrompt` still describes cluster1 as running `httpbin/sleep`, which the September rebuild ended. If that prompt is fixed in the repository before publication, re-quote it and rewrite the paragraph beneath, which depends on the staleness.


## 17-the-same-cluster-twice

- **Measured 19 September 2026, N=5**, boundaries pre-registered in `measurements/README.md` before the first run. Raw data committed. The rehearsal run was discarded and is disclosed.
- **The teardown correction is the load-bearing claim.** 18.9 s (N=1, 17 September) → 45.73 s (N=5, spread 1.24 s). Chapter 19 keeps its original dated account and points forward. Every other occurrence in the book was swept on 19 September; re-grep before print in case a new one appears.
- **The split teardown run is N=1.** The qemu-stop figure (35.60 s) and the object-removal figure (45.33 s) come from a single separating run. The five-run median covers object removal only. If the machines-stop figure is going to be quoted as often as it deserves, measure it five times too.
- **Palantir is cited about nodes, not clusters**, and the chapter says so. Do not let an editor smooth that distinction away — it is the difference between a sound citation and a misattribution.
- **DORA's under-one-hour elite threshold** is verified against DORA's own reporting; re-check the current year's report before print, since the thresholds are restated annually.
- **Three sources were cut** and should stay cut: a vendor's 90%/35% ephemeral-environment figures (no methodology), VSCO's 88% (real, but attributed to Kubernetes/gRPC/Envoy and error elimination, not cluster replacement), and three "cattle not pets" sentences that could not be located at the source they were attributed to.
- **The shared-identity demonstration is real and reproducible** but the credential and the CAs are committed demo material (`TODO.md` Sev1). If those keys are ever rotated or scrubbed, this chapter's proof needs re-running, and the transcript in it will no longer match.
- **Chapter 5 correction.** The tainted worker runs a tolerating pod normally. Chapter 5 still says the second VM "has been given nothing to do," which remains true of ordinary workloads but overstates the defect. Decide whether to amend Chapter 5 itself or leave the correction here in 17.
- **Not measured, and deliberately not claimed:** any production recovery time, any data restore, anything involving two concurrent clusters. The chapter's own "what eighty-two seconds does not mean" section is the guard; keep it if the chapter is cut for length.
- **Character names.** Anita Rao and Vikram Iyer, as elsewhere.

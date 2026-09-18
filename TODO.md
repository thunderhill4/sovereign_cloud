# TODO — sovereign_cloud platform

Open items found while writing the *Own Your Cloud* book and re-verifying the
platform on 17 September 2026. Grouped by theme; **Sev** is rough priority
(1 = do before any non-demo use, 4 = cleanup). "Source" points at the chapter,
file, or check that found it.

Demo-only trade-offs (fixed CAs, single cluster, `nats.auth` off) are recorded
in `CLAUDE.md` and are intentional for a demo; the items below are the ones
that would have to change for real use, plus plain bugs and stale docs.

---

## 1. Authentication & access control

The web console and everything it exposes currently has **no authentication**.
Anything that can reach the port can drive it. On 17 September the backend was
bound to `*:8080` / `*:8081` (all interfaces) on a host with a LAN address.

- [ ] **Sev1 — kubeui backend has no auth on any endpoint.** `GET /api/v1/nodes`,
  `/api/v1/registry/config`, etc. all return `200` with no credentials. There is
  no login, session, or token middleware in `ui/backend/main.go` — only CORS
  headers, which do not stop non-browser clients. Add real authn/authz in front
  of the API. *Source: live curl; `ui/backend/main.go`.*
- [ ] **Sev1 — the web terminal grants an unauthenticated shell.** `GET
  /api/v1/terminal` upgrades to a WebSocket and runs `pty.Start(exec.Command($SHELL))`
  in `CLAUDE_DIR`, gated only by an `Origin` check. `Origin` is empty for any
  non-browser client (curl, a script), so the check passes and hands out a shell
  on the host (or in the backend pod). Require authentication before the upgrade;
  do not rely on `CheckOrigin`. *Source: `ui/backend/handlers/terminal.go`.*
- [ ] **Sev1 — destructive endpoints are unauthenticated too:** `POST
  /api/v1/cluster/delete`, `DELETE /api/v1/cluster/target-delete`, `DELETE
  /api/v1/registry/images/{name}:{tag}`, `POST /api/v1/cluster/deploy`,
  `POST /api/v1/istio/failover`, `POST /api/v1/cross-cluster/scale`. Same fix as
  above; gate state-changing calls behind authz and an audit trail. *Source:
  `ui/backend/main.go`.*
- [ ] **Sev1 — bind to loopback by default.** The backend listens on all
  interfaces. For a laptop demo on a routable LAN that exposes all of the above
  to the network. Default to `127.0.0.1` and make external exposure opt-in.
- [ ] **Sev2 — the AI model gateway (Act 4) has no lock.** `172.18.255.203`
  answers `/api/tags`, `/api/ps`, `/api/show` and the OpenAI surface with no
  credentials, and its catch-all route also forwards Ollama admin paths
  (`/api/pull`, `/api/delete`). Add an `AuthorizationPolicy` (and/or restrict the
  route). *Source: Chapter 12; `07-istio-advanced/act4-ai-gateway/`.*
- [ ] **Sev2 — Ollama listens on `*:11434` (every interface).** Any host on the
  LAN can reach the models directly, bypassing the gateway entirely. Bind it to
  the Docker bridge / loopback, or firewall it. *Source: Chapter 12; `ss -ltnp`.*
- [ ] **Sev2 — Kiali is served with `auth.strategy: anonymous`** at
  `172.18.255.204`; anyone who can reach it browses every namespace, workload and
  identity. It is the Istio sample add-on default. Switch to a real auth strategy
  for non-demo use. *Source: Chapter 12; `07-istio-advanced/act5-observability/`.*
- [ ] **Sev3 — the local registry has no auth** (`GET /v2/_catalog` → `200`).
  Fine inside a single host; note it before exposing the registry.
- [ ] **Sev3 — the console-proxy pre-seeds the Sympozium login token into
  `localStorage`** so the embedded console is silently authenticated. Revisit
  once the console itself is behind real auth. *Source: `handlers/dashboard_proxy.go`.*

Note: the Sympozium API (`.212`) and the serving agents (`.213`/`.214`/`.222`)
**do** require a bearer token (they returned `401` without one). Good — keep it.

## 2. Secrets & sovereignty

- [ ] **Sev1 — demo CA private keys are committed and the repo is public.**
  `03-target-cluster/warm-ca/*.key` (client, server, etcd peer/server,
  request-header, service) are in git history on a public GitHub repo. Labelled
  "demo use only", but they are world-readable forever. Rotate/scrub history if
  these were ever used anywhere real; regenerate with `gen-warm-ca.sh` and keep
  them out of git. *Source: Chapter 3; `git ls-files 03-target-cluster/warm-ca`.*
- [ ] **Sev2 — a static join token and a fixed SSH public key are committed**
  in the target-cluster manifests. Demo-only, but document the blast radius, and
  do not carry them into a real build. *Source: `03-target-cluster/*.yaml`.*
- [ ] **Sev2 — no model provenance is recorded per AgentRun.** Constraint 7 of
  the strategy doc ("record which LLM + weights digest produced each output") is
  unmet; the Agent CR carries `spec.model` but runs do not record a weights
  digest. Needed for reproducing a decision later. *Source: Chapter 3;
  `Sovereign_Cloud_Agentic_Strategy.md` §4.*
- [ ] **Sev3 — observability is downloaded at install time**, not air-gapped:
  Act 5 fetches Kiali and Prometheus from a public GitHub branch. Vendor them
  locally to match the air-gap story the rest of the platform keeps. *Source:
  Chapter 12; `act5-observability/run.sh`.*

## 3. Agent guardrails (Part IV findings)

- [ ] **Sev2 — a policy-denied tool still runs.** Lab 04's run asks the agent to
  fetch a URL with a tool its `SympoziumPolicy` denies, and it succeeds on
  0.10.75. Default-deny is not enforced; only an explicit override is blocked.
  Enforce tool gating below the model. *Source: Chapter 14; `labs/04-policy/`.*
- [ ] **Sev2 — agent egress allowlist is port-only.** `agent-egress-ollama.yaml`
  allows ports 443, 6443 and 11434 to **any** destination, not to named hosts, so
  an agent can reach any HTTPS/API endpoint on those ports. Scope egress to the
  intended destinations. *Source: Chapter 14; `06-sympozium/agent-egress-ollama.yaml`.*
- [ ] **Sev2 — the `sandbox-restricted` policy selects no pods.** The sandbox
  NetworkPolicy/label the strategy assumes ("sandbox the sandbox", Constraint 4)
  is not actually applied to the chat agent. *Source: Chapter 14.*
- [ ] **Sev2 — agent runs can write in their namespace and read every secret in
  it,** including a target-cluster admin kubeconfig. Tighten the per-run Role.
  Flagged in Chapter 14's draft notes as a fix-before-publishing item; decided
  18 September to track here instead of blocking the book on it — the chapter
  describes the hole as open, not fixed. *Source: Chapter 14.*
- [ ] **Sev3 — refusal lives only in the system prompt** for `mesh-sre-agent`
  (the "metrics unavailable" RULE 1). qwen2.5:7b does not reliably obey it. If
  refusal must hold, enforce it below the model (toolGating / fail-closed egress).
  *Source: Chapter 14; `CLAUDE.md` mesh-sre-agent section.*

## 4. Correctness bugs (things that report the wrong thing)

- [ ] **Sev2 — the target-cluster worker is unusable.** It carries
  `node.cloudprovider.kubernetes.io/uninitialized=true:NoSchedule` (from
  `kubelet-arg: cloud-provider=external` with `disableCloudController: true`),
  so it accepts no workloads while every dashboard shows it `Ready`. Either run a
  cloud-controller to clear the taint, or drop the external-cloud-provider config.
  *Source: Chapter 5; `03-target-cluster/target-cluster-*.yaml`.*
- [ ] **Sev2 — Act 3's failover check counts any non-empty body as success,** so
  a total outage prints `success=20 failed=0`. Make `run.sh` count HTTP 200 (the
  console's `HandleIstioFailover` already does). *Source: Chapter 11;
  `07-istio-advanced/act3-multicluster/run.sh`.*
- [ ] **Sev2 — `verify.sh` never tests failover behaviour** — its Act 3 checks
  assert only that gateways/label/secret exist. Add a behavioural check that
  scales local endpoints to zero and asserts a 200 from the peer. *Source:
  Chapter 11; `07-istio-advanced/verify.sh`.*
- [ ] **Sev2 — the topology view is labelled `live` while drawing 14 nodes that
  don't exist** (13 shown healthy). It matches VMs by exact name (`target-cluster-cp`
  vs the real `target-cluster-cp-s9j8h`) and queries a `sample` namespace that no
  longer exists (an empty list reads as success). Match VM names by prefix; treat
  an empty/missing target as not-found. *Source: Chapter 12; `ui/backend/handlers/mesh.go`.*
- [ ] **Sev3 — the model gateway's own metrics carry no model or token labels,**
  and nothing displays them; model traffic is absent from `istio_requests_total`
  and Kiali. Add a token-by-model meter if AI cost visibility matters. *Source:
  Chapter 12.*
- [ ] **Sev3 — the AI/agent traffic doesn't use the gateway at all;** all three
  agents call the host Ollama through the cluster2 shim. Decide whether to route
  them through `.203` (needs the egress widening in §3) or drop the "one control
  point" claim. *Source: Chapter 12.*
- [ ] **Sev3 — Act 1's ListenerSet is refused for a missing `ReferenceGrant`**
  (its cert lives in another namespace) and the act prints its status instead of
  asserting it, so nobody noticed. Add a `ReferenceGrant`, or drop the beat.
  *Source: Chapter 11; `act1-gateway/06-listenerset.yaml`.*
- [ ] **Sev3 — Cluster API reports the working target cluster as
  `Available=False` (`WaitingForKthreesServer`).** k3s CAPI control-plane provider
  v0.3.0; not investigated. Check the cluster-api-k3s issue tracker. *Source: Chapter 4.*
- [ ] **Sev4 — the "locality prefers local" claim is printed without measuring;**
  healthy traffic actually split 30/30 across clusters, and the local/remote
  classifier only recognises one of three remote pods. Fix the classifier or drop
  the claim. *Source: Chapter 11; `act3-multicluster/run.sh`.*

## 5. Measurement scripts

- [ ] **Sev2 — timing scripts count nodes instead of naming them.**
  `phase-timings.sh` and `time-to-ready.sh` wait for "2 nodes Ready", which a
  ghost node baked into the `:warm` image satisfies before the worker joins;
  `phase-timings.sh` then treats a missing worker timestamp as zero, so "both
  nodes" silently equals the control-plane time. Wait for `-cp-` and `-workers-`
  by name; fail (don't zero) on a missing timestamp. A working by-name procedure
  exists (see the scratch `measure-by-name.sh`); consider committing it as
  `scripts/time-to-ready-by-name.sh`. *Source: Chapters 7 and 17; Appendix D.*
- [ ] **Sev3 — the worker never logs the "k3s started" line the instrument looks
  for,** so the worker phase row is always empty. Anchor on a line the static
  worker bootstrap actually prints. *Source: Chapter 7; `phase-timings.sh`.*
- [ ] **Sev3 — remove or fix the stale ghost node.** The `ubuntu-bake-vm-warm`
  Node object baked into the `:warm` image is cleaned up asynchronously after
  startup; that async cleanup is what makes the count bug possible. *Source:
  Chapter 7.*

## 6. Rebuild / reproducibility gaps

- [ ] **Sev2 — `:latest` and `:preinit` golden images still carry the old k3s
  v1.31.4 binary** after the September upgrade; only `:warm` was rebaked.
  `ensure-warm-image`/`pre-pull` check tag *presence*, not version, so the legacy
  `target-cluster-lite/full/parallel` paths would silently deploy the old k3s.
  Rebake them (`make bake-image build-containerdisk-preinit pre-pull-preinit`,
  and the non-preinit equivalents), or make the checks version-aware. *Source:
  Chapter 17; `CLAUDE.md`.*
- [ ] **Sev3 — `init-management-cluster.sh` has no timeout margin around its own
  readiness waits.** Corrected 18 September: the `configure-kubevirt-perf.sh` step at
  the end of the script can be skipped not because it races KubeVirt's operator (the
  original diagnosis), but because an external time limit shorter than the script's
  own up-to-600s readiness waits can kill it first — and because output is often
  piped through `tail`, that looks like success. The `supportContainerResources` fix
  is cluster state, not manifest state; verify after every `capi-init`, and give the
  script's caller enough time budget to actually finish. *Source: Chapters 8/17;
  `CLAUDE.md`.*
- [ ] **Sev3 — the minimal DataVolume was not re-imported** after the rebuild
  (confirmed still missing 18 September), and Kind node images are pinned by tag,
  not digest. `scripts/import-base-images.sh` creates it but isn't referenced from
  the Makefile or README. *Source: `CLAUDE.md` upgrade note; Chapter 17.*
- [ ] **Sev1 — the branch this book documents is not merged to `main`.**
  `upgrade/k8s-1.37-istio-1.31-sympozium-0.10.75` is 40 commits ahead of `main`
  (161 tracked files on `main` vs 312 here) — nearly all of the mesh-sre-agent work,
  the September upgrade, and most of `07-istio-advanced`/`06-sympozium` that Parts
  II-V's "Open the repo" pointers depend on. Anyone cloning `main` will not find most
  of what the book cites. Merge (open a PR) before print, or have every "Open the
  repo" block name the branch explicitly. *Source: Chapter 17, found 18 September
  while committing this file.*
- [x] **Fixed 18 September — the warm pool pointed at a stale March `:latest`
  image and had never been re-measured.** `POOL_STANDBY_MANIFEST` now defaults to
  `target-cluster-warm.yaml` (`run-ui.sh`), with CA/token seeding added to
  `buildStandby` in `pool.go` so KThrees adopts the baked material; `pool.go`'s
  `targetNodesReady` now counts nodes by name, not by count, so a ghost node from
  the `:warm` image's bake VM can't falsely satisfy readiness (same class of bug as
  §5, confirmed live in this code path too, not just the standalone scripts). Claim
  re-measured, three clean runs: 1.059 s / 0.467 s / 0.465 s — **median 467 ms**
  (was 194 ms in June). Still ~72x faster than a cold build. The pool itself remains
  opt-in (`POOL_ENABLED`, default off) — this fixes what happens when it's on, not
  the default. *Source: Chapter 9.*

## 7. Stranded / stale components

- [ ] **Sev3 — the FinOps agents can't install.** `cost-analyzer.yaml` and
  `incident-responder.yaml` (the April idle-VM and incident personas) target a
  resource type the current chart no longer ships, so `make sympozium-demo-agents`
  fails. The tool meant to measure the owner's own waste is itself broken. Rewrite
  them against the current CRDs. *Source: Chapters 16 and 18.*
- [ ] **Sev3 — the warm-up heartbeat schedule stopped** (found not running; cause
  not established, and its history shows far fewer runs than a 4-minute cadence
  implies). Restart and confirm it fires. *Source: Chapters 9 and 14.*
- [ ] **Sev3 — the security agent (`.217`) was not redeployed after the rebuild**
  and the `kubeui` namespace does not exist (the production UI manifest isn't
  applied). Decide whether these are part of the platform or demo-only. *Source:
  Chapter 16; Appendix A/B.*
- [ ] **Sev4 — `demo.sh` and `show-cluster.sh` narration is stale** (broken by the
  registry alias; describes the old topology). *Source: Chapter 5; `CLAUDE.md`.*
- [ ] **Sev4 — kubeui's agent dropdown queries the removed `SympoziumInstance`
  CRD** and falls back to listing only the default agent, so new agents never
  appear. *Source: Chapter 13; `ui/backend/handlers/ai.go`.*
- [ ] **Sev4 — seven MetalLB addresses are reserved but unused, each named in
  3–10 files** (`.200`, `.211`, `.216`–`.220`). Retire the ones that are truly
  dead to avoid a future collision. *Source: Appendix B.*
- [ ] **Sev4 — the skills-webhook is redundant** (0.10.75 copies `spec.skills`
  onto runs) but still deployed and harmlessly no-ops. Retire it. *Source: `CLAUDE.md`.*

## 8. Documentation drift

- [ ] **Sev3 — the README front-page architecture diagram shows the March
  topology** (`mc-demo`, the `sample` namespace, `.216` proxy, VM IPs `.60/.61`)
  that no longer exists. Redraw. *Source: Chapter 4; `README.md`.*
- [ ] **Sev3 — the `05-istio` cross-cluster mind map (and its byte-identical copy
  in `~/marketing/`) describes a masquerade NAT hop that was removed in the same
  commit that added the diagram.** The VMs use bridge networking. Fix or annotate.
  *Source: Chapter 10.*
- [ ] **Sev4 — `CLAUDE.md` stale spots:** `.2` is now `cluster1-control-plane`
  (not `cluster2-control-plane`); "the target CA changes on every redeploy" is not
  true for the warm path (fixed CAs); the `mesh-sre-agent` verify command still
  uses `--as=...:sympozium-agent` (per-run SAs since 0.10.75); one 0.10.75
  paragraph still says the model is "confabulating". *Source: Chapters 10, 14, 15.*
- [ ] **Sev4 — `make help` omits targets** (`kubevirt-perf`, `phase-timings`, all
  `istio-adv-*`) and still calls the default cluster a "~40s target" (measured
  33.7 s). *Source: Appendix A.*
- [ ] **Sev4 — the labs index still describes `SympoziumInstance` objects** (no
  longer shipped) and documents "observed 0.10.38 behavior" (three upgrades old).
  *Source: Appendix E; `06-sympozium/labs/README.md`.*

## 9. Book (`own-your-cloud` repo)

- [ ] **Sev2 — resolve every chapter's *Draft notes* before publication:** confirm
  primary sources (a16z, Dropbox S-1, 37signals, Flexera, Gartner, Grand View,
  AT&T/Broadcom filings, EU Data Act, GDPR/Schrems II/CLOUD Act/DPDP), and the
  hyperscaler depreciation filings for Chapter 18. Several notes flag figures the
  author must verify against primaries, not the search summaries used while drafting.
- [x] **Decided 18 September — authorship disclosure:** name the committer
  (Mahipal) in Chapter 18; how much further to describe the AI co-authorship
  (63 of 81 platform commits) is still open.
- [x] **Decided 18 September — `own-your-cloud` license:** All rights reserved.
  `LICENSE` added.
- [ ] **Sev3 — supply real cost figures for Chapter 18's worksheet,** or keep the
  "meter one workload in parallel" framing. No Meridian savings are currently claimed.
- [ ] **Sev4 — remove the duplicate `/mnt/mil/book/` copy** (now redundant with the
  `own-your-cloud` repo) or gitignore it in `sovereign_cloud`.
- [ ] **Sev4 — commit the measurement scripts** used for the live numbers (the
  `ch10`–`ch18` scratch scripts and `measure-by-name.sh`) somewhere durable, so
  Appendix D and the chapters' "Open the repo" pointers resolve.

---

*Generated 17 September 2026. Severity is a rough guide, not a commitment.
Cross-references point at `book/chapters/*` in the `own-your-cloud` repo and at
files in this repo.*

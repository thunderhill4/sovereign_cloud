# Chapter 14 — Rules the Model Cannot Break

> *If refusal must be guaranteed, enforce it below the model … not in the prompt.*
>
> — `CLAUDE.md`, the `mesh-sre-agent` section

*Meridian · a Thursday in August · the CTO's office*

Anita Rao had the strategy document open at a section called *Architecture Constraints*. Eight numbered rules, each one sensible.

"Local model first. Pinned endpoint. Address discipline. An egress allowlist. Keep the model warm. Append-only audit. Tenant isolation. A sandbox for anything that takes free-form chat." She scrolled back to the top. "These are good rules."

"They're the contract," Vikram said. "Every change to the agent layer is supposed to satisfy them."

"Then I want one more column." She turned the screen toward him. "Next to each rule, write what happens if the agent ignores it."

"The agent doesn't decide most of these."

"Then write down who does. A person. A script. The API server. The permission system. The network." She paused. "And if the honest answer is *we told the model*, write that down too."

"That's not going to be a flattering column."

"It isn't supposed to be," Anita said. "I'd rather read it from you than from an incident report."

* * *

*Build log · July–September 2026*

## Where a rule can live

A rule for an AI agent can live in several places, and they are not equally strong. From weakest to strongest:

1. **The prompt.** The model is asked.
2. **A document.** People are asked.
3. **A script.** The right thing happens, if someone runs it.
4. **Admission.** The API server refuses to create the object.
5. **Permissions.** The credentials cannot do it.
6. **The network.** The packets never arrive.
7. **Absence.** The tool, the credential, or the route does not exist.

A model can ignore the first. People can ignore the second and third. The rest do not care what anyone intended. This chapter is the column Anita asked for: each of the platform's eight rules, and the layer that actually enforces it, measured on the running platform rather than read from its documentation.

Most of the measurements were taken on 17 September 2026, on Sympozium 0.10.75 and Kubernetes 1.37, specifically for this chapter. Where they contradict what the repository says, the chapter says so.

## The rule that lived in the prompt

Start at the bottom of the ladder, because the repository has already measured what happens there.

The `mesh-sre-agent` from Chapter 15 was deliberately given no path to metrics — no route to Prometheus or Kiali. Its briefing opens with a rule meant to stop it from pretending otherwise:

*"RULE 1 — METRICS ARE UNAVAILABLE. … If asked for latency, p50/p95, request rates, error rates, throughput or saturation, do NOT run a command and do NOT estimate: reply exactly 'Metrics are not reachable from this agent …' Never invent a number."*

Measured on `qwen2.5:7b` across repeated attempts, the repository records that it *"does **not** reliably comply: it ignores 'do not run a command', and in one run answered 'we can make an educated guess based on the availability of related metrics'."* It never actually produced an invented number in testing. But moving the rule to the very top of the prompt did not fix the compliance, and the conclusion written into the repository is the sentence this chapter takes as its epigraph: *"the guard is a nudge, not a control."*

The same repository gives every agent the same closing instruction in its briefing: *"Read-only unless the user explicitly approves a change."* Hold on to that sentence. It comes back.

## What held

Some rules genuinely live below the model, and they held when tested.

**Admission refused a self-granted exception.** The platform's policy for `cluster2-agent` denies a web-fetching tool called `fetch_url`. Lab 04 tries to create a run that explicitly re-allows it. On 17 September, asked to validate that run without creating it, the API server refused, exactly as the lab recorded in July:

```
Error from server (Forbidden): admission webhook "vagentpod.sympozium.ai"
denied the request: tool "fetch_url" is denied by policy
```

That webhook is configured to *fail closed*: if it cannot be reached, the request is rejected rather than waved through. Its sibling, which validates tool packs, is configured the same way, and on 16 September it showed what that means in practice: while the platform was being reinstalled and the webhook was not yet serving, the install was refused until it came up. An inconvenience during setup; the right behavior for a guard.

**Permissions kept agents read-only outside their own namespace.** An agent run's identity was checked directly. Across the cluster, it cannot delete deployments or virtual machines, and it cannot create workloads in other namespaces. The extra grants the repository adds, for virtual machines and for the service mesh, are read-only by design and say so in their headers.

**The network closed the metrics path.** This one needed a test, because the lab written in July said the opposite (more on that below). Two short-lived pods were started in the agents' namespace: one carrying the same labels as a real agent run — `sympozium.ai/role=agent` and `app.kubernetes.io/part-of=sympozium`, confirmed against a live run pod the same morning — and an identical control pod with neither label. Both tried the same four destinations:

| Destination | Agent-labeled pod | Unlabeled control |
|---|---|---|
| Ollama, through the in-cluster Service (port 11434) | `200` | `200` |
| Kiali on `cluster1` (port 20001) | no connection | `302` |
| `example.com` over HTTP (port 80) | no connection | `200` |
| `example.com` over HTTPS (port 443) | `200` | `200` |

The control pod reached everything; the agent-labeled pod could not reach Kiali or plain HTTP. Network policy is enforced on this cluster, and the metrics endpoints really are out of an agent's reach. Whatever the model says about latency, it did not read it from Prometheus.

**The UI will only execute one action.** However an agent phrases a proposal, the web backend acts only on an allowlist written in code, and that list has one entry: deploying a target cluster. A model cannot talk its way onto that list.

Those four are real controls. Now look at the last row of the network table again.

## What did not hold

**HTTPS goes anywhere.** The agent-labeled pod reached `example.com` on port 443. The reason is in the repository's own network policy, `sympozium-allow-ollama`. It was written to open two paths agents genuinely need — Ollama, and the Kubernetes API — and it opens them *by port*, with no destination:

```
to=  ports=[{"port":11434,"protocol":"TCP"}]
to=  ports=[{"port":443,"protocol":"TCP"},{"port":6443,"protocol":"TCP"}]
```

Port 443 was opened for the Kubernetes API, and a port number cannot tell the Kubernetes API from the rest of the internet.

**A denied tool ran anyway.** Lab 04's other run creates no exception at all. It simply asks the agent to fetch `example.com` with the tool its policy denies. Re-run on 17 September, the agent made one tool call, and the answer described the real page's contents. The deny rule only blocks a run that *explicitly asks* to re-allow the tool; a run that says nothing about tools — which is how nearly every run is written — can use it freely. Put that together with open HTTPS and the result is straightforward: an agent on this platform can send a request to an arbitrary web server.

Notice the irony in the epigraph's source. The same note that says to enforce refusal below the model names policy tool gating as one way to do it. On this version, tool gating is the control that only works when nobody needs it.

**Inside their own namespace, agents can write.** The built-in `k8s-ops` tool pack grants two sets of permissions. Across the cluster, they are read-only. Inside the namespace where the run executes, they are not: `create`, `update`, `patch`, and `delete` on pods, deployments, jobs, network policies, roles, role bindings — and secrets. Checked against a real run identity from a chat with `cluster2-agent`:

```
delete pods -n sympozium-system        yes
get secrets -n sympozium-system        yes
```

That namespace is where the agent platform itself lives. It holds every agent's API key, the model credentials, and one more Secret worth naming: `target-cluster-kubeconfig`, whose user is `target-cluster-admin`, for the target cluster's API on port 6443 — which the network policy above leaves open. A `cluster2-agent` run's identity is permitted to read it.

`cluster2-agent`'s briefing says its tools talk to its own cluster only, and every briefing says *read-only unless the user explicitly approves a change.* Outside the namespace, permissions make that true. Inside it, the only thing standing between a run and those writes and those credentials is the sentence in the prompt — the layer this chapter began by measuring.

## The scorecard

Here is the column, rule by rule.

| # | Constraint | What actually enforces it today | Verdict |
|---:|---|---|---|
| 1 | Local-LLM-first | The agent's configuration; not the network, which allows HTTPS anywhere | Partly |
| 2 | Pinned model endpoint | An in-cluster Service defined in the repository | Held — at a different address than the rule names |
| 3 | Address-pool discipline | MetalLB within each cluster; only a table and a warning between the two | Partly |
| 4 | Egress allowlist | Network policy, by port rather than destination | Partly |
| 5 | Warmth is mandatory | A setting on the host, outside the repository | Partly, and by accident |
| 6 | Audit is append-only | Nothing | Not held |
| 7 | Tenant isolation | Nothing inside the shared namespace | Not held |
| 8 | Sandbox free-form chat | A policy reference that never applies | Not held |

A few rows need their evidence.

**Rule 2** names the Ollama address `172.18.0.1` and warns against a hostname that does not resolve on this kind of cluster. The agents no longer dial that address at all. The repository records that agent pods *"couldn't dial that IP directly under kindnet's NetworkPolicy enforcement,"* so it added an in-cluster Service whose endpoint points at the host. The rule's intent — one fixed, local model endpoint — held. Its text is out of date.

**Rule 3** requires every load-balancer address change to land in three places at once. Within a single cluster, MetalLB itself refuses to hand the same address to two services. But `cluster1` and `cluster2` advertise on the same network segment, and nothing stops their pools from overlapping except a table in the repository and a warning never to run one cluster's install script on the other. The pool the rule describes, `.211–.220`, has already been widened to `.211–.225`.

**Rule 5** says no persona enters the fleet without a warm-path guarantee, and names the mechanism: a heartbeat schedule. The repository ships one, set to run every four minutes. On 17 September its status recorded two runs in total, the last at 07:44 UTC the previous day, with a "next run" time that had passed seventeen hours earlier. The schedule is not firing. The model stays warm anyway, because the host's Ollama service runs with `OLLAMA_KEEP_ALIVE=-1` and never unloads a model once it is loaded. That setting lives in the host's service configuration, not in the repository (whose Ollama notes suggest 24 hours). It keeps only models that have already been loaded: on 17 September, `qwen2.5:7b` was resident, and `llama3.2`, the target-cluster agent's model, was not.

**Rule 6** is Chapter 13's finding: the audit trail is deletable, was deleted by routine scripts, and vanished entirely in the rebuild.

**Rule 7** says one namespace per tenant and one identity per agent, with no cross-tenant bindings. Every agent shares one namespace. Identities are now per run — better than the rule asked — but Chapter 15's fix bound mesh and VM read access to every service account in that namespace, and the write permissions above let a run of any agent equipped with `k8s-ops` — `cluster2-agent` or `mesh-sre-agent` — read every other agent's key.

**Rule 8** puts free-form chat under a restricted network policy allowing DNS, the Ollama host, and the Kubernetes API only. `cluster2-agent` does reference that policy. But the policy's network rules are scoped to pods labeled as sandboxed, and every agent in the repository runs with its sandbox disabled, so no agent pod carries the label. On 17 September, a real run pod carried `sympozium.ai/role=agent` and no sandbox label at all. The restricted rules exist; they select nothing.

## A correction

Lab 04, written in July, concluded that network policy was *"inert on this platform"* for two reasons. The first — the missing sandbox label — is still true. The second was that Kind's default network plugin *"has no policy engine"* and never enforces policy. On 17 September, on the rebuilt cluster, that was false: the probe pods above show enforcement plainly, and the repository's own shim Service for Ollama exists *because* a policy blocked a direct connection.

Whether the lab's second finding was wrong in July, or became wrong later, the record cannot say — the July cluster no longer exists. What it shows is the chapter's thesis turned on its authors. "Network policy is not enforced" was a claim about a control, made without a test that could have failed. The test takes two pods and a minute.

## Rules that cannot move below the model

Not every rule can be pushed down the ladder, and it is worth being clear about which.

*"Never invent a number"* is one. The network can guarantee that the agent cannot read real metrics — and here it does. It cannot stop a language model from writing a plausible latency figure out of nothing. No permission, admission rule, or firewall reaches inside the text a model generates. For rules like that, the control is not enforcement but verification: the ladder of evidence in Chapter 15, and a process that treats an agent's answer as a claim.

Everything else in this chapter's failures *can* move down, and the moves are unglamorous:

- Replace port-only egress with destinations: the API server's address, the target cluster's address, the model host.
- Run agents somewhere their credentials cannot read the platform's own secrets, or strip the tool pack's write permissions inside that namespace.
- Enable the sandbox, so the restricted policy actually selects agent pods.
- Replace the heartbeat that stopped with something whose failure someone notices, and write the host's keep-alive setting into the repository.
- Copy run records off the cluster before anything can delete them.

None of these needs a better model. All of them need someone to decide that a rule is not real until something other than the model enforces it.

* * *

*Meridian · two weeks later*

Vikram brought the column back on one page, with the verdicts in the right-hand margin: one rule held, four held partly, three did not.

Anita read the page twice, then asked the question she always asked first. "Which of these lets an agent do something we can't undo?"

"Seven. Inside its own namespace, an agent's run can write, and it can read every secret there — including an administrator credential for the target cluster."

"And what stops it today?"

"A sentence in its briefing."

She drew a box around row seven. "That one first. Then the network rule. Then the sandbox." She slid the page back. "And strike *Architecture Constraints* off the top of the document."

"And call it what?"

"*Intentions*," she said, "until each one has something under it that the model can't argue with. If the model can break it, it isn't a rule. It's a request."

## The ledger

- **Tested on 17 September, current release:** network enforcement with two probe pods; both of lab 04's runs; the real permissions of an agent run's identity; the warm-up schedule's status; the resident models; how the admission webhooks fail.
- **Held below the model:** admission refused an explicit tool override, and fails closed; permissions kept agents read-only outside their namespace; the network blocked metrics endpoints and plain HTTP; the UI backend executes one allowlisted action.
- **Did not hold:** a denied tool was used anyway; HTTPS reached an arbitrary host; agent runs can write in their own namespace and read every secret there, including a target-cluster administrator credential; the chat sandbox's policy selects no pods; the warm-up schedule stopped firing seventeen hours before the check.
- **Held by accident:** model warmth, kept by an unrecorded host setting rather than the mechanism the rule names.
- **Corrected:** lab 04's claim that network policy is inert on this cluster.
- **Cannot be enforced below the model:** "never invent a number." That one needs verification, not a control.

## Ask your team

1. **For each rule we have given our AI agents, what enforces it if the model ignores it?** Count how many answers are "the prompt."
2. **What can an agent's credentials read, and which of those secrets unlock something bigger than the agent?**
3. **When a guardrail fails — a webhook down, a policy matching nothing — does it fail open or closed, and would anyone notice?**

## Open the repo

- `Sovereign_Cloud_Agentic_Strategy.md` — "Architecture Constraints", and §4 for the reasoning behind them.
- `06-sympozium/labs/04-policy/` — both runs; re-run `run-allowed.yaml` with `--dry-run=server` to see the refusal without creating anything.
- `06-sympozium/policies/sandbox-restricted.yaml` beside `06-sympozium/agent-egress-ollama.yaml` — the destination-specific allowlist that never applies, and the port-only rule that does.
- `kubectl auth can-i --list -n sympozium-system --as=system:serviceaccount:sympozium-system:sympozium-run-<run-name>` — what a real agent run can actually do.
- `06-sympozium/mesh-sre-agent.yaml` — RULE 1, and the header explaining why the metrics path is closed.
- `ui/backend/handlers/ai.go` — `allowedActions`.
- **Without the repo:** Appendix G.5 quotes `agent-egress-ollama.yaml` whole — the port-only rules are visible in it.

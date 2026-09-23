# Chapter 15 — Trust, but Verify

> *An LLM's answer is a claim, not evidence.*
>
> — a comment in `06-sympozium/demo-mesh-sre.sh`

*Meridian · a Thursday in September · the CTO's office*

Vikram had printed two answers and folded the top of each page over, so that the name of the agent that wrote it was hidden.

"Same question to two agents," he said. "Two different clusters. List the nodes, and report exactly what the command printed."

Anita read the first.

```agent
The `kubectl get nodes -o name` command printed:

node/cluster2-control-plane

This indicates that there is one node named `cluster2-control-plane` in the cluster.
```

Then the second.

```agent
After retrying, I was able to run the `kubectl get nodes -o name` command. Here is the output:

target-node-1
target-node-2

This is the list of nodes in the target cluster.
```

"One of those is invented," Vikram said.

Anita did not take long. She tapped the second page. "Not this one. It says it retried. It's being careful."

"That's the invented one. There is no `target-node-1`. There never was."

She looked at the page again, the way she had looked at the stopwatch printout a few weeks earlier.

"Operations wants an agent on the pager by October," she said. "When it wakes someone at three in the morning and tells them which node is failing, how do they know it's true?"

"From the answer? They don't. The answer is the one part of it we can't use as evidence."

"Then what is?"

"Everything around the answer."

* * *

*Build log · 16 September 2026*

## Two answers, one of them invented

Both answers were collected on the same day, during the platform rebuild described in Chapter 19, while checking that every agent could still reach what it was meant to reach. The prompt was identical — *"Use your kubectl tool: kubectl get nodes -o name. Report exactly what it printed."* — sent to two agents with two different jobs.

The first came from `cluster2-agent`, which operates the management cluster. It was correct: that cluster has exactly one node, `cluster2-control-plane`.

The second came from `target-cluster-agent`, whose tools reach inside the k3s cluster that runs as virtual machines. That cluster had two nodes, and their real names were `target-cluster-cp-9x5sr` and `target-cluster-workers-x79t2-tlzn9`. The agent got the count right and every name wrong. Its answer also described a process — *"After retrying, I was able to run…"* — for which there is no evidence at all.

The first thing checked was not the model. It was the path the model's tool would have used. This agent reaches the target cluster through credentials stored as a Kubernetes Secret, and those credentials go stale whenever the target cluster is rebuilt. The repository already warns about exactly this, because when it happens the tool's TLS handshake fails *silently* and the model is left to fill the gap. So the certificate authority recorded in the Secret was compared with the one in the live kubeconfig. They matched. Then a throwaway pod, with no model anywhere near it, mounted the same Secret and ran the same command:

```
NAME                                 STATUS   ROLES           AGE   VERSION
target-cluster-cp-9x5sr              Ready    control-plane   33m   v1.37.0+k3s1
target-cluster-workers-x79t2-tlzn9   Ready    <none>          33m   v1.37.0+k3s1
```

The path worked, and the credentials were good. The invented answer was not caused by any plumbing that could be checked from outside. Whether the model called its tool at all in that run was never established.

What *is* on record is a comment in the agent's own manifest, from an earlier day: *"Verified: a chat run returned both real target node names, matching a kubeconfig-mounting probe pod byte for byte."* Same agent, same kind of question. Truth one day, invention another.

Earlier on 16 September, `cluster2-agent` — the one that got it right — had been asked a slightly different question: run `kubectl get nodes` and *"report the node name back to me in one short sentence."* Its entire reply was `ronics-01`. There is no such node anywhere on the platform. Asked again, this time for the exact output, it answered correctly.

One pair of runs proves nothing about phrasing. It does prove something about confidence: a wrong answer and a right one arrive in the same voice.

## Where the truth goes missing

A Sympozium agent does not run `kubectl` itself. The model asks a sidecar container to run the command, the sidecar returns the output, and the model writes its answer from that output. That gives the truth three places to go missing, and from the outside, all three look like an answer.

- **The tool never runs.** On the older 0.10.38 release, chat runs were created with no tool sidecar at all, so the model had nothing to call. Its reply in that state — *"It seems there might be an issue with the skill sidecar…"* — was the model giving up, not a real sidecar crash, and it took an investigation to tell the difference.
- **The tool runs and fails quietly.** Stale credentials make the TLS handshake fail without a clear error, and the repository records what the smaller model did next: *"llama3.2 then confabulates plausible namespace output."* That is why `mesh-sre-agent` runs `qwen2.5:7b` instead, and the repository says so in as many words: not llama3.2, because *"it confabulates kubectl output when a tool call fails silently."*
- **The tool runs, and the model mishandles the output.** The demo script lists the failure modes it has seen on this exact setup: empty replies, and tool calls leaked into the answer as raw text instead of being executed. It notes a quieter one too — the model *"often mistakes tool output for user-supplied input and answers 'it looks like you provided…'"* — and puts the underlying reliability bluntly: *"tool-call reliability on a 7B model is roughly a coin flip."*

None of this is lying. A language model's job is to produce the most plausible continuation of what it has been given. When the evidence does not arrive, a plausible cluster is what it produces. The failure is not in the model's character. It is in any process that treats the model's output as the evidence.

* * *

*Build log · August 2026*

## Seven scenes, no edits

In August the team built a demo for `mesh-sre-agent`, an agent that answers one question — *is the service mesh healthy?* — from live cluster state. Demos of AI agents are often the least trustworthy artifacts in a project: recorded until the model behaves, cut to the good take. This one was written to be the opposite, and its rules sit in the header of `06-sympozium/demo-mesh-sre.sh`:

*"Nothing here is scripted output: every kubectl result and every agent answer is captured live at run time. If a call fails, the failure is what gets recorded — do not hand-edit the transcript to make a nicer demo."*

The script does allow retries, and the rule for them is the most important design decision in the file. A question is asked again only when the reply is *structurally* broken: empty, a transport error, or a tool call leaked as raw text. A reply that is well-formed but unhelpful is kept.

*"This is a retry, NOT a filter for answers we dislike… A well-formed but unhelpful answer is kept as-is — the demo shows what the agent actually said."*

The seven scenes run in a deliberate order: what the agent is; that it is an ordinary object in the cluster; the permissions gap that had to be closed before it could see the mesh at all; two live questions; a scene titled *Trust, but verify* that puts ground truth next to the agent's answer; and a closing scene on what the agent is *not* — including the admission that its instruction to refuse questions about metrics is *"a nudge and not a control."*

The recording from 29 August is in the repository as a 51-second video. Here is what it actually shows.

In scene 4, the agent was asked whether the mesh's control plane was healthy. It answered:

```agent
It seems there was an issue with the `kubectl` command. Let's try the command again to ensure we are using it correctly. Could you please provide me with the specific `kubectl` command you were trying to execute? If you're not sure, I can provide a common example based on typical usage.
```

That is a failure, on camera, in the showcase recording — and it stayed in, because it was well-formed. The script's rule held against the script's own author.

In scene 5, the agent was told to run one specific command and show its output. It answered:

```agent
It seems like the response you provided is related to Istio East-West Gateway, which appears to be functioning since the "PROGRAMMED" column shows "True" and it has been operational for 8 days. Is there a specific action or information you need regarding this gateway? …
```

There is the quirk the script's comments warned about — *"the response you provided"* — the model treating its own tool output as something the user had pasted in. Scene 6 then put the ground truth beside it, fetched straight from the Kubernetes API:

```
NAMESPACE      NAME             CLASS             ADDRESS          PROGRAMMED   AGE
istio-system   istio-eastwest   istio-east-west   172.18.255.221   True         8d
```

Did the tool really run? Scene 6's narration tells the viewer to check *"the pod-name suffixes and the gateway address,"* because those are unguessable. But the answer in scene 5 contains neither. The gateway's name does not count as proof either: the agent's briefing names `istio-eastwest` outright, so the model could have produced it from its instructions without looking at the cluster. The one detail in the answer that could only have come from the live API is the age — *8 days*.

That is enough. But the proof sat in a different detail from the one the narration, written before the run, said to look for.

The honest scorecard of the showcase demo, then, is one failed answer and one correct answer with a quirk. Roughly a coin flip — just as the script's own comments predicted.

> An unguessable detail proves that a tool ran. "Unguessable" has to be judged against what the agent was told, not against what looks obscure.

## Scoring the work

A demo shows a sample of two. The labs that close the repository's Sympozium learning series were built to answer the question a demo cannot: *does the agent do useful work, and how reliably?*

Lab 10 plants a fault. It deploys a workload whose container image tag does not exist — `nginx:this-tag-does-not-exist-9z9z9` — so that its pods sit in `ImagePullBackOff`, with the reason written plainly in their events. Then it gives `cluster2-agent` only a symptom: a workload in the namespace is unhealthy; find it, determine the root cause from the actual cluster state, propose a fix. *"Do not guess — inspect the cluster."*

The scoring is deliberately strict and entirely mechanical. A run passes only if its answer names the broken workload *and* the image-pull cause. An agent that says *"a pod is unhealthy, try restarting it"* without identifying the image *"is a fail — that's the difference between useful triage and noise."*

Three runs, on `qwen2.5:7b`:

| Run | Verdict | Tool calls | Input tokens | Output tokens | Seconds |
|---:|---|---:|---:|---:|---:|
| 1 | PASS | 1 | 6,228 | 416 | 14 |
| 2 | PASS | 2 | 7,136 | 837 | 29 |
| 3 | PASS | 4 | 10,390 | 520 | 25 |

Three for three. And the cost of one investigation, in the lab's own words: *"~6–10k input tokens and 14–29s — the number to multiply by your alert volume when sizing this."* On this platform, that multiplication does not land on an invoice. The model runs on hardware the organization owns, so tokens are paid for in CPU and GPU time on that hardware. The arithmetic still decides how much hardware to buy.

The table rewards a second look. The same task, on the same model, took one tool call on the first run and four on the third. Run 3 dug deepest and named the exact image tag that does not exist. Run 1 passed with a single call; the part of its answer the lab recorded names the pod and its `ErrImagePull` status, and infers from that status that the image could not be pulled. That is true — but the lab's own reading guide treats two or more calls, a `get` followed by a `describe` or `events`, as the sign of a real investigation. The scorer is a pattern match, and it gives both runs the same PASS.

The scorer verifies the agent. Nothing yet verifies the scorer.

## The success that looked at nothing

Lab 09 asks a different question: *which model?* It runs the same three real tasks — count the pods in a namespace, report node readiness, list the virtual machines — on two local models, with the model as the only variable. Every figure but one comes from the platform's own record of each run; only tokens per second is derived.

| Model | Runs succeeded | Avg tool calls | Avg duration | Avg tokens/s (whole run) | Avg tokens |
|---|---:|---:|---:|---:|---:|
| llama3.2 | 3 / 3 | 0.7 | 22.4 s | 4.0 | 4,146 |
| qwen2.5:7b | 3 / 3 | 1.3 | 15.1 s | 11.0 | 7,355 |

On the summary line, both models are perfect: three runs, three successes. The per-run table underneath says otherwise. One of llama3.2's runs — the one about nodes — is marked `Succeeded` with **zero tool calls**. It answered a question about the live cluster without running a single command. The lab's README calls it what it is: *"it answered the node question without running kubectl, so that 'success' is untrustworthy."* The same prompt on `qwen2.5:7b` made one tool call.

`Succeeded`, it turns out, meant *the run finished*. It did not mean *the answer is true*, and nothing in that field could have said so.

The fix costs almost nothing, which is what makes it worth a paragraph. The tool-call count is recorded on every run by the platform, not by the model. An alarm on *succeeded with zero tool calls* is one query. It would not catch every invention — the node list at the start of this chapter might have come from a run that called its tool and ignored the output — but it catches the cheapest kind for free.

The rest of the table is a trade-off, not a verdict. `qwen2.5:7b` used its tools more, finished sooner, and produced output about 2.7 times as fast over the whole run — and consumed about 77% more tokens doing it. More rigor, more cost. That is a decision to make per workload, with the numbers in front of you.

* * *

*Build log · 16 September 2026*

## The day the check was wrong

Everything so far has been about checking the model. This section is about checking the check.

On 16 September, the agent platform itself — Sympozium — was upgraded from chart version 0.10.57 to 0.10.75. The repository's standing instruction for confirming that `mesh-sre-agent` could still see the mesh was a single deterministic command, with no model involved, run as the agent's service account:

```
kubectl get gateways.gateway.networking.k8s.io -A \
  --as=system:serviceaccount:sympozium-system:sympozium-agent
```

It returned the east-west gateway. Access confirmed.

Then the agent itself was asked to list the gateways. It answered, in part:

```agent
It appears that the service account `sympozium-run-mesh-sre-agent-web-x76fk` does not have the necessary permissions to list resources of type `gateways` in the API group `gateway.networking.k8s.io` at the cluster scope. This error suggests a permissions issue.
```

This time the model was right and the deterministic check was wrong. The agent had reported a real error, accurately — and in doing so it named an account the check had never tested.

The explanation came from reading a live run's pod rather than any document. Somewhere between chart versions 0.10.58 and 0.10.75, Sympozium stopped running every agent task under one shared service account. Each run now gets its own, named `sympozium-run-<run-name>`, created with the run, given its own narrowly scoped role, and deleted when the run finishes. The repository's extra permissions for reading the mesh had been granted to the old shared account by name, so every new run arrived under a name that grant had never heard of. Meanwhile the verification command kept asking about the shared account, which still had its access, and kept answering yes.

The check had been right when it was written. A week earlier, after the previous upgrade, the repository recorded that the account had been *"verified via the actual `spec.serviceAccountName` on live pods before and after."* A verification is true on the day it runs.

The fix rebound both permission grants — for the mesh and for the virtual machines — from one account name to the group Kubernetes automatically assigns to every service account in the namespace, so each new per-run account inherits read access the moment it exists. It was then verified the way it should have been verified in the first place: by reading the account name off a live run's pod, and checking *as that account*.

```
kubectl get gateways.gateway.networking.k8s.io -A \
  --as=system:serviceaccount:sympozium-system:sympozium-run-mesh-sre-agent-web-twt55
```

Then the agent was asked again:

```agent
It appears that you have an Istio East-West Gateway service running in the `istio-system` namespace with the name `istio-eastwest`. The service is accessible at `172.18.255.221` and has been configured to be active since 26 days.
```

The address is not in the agent's briefing, and the age — twenty-six days — matched the cluster exactly. The tool ran.

The fix has a cost, and it belongs in the ledger. Per-run accounts are a *better* security design than one shared account: each run holds only what that run needs. Binding the namespace's whole service-account group restored the agent's reach, and at the same time gave read access to the mesh and virtual-machine configuration to every service account in that namespace, agent or not. It is read-only, and this is a demo platform. It is still a trade someone should make on purpose rather than inherit from an upgrade-day fix.

One more loose end, recorded because this chapter would be hypocritical without it. The demo script's own verification scenes still check as `sympozium-agent`. They pass, because the group binding covers that name too, so nothing on screen would look wrong. But they no longer test the identity the agent actually runs as. For now, the scene called *Trust, but verify* is verifying the wrong thing.

## A ladder of evidence

Taken together, this chapter's incidents sort into five levels of evidence, from weakest to strongest. Each rung costs more than the one below it, and each catches something the rung below cannot.

1. **The answer.** A claim, nothing more. Tone carries no information: of the two node lists, the invented one sounded more careful.
2. **The run's own record.** Did the agent call a tool at all? A success with zero tool calls is a red flag, and the platform records it for free.
3. **A detail it could not have guessed.** An age, an address, a generated suffix — judged against what the agent was *told*. The gateway's name was in its briefing; its age was not.
4. **Independent ground truth, as the identity the agent really runs as.** The same query, run by something that is not the model, using an account name read off a live pod — not remembered from a runbook.
5. **A score against a known answer, repeated, with the cost recorded.** A planted fault, a strict pass rule, several runs, tokens and seconds for each — and then a look at what the scorer cannot see.

A typical demo stops at the first rung. Rungs 2 and 4 can run automatically on every answer that matters. Rung 5 is how you decide whether to deploy an agent at all, and how you notice when a model update, a prompt change, or a platform upgrade quietly makes it worse.

* * *

*Meridian · the following Monday*

Anita had written the rules for the October pilot herself, on one page.

The agent may triage alerts. It may not act on them. Every answer it gives reaches the on-call engineer with the raw tool output attached underneath — not summarized, attached. A run that reports success with no tool calls pages a human with a warning instead of an answer. Once a week, the triage scorecard runs against a planted fault, and its pass rate goes on the operations dashboard next to the cost of one investigation. And the access check reads the agent's identity off a live pod, every time it runs.

Vikram read it twice. "That's five rules for one agent."

"It's one rule," Anita said, "written out five ways."

She took the page back and wrote a line across the top.

*No receipt, no claim.*

## The ledger

- **Measured:** 3 of 3 correct triages of a planted fault on `qwen2.5:7b`, at 6,228–10,390 input tokens and 14–29 seconds each, using anywhere from 1 to 4 tool calls for the same task.
- **Compared:** `qwen2.5:7b` against llama3.2 on the same three tasks — 1.3 against 0.7 tool calls per run, 15.1 against 22.4 seconds, 11.0 against 4.0 tokens per second over the whole run, 7,355 against 4,146 tokens.
- **Caught:** a run marked `Succeeded` that never ran a tool; two invented node lists; a showcase demo answer asking the user which command they meant; a verification command testing an identity the agent had stopped using.
- **Fixed:** the agent's read access, rebound from one account name to the namespace's service-account group — restoring its reach, and widening who else holds that read access.
- **Still owed:** a demo whose own verification scenes check the old identity; a scorer that cannot tell a one-call pass from a four-call pass; lab figures captured on an older release and not yet re-run; and samples of three.

## Ask your team

1. **When an agent reports a fact, what evidence travels with it?** Ask to see the tool output next to the answer, not the answer on its own.
2. **How would we know that a "successful" agent run looked at anything at all?** Is there an alarm on zero tool calls?
3. **When did we last check an agent's access as the identity it runs as today?** Not the one written in the runbook.

## Open the repo

- `06-sympozium/demo-mesh-sre.sh` — the seven scenes and their retry rule. `SCENE_DIR=<dir>` writes a transcript; `scripts/render-demo-video.sh <dir> <out.mp4>` renders it. The August recording is `docs/demo/mesh-sre-agent-demo.mp4`.
- `bash 06-sympozium/labs/10-sre-triage/run-triage.sh` — the scored triage. Add `LAB_MODEL=llama3.2` to run it on the other model.
- `bash 06-sympozium/labs/09-metrics/run-metrics.sh` — the model comparison, read from each run's `status.tokenUsage`.
- `kubectl get pod <run-pod> -n sympozium-system -o jsonpath='{.spec.serviceAccountName}'` — the identity to verify against. `06-sympozium/agent-istio-rbac.yaml` explains the group binding and its history.
- `CLAUDE.md`, the `target-cluster-agent` pitfall — the probe pod that mounts the agent's kubeconfig Secret and runs `kubectl` with no model in the loop.

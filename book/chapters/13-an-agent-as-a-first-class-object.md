# Chapter 13 — An Agent as a First-Class Object

> *Every `AgentRun` remains queryable as a first-class Kubernetes object.*
>
> — Architecture Constraint 6, `Sovereign_Cloud_Agentic_Strategy.md`

*Meridian · a Wednesday in August · the CTO's office*

The vendor's last slide said *AI SRE copilot — connected in minutes*, and the account executive let it sit on the screen while she waited for questions.

Anita Rao had three, written on a pad before the meeting started.

"Where does the model run?"

"In our cloud. Your cluster connects out to us through a lightweight agent. Nothing inbound."

"What did it do last Tuesday — exactly?"

"Every action is in the activity feed in our console, retained according to your plan, and exportable."

"If we cancel, what do we keep?"

"Your exports."

Anita thanked her, and waited for the door to close. Then she turned to Vikram Iyer, who had said nothing for forty minutes. "Same three questions. Ours."

He put a terminal on the wall screen and typed.

```
$ kubectl get agents.sympozium.ai -n sympozium-system
NAME                   PHASE     ACTIVE AGENTS   TOTAL RUNS   AGE
cluster2-agent         Serving   1                            12h
mesh-sre-agent         Serving   1                            12h
target-cluster-agent   Serving   1                            12h
```

"Where does the model run? On a machine in this building. What did it do last Tuesday? Every conversation creates an object in the cluster — the task, the answer, how many commands it ran, how many tokens it used, which account it ran as. You list them with the same command you'd use to list anything else. And if we cancel —" he shrugged. "There's nothing to cancel. It's our cluster's API."

Anita looked at the table for a while.

"So the agent isn't a service we call," she said. "It's a workload we run."

"Under the same permission system as everything else we run."

"Then I'm not choosing an AI vendor," she said. "I'm choosing rules."

* * *

*Build log · February–September 2026*

## From a chat window to an object

The platform's AI tab did not start as a control plane. In February 2026 it was a chat window backed by kagent, talking over an agent-to-agent JSON-RPC protocol. In April the first Sympozium personas appeared — a cost analyzer and an incident responder, written as demo agents. By late June, the tab's chat proxy had been switched to Sympozium and spoke the OpenAI Chat Completions format, the request shape nearly every AI client library already understands. By August the AI tab no longer rendered its own chat at all; it embedded Sympozium's console directly.

Each step moved the agent further from being a feature of the UI and closer to being a thing in the cluster. That is the idea this chapter is about, and the strategy document states the ambition behind it in one sentence:

*"You are not building 'Claude for Kubernetes.' You are building a sovereign, air-gap-capable, VM-native, mesh-aware agentic control plane — a category that barely exists in the open-source world today."*

The same document names the three assumptions this platform breaks. Most agentic Kubernetes work assumes pod-native workloads on a single cluster with cloud models. Here, virtual machines are first-class; there are *"two control planes"*, and an incident on one is not an incident on the other; and *"sovereignty is a hard constraint. External LLM egress is often forbidden."* An agent that operates this platform has to live inside the same boundary as the platform.

## Five nouns

Sympozium describes agents the way Kubernetes describes everything else: as custom resources, reconciled by controllers. Five matter for this part of the book.

- **`Agent`** — a persona. Which model, where to reach it, which tools, which policy, and a standing briefing about the environment.
- **`AgentRun`** — one execution of that persona. It behaves like a Job: it is created, it runs a pod, it finishes, and its result stays behind.
- **`SkillPack`** — a bundle of tools, delivered as a sidecar container with its own permissions.
- **`SympoziumPolicy`** — the rules an execution is admitted under. Chapter 14 is about these.
- **`SympoziumSchedule`** — a trigger that creates runs on a timer.

Here is `cluster2-agent`, the agent that operates the management cluster, as it sits in the repository:

```yaml
apiVersion: sympozium.ai/v1alpha1
kind: Agent
metadata:
  name: cluster2-agent
  namespace: sympozium-system
spec:
  agents:
    default:
      model: qwen2.5:7b
      baseURL: http://host-ollama.sympozium-system.svc.cluster.local:11434/v1
  authRefs:
    - provider: ollama
      secret: llm-credentials
  skills:
    - skillPackRef: web-endpoint   # serve it as an HTTP endpoint
    - skillPackRef: k8s-ops        # give its runs kubectl
  policyRef: sandbox-restricted
  memory:
    enabled: false
    systemPrompt: |
      You operate "cluster2", the management cluster of a KubeVirt
      multi-cluster platform. ...
```

Read it as a CxO would read a contract. The model is a 7-billion-parameter model served by Ollama on hardware inside the building, reached through an in-cluster address. The tools are named. The policy is named. The briefing is text anyone can review in a pull request. None of it is a setting in someone else's console.

### The object the documentation described

The chapter would be dishonest without the correction that sits at the center of the repository's Sympozium labs. The strategy document's own table of these nouns lists the persona object as a `SympoziumInstance`, and for months the repository kept its agents in exactly that form.

In July, while building the labs, the team listed the controllers actually running inside Sympozium's controller manager: `Agent`, `AgentRun`, `Ensemble`, `MCPServer`, `SkillPack`, `SympoziumPolicy`, `SympoziumSchedule`. No `SympoziumInstance`. The resource definition existed and accepted objects, but nothing reconciled them. The labs index calls it *"the one correction that matters most"*: the objects the repository had been editing were *"inert documentation of intent."* The agents worked only because a separate `Agent` object with the same name happened to exist beside each one.

It got stranger. The resource definition existed on that cluster only because an older release had installed it; the current chart does not ship it at all. And typing the obvious command, `kubectl get agent`, did not return Sympozium agents — it resolved to a definition left behind by kagent, the platform's previous agent framework, because two products had each defined something called *agent*. The labs have used the fully qualified name, `agents.sympozium.ai`, ever since.

> A name is not an identity. In Kubernetes, the API group is part of the name — and in a platform that has changed vendors, it is the part that matters.

## One execution, one object

An `AgentRun` is where the idea becomes concrete, and lab 02 is built around one. On 17 September it was run again on the current versions — Sympozium 0.10.75 on Kubernetes 1.37 — to check that its lessons still hold.

The first lesson is that the schema flatters you. It suggests that naming an agent and giving it a task is enough. Asked to validate exactly that, the cluster refused, just as the lab recorded two releases earlier:

```
The AgentRun "book-ch13-dryrun" is invalid:
* spec.agentId: Required value
* spec.model: Required value
* spec.sessionKey: Required value
```

That check was a server-side dry run; it created nothing. When a controller creates a run on your behalf — for a chat message, a schedule, an ensemble — it fills these fields in. When you write a run yourself, you supply them.

With the fields supplied, lab 02's run asks the agent to list the cluster's nodes and virtual machines and summarize their health. It finished in about thirty seconds, and everything worth knowing about it was recorded on the object and the pod it created:

| | |
|---|---|
| Phase | `Succeeded` |
| Containers in the pod | `agent`, `ipc-bridge`, `skill-k8s-ops` |
| Identity it ran as | `sympozium-run-lab-agentrun-health` |
| Tool calls | 4 |
| Tokens | 7,418 in, 226 out |
| Model time | 21.4 s |

Its answer was a small table — nodes ✅, KubeVirt VMs ✅ — and a sentence: *"The cluster appears healthy with no issues reported for nodes or KubeVirt VirtualMachines."* It was true: one node, `Ready`; two virtual machines, `Running`. It also contained no detail the model could not have guessed, which is exactly the problem Chapter 15 takes up. For this chapter, the point is different. The record exists, and anyone with read access to the namespace can inspect it.

The lab's history holds the more instructive run. The first time it ran, back on release 0.10.38, the `skills` list was left out on the assumption that the run would inherit it from the agent. It did not. The pod came up with two containers instead of three, and the model's six attempts to run `kubectl` were written to a channel no tool was listening on:

```
04:54:05 tool_call [1]: execute_command args={"command":"kubectl get nodes; ..."}
04:54:05 Wrote exec request 1782968045241295119: kubectl get nodes; ...
... (6 tool calls, all silently dropped) ...
04:58:25 LLM call succeeded (tokens: in=11457 out=752, tool_calls=6)
```

After 262 seconds and 12,209 tokens, the model concluded: *"Given the persistent issues with command execution, let's assume there is a connectivity or availability problem with the skill sidecar."* The lab's verdict is one line: *"It was right."*

## How a model gets hands

That failed run shows how tools actually reach a model, and it is worth understanding, because it decides what an agent can touch.

The model never runs a command itself. It emits a structured request — `execute_command`, with a command string. The `ipc-bridge` container writes that request to a shared channel. A separate sidecar container, supplied by a `SkillPack`, picks it up, runs `kubectl` with its own credentials, and writes the output back. Only then does the model see anything real.

So what an agent can do is not decided by what it is told. It is decided by which sidecar containers its run receives, and by the permissions those containers hold. Remove the sidecar and the model is talking into an empty room, however confident it sounds. Narrow the sidecar's permissions and the model's reach narrows with them.

That design paid for itself twice in this repository, and cost twice.

It paid when permissions turned out to be too narrow. The built-in `k8s-ops` tool pack could read nodes and pods but not virtual machines — its bundled permissions include no `kubevirt.io` group. The fix was an additive grant, reviewed in a file (`06-sympozium/agent-kubevirt-rbac.yaml`), not an instruction to the model. The service-mesh agent needed the same for Istio's resources.

It cost when the rules for delivering sidecars changed between releases. On 0.10.38, runs created through Sympozium's API received no tools at all, so the repository added a small admission webhook to attach the right tool pack to each agent's runs. On 0.10.47, the API began copying the agent's own skills list onto the runs it created. The agents listed only the serving skill at the time, so their runs now carried a non-empty skills list with no tool in it — and the webhook, which only filled *empty* lists, stood aside. Every tool call failed with *"the Kubernetes sidecar container is not currently running."* The fix was to list the tool packs on the `Agent` itself, which is why `cluster2-agent` names `k8s-ops` today. The webhook is still deployed, and now does nothing.

## An agent that speaks the standard protocol

The first skill in `cluster2-agent`'s list, `web-endpoint`, is not a tool. It turns the agent into a service. The skill pack is marked as requiring a server, so listing it makes the controller create a long-running run in `server` mode and a Deployment named `cluster2-agent-web-endpoint-server`. The agent gets its own access key, stored in a Secret the platform generates, and an endpoint that answers the same requests as a commercial model API.

Asked what models it offers, on the current release, it answers:

```json
{"object":"list","data":[{"id":"qwen2.5:7b","object":"model","owned_by":"sympozium"}]}
```

The integration consequence is the one the lab emphasizes: *"anything that already speaks the OpenAI protocol … can use a Sympozium agent as a drop-in backend."* An existing client library, pointed at a different address, is now talking to a model running inside the building.

The lab also records the catch, and it is the right catch to know: *"You're talking to an agent, not a bare LLM."* Asked to count from one to five, the agent did not print five digits. It wrote them to a file called `/tmp/count.txt` in its sandbox and reported that it had done so. An endpoint that looks like a model API is attached to something that acts.

Behind the endpoint, every chat message becomes a short-lived `AgentRun` of its own. A conversation is not a stream of text that disappears. It is a series of objects.

## Three windows onto the same objects

The platform's web UI has three modes, and each is a different window onto the same cluster.

**SRE mode** is the operator's dashboard: nodes, pods, virtual machines, events, the cluster-deployment flow, and a web terminal.

**AI mode** embeds Sympozium's own console. The backend runs a small reverse proxy on a separate port, 8081, which hides the vendor's top bar and pre-loads the login token and the right namespace. Both details came from experience. The console's single-page app uses absolute paths that would collide with the UI's own routes, which is why it needs its own port. And the console defaults to a namespace where nothing lives, so without the pre-loaded setting its agent, run, and schedule panels simply appear empty. The proxy is also unauthenticated — acceptable for a demo platform, and recorded as such in its own comments.

**Visual mode** is the service-mesh view, and its command center still carries the platform's original chat panel. That panel talks to agents through the backend: a two-minute limit per request, a streaming reader sized for large responses, and each agent's key read from its Secret. It can also do one thing more than chat. If an agent's answer contains a structured proposal for an action, the backend turns it into something a person can act on, and it will only ever execute actions on an explicit allowlist. Today that allowlist has one entry:

```go
// Keeping this tight: any new agent-driven action must be added explicitly.
var allowedActions = map[string]bool{
	"deploy_target_cluster": true,
}
```

One honest defect belongs here. The agent picker in that panel still asks the cluster for `SympoziumInstance` objects, which no longer exist, gets an error, and falls back to listing only the default agent. The service-mesh agent is running and serving; the picker cannot see it.

## Why it matters that it is an object

Return to Anita's three questions, because the architecture answers each of them without a vendor in the room.

*Where does the model run?* On an Ollama server on the platform's own host, reached through an in-cluster address. No prompt, no tool output, and no answer leaves the boundary on the way.

*What did it do last Tuesday?* Every execution is an `AgentRun`: the task it was given, the answer it produced, its phase, its token and tool-call counts, its duration, the pod it ran in, and — since the upgrade described in Chapter 15 — a service account created for that run alone. Access to those records is governed by the same role-based permissions as every other object in the cluster, and they can be created by the same GitOps pipelines. There is no second audit system to buy, learn, or trust.

*If we cancel, what do we keep?* Everything. There is no separate service to cancel.

That is what *first-class object* buys: the agent inherits the governance machinery the organization already runs, instead of arriving with its own.

It does not buy immortality, and this is where the build log has to be exact. The strategy document's sixth constraint says the trail is append-only: *"No design may erase, short-circuit, or replace that trail."* Nothing enforces that. The labs delete their runs when they clean up. The repository's own repair script for serving agents works by deleting their runs so the controller will recreate them. And on 16 September, when both host clusters were deleted and rebuilt for the upgrade in Chapter 19, every run record that existed that morning — fifty-three of them, the oldest nearly four weeks old — was deleted with the cluster that held it.

An object is auditable for exactly as long as it exists. The same constraint already names the answer — durable storage off the cluster that *"mirrors `AgentRun`s, it does not replace them"* — and it has not been built.

* * *

*Meridian · the following Monday*

Anita's approval for the agent pilot ran to four lines.

*No external model. No vendor copilot. Every agent run stays a cluster object, and every run is copied off the cluster before anything is allowed to delete it. Access to the run records is reviewed like access to production.*

Vikram read the third line twice. "The copy doesn't exist yet."

"Then the pilot starts when it does," she said. "An agent we can't list is an agent we can't answer for."

## The ledger

- **Built:** three serving agents as `Agent` objects — `cluster2-agent`, `target-cluster-agent`, and `mesh-sre-agent` — each with a Deployment, an endpoint in the standard chat-completions format, and its own access key.
- **Measured on 17 September, current release:** one hand-written run — three containers, a per-run identity, 4 tool calls, 7,418 input and 226 output tokens, 21.4 seconds of model time.
- **Learned the hard way:** the persona object the documentation described had no controller; runs a controller creates for you and runs you write yourself get different defaults; and tools reach a run only through its skills list, whose delivery rules changed between releases.
- **Kept:** every run as a queryable object, governed by existing permissions — for exactly as long as nobody deletes it.
- **Lost:** fifty-three run records in the 16 September rebuild.
- **Still owed:** an off-cluster copy of the run records; authentication on the embedded console; an agent picker that can see the service-mesh agent; a strategy document that still names the old persona object.

## Ask your team

1. **Where does our AI model run, and what crosses our boundary when an agent answers a question?** Ask for the network path, not the vendor's assurance.
2. **Can we list every action an agent took last week — with the identity it used and what it cost — without asking a vendor?** If not, who can?
3. **What can our agents touch, and is that decided by a permission system or by a prompt?** Only one of those is a control.

## Open the repo

- `06-sympozium/cluster2-agent.yaml` — an agent, declared: model, skills, policy, and briefing, with the history of skills delivery in its comments.
- `06-sympozium/labs/01-chat-serving/` — the standard chat endpoint, including a 30-line client.
- `06-sympozium/labs/02-agentrun/` — one execution as an object: the required fields, and the run whose tools never arrived.
- `kubectl get agents.sympozium.ai,agentruns.sympozium.ai -n sympozium-system` — the live records, by their fully qualified names.
- `ui/backend/handlers/ai.go` (the chat proxy and its action allowlist) and `ui/backend/handlers/dashboard_proxy.go` (the embedded console).
- `06-sympozium/labs/README.md` — "the one correction that matters most."
- **Without the repo:** Appendix G.6 quotes the `cluster2-agent` Agent object whole, briefing included.

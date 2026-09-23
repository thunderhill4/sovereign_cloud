# Chapter 12 — The Model Is Just Another Service

> *Model traffic now carries mesh policy + telemetry, same as any service.*
>
> — `07-istio-advanced/act4-ai-gateway/run.sh`

*Meridian · a Wednesday in September · the boardroom, the evening before the board meets*

Anita had the board pack open at the page headed *Artificial intelligence*. It asked two questions, the same two the board had asked every quarter since the first pilot: *Who can use it?* and *What does it cost?*

"This quarter we answer them with a screen, not a paragraph," she said.

Vikram put the console on the wall. "Every model call goes through an AI gateway now. Same as any other service in the mesh — it gets policy, and it gets measured. And the visual mode shows the whole estate, live."

"Then tomorrow I can say every model call passes one control point, and the board can watch the traffic on that screen."

"That's the design."

Anita looked at the screen for a while. It was full of green. "Before I say it to the board," she said, "show me yesterday's model calls on it. And show me who was allowed to make them."

* * *

## A gateway for models

*Build log · 21 August 2026*

Act 4 of `07-istio-advanced/` is the one its README calls Istio's headline feature for this release: an **AI gateway**.

It uses the same Gateway API object as Chapter 11's front door, with one field changed. `gatewayClassName: istio-agentgateway` swaps the general-purpose proxy for **agentgateway**, *"a proxy purpose-built for AI/agent traffic."* The class is experimental and has to be switched on when Istio is installed, one more day-one decision. The gateway takes the address `172.18.255.203`, and one route sends everything it receives to the models.

The models are not in the cluster. They run on the host's own GPU, under Ollama, and the cluster reaches them through a shim: a Service with a hand-written endpoint pointing at the host, `172.18.0.1`. This is Chapter 10's pattern, used here on purpose, because keeping models off the cluster *"keeps ~4GB of RAM per replica free."*

For a board, the reason to put a gateway in front of models is the reason to put one in front of anything: a single place to decide who may call which model, to count what they use, and to switch providers without touching every caller. On sovereign infrastructure that matters twice over, because the models, the GPU and the data never leave the building.

The act ends by asserting the result in one line, which is this chapter's epigraph: *"Model traffic now carries mesh policy + telemetry, same as any service."* The console's AI Gateway panel says the same, and it fetches the model list *through* the gateway, *"so it proves the data path, not just the config."*

That is three claims: a data path, policy, and telemetry. They were tested separately.

## The data path is real

*Build log · 17 September 2026*

Through the gateway, the model list came back with all 14 of the host's models, and a chat completion returned `ok`.

The more useful question is what the extra hop costs. With `qwen2.5:7b` already loaded, the same short request was sent seven times directly to Ollama and seven times through the gateway, alternating between the two:

| Path | Median | Range |
|---|---|---|
| Direct to host Ollama | 188.4 ms | 173.0–216.4 ms |
| Through the AI gateway | 179.7 ms | 164.9–198.7 ms |

The gateway adds nothing measurable. The small difference in its favour is noise, not speed.

Streaming — the way the agents actually receive answers, a token at a time — passed through intact. A request to count to twenty arrived as 51 chunks by both paths, with the first byte after 0.163 seconds direct and 0.171 seconds through the gateway.

So the first claim holds, and it is good news. Putting a control point in front of the models costs nothing a user would notice.

## A door with no lock

On 17 September there was no authorization policy of any kind in the gateway's namespace. With no credential, from the host:

| Request through the gateway | Result |
|---|---|
| `GET /api/version` | `200` |
| `GET /api/ps` — which model is loaded right now | `200` |
| `POST /api/show` — a model's full configuration | `200` |

The route forwards every path, so Ollama's administrative endpoints — pulling new models, deleting models, creating them — are behind the same route. Those were deliberately not called.

The gateway is also not a mesh workload in the way the echo services are. Its pod carries the label `istio.io/dataplane-mode=none`, and the hop from the gateway to the model is plain HTTP to the host, which no mesh can enrol. Compare the agent platform's own serving endpoints: Chapter 11's verification script cannot even check them without first fetching a bearer token.

The gateway did not create this exposure. Ollama on the host listens on every interface (`*:11434`), so anything that can reach the host can reach the models without going near the gateway. But the gateway did not close the exposure either. A control point is where a policy *could* live, and on 17 September none did.

## Counted under another name

Chapter 11's verification script and the console's Observability panel both judge "is telemetry flowing?" the same way: the number of `istio_requests_total` series in Prometheus. On 17 September the panel reported 16 series, labelled `source: live`.

About forty requests had just been sent through the AI gateway. Not one of those sixteen series mentioned the gateway or the model backend. Every series came from Act 1's front door or Act 2's waypoint — including, as it happens, the `503`s from Chapter 11's failover experiments. The per-node mesh component recorded nothing either: no connection metrics had the gateway as their source.

Kiali, the mesh map from Act 5, agreed. Its graph for the last hour had **0 nodes and 0 edges** for `ai-gateway`. For `demo-apps`, it had 10 nodes and 14 edges.

The gateway does keep its own count. Prometheus was scraping it successfully, and a family of `agentgateway_*` metrics recorded requests by method, status, route and backend. The problem is what they omit: there is **no model name and no token count**. Those are the two dimensions that turn a request count into a cost. And nothing reads these metrics — not the console panel, not Kiali, not the verification script.

So the model traffic is measured, but in a metric nobody displays, and without the two labels that would answer the board's second question.

## Nobody uses it

Then there is the question of whose traffic this is.

The platform's three serving agents run on cluster2, and each names its model endpoint in its own configuration:

```yaml
baseURL: http://host-ollama.sympozium-system.svc.cluster.local:11434/v1
```

That is a second shim, on cluster2, going straight to the host. The AI gateway is on cluster1. None of the three agents sends a single request through it. The comment in the gateway's route file was carefully worded: the endpoint the agents speak *"is reachable through the mesh."* Reachable, yes. Reached, no.

The act's own check even gets in the agents' way. By default it asks a different model, `qwen3.5:4b`. The GPU on this host holds one model at a time (Chapter 9). On 17 September the act's single completion took 8.2 seconds, reasoned through 556 characters of hidden thinking before replying `ok`, and evicted `qwen2.5:7b` — the model two of the agents use. Putting it back cost a cold load of 3.30 seconds, which the next agent request would have paid.

For a CxO, this is the chapter's central finding. The governance layer for AI exists, and it works. It stands *beside* the AI fleet, not in front of it.

## An honest limitation about the wrong thing

Act 4 has a second tier, and it comes with the most candid passage in the repository:

> *HONEST LIMITATION, worth saying out loud on stage: the EPP schedules on KV-cache utilization and queue depth, which vLLM and Triton export and Ollama does NOT. What follows demonstrates the routing API surface, not genuine load-aware scheduling.*

Tier 2 is the Gateway API **Inference Extension**. It groups model servers into a pool and picks a server for each request based on how busy each one is. The console repeats the limitation in an amber box. It is true: Ollama does not publish those load figures.

But Tier 2 never ran. The Inference Extension was not installed on either cluster. Its installer script and its manifest are referenced by nothing — not the act, not the Makefile, not the demo runner. The act checks whether the extension exists, finds it missing, prints the limitation, and moves on.

And the manifest could not have been applied. Checked against the schemas in the extension version its own installer pins (v1.0.1):

- **The pool** is declared at version `v1`, but uses field names from the older `v1alpha2` (`targetPortNumber`, `extensionRef`). Version `v1` requires `targetPorts` and `endpointPickerRef`.
- **The objective** is declared in an API group that does not exist in that release, and uses two fields, `modelName` and `criticality`, that do not exist in the one that does. Its real fields are `poolRef` and `priority`.
- **The pool's selector** matches no pods: the model backend is a shim Service, not a set of pods.
- **The endpoint picker** it names, `ollama-epp`, is not deployed anywhere.

The limitation is real, but it is a limit the demo would only have reached after four more basic problems. Honesty about a limit you would hit later can hide the ones you hit first.

## The stage

*Build log · 21 August and 17 September 2026*

Act 5 installs **Kiali**, the mesh map, and **Prometheus**, which stores its metrics, on cluster1. The act states a principle this book would sign: *"A Kiali screenshot is not evidence; the metric count is."* The console's Observability panel follows it, showing the series count rather than just whether the pods are up — because *"a Ready pod alone does not mean telemetry is flowing."*

Three facts about that stage belong in front of a board:

- **The map is open to anyone.** Kiali at `172.18.255.204` reported its authentication strategy as `anonymous`. Anyone who can reach the address can browse every namespace, workload and identity in the mesh, and a map of your systems is itself sensitive. This is the default in Istio's sample installation, which is what the act uses.
- **The history is temporary.** Prometheus is set to keep fifteen days of data, on storage that disappears whenever its pod restarts. On 17 September its history began on 16 September at 07:29 UTC, the moment of Chapter 19's rebuild.
- **The stage is downloaded, not owned.** The act fetches both installations at install time from Istio's public GitHub branch for the release, not from a pinned, local copy. The mesh's root certificate was created so the demo *"works air-gapped"*. Its observability does not.

## The screen that says live

The console's visual mode was rebuilt on 21 August. It retired six panels that made no API calls (Chapter 10) and put up a comment saying *"Everything below reads live cluster state."* Most of it does. The mesh overview reads whether both clusters share a root certificate. The AI Gateway panel lists models through the gateway. The Observability panel counts series. Each reports where its data came from.

One view was kept from before, with a note that it *"is real."* It is the topology view: the network map, drawn as a force-directed graph. On 17 September its API returned 21 nodes, labelled `source: live`.

Fourteen of those nodes do not exist:

- In cluster1, the `mc-demo` namespace: `httpbin`, its pod, `sleep`, the load balancer on `.200`, and the ServiceEntry for `nginx.target-cluster.global`.
- In cluster2, the proxy on `.216`.
- In the target cluster, the whole Istio installation and sample application: `istiod`, two ztunnels, `nginx`, its NodePort, its pod, `sleep`, and the ServiceEntry back to cluster1.

That is Chapter 10's March picture. The graph showed thirteen of the fourteen as **healthy**. The one exception is a ztunnel marked `error` with the note *"Not Ready"*, typed into the source code. The worker virtual machine appeared as `degraded`, also typed in. The real worker was `Running` and ready.

The mechanism is simple, and it is worth understanding because it is so common. The handler starts from a fixed drawing and tries three live lookups. It labels the result `live` if none of those lookups *returns an error*:

- **Cluster2's node list** succeeded, and correctly updated one node.
- **The virtual machines** were found, but they are matched to the drawing by exact name: `target-cluster-cp` and `target-cluster-worker`. The real machines are `target-cluster-cp-s9j8h` and `target-cluster-workers-rz9xt-m5x6z`. Nothing matched, so nothing changed.
- **The target cluster** was queried for Services in a namespace called `sample`, which no longer exists. In Kubernetes, listing an empty or missing namespace returns an empty list, not an error. The lookup "succeeded", and the target cluster was marked healthy.

The only live number the target side received was `ztunnelPods: 0/0 ready`, attached to a node drawn as healthy.

The same picture shows up somewhere else, too. The briefing given to one of the platform's AI agents still tells it that cluster1 runs *"Istio ambient mesh, httpbin/sleep."*

For a CxO, the lesson is general. *Live* on a screen is a claim about the plumbing, not the picture. What a status label needs to tell you is not whether its queries ran, but whether the thing drawn was found.

* * *

*Meridian · later that evening*

Vikram's answers fitted on one side of a page. Anita read them out loud, one at a time.

"Who can use the models: anyone who can reach the network. What it costs: the gateway counts requests, but not which model and not tokens. How much of our AI traffic goes through the gateway: none of it — the agents go around. And the screen." She looked up. "Fourteen of the things on that map don't exist."

"Thirteen of them are drawn green."

She was quiet for a moment. "The design is right," she said at last. "A model *is* just another service. So it gets the same three things every service gets from us: an identity, a lock, and a meter. The meter counts tokens by model, because that's the bill. The agents go through the gateway, or we stop calling it the AI control point. And nothing goes on a board screen unless every box on it traces back to a lookup that could have come back empty."

"And tomorrow?"

"Tomorrow I tell them the truth," Anita said. "We have built the control point. Our AI doesn't use it yet. I'd rather present a gap we've measured than a screen we haven't."

![Topology panel badged LIVE: seven solid nodes and fourteen dashed ones, with two causes annotated.](../figures/fig-12-2-phantom-topology.svg)

*Figure 12.2 — A view labelled "live", drawing fourteen things that do not exist.*

## The ledger

- **Built (21 August 2026):** an AI gateway on cluster1 in front of the host GPU's models; a load-aware second tier designed and documented, with its limitation stated; Kiali and Prometheus; a visual mode rebuilt around live data.
- **Measured (17 September):** no measurable latency cost through the gateway (179.7 ms against 188.4 ms, median, warm); streaming intact (51 chunks, first byte 0.17 s).
- **Found:** no authentication or authorization on the model gateway; model calls absent from the mesh's request metric and from Kiali (0 nodes); the gateway's own metric has no model or token labels, and nothing reads it; none of the three agents routes through the gateway; the act's check evicts the agents' model (3.30 s to reload); Tier 2 never installed, and its manifest invalid against its own pinned version; Kiali open anonymously; Prometheus history on temporary storage; a topology view labelled `live` in which 14 of 21 nodes do not exist.
- **Restored:** `qwen2.5:7b` reloaded onto the GPU and confirmed. No cluster objects were created or changed.

## Ask your team

1. **What share of our AI calls passes through the control point we show the board — and which calls go around it?**
2. **Can we say, per model, how many tokens we used last month, and from which system?**
3. **On our operations screens, what does "live" actually mean — and what would the screen show if the thing it draws were deleted?**

## Open the repo

- `07-istio-advanced/act4-ai-gateway/01-agentgateway.yaml`, `00-host-ollama.yaml`, and `run.sh` — the gateway, the shim, the claim, and the honest limitation.
- `07-istio-advanced/act4-ai-gateway/02-inference-pool.yaml` and `install-inference-ext.sh` — compare against the v1.0.1 CRD schemas.
- `06-sympozium/cluster2-agent.yaml` (and the other agent manifests) — `baseURL`, where the agents' model traffic really goes.
- `07-istio-advanced/act5-observability/run.sh` — the stage, and where it is downloaded from.
- `ui/backend/handlers/mesh.go` — the fixed drawing, `vmNodeID`, and `applyTargetOverlay`.
- `ui/backend/handlers/istio.go` — `HandleIstioAIGateway` and `HandleIstioObservability`, the panels that do read live state.
- `curl -s localhost:8080/api/v1/mesh/topology` with the console backend running — the graph, node by node.
- Prometheus query `agentgateway_requests_total` — the metric that counts model traffic, and the labels it lacks.

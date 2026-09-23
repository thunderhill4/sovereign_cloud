# Chapter 10 — Hardcoded IPs and Other Confessions

> *`05-istio/` is left untouched. It stays as the "before" picture.*
>
> — `07-istio-advanced/README.md`

*Meridian · a Tuesday in September · the executive briefing centre*

The prospect was a regional bank, and their architect asked the question every bank asks: if one site goes down, what happens to the other?

Vikram had the answer on the wall screen. Two clusters, drawn as boxes. Four live probes between them, all green. "And the traffic between the sites is encrypted?" the architect asked. "It all goes through the service mesh," Vikram said, and pressed a button that switched off a service in the first cluster. One probe went red. A moment later an amber badge lit across the top of the diagram: **⚡ FAILOVER ACTIVE**.

The architect nodded and wrote something down.

Afterwards, Anita stayed in the room. "Which service took over?"

"None of them." Vikram brought the code up on his laptop. "The badge comes on when a service here is down *and* a call to a different service over there still works. The two aren't connected."

"So the screen showed two true facts, and a word that joined them."

"Yes."

"And the word is what the bank wrote down." She looked at the diagram a while longer. "How much of the rest of this picture is like that?"

"I don't know yet."

"Find out. Every box, every arrow, every word on it. I don't want it fixed this week — I want a list. Some of it we will keep; a before picture is worth having. But I want to know which parts are the before picture, and which parts are just wrong."

* * *

*Build log · 12 March 2026*

## Eight steps, one commit

Shortly after midnight on 12 March, a single commit added the platform's whole cross-cluster story: an eight-step interactive demo script, a mind-map diagram of the traffic path, a "Live Demo" tab in the web console with real probes and failure controls, and six more console panels for the service mesh. Thirty-six files; 5,362 lines added.

The problem it took on was genuinely hard. `cluster1` is a Kind cluster on the host. The target cluster is k3s running inside two KubeVirt virtual machines, which run inside pods of a *second* Kind cluster, `cluster2`. For a program in cluster1 to call a program in the target cluster, a request has to leave one cluster, find the right virtual machine inside another, and get in.

And there is no shared address plan to lean on. cluster1 and cluster2 both number their pods from the same range, `10.244.0.0/24` — checked on both clusters on 17 September. The target cluster's control-plane machine holds the address `10.244.0.115`. Typed into cluster1, that address names a slot in cluster1's *own* pod network, not a machine in the other cluster.

So the demo built a path by hand, and prints it at the end:

```
sleep pod → ztunnel → ServiceEntry → MetalLB 172.18.255.216
→ cluster2 proxy Svc → virt-launcher pod:30080
→ masquerade NAT → k3s VM → nginx
```

In plain language, three pieces of plumbing:

- **A door on every machine.** In the target cluster, nginx is published on port 30080 of every node — a *NodePort*.
- **A fixed address on the shared network.** In cluster2, a Service is given the address `172.18.255.216`, typed into an annotation for MetalLB to honour. Behind it, instead of letting Kubernetes find the backends, the script writes the destination in itself: the current address of the virtual machine's pod, port 30080.
- **An entry in the mesh's address book.** In cluster1, an Istio *ServiceEntry* tells the mesh that the name `nginx.target-cluster.global` lives at `172.18.255.216:8000`. A matching entry in the target cluster points back at `172.18.255.200`, cluster1's httpbin.

The eight steps then tell a story with the shape of every good demo. Show the two clusters. Deploy the services. Prove each works locally. Show that the other cluster's service cannot be reached by name — *"✗ Connection failed — service not discoverable."* Create the ServiceEntries. And then, in the script's own words: *"Now the magic: cluster1 calls target-cluster and vice versa."* Step 7 adds a traffic policy, a connection pool and a circuit breaker. Step 8 shows the logs.

Fail, fix, succeed. It ran, and because it ran, nobody questioned it.

It was also, in six distinct places, not what it said it was. In August, when the platform built the proper answer, it kept this directory untouched, on purpose, as the "before" picture. On 17 September the path was re-plumbed by hand on the rebuilt clusters, using the script's own YAML, to see which of its claims still held; everything created was measured and then removed. What follows is the list Anita asked for.

## The diagram was wrong before it was committed

The mind map is this chapter's figure, and it is a good diagram: clear boxes, the forward path in one colour, the reverse path in another, and a panel of key concepts. One of those concepts reads:

> **KubeVirt Masquerade** — VM gets static IP 10.0.2.2. NAT via virt-launcher pod. All ports forwarded (DNAT).

*Masquerade* is one of two common ways to connect a virtual machine to a Kubernetes network. The machine sits behind its pod like a laptop behind a home router: it has a private address, `10.0.2.2`, and the pod translates traffic in and out. The other way, *bridge*, has the machine take the pod's address as its own, with nothing translated.

The same commit that added the diagram switched the target cluster from one to the other:

```diff
                   interfaces:
-                    - masquerade: {}
+                    - bridge: {}
```

It also deleted the firewall rule that translated traffic for `10.0.2.2`. So the hop the diagram draws had already been removed before the diagram reached the repository. It is still absent today. The target cluster's control-plane node reports its internal address as `10.244.0.115`, exactly the address of the pod that hosts it. There is no `10.0.2.2`, and no translation step. The script's narration says "masquerade NAT" anyway, once at the start of the demo and again in the closing summary.

Nothing failed, which is exactly why nobody noticed: a wrong diagram of a working system raises no error. And diagrams travel. The copy of the mind map kept with the briefing material is the same file, byte for byte.

## Step six did not need step five

Look again at the demo's logic. Step 4 tries the *name*, and fails. Step 5 teaches the mesh the name. Step 6 succeeds. The audience draws the obvious conclusion: the ServiceEntry made the call possible.

But Step 6 does not call the name. It calls the address:

```bash
curl -s "http://${TARGET_LB}:8000"
```

`TARGET_LB` is `172.18.255.216`. Any pod in cluster1 can reach that address with or without a ServiceEntry, because it is simply an address on the shared network.

*Build log · 17 September 2026*

On the re-plumbed path, a pod enrolled in cluster1's mesh called the address five times *before any ServiceEntry existed*. All five returned `200`, in 1.1–1.7 milliseconds. Then the demo's ServiceEntry was applied, verbatim. Calls to the address carried on returning `200`, in 1.3–1.6 ms. Calls to the *name* — which a moment earlier had failed with "could not resolve host" — now succeeded too, in 1.7–2.1 ms. The mesh answered the lookup with `240.240.0.1`, an address Istio assigns to such entries by itself.

So the name works, at least on today's mesh. The demo never used it. The step that looked like the cause of the success sat beside the success, not inside it.

## A policy nothing read

Step 7 applies a *DestinationRule* to the remote nginx. Among its settings: one request per connection, and eject a backend after three server errors in a row. The script prints *"Traffic policy applied: connection pooling + circuit breaker"*, and then proves it with one request that returns `200`.

That request would have returned `200` without the policy. And in *ambient* mode — the mesh design used here, which puts no proxy inside the application's pods — the component on each node, ztunnel, handles connections, not individual HTTP requests. Request-level rules like these need a separate layer-7 proxy, a *waypoint*, and the demo never deployed one. When the "after" picture was built in August, the repository's own comparison table filled in the before column with a single word:

> **L7 policy** — none (ztunnel is L4-only)

The same document, describing a different kind of policy, warns that a rule enforced at the connection layer *"will silently ignore HTTP path and method rules."* Silence is the failure mode again, just as with the diagram. An unenforced policy looks exactly like an enforced one until the day it is needed.

## The one leg without a lock

The re-plumbed run also captured what cluster1's ztunnel recorded for the cross-cluster calls. Here is one of those log lines, trimmed, beside an ordinary call between two workloads inside cluster1's mesh:

```
cross-cluster   src.workload="sleep" src.namespace="ch10-before" src.cluster="cluster1"
                dst.addr=172.18.255.216:8000 dst.service="nginx.target-cluster.global"
                dst.workload="nginx-target-cluster" dst.namespace="ch10-before"
                dst.cluster="cluster1"

inside cluster1 src.workload="waypoint-…" src.namespace="demo-apps"
                src.identity="spiffe://cluster.local/ns/demo-apps/sa/waypoint"
                dst.addr=10.244.0.25:15008 dst.service="echo-v1.demo-apps.svc.cluster.local"
                dst.identity="spiffe://cluster.local/ns/demo-apps/sa/echo"
```

Two differences matter.

The call inside cluster1 carries an **identity** at both ends. That is a cryptographic name for each workload, and it is what mutual TLS authenticates. The call also travels to port 15008, the mesh's encrypted tunnel. The cross-cluster call has no identity at either end and uses no tunnel. The project's own notes say it plainly: *"mTLS terminates at cluster edge; cross-MetalLB traffic is plain HTTP."* The one leg of the journey that crosses the shared network between clusters is the one leg with no encryption and no identity.

The second difference is quieter: `dst.cluster="cluster1"`, `dst.namespace="ch10-before"`. A ServiceEntry is a local record, so the mesh files the target cluster's nginx as a workload *in cluster1*, in the caller's own namespace. The mesh's own logs put the remote service in the wrong cluster.

For a board, this is the sentence to carry out of the chapter: when someone says traffic between sites is "in the mesh" or "encrypted in transit", ask which legs they mean.

## A word that joined two facts

The console's Live Demo tab, added in the same commit, runs four real probes every five seconds: each cluster calls its own service, and each calls across to the other. Its failure controls scale a service down to zero. The amber badge from Meridian's briefing is computed in `ui/backend/handlers/cross_cluster.go`:

```go
httpbinAlive := ok(probes[0])
nginxAlive := ok(probes[2])
// Failover: a local service is down but the cross-cluster path is still up
failoverActive := (!httpbinAlive && ok(probes[1])) || (!nginxAlive && ok(probes[3]))
```

`probes[1]` is cluster1 calling the *target cluster's nginx*. So when cluster1's httpbin is scaled to zero, and an unrelated nginx in another cluster still answers, the console logs *"Cross-cluster FAILOVER now active — remote service handling traffic."* No traffic meant for httpbin goes anywhere. There is no second httpbin to go to, and nothing reroutes. The probes are real and the facts they report are true; only the word joining them is false. That code is unchanged since March, and the tab is still in the console today, kept as the legacy demo.

The six mesh panels from the same commit went further. None of them imported anything except React and an icon library: no API client, no data source of any kind. The pod addresses they displayed, such as `10.42.0.12` and `10.244.0.60`, and the services named `target-cluster-nginx`, were written into the page. They were retired on 21 August, and the comment left where they used to be is the most honest sentence in this chapter:

> *none made an API call, and several hardcoded invented pod names and IPs for a target-cluster mesh that no longer exists.*

## An address with an expiry date

The last confession is the one in the chapter's title. The proxy in cluster2 reaches the virtual machine through its pod's address, typed in by the script. The Service has no selector, so no controller will ever correct that address. The pod it names was created when that morning's target cluster was built. A rebuild — 33.7 seconds, as Chapter 7 measured — replaces the pod, and the address goes with it. Traffic reaches the pod at all only because it arrives from outside the cluster and is forwarded by the node; as Chapter 8 found, other pods cannot dial a virtual machine's pod address directly.

In fairness, the script looks the address up again each time it runs. But it does not remove what it creates: it prints five cleanup commands and exits. Whatever is left behind holds a snapshot of an address that expires with the cluster.

Three smaller things came with it:

- **A comment the code does not keep.** The selection loop is introduced with *"We select the CP specifically,"* then takes the first matching virtual machine, control plane or worker. It picks the control plane only because `cp` sorts before `workers`.
- **A warning sent nowhere.** Writing endpoints by hand uses an API that Kubernetes deprecated in version 1.33. On today's 1.37, `kubectl` says so — *"v1 Endpoints is deprecated in v1.33+; use discovery.k8s.io/v1 EndpointSlice"* — and the script's `2>/dev/null` throws the message away.
- **An address that has to be defended.** `172.18.255.216` has had its own row in the project's IP allocation table since the March commit. In August, when cluster2 needed a new address, the pool was widened rather than reuse `.216`, because *"double-booking it would make the two demos mutually exclusive."* A hardcoded address is not a line of configuration. It is a reservation that somebody has to keep honouring.

## Why the before picture stayed

*Build log · 21 August 2026*

The proper answer arrived in August as `07-istio-advanced/`: five acts, each of which *"ends with assertions, not screenshots."* Its README opens with a table that is, in effect, this chapter's summary:

| | `05-istio/` (before) | `07-istio-advanced/` (after) |
|---|---|---|
| Istio | 1.28.0, ambient, one cluster | 1.30.3, ambient, two clusters |
| L7 policy | none (ztunnel is L4-only) | waypoint + `AuthorizationPolicy` on SPIFFE identity |
| Cross-cluster | `ServiceEntry` + hardcoded MetalLB IP + NodePort proxy | `istio.io/global: "true"` |

A single label on a Service replaces the address book, the fixed address and the hand-written proxy. Chapter 11 tells that story, including the day a waypoint silently broke it. (The "after" column has since moved to Istio 1.31; Chapter 19.)

It would have been easy to delete `05-istio/` that same day. Instead the README records that it *"is left untouched. It stays as the 'before' picture."* A company keeps last year's accounts for the same reason: an improvement you cannot compare against is only a claim.

There is one honest limit, and the README states it too. Act 3 connects **cluster1 and cluster2** — not the target cluster. The target cluster is *"k3s nested in KubeVirt VMs (pod IPs unreachable from cluster1 without a NodePort proxy) and the warm pool destroys and rebuilds it, which would break mesh membership each time."* On 17 September the target cluster had no service mesh at all: four namespaces, none of them Istio's. The virtual-machine clusters are the product this book is about. The only cross-cluster path they have ever had is the one in this chapter.

* * *

*Meridian · the following week*

Vikram's list had six lines. Anita read them in order and stopped at the fourth.

"Plain HTTP between the sites."

"Between the clusters, on our own network. Inside each cluster, it's encrypted. The crossing isn't."

"You told the bank it all went through the mesh."

"I told the bank what the diagram said."

She didn't cross anything out. "Here is what we do. The before picture stays. It is the best evidence we have that the new design is better. But every screen that shows it gets a label: *hand-plumbed; not encrypted between clusters; not failover*. The badge loses the word. The diagram comes out of the sales folder until it matches the machines. And the VM clusters get their own line on the roadmap, because the new design does not reach them yet."

"And the bank?"

"We call them before they find it," Anita said. "I would rather show a customer our confession than have them discover it."

## The ledger

- **Built (12 March 2026):** cross-cluster calls in both directions between a Kind cluster and the VM-hosted target cluster, over a path plumbed by hand. The eight-step demo, the mind map, a live console tab and six mesh panels all arrived in one commit: 36 files, 5,362 lines.
- **Measured (17 September, path re-plumbed by hand; Kubernetes 1.37, Istio 1.31 on cluster1):** `200` in 1.1–1.7 ms by address *before* any ServiceEntry existed; 1.7–2.1 ms by name after it.
- **Found:** a diagram describing a translation step that its own commit removed; a demo step that did not depend on the step before it; a request-level policy given to a layer that sees only connections; a cross-cluster leg with no identity and no encryption; a failover badge joining two unrelated probes; six panels with no data source; a deprecation warning discarded.
- **Kept on purpose:** the whole before picture, as the baseline Chapter 11 is measured against.
- **Still open:** the VM-hosted target cluster has no service mesh, and no after picture.
- **Cleaned up:** everything created for the measurement was deleted; `.216` was released.

## Ask your team

1. **Which step of our best demo would still succeed if we skipped the step before it?**
2. **When a dashboard says *failover*, *encrypted* or *healthy*, what exactly does the code behind that word measure — and when did someone last read it?**
3. **Which addresses have we typed in by hand, and what happens to each of them the next time we rebuild?**

## Open the repo

- `05-istio/cross-cluster-demo.sh` — the eight steps. Step 5 (the ServiceEntries) begins at line 460; Step 6's call by address is at line 537.
- `05-istio/cross-cluster-mindmap.html` — the chapter figure. A byte-identical copy sits in the author's briefing material (Appendix H), so correcting one leaves the other wrong.
- `git show 44aa28a -- 03-target-cluster/target-cluster.yaml` — `masquerade` becomes `bridge` in the same commit as the diagram.
- `ui/backend/handlers/cross_cluster.go` (line 151) — the failover rule; `ui/frontend/src/components/visual/CrossClusterDemo.tsx` — the badge.
- `ui/frontend/src/pages/VisualDashboard.tsx` — the retirement note for the six panels.
- `07-istio-advanced/README.md` — the before/after table, and the Act 3 note on why the target cluster is left out.
- `kubectl --context kind-cluster1 logs -n istio-system -l app=ztunnel | grep "connection complete"` — check any connection for `src.identity` and `dst.identity`.

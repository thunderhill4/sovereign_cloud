# Chapter 11 — One Label

> *Marking a Service global is the ENTIRE cross-cluster config.*
>
> — `07-istio-advanced/act3-multicluster/03-global-service.yaml`

*Meridian · a Monday in September · the platform team's room*

Vikram had cleared the whiteboard of the old diagram and written two words where it used to be: *one label*.

"That's the replacement," he said. "No fixed addresses, no hand-written proxy, no address book. We mark a service as global, and the mesh in each site learns where the copies in the other site are. If a site loses its copies, calls go to the other one."

Anita had the confessions list from the week before in her hand. "The last design had a demo that worked too."

"This one is different. Every step ends in a check that reads the answer back from the system. Not a screenshot."

"Good. Then show me one of those checks going red."

"Red?"

"Take the service down in both sites. Nowhere left to fail over to. If the check is real, it fails." She put the list down. "I've been shown plenty of green. I want to see what red looks like before I believe the green."

* * *

## Decisions you can only make once

*Build log · 21 August 2026*

The replacement arrived as `07-istio-advanced/`: five acts, each of which, in its README's words, *"ends with assertions, not screenshots."* This chapter covers the first three. Acts 4 and 5 are Chapter 12.

Before any act, the README opens with a section called *"Order is not optional."* It is the most important page for a board, because it is about decisions that can only be taken on the first day.

Joining two clusters into one mesh needs four things fixed when the mesh is installed: a mesh name, a name for each cluster, a network name for each, and a flag that switches on the multi-network design. It also needs both clusters to trust one **root certificate** — the single authority from which every workload's identity is issued. Istio reads that root only when it starts. The install script says what happens if it starts without one:

> *if istiod starts without it, it self-signs its own root, and the only repair is to reinstall, which reissues every workload certificate in the mesh.*

So the root is generated first, and both meshes are installed ready for two clusters, even though the second cluster isn't demonstrated until Act 3. The root is created locally, with no outside certificate authority involved: *"that is the sovereign-cloud story, and it also means the demo works air-gapped."*

For a CxO the point is general. Some architecture choices are cheap on day one and a full rebuild on day four hundred. Whether you will ever run more than one site is one of them. On 17 September both clusters still reported the same root fingerprint, and each control plane reported its peer as `synced`.

## The first refusal

Act 1 puts a front door on cluster1 using the Kubernetes **Gateway API**. The door is declared as a single object, a `Gateway`. There is no installer and no hand-built proxy. Istio reads the object and creates the proxy Deployment and its Service itself. On 17 September both existed, as `edge-istio`, serving `172.18.255.201`.

The act's real subject is *who may do what*. Three namespaces play three roles. `gateway-infra` belongs to the platform team, which owns the Gateway and its certificate. `demo-apps` belongs to an application team, which owns the routes that send traffic to its services. `rogue-tenant` is labelled untrusted. The Gateway accepts routes only from namespaces labelled as applications.

Then the rogue tenant tries to take over the application team's hostname. It writes a route claiming `echo.demo.istio.local` and pointing it at a service of its choosing. The Kubernetes API accepts the object: the tenant is allowed to write objects in its own namespace. The Gateway refuses to attach it. This is the route's status, read back from the cluster on 17 September:

```
Accepted=False  NotAllowedByListeners: hostnames matched parent hostname
"*.demo.istio.local", but namespace "rogue-tenant" is not allowed by the parent
```

Traffic to the hostname carried on reaching the application team's service.

That is governance in the platform, not in a wiki page. Permission to write configuration is not permission to receive traffic, and the refusal is written down where anyone can read it. The rest of Act 1 is ordinary good practice, and it was verified again on 17 September: HTTPS with a locally rooted certificate, `200`; a 90/10 canary release, which measured 8% on 60 requests; and a header that pins internal users to the new version, which held on 10 of 10 requests.

## The second refusal

Act 1 has one more beat, and it did not appear in the August commit's list of verified results.

A **ListenerSet** lets an application team add its own entrance to the platform team's Gateway — here, a listener for `tenant-a.demo.istio.local` on port 8443 — without editing the Gateway itself. The act's ListenerSet borrows the platform team's certificate, which lives in `gateway-infra`. On 17 September its status read:

```
Accepted=False  certificateRef echo-tls/gateway-infra not accessible to a Gateway
in namespace "demo-apps" (missing a ReferenceGrant?)
```

The Gateway had refused the demo's own tenant, for the same reason it refused the rogue one. Borrowing something from another team's namespace requires that team to sign a permission slip first, a `ReferenceGrant`. Nothing in the repository contains one.

The refusal is correct; the demo is what's wrong. And nobody noticed, because this step prints a status instead of asserting one. It then prints *"Gateway now reports attached ListenerSets in its status"*, followed by the Gateway's listener names — `http https`, with nothing attached — and moves on. The verification script does not check it. Both refusals come from the same rule. Only one of them was ever checked.

## Identity, not address

Act 2 is the fix for the policy in Chapter 10 that nothing enforced.

In ambient mode, the component on each node (ztunnel) works with connections, not HTTP requests. For request-level rules, Act 2 adds a **waypoint**: one proxy per namespace, declared, like the front door, as a Gateway object. No proxy is placed inside any application pod. On 17 September each echo pod ran exactly one container, `echo`.

Two client deployments are identical in every respect except their Kubernetes service account, `trusted` or `untrusted`. The mesh turns that account into a cryptographic identity: `spiffe://cluster.local/ns/demo-apps/sa/trusted`. The policy names identities, paths and methods, not IP addresses. Measured on 17 September:

| Caller | Request | Result |
|---|---|---|
| `trusted` | `GET /secure` | `200` |
| `untrusted` | `GET /secure` | `403` — *"RBAC: access denied"* |
| `trusted` | `POST /secure` | `403` |
| `untrusted` | `GET /public` | `200` |

The application behind the waypoint also learned who was calling, without doing any cryptography itself. It received a header naming both the waypoint and the original caller:

```
x-forwarded-client-cert=By=spiffe://cluster.local/ns/demo-apps/sa/waypoint;URI=spiffe://cluster.local/ns/demo-apps/sa/trusted
```

The act also switches the trusted client's service account to `untrusted`, and access drops to `403` with no change to the policy and no restart of the application. That beat was not re-run for this chapter.

The README flags the one mistake that undoes all of this. If the policy is attached to the workloads rather than to the waypoint, it is enforced by ztunnel, which cannot see paths or methods — and it *"will silently ignore HTTP path and method rules."* That is the same shape of failure as Chapter 10's circuit breaker: a rule that looks enforced and isn't.

For a board, this is the counterpart to Chapter 10's plaintext link between clusters. Access is decided by a verifiable identity for the caller, not by where the caller happens to sit on the network. And it can be revoked without touching the application.

## One label

Act 3 joins cluster1 and cluster2. It needs only three things, because the hard decisions were taken at install time.

- **An east-west gateway on each side** — the doorway through which one cluster's mesh reaches the other's, over the mesh's encrypted tunnel on port 15008. cluster1's is at `172.18.255.202`; cluster2's is at `.221`, because `.216` still belongs to Chapter 10.
- **A credential each way**, so each control plane can read the other cluster's list of services. The README records a trap here: on this setup, unless the command is told the real API server address, the credential embeds `127.0.0.1`, and *"the peer cluster then sits at `STATUS=timeout` while everything else looks healthy."* The act waits for both directions to report `synced` before it continues.
- **One label** on the Service, in both clusters:

```yaml
metadata:
  name: echo
  labels:
    istio.io/global: "true"
```

Set that against Chapter 10: a ServiceEntry in each direction, a MetalLB address typed into an annotation, a proxy Service with a pod address written in by hand, and a NodePort on every node. All of it, replaced by one line.

Then comes the moment the act calls *"THE MONEY SHOT."* Every copy of echo in cluster1 is scaled to zero, with no configuration change anywhere. The same service name is called twenty more times, and the act prints the tally.

## Morning and evening

*Build log · 21 August 2026*

The commit that added the acts landed at 06:36 and reported the tally: *"Act 3 success=20 failed=0 — cluster1 scaled to zero, traffic failed over to cluster2 with no config change."* The verification script: 22 passed, 0 failed.

At 18:34 the same day, a second commit recorded a problem. Act 2 enrols the whole namespace in its waypoint. A waypoint, it said, resolves only *local* endpoints, so once the namespace is enrolled, scaling the local copies to zero gives *"503 on every request with the waypoint in path."* The evidence was the waypoint's own endpoint list, which held only cluster1's pod addresses. The fix was to take the global service out of the waypoint's path during failover. The console's failover button was built to do exactly that, and to put it back afterwards.

Put those two commits side by side. The morning's run came straight after Act 2, with the namespace already enrolled. The evening says that configuration fails every request. Both cannot be describing working failover — unless the morning's tally was measuring something other than success.

## What the counter counted

*Build log · 17 September 2026*

Here is how Act 3 decides that a call succeeded:

```bash
call() { $K1 exec -n "$NS_APPS" deploy/client-trusted -c curl -- \
         curl -s --max-time 10 http://echo/ 2>/dev/null || true; }
...
b=$(call)
if [[ -n "$b" ]]; then okc=$((okc+1)); else failc=$((failc+1)); fi
```

A call counts as a success if it returns *any text at all*. The HTTP status is never examined.

On 17 September Anita's test was run for real. Echo was scaled to zero in **both** clusters, leaving nothing anywhere to fail over to, and the service was called twenty times. Every call returned `503`, with a nineteen-byte body: *"no healthy upstream."* Under Act 3's rule, nineteen bytes count as success. The act would have printed:

```
success=20 failed=0
```

— the same line, character for character, that it prints for a perfect failover. The check could not go red.

The verification script didn't cover the gap either. Its Act 3 section asserts four things: that both east-west gateways are programmed, that a credential is present, and that the label is present. It checks that the label *exists*, not that it *does* anything. None of its twenty-two checks sends a request to a scaled-down service.

A correct counter did exist. The console's failover action, written that evening, counts only status `200`, and it measures how long convergence takes instead of assuming it. It is in a different file, and `run.sh` was never updated to match.

What the morning's twenty calls actually returned cannot be recovered now. The evening's measurement says `503`; the morning's counter would have printed the same line either way.

## What is actually true

Measured on 17 September by HTTP status, on Istio 1.31 and Kubernetes 1.37, with every change put back afterwards:

**Failover works — even with the waypoint in the path.** The namespace was left enrolled in the waypoint, exactly as the acts leave it, with no opt-out label. cluster1's copies of echo were scaled to zero. All 30 of the next calls returned `200`, and every one was answered by a pod in cluster2. The first call after the last local pod disappeared was already one of them. The waypoint's endpoint list explains why. In August it held only local pod addresses. Now it also holds cluster2's east-west gateway:

```
envoy://inner_connect_originate/172.18.255.221:15008   HEALTHY   inbound-vip|80|http|echo.demo-apps.svc.cluster.local
```

The problem recorded that August evening no longer reproduces. The opt-out the console applies is now unnecessary, though harmless. What changed is not established. The Istio upgrade of Chapter 19 is the obvious candidate.

**The other August gotcha still reproduces.** Act 2 also leaves a route that splits in-mesh traffic between the `echo-v1` and `echo-v2` Services — and those Services are not global. With that route re-applied and cluster1's copies scaled to zero, 20 of 20 calls returned `503`. After the route was deleted, the second call returned `200`, 0.15 seconds later, and the next 20 all did. A route that points at local-only services overrides global routing. Act 3 deletes the route for exactly this reason.

**Healthy traffic does not prefer home.** The Service's manifest says *"Local endpoints are preferred; remote ones are used when no healthy local endpoint exists."* After its baseline calls, the act prints *"traffic served (locality-aware routing prefers in-cluster endpoints)"* — without comparing any numbers. Measured with everything healthy, 60 calls split exactly **30 to cluster1 and 30 to cluster2**. The act's own local-or-remote classifier recognises only one of cluster2's three pod names. That day every remote call happened to land on a different one, so the act would have counted all of them as local.

On one host, the price of that split is a fraction of a millisecond: 2.67 ms mean for calls served locally, 2.96 ms for calls served across. Between two real sites, it is a network hop for half of every service's traffic, on a day when nothing is wrong. That is a latency and data-transfer bill, set by a default nobody chose.

## What one label bought

It is worth being precise about the verdict, because both halves are true.

The label is real. One line replaced every piece of Chapter 10's plumbing. Failover works and was measured by status code, and the connection between clusters carries identities through an encrypted tunnel. The front door refuses tenants it should refuse — including, by accident, the demo's own.

But the simpler the configuration became, the more the risk moved into the checks. Chapter 10's demo could not fail because its steps didn't depend on each other. This one could not fail because its counter could not tell a response from an error page. The replacement was far better engineering, and it inherited the older demo's habit of printing green.

One more limit carries over from Chapter 10 unchanged. Act 3 joins the two Kind clusters. The target cluster — the virtual-machine clusters this book is about — is not in the mesh, for the reasons given there.

* * *

*Meridian · two weeks later*

Vikram put two terminal screenshots side by side on the wall screen. Both ended with the same line: `success=20 failed=0`.

"The left one is failover working," he said. "The right one is both sites down."

Anita looked for a long time. "So the check I asked to see go red can't."

"It couldn't. It counts replies, and an error page is a reply. The failover itself works — we measured it properly. Every one of thirty calls came back from the other site."

"And when nothing is broken?"

"Half our traffic crosses to the other site anyway. Nobody set that. It's the default."

"That one isn't an engineering question," Anita said. "That's a bill." She took a pen to the list from the week before and added a line at the bottom. "New rule. No check goes on a dashboard of mine until someone has watched it fail on purpose. And the traffic split comes to me with a cost attached, and we choose it."

![Two terminal panels, working failover and total outage, printing the identical success line.](../figures/fig-11-3-two-identical-screens.svg)

*Figure 11.3 — One check, two opposite realities, the same output.*

## The ledger

- **Built (21 August 2026):** a Gateway API front door with a team ownership split; request-level authorization on cryptographic identity, with no proxy in any application pod; one mesh across two clusters under a single local root certificate; a cross-cluster service declared with one label.
- **Re-verified (17 September; Istio 1.31.0, Kubernetes 1.37):** verification script 22 passed, 0 failed. Rogue route refused. Identity rules 200 / 403 / 403 / 200. Caller identity delivered to the application.
- **Measured by HTTP status:** cluster1 scaled to zero → 30 of 30 `200`, all from cluster2. Both clusters at zero → 20 of 20 `503`, which Act 3's counter reports as `success=20 failed=0`.
- **Found:** the ListenerSet beat refused for a missing `ReferenceGrant`, and never asserted; healthy traffic split 30/30 across clusters, despite the manifest's comment; a local/remote classifier that sees one of three remote pods; a verification section that checks configuration, not behaviour; one August gotcha that no longer reproduces, and one that still does.
- **Corrected elsewhere:** Chapter 19 had cited `success=20 failed=0` as proof of failover. It now cites the status-code measurement.
- **Cleaned up:** every replica count, label and route changed for measurement was put back, and checked.

## Ask your team

1. **When did we last watch this check fail — deliberately?**
2. **If our primary site and our backup were both down, which of our dashboards would still be green?**
3. **How much of our traffic crosses between sites when nothing is wrong, and who decided that?**

## Open the repo

- `07-istio-advanced/README.md` — the before/after table, "Order is not optional", and "Gotchas found while building this".
- `07-istio-advanced/01-install/gen-mesh-ca.sh` and `istio-values-cluster1.yaml` — the day-one decisions.
- `07-istio-advanced/act1-gateway/03-gateway.yaml`, `05-rogue-route.yaml`, `06-listenerset.yaml` — both refusals.
- `07-istio-advanced/act2-waypoint/01-authz.yaml` — the identity policy, and the comment about attaching it to the waypoint.
- `07-istio-advanced/act3-multicluster/run.sh` — `call()` and the tally; `03-global-service.yaml` — the label.
- `ui/backend/handlers/istio_actions.go` — `HandleIstioFailover`: the counter that checks for `200`, with convergence measured.
- `07-istio-advanced/verify.sh`, section 5 — what the Act 3 checks do and do not assert.
- `istioctl proxy-config endpoints deploy/waypoint.demo-apps --context kind-cluster1 | grep echo` — whether the waypoint can see the other cluster.
- **Without the repo:** Appendix G.4 quotes both counters side by side — the act's non-empty-body check, and the console's status-code check.

# On-demand target cluster: speaker script

About 4 minutes. Lines in **bold** are actions to take; the rest is what you say.
Background: the "On-demand mode" section of `CLAUDE.md`.

## Setup (before the audience is watching)

Terminal 1, the backend with a 3-minute idle limit:

```bash
cd /mnt/mil
POOL_IDLE_TTL=3m ./run-ui.sh
```

Wait for `Target front door on 172.18.0.1:16443` in the log, then open
`http://localhost:5173` → SRE → Cluster Manager.

Terminal 2, the live view:

```bash
watch -n1 'kubectl --context kind-cluster2 get cluster,vmi -A 2>/dev/null; echo; curl -s localhost:8080/api/v1/cluster/pool-status'
```

Terminal 3, put the cluster to sleep if one exists:

```bash
curl -s -N -X POST localhost:8080/api/v1/cluster/delete >/dev/null
```

Wait until Terminal 2 is empty and the UI shows "Sleeping".

Things that break the demo:

- **Don't run `make verify` beforehand.** It points the kubeconfig at `.215`, which
  bypasses the front door, so nothing wakes.
- **Don't wake it during a teardown.** Wait for Terminal 2 to be empty, or the wake takes
  about 30s instead of 15s.
- **Skip the agent chat.** `target-cluster-agent`'s serving pod is crash-looping, so that
  path isn't demoable yet.
- **Afterwards, restart `run-ui.sh` without `POOL_IDLE_TTL`** to go back to one hour.

---

## Opening (about 30s)

> "A Kubernetes cluster that nobody is using still costs you: VMs, memory and CPU, all
> day. Today I'll show a target cluster that only exists while someone is using it. You
> access it and it builds itself. You stop using it and it deletes itself."

**Point at the UI ("Sleeping") and at Terminal 2 (empty).**

> "Right now there is no target cluster. No control plane, no worker, no VMs. Zero
> resources."

## Nothing is there (about 30s)

**Run:** `kubectl --context kind-cluster2 get cluster target-cluster`

> "The management cluster agrees: not found."

**Run:** `grep server: target-cluster-kubeconfig`

> "But I still have my kubeconfig. It doesn't point at the cluster. It points at a front
> door, `.223`, which the platform runs for me. That front door is the trick."

## The cold wake (about 45s)

**Run:** `time kubectl --kubeconfig target-cluster-kubeconfig get nodes`

> "I'm just going to use the cluster like I normally would. Plain kubectl, nothing
> special."

**While it waits, point at Terminal 2 as the VMs appear.**

> "The front door saw a request from someone holding a valid certificate, so it started a
> build. The control plane and worker boot in parallel from a pre-baked image. kubectl is
> simply waiting. I'm not retrying anything."

**When the nodes print:**

> "About fifteen seconds, from nothing to an answering API server. Those are real nodes in
> a brand-new cluster."

## The idle clock (about 30s)

**Point at the UI countdown.**

> "The platform now tracks usage. Every request through the front door resets the clock.
> For the demo I've set the limit to three minutes. In normal use it's one hour."

**Run** `kubectl --kubeconfig target-cluster-kubeconfig get nodes` **again.**

> "I used it again, so the clock resets."

## Idle delete (about 45s; fill the wait)

> "Now I'll stop using it. While we wait: the cluster's own internal traffic doesn't count
> as use. Only real users and agents coming in through the front door do. So a cluster
> nobody touches really does go away."

**When Terminal 2 empties and the UI shows "Sleeping":**

> "And it's gone. VMs deleted, resources back. My kubeconfig is still here."

## Wake it again (about 30s)

**Run the same** `time kubectl --kubeconfig target-cluster-kubeconfig get nodes`.

> "Same file, same command. It's a completely new cluster, but my old credentials still
> work because the cluster is built with the same fixed certificate authorities every
> time."

## Close (about 15s)

> "So: no cost while it's idle, about fifteen seconds to come back, and it cleans up after
> itself. The user just runs kubectl."

---

## If someone asks

- **"Why not keep one warm?"** "You can. Standby mode hands over a pre-built cluster in
  half a second, but it keeps running all day. It's a trade-off between speed and cost."
- **"What if the platform is down?"** "Then the front door is down too. Waking and
  auto-delete both need the backend running."
- **"Is this production-ready?"** "No. It's a demo: the certificate authorities are fixed
  and there's one cluster at a time."

Measured numbers behind the script: first `kubectl` answered in 15.3 / 15.3 / 15.7s from no
cluster; both nodes Ready at about 20.5s; idle teardown about 14s.

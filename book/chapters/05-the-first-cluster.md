# Chapter 5 — The First Cluster

> *Spins up a full Kubernetes cluster where each node is a Virtual Machine running inside our management cluster.*
>
> — `demo.sh`, "What this demo does"

*Meridian · a Wednesday in September · the fourth-floor boardroom*

The newest member of the board had asked a fair question at dinner — *what exactly did we buy?* — and Anita Rao had promised him the demo that had won the pilot its funding in February.

"From the beginning," she told Vikram. "The version where the machine explains itself."

Vikram started the script. The screen filled with a banner, a sentence about virtual machines running inside a management cluster, and a checklist.

```
    ✓ KubeVirt installed
    ✓ CDI installed
    ○ Container registry (172.18.0.2:5000) — not ready
```

Then nothing. The script had stopped.

"It's the demo," Vikram said, into the silence. "Not the platform. The platform's fine."

The board member smiled politely. Anita did not.

"Then show him the platform," she said, "and afterwards, tell me how the thing we show people stopped working without anyone noticing."

* * *

*Build log · February–September 2026*

## The demo that stopped at step one

`demo.sh` was written on 11 February 2026, the platform's first day in the repository, and last changed on 28 February. It is a good piece of work: nine steps, each announced with a banner and a paragraph explaining what is about to happen and why — check the prerequisites, make sure the load balancer is running, make sure the Cluster API providers are ready, pre-pull the VM image, show the golden image, deploy the cluster, wait for the VMs, wait for the API server, verify.

Run on 17 September, it printed two green checks and stopped at the third.

The cause is Chapter 6's. The script confirms the registry is up by sending a plain HTTP request to `172.18.0.2:5000` — the alias every image reference uses, which only the container runtime knows how to translate. A plain request to that address now reaches a Kubernetes node with no registry on it. The check fails, and because the script is written to stop on the first error, the demo ends before it begins.

Nothing in the demo is wrong about February. Its narration describes VMs with two CPUs and 4 GiB of memory; today's default VMs have four cores and 8 and 6 GiB. It pre-pulls the image tagged `:latest`, which Chapter 6 found was last built in March. The platform kept moving, and the story about the platform stayed where it was.

What follows walks the same ground on the running platform instead.

## Three clusters, one host

The platform runs on a single workstation, and it is built from three Kubernetes clusters.

`cluster1` and `cluster2` are Kind clusters: Kubernetes whose "nodes" are Docker containers. `cluster1` hosts the service-mesh demonstrations in Part III. `cluster2` is the **management cluster** — the factory. It runs Cluster API, which manages clusters as Kubernetes objects; KubeVirt, which runs virtual machines as Kubernetes workloads; CDI, which imports disk images; MetalLB, which hands out load-balancer addresses; and the agent platform from Part IV.

The third cluster, `target-cluster`, is the product. It is a k3s cluster whose control plane and worker are virtual machines running *inside* `cluster2`.

The repository builds the factory in numbered stages — `00-prereqs` installs the `clusterctl` tool, `01-metallb` the load balancer, `02-capi-init` the Cluster API providers, `03-target-cluster` the target cluster, `04-verify` a health check — with `make all` running the first four. As Chapter 19 found, the stages assume a few things already exist: the Kind clusters themselves, KubeVirt, and CDI are installed by hand.

## One apply

The default target cluster is one file: `03-target-cluster/target-cluster-warm.yaml`, 258 lines declaring seven objects.

| Object | What it declares |
|---|---|
| `Cluster` | the cluster exists, and which objects implement it |
| `KubevirtCluster` | its machines are KubeVirt virtual machines |
| `KThreesControlPlane` | a one-node k3s control plane |
| `KubevirtMachineTemplate` ×2 | the shape of each VM: cores, memory, disk image |
| `Secret` | the worker's startup configuration |
| `MachineDeployment` | one worker, kept at one |

Before it is applied, a script seeds four more secrets — certificate authorities and a join token — so the cluster's identity is known in advance (Chapter 8 explains why). Then `kubectl apply`, and the file's work is done. From here on, the cluster exists only as a declaration, and controllers make it true.

Four controllers take part: Cluster API's core controller, the KubeVirt infrastructure provider, and k3s's bootstrap and control-plane providers — with KubeVirt itself underneath. Counted on the running platform on 17 September, those seven objects and four secrets had produced about eighteen more: a bootstrap configuration, a MachineSet, two Machines, two KubevirtMachines, two VirtualMachines, two running VM instances, two launcher pods, a LoadBalancer Service holding the API address `172.18.255.215`, and five further Secrets, including the kubeconfig that grants access to the new cluster.

Followed through its owner references, the control plane's chain reads from the top down:

```
Cluster/target-cluster
  └─ KThreesControlPlane/target-cluster-control-plane
       └─ Machine/target-cluster-control-plane-qkckl
            └─ KubevirtMachine/target-cluster-cp-s9j8h
                 ┆ (no owner reference)
                 VirtualMachine/target-cluster-cp-s9j8h
                   └─ VirtualMachineInstance/target-cluster-cp-s9j8h
                        └─ Pod/virt-launcher-target-cluster-cp-s9j8h-…
```

One link is missing. The VirtualMachine has no owner reference back to the `KubevirtMachine` that created it; the provider manages it without one. So deleting a cluster is not a single cascade that Kubernetes performs on its own. It is a sequence of controllers noticing, finalizing, and deleting, one object at a time — which is part of why, in Chapter 19, tearing a cluster down took nearly nineteen seconds.

## Six layers deep

The chain above describes ownership. The picture below describes where the software actually runs, and it is the thing to hold in mind for the rest of Part II:

1. **A workstation.**
2. **A Docker container**, `cluster2-control-plane` — a Kind node running Kubernetes 1.37.
3. **A pod** in that cluster, `virt-launcher-target-cluster-cp-…`, whose job is to run one virtual machine.
4. **A virtual machine** inside the pod: QEMU running Ubuntu 24.04.5, with four cores and 8 GiB.
5. **A Kubernetes node** inside the VM: `target-cluster-cp-s9j8h`, running k3s v1.37.0 and its own container runtime.
6. **Pods** on that node — today, CoreDNS, the metrics server, a storage provisioner.

The layers show up in small, strange ways. The target cluster reports its nodes' internal addresses as `10.244.0.115` and `10.244.0.114`, and those are not addresses in the target cluster at all: they are pod addresses in the management cluster, because each VM lives in a pod. Everything outside reaches the target cluster's API through one load-balancer address that `cluster2` hands out.

Each layer is a boundary, and the virtual-machine layer is the one that matters for tenants: it gives each target cluster its own kernel, not just its own containers.

## The victory lap

`show-cluster.sh`, written on 17 February, is the demo's second half. It walks every object in the chain, looks inside the target cluster, then proves the cluster works by deploying a web server. It was run as-is on 17 September.

Most of it still told the truth. It listed both VMs — four cores and 8 GiB for the control plane, four cores and 6 GiB for the worker — and the load-balancer Service at `172.18.255.215`. Some of it had aged. Its disk section announced that *"each VM disk is a PVC cloned from the golden image via CDI DataVolumes"* and then found none, because disks have been container images since February (Chapter 6).

It also reused a kubeconfig file it found in `/tmp`, written the previous afternoon for a cluster that had since been deleted and rebuilt six times. The file still worked. The warm image's fixed certificates (Chapter 6) give every rebuilt cluster the same identity, so a credential issued for yesterday's cluster opens today's. For a single-cluster demo that is convenient. It is also a credential that outlives the thing it was issued for.

Then the web server. The script created a namespace, deployed two replicas of nginx, and printed its heading for the result:

```
==> Pods (should be spread across nodes):
    nginx-demo-…-7kmpx   1/1   Running   0   8s   10.42.1.7   target-cluster-cp-s9j8h
    nginx-demo-…-pf24z   1/1   Running   0   8s   10.42.1.6   target-cluster-cp-s9j8h
```

Both replicas ran on the control plane. Neither ran on the worker. The script then reported success — *"✓ nginx responded: `<title>Welcome to nginx!</title>`"* — from a request that went to `localhost` inside one of the pods, which proves that nginx starts, and nothing about the second node or the Service in front of it. The namespace was deleted afterwards, as the script instructs.

## Ready, but not allowed to work

The worker was not busy and not broken. It was forbidden.

```
target-cluster-workers-rz9xt-m5x6z:
  taints=[{"effect":"NoSchedule",
           "key":"node.cloudprovider.kubernetes.io/uninitialized",
           "value":"true"}]
```

That taint tells the Kubernetes scheduler to place no ordinary workload on the node. On 17 September the worker ran nothing at all: every pod in the target cluster — the four system pods and both web servers — was on the control plane.

The cause is one line in the worker's startup configuration, the Secret in the manifest:

```yaml
kubelet-arg:
  - cloud-provider=external
```

That setting tells the worker's kubelet that an external *cloud controller* will initialize the node, and to keep the node tainted until it does. But the control plane is configured with its cloud controller disabled. Nothing on the platform ever initializes the worker, so the taint never comes off.

And every signal says the worker is fine. Its Machine is `Running`. Its node is `Ready`. Its provider ID matches the Machine's, so Cluster API links them. The verification script lists it as `Ready`. The stopwatch in Chapter 7 waits for it. By every automated measure on the platform, the target cluster has two working nodes. It has one — and a second virtual machine holding four cores and 6 GiB of memory that has been given nothing to do.

This static worker configuration was introduced in June, to let the worker boot at the same time as the control plane (Chapter 8), and the same line is in the default path's manifest today. This chapter verified only the cluster that was running on 17 September; whether every cluster since June has had an idle worker was not checked, and the fix — removing the setting, or running something that initializes nodes — has not been tried.

The only check on the platform that ever gave the worker real work was the victory lap. Its heading said the pods *should be spread across nodes*. It printed that they were not, and declared success anyway.

* * *

*Meridian · the following Monday*

"It's a two-node cluster," Vikram said, "with one node allowed to work. It's been that way since June, as far as I can tell. Every dashboard we have says both are Ready."

Anita let that sit. "And the demo?"

"The demo stopped because of a registry address. The victory lap still runs. It even prints the evidence — it says the pods should be on both nodes and shows them on one. Nobody reads that line."

"So we have been timing, to a tenth of a second, the arrival of a machine that isn't allowed to do anything." She was quiet for a moment. "Change one word on the dashboard. Stop saying *Ready*. Say *working*, and don't show it until something we gave that node has actually run on it."

"And the demo?"

"Fix it, and run it from the first step every Monday." She stood. "The thing we show people is the thing people believe. It had better still be true."

![Six nested boxes: workstation, container, pod, virtual machine, pod, container.](../figures/fig-05-2-six-layers.svg)

*Figure 5.2 — Six layers: a pod inside a VM inside a pod inside a container.*

## The ledger

- **Declared:** seven objects in a 258-line manifest, plus four pre-seeded secrets.
- **Produced by controllers:** about eighteen more — Machines, VMs, launcher pods, a load-balancer Service, and Secrets including the new cluster's kubeconfig.
- **Nesting:** six layers, from the workstation to a pod inside a VM inside a pod inside a container.
- **Found on 17 September:** the origin demo stops at its first check; the victory lap narrates a disk design replaced in February; a day-old credential opens a newly rebuilt cluster; and the worker node, `Ready` by every measure, carries a taint that keeps every ordinary workload off it.
- **Idle capacity:** one VM, four cores and 6 GiB.
- **Still owed:** a demo that runs from step one; a definition of "done" that schedules work on every node; a worker configuration that does not wait for a cloud controller the platform does not run.

## Ask your team

1. **When a dashboard says a system is ready, what did anyone check besides the system's own report?**
2. **Does our flagship demo still run today, from the first step?**
3. **How much capacity are we paying for that has never done any work?**

## Open the repo

- `demo.sh` — the nine narrated steps; `show-cluster.sh` — the victory lap.
- `00-prereqs` through `04-verify`, and `make all`.
- `03-target-cluster/target-cluster-warm.yaml` — the seven objects, and the worker's `kubelet-arg` in its bootstrap Secret.
- On the management cluster: `kubectl get cluster,kthreescontrolplane,machinedeployment,machine,kubevirtmachine,vm,vmi`.
- On the target cluster: `kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name} {.spec.taints}{"\n"}{end}'`.
- **Without the repo:** Appendix G.1 quotes the manifest's seven objects, including the worker bootstrap Secret and its `kubelet-arg`.

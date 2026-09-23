# Chapter 4 — Kubernetes That Runs Kubernetes

> *Cluster API (CAPI) is elegantly defined as "using Kubernetes to manage Kubernetes."*
>
> — *Reclaiming Margins with Sovereign Cloud*, strategic briefing, February 2026

*Meridian · a Saturday in January 2026 · the board strategy day*

The board gave the technology session forty minutes, and Anita had been told to use no more than one slide.

The slide showed three roads. On the left, *stay with what we know*: Meridian's virtualization supplier, which now also sold Kubernetes built into its platform. In the middle, *rent the pieces and run them ourselves* on one cloud. On the right, *build our own*.

The chair asked the question she had been waiting for. "I understand the left-hand road. I understand renting. What is the right-hand one, in a sentence? What would we actually be building?"

"A factory," Anita said. "Not for products — for computing environments. We write down what we want: how many machines, what size, which software. The factory reads the order and builds it, and it keeps checking that what exists still matches what we wrote. If a machine dies, it builds another. If we change the order, it changes the environment."

"And who builds the factory?"

"The same factory builds itself, the same way. That's the point, and it's the part that sounds like a trick." She looked around the table. "The technology is called Cluster API. It's Kubernetes, managing Kubernetes."

* * *

## A factory for clusters

*Briefing · February 2026*

Kubernetes is already a factory of a kind. The strategy deck explains it in one line: *"Normally, you write YAML to say 'I want 3 web servers running' — Kubernetes makes it happen automatically."* You declare what you want, and a control loop keeps working until reality matches the declaration.

**Cluster API** applies the same idea one level down. In the deck's words: *"With CAPI, you write code to say 'I want 3 Kubernetes clusters with these hardware settings' — and it creates them."* A central **management cluster** *"acts as the command center, bootstrapping workload clusters across providers."*

The words that matter for a board are *declaration* and *reconciliation*:

- **Declaration** means a cluster is described in a file. The file can be reviewed, approved, versioned and audited like any other document. The question "how is the payments cluster configured?" has an answer you can read.
- **Reconciliation** means the management cluster keeps comparing the description with what exists, and acting on any difference. A cluster is not built once and then left to drift. It is continuously held to its description.

The deck calls CAPI *"a universal translator"*, because the same declaration style works whether the machines come from a public cloud, a private one, or bare metal. A different **provider** plugin translates the declaration for each kind of infrastructure. That is where the escape from lock-in lives: not in the promise that nothing changes when you move, but in the fact that the *shape* of the declaration stays the same.

## The spectrum

*Briefing · February 2026*

The deck lays the options out as a spectrum from convenience to control, using three real products as markers:

- **VMware Tanzu, "The Familiar Choice"** — aimed at *"organizations with 15+ years of VMware estate"*; Kubernetes built into the virtualization platform they already run. *"No staff retraining required."* The warning attached: *"High vendor lock-in persists. If Broadcom raises prices or you want bare metal elsewhere, you're still tangled in proprietary web."*
- **CAPZ, "The DIY Power User"** — the open-source Cluster API provider for Azure. *"You manage Azure VMs, networking, and cluster resources yourself. No Microsoft support ticket for internal cluster logic."*
- **Kubermatic Kubernetes Platform (KKP), "The Multi-Cloud Orchestrator"** — *"a unified control plane across AWS, Azure, GCP, and on-premises infrastructure."*

The deck's summary line: *"From safe but locked (Tanzu) → powerful but complex (CAPZ) → truly sovereign (KKP)."* Past the end of the spectrum, it adds a fourth option — *"The Self-Managed Sovereign Platform"*: *"building your own cloud platform using open standards."*

Two cautions before a board uses this slide. First, the labels are the deck's opinions, not measurements; *"truly sovereign"* in particular depends on who runs the platform, under what licence, and with what support. Second, the four options are not exclusive. The platform in this book is the fourth kind, built from the same open-source Cluster API the middle options use. Meridian's three roads share more pavement than the slide suggests.

## What the slide leaves off

*Not independently verified · September 2026*

A three-marker spectrum is a slide, not a market. A board evaluating this decision will meet several other options, and it is worth naming them so that nobody mistakes the deck's three for the field.

**A caution that applies to this whole section.** What follows is a map, assembled from each project's own public description of itself. **None of it was installed, benchmarked, or otherwise verified for this book.** Only one option here was actually built and measured — the one the rest of the book is about — and a reader should treat every other line as a pointer to evaluate, not a finding. Capabilities change release by release; check current documentation before deciding anything.

- **Harvester** is the closest thing to a productized version of what this book builds by hand: an open-source hyperconverged infrastructure product from SUSE that packages KubeVirt with storage and networking, and integrates with Rancher for cluster management. An organization that wants this architecture without assembling it is looking for this category. The trade is the usual one — less to build, less to change.
- **OpenStack** is the long-established open-source answer to "run our own infrastructure-as-a-service," predating the Kubernetes era and widely deployed in telecommunications and public-sector estates. It solves a broader problem than this book's platform and is correspondingly larger to operate. Organizations that already run it should be asking how Kubernetes sits *on* it, not whether to replace it.
- **Talos Linux** takes a different cut at the same sovereignty argument: a minimal, immutable, API-managed operating system built only to run Kubernetes, with no shell and no package manager. It addresses the layer this book mostly inherits — the golden image and what is inside it (Chapter 6).
- **Distributions** — RKE2, OpenShift, Anthos, EKS Anywhere — are the "supported Kubernetes you run yourself" tier, and they are the most likely real alternative for an enterprise with a support requirement. They differ enormously in licence and price, and two of them are paths back toward the vendor whose bill started this book. That is not an objection; it is the trade being made deliberately, which is the whole subject of Chapter 2.
- **OpenTofu and Terraform** are the declarative alternative to Cluster API itself, and the comparison is genuinely interesting for a board. Both describe infrastructure as code. The difference is *where the loop lives*: Terraform and OpenTofu run a plan when a person or a pipeline runs them, while Cluster API runs a controller inside a cluster that continuously reconciles reality against the declaration. Chapter 18 makes the case for the continuous kind, and Chapter 19 shows what happens without it. Many organizations use both — one to build the foundation, the other to manage what runs on it.

The book's own choice is not presented as the winner of this list. It is the option that best matched one set of constraints — open source throughout, virtual machines as first-class objects, no dependency on a provider's control plane — and the rest of the book is what happened when someone actually built it.

## Why virtual machines still matter

For an organization like Meridian, the most important word on the Tanzu marker is *estate*. Fifteen years of virtualization means thousands of virtual machines running systems that will never be rewritten as containers: vendor appliances, licensed databases, old applications nobody dares to touch.

A Kubernetes strategy that has nothing to say about virtual machines asks those systems to wait for a future that will not arrive. That is why the platform in this book is built on **KubeVirt**, an open-source project that runs virtual machines *as Kubernetes objects*. A virtual machine sits beside containers, is declared in the same kind of file, and is scheduled, networked and monitored by the same system.

The platform then does something a little dizzying with that. The workload clusters it manufactures are Kubernetes clusters **whose machines are themselves virtual machines**, run by KubeVirt on the management cluster. Kubernetes manages virtual machines, which run Kubernetes.

One honest limit belongs here, before Part II begins. The repository contains no tooling for *moving* existing VMware virtual machines — none of the usual import or conversion tools appears anywhere in it. This book shows the destination, and measures it. The migration of an existing estate is a separate project, and this book does not pretend to have done it.

That tooling does exist, and a board should know the names even though this book cannot report on them. **Forklift** is the open-source migration project for bringing virtual machines from VMware and other sources into a KubeVirt environment; **virt-v2v** is the older, lower-level converter that turns a virtual machine's disk into a form a different hypervisor can boot. Neither was run here, so this book has nothing to say about how well they work on a real estate, how long a migration takes, or what fraction of machines need hand-holding — and those three questions, not the destination architecture, are what determine whether a migration is affordable. Treat the names as the start of an evaluation. An organization serious about this decision should migrate a representative sample early, because the answer will shape the business case far more than the thirty-four seconds in Chapter 7.

## The factory, running

*Build log · 17 September 2026*

This is the factory as it stood on the day this chapter was checked.

**The machinery is open source, with version numbers.** On the management cluster: Cluster API v1.14.2; the KubeVirt infrastructure provider (CAPK) v0.11.2; the k3s bootstrap and control-plane providers v0.3.0; KubeVirt v1.9.0; and CDI v1.66.0, which imports disk images for the virtual machines.

**The order is a 258-line file.** The default declaration, `03-target-cluster/target-cluster-warm.yaml`, contains seven objects:

| Object | What it declares |
|---|---|
| `Cluster` | The cluster itself, and what it is made of |
| `KubevirtCluster` | The infrastructure it runs on — here, KubeVirt |
| `KThreesControlPlane` | The control plane: one k3s server, version v1.37.0+k3s1 |
| `KubevirtMachineTemplate` ×2 | The virtual-machine shapes, one for the control plane and one for workers |
| `MachineDeployment` | The worker pool: how many, from which template |
| `Secret` | A pre-shared join token, so the worker can boot in parallel with the control plane |

**The factory built what the order said.** Cluster API recorded two Machines. Each is backed by a KubeVirt virtual machine, `target-cluster-cp-s9j8h` and `target-cluster-workers-rz9xt-m5x6z`, both `Running` and ready. From the moment the file is applied, both machines are ready in a median of **33.7 seconds**, and the whole cluster is torn down in **45.7 seconds** (Chapters 7 and 17).

**The factory fits on one host.** The management cluster, the cluster it built, a second cluster for the service mesh, and the AI agents all ran on one host (the prologue lists it). The manufacturing pattern is the same one a data centre would use. The capacity is not.

## The gauge that disagrees

One more reading from the same day belongs in a chapter for a board, because it is the first appearance of this book's most persistent theme.

While the target cluster was serving workloads normally — including the measurements in Chapter 10, run that morning — Cluster API's own summary of it read:

```
AVAILABLE   False
ControlPlaneAvailable=False  (WaitingForKthreesServer)
ControlPlaneComponentsHealthy=False  Following machines are reporting control plane errors
```

The factory reported its product as *not available* while the product was working. The k3s control-plane provider, at version 0.3.0, is a young project, and the cause was not investigated for this chapter. Chapter 5 will find the mirror image of the problem: a status that said *Ready* about a machine that could not do its job.

The lesson for a board is the same in both directions. A factory's gauges are part of the factory. They need to be checked against the product as carefully as the product is checked against the order.

A smaller example of the same drift sits at the top of the repository. Its front-page architecture diagram still shows the platform as it was in March — a namespace, addresses and a cross-cluster proxy that no longer exist (Chapters 10 and 12). Documentation, like a gauge, describes the factory only until the factory changes.

## From money to machinery

Part I has argued from the invoice outwards: that renting stops being obviously cheaper for steady work (Chapter 1); that every landlord controls something you need (Chapter 2); and that for some organizations, where data and AI live is not a preference but a constraint (Chapter 3). This chapter is the bridge: a way to own the brain of your clusters, declaring them in files and manufacturing them on hardware you control.

The rest of the book is the machinery, measured. **Part II, The Factory**, is how the clusters are made, and how long it takes. **Part III, The Fabric**, is how they talk to each other. **Part IV, The Brain**, is what AI agents can safely do with them. **Part V, The Dividend**, returns to the money.

For a CxO, four questions carry through all of it. How long does the factory take to build? How long to destroy? What exactly does the order say? And how do you know the gauges are telling the truth?

* * *

*Meridian · the same Saturday, forty minutes later*

The board did not choose a road. It chose a test.

"Small," the chair said. "Owned hardware, open-source software, a single team. Show us that the factory can build an environment from a written order faster than we can rent one, and show us how you measured it. Then we'll talk about roads."

Anita wrote the commitment into the minutes herself, in words the board could hold her to.

*The platform team will build a small cluster factory on hardware Meridian owns, using open-source Cluster API. It will demonstrate an environment built from a written declaration, and report how long it takes, measured, by February.*

Vikram read it over her shoulder. "*Measured* is doing a lot of work in that sentence."

"It's the only word in it I care about," Anita said.

## The ledger

- **The idea (briefing):** Cluster API extends Kubernetes' declaration-and-reconciliation model to whole clusters, run from a management cluster, with providers for different infrastructure.
- **The spectrum (briefing):** Tanzu, CAPZ and KKP as markers from convenience to control, and a fourth option — self-managed on open standards. The labels are the deck's opinions.
- **Virtual machines (build log):** KubeVirt runs VMs as Kubernetes objects; this platform's workload clusters run on VMs. There is no VMware migration tooling in the repository.
- **The factory, 17 September 2026 (build log):** Cluster API v1.14.2, CAPK v0.11.2, k3s providers v0.3.0, KubeVirt v1.9.0, CDI v1.66.0. A 258-line declaration of seven objects produces two VM-backed machines, ready in 33.7 s median and torn down in 45.7 s.
- **Found:** Cluster API reported the working cluster as `Available=False`; the repository's architecture diagram still shows March.

## Ask your team

1. **Can we point to the file that describes each of our production environments — and does the environment still match it?**
2. **Which of our virtual machines will never become containers, and what is our plan for running them next to the ones that will?**
3. **When a status screen says an environment is healthy or unavailable, how do we know it is right?**

## Open the repo

- `03-target-cluster/target-cluster-warm.yaml` — the order: seven objects, 258 lines.
- `kubectl --context kind-cluster2 get clusters,machines,kubevirtmachines -A` and `kubectl --context kind-cluster2 get vm -A` — the factory and what it built.
- `kubectl --context kind-cluster2 get clusters.cluster.x-k8s.io target-cluster -o wide` — the gauge.
- `02-capi-init/init-management-cluster.sh` — how the factory itself is set up.
- `README.md` — the architecture diagram, and how far it has drifted.
- The strategy deck, slides 14–16; the recorded factory demonstration (8 min 56 s), end to end.

# Chapter 6 — Ten Minutes Becomes Two

> *The golden image pre-bakes k3s binary, airgap images, CA certificates, and kernel modules into a DataVolume, reducing cluster spin-up from ~10 minutes to ~2 minutes.*
>
> — `README.md`

*Meridian · a Wednesday in September · the CTO's office*

The supply-chain questionnaire from the regulator had forty questions, and Anita Rao had flagged one.

*For each new compute environment you create: where does its software come from, and when was it obtained?*

"Walk me through a new cluster," she said.

"Nothing is downloaded when a cluster is created," Vikram said. "Everything it needs — Kubernetes, its container images, its system settings — was downloaded once, when we baked the golden image. Every cluster boots from a copy of that image. No internet at boot."

"That's a good answer for a sovereign platform." She wrote it down. "Second half. When was the image baked?"

"The one we deploy from, yesterday. The one the standby cluster uses —" He checked. "The third of March."

Anita stopped writing. "So some of the clusters we create today are six months old the moment they're born."

"In their software, yes."

"Then an image isn't a file," she said. "It's a decision someone made on the day it was baked, copied into every cluster after. I want to know what's inside each one, and when it was decided."

* * *

*Build log · February–September 2026*

## Why a cluster took ten minutes

A virtual machine has to get its software from somewhere. In the platform's earliest form, each VM booted a stock Ubuntu cloud image and then set itself up: fetched k3s and the container images it needs, and prepared itself to run a cluster — over the network, on every boot, for every node.

On 28 February 2026 the repository gained two scripts that changed that: `bake-golden-image.sh` and `build-containerdisk.sh`, committed together as *"optimized images for cdi and container images."* The README states the result in one line — cluster spin-up *"from ~10 minutes to ~2 minutes"* — and the same pair of numbers appears in the developer guide and the strategy document. The user guide is more cautious: *"~2-5 minutes (with golden image) or ~10 minutes (without)."*

Those figures deserve the same honesty as the rest of Part II. They are stated in four documents, and no run log or script output in the repository records the measurements behind them. They date from before the platform had an instrument (Chapter 7 tells that story). What is not in doubt is the direction, and the reason: the slowest thing a new machine can do is download its own software.

## Baking

Baking a golden image is a four-step job run by `bake-common.sh`, and on 16 September it was run from scratch during the rebuild in Chapter 19.

1. **Clone the base image.** A stock Ubuntu Noble cloud image, already imported into the cluster as a DataVolume by the Containerized Data Importer, is cloned into a new volume. On 16 September: about 50 seconds.
2. **Boot a bake VM** from the clone, with a cloud-init script that does the real work.
3. **Wait for the VM to finish and power itself off.** On 16 September: 160 seconds.
4. **Delete the VM and keep the disk.**

Inside the VM, the script's own log lines tell the recipe. It installs the k3s binary. It downloads k3s's air-gapped container images. It writes the systemd units k3s will run under. It pre-loads kernel modules and sets kernel parameters. It masks services that slow a boot down and are useless here — `snapd`, `multipathd`, the automatic `apt` timers, `motd-news`, unattended upgrades. Depending on the image variant, it starts k3s once to warm its datastore, then stops it. It creates the login user and an air-gapped install script. And finally it resets cloud-init's state and the machine ID, so that every VM booted from this disk believes it is a new machine, and powers off.

That last step is worth noticing for what it does *not* reset. It wipes the operating system's memory of the bake VM, not k3s's. The warm image's datastore still contains a Node object for the machine it was baked on — the ghost that, in Chapter 7, fooled a stopwatch into stopping early.

The sovereignty consequence is the one Vikram gave the regulator. The internet is touched during the bake — k3s from its install endpoint, the air-gapped images from the project's release page — and never again. The cluster manifests declare `airGapped: true`, and every cluster after that boots from what the image already holds. The dependency on the outside world moves from every boot of every node to one event, on one day, that can be reviewed.

## A disk that ships like a container

A baked disk still has to reach each new VM. The first approach cloned the golden DataVolume for every virtual machine — a full copy of a 20 GiB volume per node. `build-containerdisk.sh` replaced that with a KubeVirt *containerDisk*: the VM's disk packaged as an ordinary container image.

The script does it in four steps. A helper pod mounts the baked volume and converts the disk to a compressed `qcow2` file. The file is copied out. A two-line container image is built around it:

```
FROM scratch
ADD --chown=107:107 disk.qcow2 /disk/disk.qcow2
```

And the image is pushed to the platform's local registry. On 16 September, the rebuilt warm image came out at **1.35 GB — against a 20 GiB volume.**

A containerDisk behaves like any container image. It is pulled to a node once and reused; it needs no per-VM volume clone; and it is ephemeral — every start of the VM begins from the pristine image, which is why, in Chapter 7, a restarted VM counts as a genuine first boot.

Measured on 17 September, pulling a 1.29 GB containerDisk that was not yet on the node took **4.2 seconds** from the local registry, and pulling it again once cached took **0.1**. That is the value of the Makefile's `pre-pull` targets on this host: a few seconds per image, paid once. Worth having. Not, as Chapter 8 showed, where the time in a cluster build actually goes.

The containerDisk script's header makes its own promise: *"~30-60 second cluster spin-up (vs 1.5-2.5 min with DV cloning)."* It was written before the instrument existed. When the repository wrote down its baseline in June, cold builds took 90 to 150 seconds, with the worker waiting for the control plane before it even started. The images were not the reason. Chapter 8 tells what was.

## The address that isn't

Every image reference on the platform names its registry the same way:

```
172.18.0.2:5000/ubuntu-noble-k3s:warm
```

`172.18.0.2` is not the registry's address. It never has been.

The two Kind clusters and the registry all run as containers on one Docker network, and Docker hands out addresses in the order containers join it. `.2` goes to whichever joined first. On 17 September the network looked like this:

```
cluster1-control-plane   172.18.0.2
cluster2-control-plane   172.18.0.3
registry                 172.18.0.4
```

Every image reference on the platform points at the control-plane node of `cluster1` — a Kubernetes node that runs no registry. Before the rebuild in Chapter 19, the repository's notes recorded `.2` as `cluster2`'s node instead, and in June as `cluster1`'s. The address in the image names has moved from one neighbor to another and back. In every record the repository keeps, it has never been the registry's own.

Images still pull, because the container runtime on each `cluster2` node is given a small translation file:

```
[host."http://172.18.0.4:5000"]
  capabilities = ["pull", "resolve", "push"]
  skip_verify = true
```

containerd reads it and quietly sends every request for `172.18.0.2:5000` to `172.18.0.4:5000` instead. Nothing else knows about the file. Asked directly from the same node on 17 September, a plain HTTP request to the alias failed to connect at all; the same request to the registry's real address returned `200`.

That arrangement fails in two ways, and both have happened.

**The registry's address drifts.** When containers reconnect to the network, the registry can come back at a different address. The translation file still points at the old one, and new VMs sit in `ImagePullBackOff`, failing to reach `172.18.0.2:5000`.

**The file disappears.** It lives inside the node's container, so recreating a node — as Chapter 19 did to both clusters — deletes it.

The repair is `scripts/fix-registry-hosts.sh`, added in June. It looks up the registry's current address and rewrites the translation file on every `cluster2` node. The Makefile runs it before the deploy and pre-pull targets. The UI's own deploy path does not.

The alias claimed one more casualty. The UI has a Registry tab that lists images, and it dialed the alias the way any program would — over plain HTTP, without containerd's translation. It reported `failed to connect to registry` and showed the registry as disconnected, both from the host and from inside the cluster. The fix split one setting into two: `REGISTRY_URL`, a real address to dial, and `REGISTRY_ALIAS`, the name to *display* — because, in the handler's words, the alias *"is what image references use."*

The platform works. But hundreds of references name a machine that is not the registry, and every consumer that needs to reach it must be told, separately and locally, where the registry really is.

## An image is a decision, frozen

By 17 September the registry held three tags of the platform image, and the gap between them is the chapter's real subject.

`:warm` was built on 16 September, the day of the upgrade. `:preinit` was built on 18 April. `:latest` was built on **3 March**. After the rebuild, only `:warm` had been pulled onto the new `cluster2` node.

Everything inside an image was decided on the day it was baked, and every cluster built from it inherits those decisions:

- **A Kubernetes version.** Chapter 19 had to rebake `:warm` because every tag still carried k3s v1.31, and the variable the bake script documented for choosing the version turned out never to have controlled anything.
- **A ghost.** The warm image's Node object for its own bake VM (Chapter 7).
- **Fixed certificates and a join token**, baked into the warm image on purpose, safe only while one cluster runs at a time.
- **An Ubuntu base** as it stood on the day it was imported.

And the checks that decide whether to rebuild an image look only for its presence. `ensure-warm-image` asks whether an image tagged `warm` exists on the node, not what it contains. The standby cluster in Chapter 9 is built from `:latest` — whose name promises "the newest," and whose contents are six months old.

For a CxO, a golden image is what a regulator would call a baseline: the approved state every new system starts from. It earns its keep only if its age and contents are as visible as the systems it produces.

* * *

*Meridian · the following Wednesday*

Vikram's answer to question fourteen ran to a table: one row per image, with the date it was baked, the Kubernetes version inside, the base operating system, and what had been downloaded to build it.

Anita read the row for `:latest` twice. "March."

"It's scheduled for a rebake, or for deletion."

"And the address in every image name that belongs to another machine?"

"It works because each cluster node has a file that translates it. We've added the file to the list of things a rebuild must recreate."

"Then call it what it is." Anita made a note in the margin. "Anything that only works because something else translates it for us is a dependency. It goes on the dependency list, not in a script comment." She signed the page. "And every image gets a birth certificate: what's in it, where it came from, and the date after which we stop building from it."

"An expiry date?"

"Every cluster we build is as old as the image it came from," she said. "I'd like to decide how old that's allowed to be."

## The ledger

- **Claimed:** cluster spin-up from about ten minutes to about two with the golden image — stated in four documents, with no measurement recorded.
- **Measured (16–17 September):** base-image clone about 50 s; bake VM 160 s; a 20 GiB disk packaged as a 1.35 GB containerDisk; a cold pull of a 1.29 GB containerDisk in 4.2 s from the local registry, 0.1 s once cached.
- **Moved:** the platform's internet dependency, from every node's boot to one bake.
- **Found:** every image reference names an address that belongs to a Kubernetes node, not the registry; pulls work through a per-node translation file that the UI's deploy path never refreshes.
- **Frozen inside the images:** a Kubernetes version, a ghost Node, fixed certificates, and — in the tag the standby uses — a build date of 3 March.
- **Still owed:** a recorded bake date and contents for each image; checks that verify what an image contains, not just that it exists; a registry reference that is the registry's own address.

## Ask your team

1. **When was the image behind our newest system built, and what is inside it?**
2. **Which of our systems only work because something else quietly translates a name or an address for them?**
3. **When a new machine boots, what does it download, and from whom?**

## Open the repo

- `bake-golden-image.sh` and `bake-common.sh` — the bake; the in-VM recipe is the sequence of `[bake] Step` log lines.
- `build-containerdisk.sh` — from baked volume to container image.
- `scripts/import-base-images.sh` — the base Ubuntu images the bake starts from.
- `scripts/fix-registry-hosts.sh` and `make registry-fix` — the alias translation, rewritten on demand.
- `ui/backend/handlers/registry.go` — `REGISTRY_URL` versus `REGISTRY_ALIAS`.
- `docker network inspect kind` — who actually holds `172.18.0.2` today.

# Figures

Draft figures for *Own Your Cloud*. **These are drafts, meant to be replaced.** They exist so that the manuscript can be read with its figures in place and so that a designer has an unambiguous specification of what each one must show.

## Status

Five figures are drawn. The remaining twenty-five are specified below but not drawn — the chapters' draft notes name them, and this file is the brief.

| Figure | Chapter | File | Status |
|---|---|---|---|
| 5.2 | 5 | `fig-05-2-six-layers.svg` | **drawn** |
| 7.1 | 7 | `fig-07-1-boot-timeline.svg` | **drawn** |
| 8.1 | 8 | `fig-08-1-median-and-spread.svg` | **drawn** |
| 11.3 | 11 | `fig-11-3-two-identical-screens.svg` | **drawn** |
| 12.2 | 12 | `fig-12-2-phantom-topology.svg` | **drawn** |
| all others | — | — | specified below, not drawn |

## House rules for every figure

These apply to the drawn drafts and to anything that replaces them.

1. **Self-contained SVG.** No external fonts, no external images, no scripts. Generic font stacks only (`'Source Serif 4', Georgia, serif` for prose, `'IBM Plex Mono', Menlo, Consolas, monospace` for anything quoted from a terminal or a file).
2. **Theme-aware.** Define the light palette on `:root, svg`, then redefine the same custom properties under `@media (prefers-color-scheme: dark)`. Never give a colour its only definition inside a media query. The pages render in both themes.
3. **Accessible.** Every figure carries `role="img"`, a `<title>` and a `<desc>`, with the `<desc>` stating every number in the figure so the figure is readable without seeing it.
4. **No number that is not in the book.** Every value in a figure must appear in the chapter it belongs to, measured and dated. If a figure needs a number the book does not have, the figure is wrong, not the book.
5. **Honest encoding.** Phantom or unverified things are drawn dashed and in the flag colour; measured things are solid. A figure must not make an uncertain thing look certain.

### Palette

| Token | Light | Dark | Use |
|---|---|---|---|
| `--ink` | `#1B2430` | `#E6E8E3` | primary text |
| `--ink2` | `#4A5563` | `#A9B2BA` | labels |
| `--ink3` | `#737C86` | `#7E8892` | captions, axis text |
| `--rule` | `#D6DAD3` | `#2A3642` | hairlines, grids, box strokes |
| `--accent` | `#1F5F4A` | `#6FB79A` | measured data, real things |
| `--accent-soft` | `#E3EDE7` | `#17302A` | fills behind accent |
| `--flag` | `#A8442A` | `#E08A6B` | findings, phantoms, things that are wrong |
| `--panel` | `#EAEEE8` | `#1B2732` | terminal and code panels |

These match `tools/build_chapter.py`'s CSS, so figures sit in the page rather than on it.

## The drawn five

Each entry gives the data source, so a designer can rebuild the figure faithfully rather than tracing it.

### 5.2 — Six layers
**Shows:** six nested boxes — workstation → container (Kind node) → pod (virt-launcher) → virtual machine (KubeVirt/QEMU, a target-cluster node) → pod → container (the workload).
**Data source:** Chapter 5 ledger, "Nesting: six layers"; launcher-pod reservation `8484Mi` measured on cluster2, 18 September 2026; host `24` threads / `29.96 GiB` from the prologue.

### 7.1 — Where fifty seconds went
**Shows:** the median 50.0 s build as a stacked timeline of six phases, with a bracket over the first two marking 16.6 s / 36% elapsing before the guest kernel starts.
**Data source:** Chapter 7's phase table (median of three clean runs, 9 September 2026): 6.6 / 10.0 / 3.7 / 5.5 / 7.0 / 11.8 s; runs 52.7 / 50.0 / 46.6.

### 8.1 — The median falls and the spread collapses
**Shows:** a dot plot of three runs at each of three stages, with the median as a bar and the spread as a thin line.
**Data source:** Chapter 8. Baseline 52.7 / 50.0 / 46.6 (median 50.0, spread 6.1); after `advertise-address` 42.7 / 42.8 / 41.2 (median 42.7, spread 1.6); after `supportContainerResources` 34.5 / 34.2 / 34.7 (median 34.5, spread 0.5).

### 11.3 — Two identical screens
**Shows:** two terminal panels, one during working failover and one during a total outage, printing the identical line `success=20  failed=0`; beneath, the shell line that causes it.
**Data source:** Chapter 11; the counter quoted verbatim in Appendix G.4 from `07-istio-advanced/act3-multicluster/run.sh`.

### 12.2 — The phantom topology
**Shows:** twenty-one nodes under a `LIVE` badge, seven solid and fourteen dashed, with the two causes annotated.
**Data source:** Chapter 12; `ui/backend/handlers/mesh.go`, read 17 September 2026. Exact-name matching (`target-cluster-cp` vs `target-cluster-cp-s9j8h`) and an empty list from a namespace that no longer exists.

## Specified but not drawn

From the chapters' draft notes. Each line is the brief.

**Chapter 4** — 4.1: management cluster → declaration → workload cluster on VMs. 4.2: the deck's spectrum, relabelled neutrally. 4.3: the two gauges side by side, `Available=False` here against `Ready` in Chapter 5.

**Chapter 5** — 5.1: the ownership chain, with the missing link drawn dashed.

**Chapter 6** — 6.1: the pipeline, base image → bake VM → baked volume → qcow2 → container image → registry → node cache → VM. 6.2: the alias — three containers on one network, and the translation file that makes one address mean another.

**Chapter 7** — 7.2: the ghost — a node list at the moment the control plane turns `Ready`, with `ubuntu-bake-vm-warm` sitting in it.

**Chapter 8** — 8.2: reachability sketch — node → control-plane VM pod IP (`200`, 0.3 ms); worker VM → control-plane VM pod IP (timeout); worker VM → VIP `172.18.255.215` (reachable).

**Chapter 9** — 9.1: the standby lifecycle, `(none) → WARM → CLAIMED` with the background rebuild arrow. 9.2: the two inventories as a diagram — shelves one item deep.

**Chapter 10** — 10.1: the mind map as drawn, with the masquerade concept annotated as removed. 10.2: the path as measured (bridge binding, no NAT hop). 10.3: the two ztunnel log lines side by side, identity fields highlighted.

**Chapter 11** — 11.1: the Chapter 10 plumbing next to the one label. 11.2: the two refusal status lines. 11.4: the 60-call distribution (three cluster1 pods at 10 each; all 30 remote calls on one cluster2 pod).

**Chapter 12** — 12.1: two model paths — agents → cluster2 shim → host (used), and the AI gateway on cluster1 (unused). 12.3: Kiali's empty `ai-gateway` graph beside `demo-apps`.

**Chapter 13** — 13.1: the object map — `Agent` → `AgentRun` → pod (`agent`, `ipc-bridge`, skill sidecar) → per-run identity and permissions → model endpoint. 13.2: the three UI modes as windows onto one set of objects.

**Chapter 14** — 14.1: the seven-layer ladder (prompt, document, script, admission, permissions, network, absence) with each of the eight rules placed where it actually lives. 14.2: the probe results as a two-column reachability diagram.

**Chapter 15** — 15.1: the two node lists beside the probe pod's output. 15.2: the ladder of evidence. 15.3: stills of scenes 5 and 6 side by side, extractable from `docs/demo/mesh-sre-agent-demo.mp4` with `ffmpeg`.

**Chapter 16** — 16.1: the catalog as 29 marks across the three planes, with three lit. 16.2: the two-agent pipeline with its token counts, and the analyst's wrong claim passing through the reviewer unchanged.

**Chapter 17** (new) — no figures specified. The Transfers / Does not transfer table may be worth setting as one.

## Embedding

Figures are referenced from the manuscript as standard Markdown images with a caption line beneath:

```markdown
![Six nested boxes: workstation, container, pod, virtual machine, pod, container.](../figures/fig-05-2-six-layers.svg)

*Figure 5.2 — Six layers: a pod inside a VM inside a pod inside a container.*
```

`tools/build_chapter.py` rewrites the relative path for the rendered HTML, wraps the image and its caption in a `<figure>`, and lets the SVG size itself to the column.

# Sympozium chart upgrade notes

The chart version `install-sympozium.sh` pins (`SYMPOZIUM_CHART_VERSION`) is the one
these manifests and labs are verified against. History of that pin:
0.10.38 -> 0.10.47 (2026-08-21) -> 0.10.57 (2026-09-09) -> 0.10.75 (2026-09-16) ->
0.10.87 (2026-09-30) -> 0.12.0 (0.12.2 was pinned in 5a09eb9 but never installed; pin reverted 2026-11-27).

The 0.10.38 pin's original rationale (newer charts dropped the `SympoziumInstance` CRD)
no longer applies: the `SympoziumInstance` objects were removed from
`cluster2-agent.yaml` and `target-cluster-agent.yaml`, which now carry only `Agent` CRs.

## How to upgrade

`install-sympozium.sh` does **not** upgrade in place. Helm never upgrades CRDs in
`crds/`, and the controller mutates its own built-in SkillPacks after install
(`spec.sidecar.mountWorkspace`), so a plain `helm upgrade` conflicts. Apply the CRDs
first, then upgrade server-side, **without `--wait`** (see 0.10.47 -> 0.10.57, item 2):

```bash
helm template sympozium-crds sympozium/sympozium-crds \
  --version <VERSION> | kubectl apply --server-side --force-conflicts -f -
helm upgrade sympozium sympozium/sympozium --version <VERSION> \
  -n sympozium-system -f 06-sympozium/values.yaml \
  --server-side=true --force-conflicts --timeout 12m
```

Note `--server-side=true` (with a value): Helm 4 made `--server-side` take an argument,
so a bare `--server-side --force-conflicts` fails with
`invalid/unknown release server-side apply method: --force-conflicts`.

### After ANY chart upgrade

1. `fix-web-proxy-image.sh`: the image tag reverts to the mutable `:latest` on every
   upgrade (Helm re-applies the chart's own SkillPack default).
2. `fix-missing-builtin-skillpacks.sh`: checks for the admission-webhook race; harmless
   no-op if nothing is missing.
3. `sympozium-lb-setup.sh`: the AgentRun controller recreates Services as ClusterIP when
   it regenerates the serving Deployments.
4. Re-check the per-run ServiceAccount identity (see 0.10.57 -> 0.10.75).
5. `make sympozium-verify`.

## 0.10.47 -> 0.10.57

Manual upgrade, not this script. Three things broke on the way, all worth knowing
before touching this again:

1. **New chart-templated `sympozium-agent` ServiceAccount collided** with an existing one
   the controller had created out-of-band (managed-by: sympozium, no Helm ownership
   annotations). `helm upgrade` refuses to adopt it. Fix (idempotent, apply before
   upgrading if this is still the installed base):

   ```bash
   kubectl label sa sympozium-agent -n sympozium-system \
     app.kubernetes.io/managed-by=Helm --overwrite
   kubectl annotate sa sympozium-agent -n sympozium-system \
     meta.helm.sh/release-name=sympozium \
     meta.helm.sh/release-namespace=sympozium-system --overwrite
   ```

2. **`--wait` polls EVERY MCPServer's custom status** for readiness, including ones
   deliberately left Suspended (e.g. `postgres`, an unconfigured demo example:
   `Ready:false` is its correct steady state, not a fault). Any `--wait` upgrade on this
   chart times out on it regardless of version. Drop `--wait` and verify readiness
   manually instead (Deployments + Agent phase + a real chat completion, not the helm
   exit code).

3. **0.10.57 defaults `nats.auth.enabled: true`** and wires `NATS_USERNAME`/`PASSWORD`
   into the controller and llmfit-daemonset, but NOT into the built-in web-endpoint
   SkillPack's sidecar env (still bare `EVENT_BUS_URL`, no credentials). Every served
   agent's web-proxy then dials NATS unauthenticated, gets rejected ("authentication
   error" in the nats pod's own log), and crash-loops with exit 2 and zero log output:
   same signature as the `:latest` tag-skew landmine, different cause.
   `values.yaml` sets `nats.auth.enabled: false` to restore the known-working behavior;
   see the comment there for why that's an acceptable trade-off on this cluster and when
   to revisit it.

## 0.10.57 -> 0.10.75

Clean helm upgrade (SA already Helm-adopted from the last upgrade; no new collisions).
CRDs/RBAC diff against 0.10.57 is additive only (new Celln-fleet + model-gateway CRDs,
both `enabled: false` by default and untouched by `values.yaml`) plus a real bug fix
(generated NATS passwords could start with a digit and break `nats.conf` parsing).
Nothing here required a `values.yaml` change.

One real breaking change, found by testing, not by the diff (it's a controller behavior
change, not a template one): **AgentRun task pods now run as a per-run ServiceAccount**
(`sympozium-run-<agentrun-name>`, bound to a matching per-run Role/RoleBinding the
controller generates and deletes with the run) instead of the shared `sympozium-agent`
SA every skill sidecar used to run as. That silently broke `agent-istio-rbac.yaml` and
`agent-kubevirt-rbac.yaml`, which bound only the literal `sympozium-agent` subject:
mesh-sre-agent's live tool calls degraded to "permission issue" confabulation while
`kubectl get gateways --as=...sympozium-agent` still looked fine (wrong identity to
check). Fixed by rebinding both ClusterRoleBindings to the
`system:serviceaccounts:sympozium-system` Group instead of one SA name, so every present
and future per-run SA inherits the grant automatically. Verified with
`--as=system:serviceaccount:sympozium-system:sympozium-run-<real-run-name>` (not the old
`sympozium-agent` name) and a live chat completion whose answer matched real cluster
state. Re-run this same check after any future upgrade: the per-run identity model is
new enough here that it could change again.

## 0.10.75 -> 0.10.87

CRDs server-side, then helm upgrade without `--wait`. Values diff is additive only
(Celln mediation/fleet settings, all off by default); nothing in `values.yaml` changed.
Two breakages, both in the post-upgrade step, both now handled by
`fix-web-proxy-image.sh`:

1. Recreated serving Deployments come back with `readOnlyRootFilesystem: true`, so every
   web-proxy crash-loops (exit 2, no logs). mesh-sre-agent was never in any
   `fix-web-proxy-rootfs.sh` caller list and had been crash-looping since 0.10.75 for
   this reason. (`install-sympozium.sh` now patches it too.)
2. cluster2-agent's recreated serving AgentRun raced its Deployment and went Failed
   ("server Deployment not found") with no retry; deleting it once more fixes it.

Verified: istio-adv-verify 22/22, all three agents Serving with 0 restarts, live chat
completions with real kubectl tool calls through all three.

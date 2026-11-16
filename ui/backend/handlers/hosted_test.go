package handlers

import "testing"

func TestCountReadyTargetNodes(t *testing.T) {
	cases := []struct {
		name string
		out  string
		want int
	}{
		{"hosted: two Ready agents", "target-cluster-workers-ab12c-x1=True\ntarget-cluster-workers-ab12c-x2=True\n", 2},
		{"hosted: one still NotReady", "target-cluster-workers-ab12c-x1=True\ntarget-cluster-workers-ab12c-x2=False\n", 1},
		{"vm: control plane + worker", "target-cluster-cp-abcde=True\ntarget-cluster-workers-ab12c-x1=True\n", 2},
		{"ghost bake node never counts", "ubuntu-bake-vm-warm=True\ntarget-cluster-cp-abcde=True\n", 1},
		{"stale node of another cluster", "other-cluster-workers-x=True\n", 0},
		{"empty / unknown status", "\n  \ntarget-cluster-workers-x=Unknown\nmalformed-line\n", 0},
	}
	for _, c := range cases {
		if got := countReadyTargetNodes(c.out, "target-cluster"); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestImageAllowed(t *testing.T) {
	cases := []struct {
		mode, image string
		want        bool
	}{
		{cpModeHosted, hostedImageKey, true},
		{cpModeHosted, "warm", false}, // would claim the VIP the hosted server owns
		{cpModeHosted, "noble", false},
		{cpModeVM, "warm", true},
		{cpModeVM, "minimal", true},
		{cpModeVM, hostedImageKey, false}, // no hosted server to join
	}
	for _, c := range cases {
		if got := imageAllowed(c.mode, c.image); got != c.want {
			t.Errorf("imageAllowed(%q, %q) = %v, want %v", c.mode, c.image, got, c.want)
		}
	}
}

func TestCPModeDefaults(t *testing.T) {
	t.Setenv("TARGET_CP_MODE", "")
	if cpMode() != cpModeHosted || defaultImageKey() != hostedImageKey ||
		defaultStandbyManifest() != "03-target-cluster/target-cluster-hosted.yaml" {
		t.Errorf("unset TARGET_CP_MODE: want hosted defaults, got mode=%s image=%s manifest=%s",
			cpMode(), defaultImageKey(), defaultStandbyManifest())
	}
	t.Setenv("TARGET_CP_MODE", "vm")
	if cpMode() != cpModeVM || defaultImageKey() != warmImageKey ||
		defaultStandbyManifest() != "03-target-cluster/target-cluster-warm.yaml" {
		t.Errorf("TARGET_CP_MODE=vm: want VM defaults, got mode=%s image=%s manifest=%s",
			cpMode(), defaultImageKey(), defaultStandbyManifest())
	}
	if _, ok := imageVariants[hostedImageKey]; !ok {
		t.Error("imageVariants has no hosted entry")
	}
}

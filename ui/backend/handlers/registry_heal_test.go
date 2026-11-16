package handlers

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func pullPod(ns, name, image, reason, owner string) corev1.Pod {
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
			Image: image,
			State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason}},
		}}},
	}
	if owner != "" {
		p.OwnerReferences = []metav1.OwnerReference{{Kind: owner, Name: "x"}}
	}
	return p
}

func TestStuckOnRegistryAlias(t *testing.T) {
	const alias = "172.18.0.2:5000"
	pods := []corev1.Pod{
		pullPod("capk-system", "capk", alias+"/capk-manager:v0.11.2-kubeui2", "ImagePullBackOff", "ReplicaSet"),
		pullPod("default", "launcher", alias+"/ubuntu-noble-k3s:agent", "ErrImagePull", "VirtualMachineInstance"),
		pullPod("kube-system", "other", "quay.io/foo:1", "ImagePullBackOff", "ReplicaSet"),
		pullPod("kube-system", "crash", alias+"/bar:1", "CrashLoopBackOff", "ReplicaSet"),
		pullPod("kube-system", "prefix", "172.18.0.2:50001/bar:1", "ImagePullBackOff", "ReplicaSet"),
	}
	stuck := stuckOnRegistryAlias(pods, alias)
	if len(stuck) != 2 || stuck[0].Name != "capk" || stuck[1].Name != "launcher" {
		t.Fatalf("stuck = %+v", stuck)
	}
	if !stuck[0].restartable || stuck[1].restartable {
		t.Fatalf("only ReplicaSet-owned pods are restartable: %+v", stuck)
	}
}

func TestStuckOnRegistryAliasInitContainer(t *testing.T) {
	p := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "b"}}
	p.Status.InitContainerStatuses = []corev1.ContainerStatus{{
		Image: "172.18.0.2:5000/x:1",
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}},
	}}
	if got := stuckOnRegistryAlias([]corev1.Pod{p}, "172.18.0.2:5000"); len(got) != 1 {
		t.Fatalf("init container pull failure missed: %+v", got)
	}
}

func TestRegistryHealCooldown(t *testing.T) {
	var h registryHealer
	now := time.Now()
	if !h.mayRun(now) {
		t.Fatal("first run refused")
	}
	if h.mayRun(now.Add(time.Minute)) {
		t.Fatal("ran again inside the cooldown")
	}
	if !h.mayRun(now.Add(registryHealCooldown + time.Second)) {
		t.Fatal("refused after the cooldown")
	}
}

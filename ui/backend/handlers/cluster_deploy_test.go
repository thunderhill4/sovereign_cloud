package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFinishDoneLogsElapsedTime(t *testing.T) {
	d := &deployManager{state: "idle"}
	d.resetForNewRun()
	time.Sleep(10 * time.Millisecond)
	d.finish("done")

	if len(d.logs) == 0 {
		t.Fatalf("expected at least one log entry, got none")
	}
	last := d.logs[len(d.logs)-1]
	if !strings.Contains(last.Message, "Total time") {
		t.Errorf("expected final log to report elapsed time, got %q", last.Message)
	}
}

func TestFinishFailedDoesNotLogElapsedTime(t *testing.T) {
	d := &deployManager{state: "idle"}
	d.resetForNewRun()
	d.addLog("error", "boom")
	d.finish("failed")

	last := d.logs[len(d.logs)-1]
	if strings.Contains(last.Message, "Total time") {
		t.Errorf("did not expect elapsed time log on failure, got %q", last.Message)
	}
}

func TestStderrTailKeepsLastErrorLine(t *testing.T) {
	var w stderrTail
	w.Write([]byte("Warning: cluster.x-k8s.io/v1beta1 Cluster is deprecated\n"))
	w.Write([]byte("Error from server (InternalError): failed calling webhook \"x\": context deadline exceeded\n"))
	w.Write([]byte("Warning: trailing deprecation notice\n\n"))
	if got := w.last(); got != `Error from server (InternalError): failed calling webhook "x": context deadline exceeded` {
		t.Fatalf("last() = %q", got)
	}
}

func TestRunShellErrorCarriesStderr(t *testing.T) {
	err := runShell("echo 'Warning: noise' >&2; echo 'Error: the real cause' >&2; echo 'Warning: more noise' >&2; exit 1")
	if err == nil || err.Error() != "exit status 1: Error: the real cause" {
		t.Fatalf("err = %v", err)
	}
	if err := runShell("echo 'Warning: only noise' >&2; exit 3"); err == nil || err.Error() != "exit status 3: Warning: only noise" {
		t.Fatalf("all-warning stderr: err = %v", err)
	}
	if err := runShell("echo fine >&2"); err != nil {
		t.Fatalf("success: err = %v", err)
	}
}

func TestPickBackendKubeconfigNeverUsesFrontDoor(t *testing.T) {
	dir := t.TempDir()
	direct := filepath.Join(dir, "direct")
	published := filepath.Join(dir, "published")
	write := func(p, server string) {
		if err := os.WriteFile(p, []byte("clusters:\n- cluster:\n    server: https://"+server+":6443\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// After an idle delete: /tmp copy gone, repo copy points at the front door.
	write(published, targetFrontDoorIP)
	if got := pickBackendKubeconfig(direct, published); got != "" {
		t.Fatalf("front-door kubeconfig used for a backend probe: %q", got)
	}
	// Standby mode: the repo copy is direct, so it is a valid fallback.
	write(published, targetLBIP)
	if got := pickBackendKubeconfig(direct, published); got != published {
		t.Fatalf("direct repo copy not used: %q", got)
	}
	// The /tmp copy always wins.
	write(direct, targetLBIP)
	if got := pickBackendKubeconfig(direct, published); got != direct {
		t.Fatalf("direct copy not preferred: %q", got)
	}
}

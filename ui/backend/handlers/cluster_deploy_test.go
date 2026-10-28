package handlers

import (
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

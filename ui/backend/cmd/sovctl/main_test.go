package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestReadSSE(t *testing.T) {
	cases := map[string]struct {
		in   string
		want []string
	}{
		"deploy stream: keepalives, ends at close": {
			": keepalive\n\ndata: {\"type\":\"step\",\"message\":\"a\"}\n\ndata: {\"type\":\"done\",\"message\":\"done\"}\n\n",
			[]string{`{"type":"step","message":"a"}`, `{"type":"done","message":"done"}`},
		},
		"ai stream: stops at [DONE]": {
			"data: {\"type\":\"text\",\"text\":\"hi\"}\n\ndata: [DONE]\n\ndata: {\"type\":\"text\",\"text\":\"after\"}\n\n",
			[]string{`{"type":"text","text":"hi"}`},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var got []string
			if err := readSSE(strings.NewReader(c.in), func(d []byte) error { got = append(got, string(d)); return nil }); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestParseTTL(t *testing.T) {
	ok := map[string]int{"15m": 900, "1h30m": 5400, "90": 5400, "never": 0, "off": 0, "24h": 86400, "1m": 60}
	for in, want := range ok {
		if got, err := parseTTL(in); err != nil || got != want {
			t.Errorf("parseTTL(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"30s", "25h", "abc", "-5m", "1.5"} {
		if _, err := parseTTL(in); err == nil {
			t.Errorf("parseTTL(%q) should fail", in)
		}
	}
}

func TestExtractGlobals(t *testing.T) {
	defer func() { serverURL, outputMode, noColor = "http://localhost:8080", "table", false }()
	rest, err := extractGlobals([]string{"pods", "-o", "json", "-n", "kube-system", "--server=http://x:1/", "--no-color"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rest, []string{"pods", "-n", "kube-system"}) || outputMode != "json" || serverURL != "http://x:1" || !noColor {
		t.Fatalf("rest=%q output=%s server=%s noColor=%v", rest, outputMode, serverURL, noColor)
	}
	if _, err := extractGlobals([]string{"-o", "yaml"}); err == nil {
		t.Fatal("-o yaml should be rejected")
	}
}

func TestParseFlagsInterspersed(t *testing.T) {
	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	yes := fs.Bool("y", false, "")
	pos, err := parseFlags(fs, []string{"15m", "-y"})
	if err != nil || !*yes || !reflect.DeepEqual(pos, []string{"15m"}) {
		t.Fatalf("pos=%q yes=%v err=%v", pos, *yes, err)
	}
}

func TestTargetState(t *testing.T) {
	useColor = false
	cases := []struct {
		p    poolStatus
		want string
	}{
		{poolStatus{Mode: "ondemand", State: "none"}, "Asleep"},
		{poolStatus{Mode: "ondemand", State: "none", Stopped: true}, "Stopped"},
		{poolStatus{Mode: "ondemand", State: "building"}, "Waking…"},
		{poolStatus{Mode: "ondemand", State: "claimed", ClusterReady: true, IdleTTLSeconds: 900}, "Up"},
		{poolStatus{Mode: "ondemand", State: "warm", ClusterReady: true, Deleting: true, DeletingSeconds: 10}, "Deleting…"},
		{poolStatus{Mode: "ondemand", State: "warm", Deleting: true, DeletingSeconds: 600}, "Deleting (stuck)"},
		{poolStatus{Mode: "standby", State: "warm", ClusterReady: true}, "Standby ready"},
	}
	for _, c := range cases {
		if got, _ := targetState(&c.p); got != c.want {
			t.Errorf("%+v -> %q, want %q", c.p, got, c.want)
		}
	}
}

// captureStdout runs f and returns what it printed.
func captureStdout(t *testing.T, f func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := f()
	w.Close()
	os.Stdout = old
	b, _ := io.ReadAll(r)
	return string(b), err
}

func stubBackend(t *testing.T, h http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	old := serverURL
	serverURL = srv.URL
	t.Cleanup(func() { serverURL = old })
	useColor = false
}

func TestTimerSetSendsSeconds(t *testing.T) {
	var gotBody map[string]int
	stubBackend(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/cluster/pool-status":
			fmt.Fprint(w, `{"state":"none","mode":"ondemand","idleTTLSeconds":3600,"idleTTLSource":"env"}`)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/cluster/pool-settings":
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			fmt.Fprintf(w, `{"state":"none","mode":"ondemand","idleTTLSeconds":%d,"idleTTLSource":"ui"}`, gotBody["idleTTLSeconds"])
		default:
			http.NotFound(w, r)
		}
	})
	set := timerCmd.find("set")
	out, err := captureStdout(t, func() error { return set.run([]string{"15m"}) })
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["idleTTLSeconds"] != 900 || !strings.Contains(out, "15m") || !strings.Contains(out, "set from UI/CLI") {
		t.Fatalf("body=%v out=%q", gotBody, out)
	}
}

func TestDeployFollowFailsOnFailedRun(t *testing.T) {
	var gotQuery string
	stubBackend(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"step\",\"message\":\"Deploying\"}\n\n: keepalive\n\ndata: {\"type\":\"error\",\"message\":\"boom\"}\n\ndata: {\"type\":\"done\",\"message\":\"failed\"}\n\n")
	})
	deploy := clusterCmd.find("deploy")
	out, err := captureStdout(t, func() error { return deploy.run([]string{"--profile", "lite", "--image", "warm"}) })
	if err == nil {
		t.Fatal("a failed run must return an error")
	}
	if gotQuery != "image=warm&profile=lite" || !strings.Contains(out, "Deploying") || !strings.Contains(out, "✗ boom") {
		t.Fatalf("query=%q out=%q", gotQuery, out)
	}
}

func TestChatProposalAndErrors(t *testing.T) {
	var actionBody map[string]string
	stubBackend(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ai/chat":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["agent"] != "target-cluster-agent" {
				t.Errorf("agent not sent: %v", body)
			}
			fmt.Fprint(w, "data: {\"type\":\"text\",\"text\":\"Sure.\"}\n\ndata: {\"type\":\"proposal\",\"proposal\":{\"action\":\"deploy_target_cluster\",\"profile\":\"full\"}}\n\ndata: [DONE]\n\n")
		case "/api/ai/action":
			_ = json.NewDecoder(r.Body).Decode(&actionBody)
			fmt.Fprint(w, "data: {\"type\":\"tool_step\",\"step\":{\"type\":\"done\",\"message\":\"done\"}}\n\ndata: [DONE]\n\n")
		case "/api/v1/registry/images/x:1":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error":"image not found"}`)
		case "/api/v1/cross-cluster/scale":
			http.Error(w, "replicas out of range", http.StatusBadRequest)
		}
	})
	chat := aiCmd.find("chat")
	out, err := captureStdout(t, func() error { return chat.run([]string{"--agent", "target-cluster-agent", "--yes", "deploy", "it"}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Sure.") || actionBody["action"] != "deploy_target_cluster" || actionBody["profile"] != "full" {
		t.Fatalf("out=%q action=%v", out, actionBody)
	}

	// Both backend error formats surface their message.
	if _, err := call(http.MethodDelete, "/api/v1/registry/images/x:1", nil); err == nil || !strings.Contains(err.Error(), "image not found") {
		t.Fatalf("json error = %v", err)
	}
	if _, err := call(http.MethodPost, "/api/v1/cross-cluster/scale", map[string]int{}); err == nil || !strings.Contains(err.Error(), "replicas out of range") {
		t.Fatalf("text error = %v", err)
	}
}

func TestEveryCommandHasSummary(t *testing.T) {
	var walk func(c *command, path string)
	walk = func(c *command, path string) {
		if c.summary == "" {
			t.Errorf("%s has no summary", path)
		}
		if c.run == nil && len(c.sub) == 0 {
			t.Errorf("%s does nothing", path)
		}
		for _, s := range c.sub {
			walk(s, path+" "+s.name)
		}
	}
	walk(root, "sovctl")
}

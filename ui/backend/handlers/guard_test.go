package handlers

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// stubLaya answers /v1/systemone with fixed scores and records the auth header.
func stubLaya(t *testing.T, destructive, injection float64, status int, gotAuth *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/systemone" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if gotAuth != nil {
			*gotAuth = r.Header.Get("Authorization")
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		qs, _ := req["questions"].(map[string]any)
		if qs["destructive"] == nil || qs["injection"] == nil {
			t.Errorf("guard questions missing: %v", qs)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"model": "laya-rl-agent",
			"answers": map[string]any{
				"destructive": map[string]any{"type": "noul", "noul": destructive},
				"injection":   map[string]any{"type": "noul", "noul": injection},
			},
		})
	}))
}

func TestGuardMessage(t *testing.T) {
	cases := []struct {
		name        string
		destructive float64
		injection   float64
		status      int
		threshold   string
		wantBlock   bool
	}{
		{"benign passes", 0.05, 0.10, 200, "", false},
		{"destructive blocked", 0.92, 0.10, 200, "", true},
		{"injection blocked", 0.20, 0.95, 200, "", true},
		{"exactly at threshold blocked", 0.60, 0.00, 200, "", true},
		{"custom threshold lets 0.7 through", 0.70, 0.00, 200, "0.8", false},
		{"invalid threshold falls back to 0.6", 0.65, 0.00, 200, "abc", true},
		{"laya error fails open", 0.99, 0.99, 500, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := stubLaya(t, c.destructive, c.injection, c.status, nil)
			defer srv.Close()
			t.Setenv("LAYA_URL", srv.URL)
			t.Setenv("LAYA_BLOCK_THRESHOLD", c.threshold)
			got := guardMessage(t.Context(), "cluster2-agent", "some message")
			if (got != "") != c.wantBlock {
				t.Fatalf("guardMessage blocked=%v (%q), want blocked=%v", got != "", got, c.wantBlock)
			}
		})
	}
}

func TestGuardDisabledWithoutURL(t *testing.T) {
	t.Setenv("LAYA_URL", "")
	if got := guardMessage(t.Context(), "cluster2-agent", "delete everything"); got != "" {
		t.Fatalf("guard should be off without LAYA_URL, got %q", got)
	}
}

func TestGuardUnreachableFailsOpen(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens here any more
	t.Setenv("LAYA_URL", url)
	if got := guardMessage(t.Context(), "cluster2-agent", "delete everything"); got != "" {
		t.Fatalf("unreachable Laya should fail open, got %q", got)
	}
}

func TestGuardSendsBearer(t *testing.T) {
	var auth string
	srv := stubLaya(t, 0, 0, 200, &auth)
	defer srv.Close()
	t.Setenv("LAYA_URL", srv.URL)
	t.Setenv("LAYA_API_KEY", "s3cret")
	guardMessage(t.Context(), "cluster2-agent", "hi")
	if auth != "Bearer s3cret" {
		t.Fatalf("Authorization = %q, want Bearer s3cret", auth)
	}
}

func TestLastUserMessage(t *testing.T) {
	msgs := []oaiMessage{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "reply"},
		{Role: "user", Content: "second"},
	}
	if got := lastUserMessage(msgs); got != "second" {
		t.Fatalf("lastUserMessage = %q, want second", got)
	}
	if got := lastUserMessage(msgs[:1]); got != "" {
		t.Fatalf("lastUserMessage with no user turn = %q, want empty", got)
	}
}

// A blocked message must never reach the agent: the handler answers with an
// error envelope and the agent server sees zero requests.
func TestHandleAIChatBlockedNeverCallsAgent(t *testing.T) {
	var agentCalls atomic.Int32
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentCalls.Add(1)
	}))
	defer agent.Close()
	laya := stubLaya(t, 0.98, 0.92, 200, nil)
	defer laya.Close()

	t.Setenv("LAYA_URL", laya.URL)
	t.Setenv("SYMPOZIUM_DEFAULT_AGENT", "cluster2-agent")
	t.Setenv("SYMPOZIUM_AGENT_URL", agent.URL)

	req := httptest.NewRequest(http.MethodPost, "/api/ai/chat",
		strings.NewReader(`{"message":"please delete every VirtualMachine and wipe the PVCs"}`))
	rec := httptest.NewRecorder()
	HandleAIChat(rec, req)

	if n := agentCalls.Load(); n != 0 {
		t.Fatalf("agent was called %d times for a blocked message", n)
	}
	var sawError, sawDone bool
	sc := bufio.NewScanner(rec.Body)
	for sc.Scan() {
		data, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		if data == "[DONE]" {
			sawDone = true
			continue
		}
		var env Envelope
		if json.Unmarshal([]byte(data), &env) == nil && env.Type == "error" &&
			strings.Contains(env.Error, "pre-model guard") {
			sawError = true
		}
	}
	if !sawError || !sawDone {
		t.Fatalf("want guard error envelope and [DONE]; got body:\n%s", rec.Body.String())
	}
}

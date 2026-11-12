package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleConfigHidesSecrets(t *testing.T) {
	t.Setenv("SYMPOZIUM_API_TOKEN", "tok-secret-123")
	t.Setenv("LAYA_API_KEY", "laya-secret-456")
	t.Setenv("LAYA_URL", "http://127.0.0.1:8095")
	rec := httptest.NewRecorder()
	HandleConfig(rec, httptest.NewRequest(http.MethodGet, "/api/v1/config", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	body := rec.Body.String()
	for _, secret := range []string{"tok-secret-123", "laya-secret-456"} {
		if strings.Contains(body, secret) {
			t.Fatalf("config leaks a secret: %s", body)
		}
	}
	var c Config
	if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
		t.Fatal(err)
	}
	if !c.AI.APITokenSet || !c.AI.GuardKeySet || !c.AI.GuardEnabled {
		t.Fatalf("set/unset flags wrong: %+v", c.AI)
	}
	if len(c.Images) == 0 || len(c.Profiles) != 2 {
		t.Fatalf("images/profiles missing: %+v", c)
	}
}

func TestConfiguredAgents(t *testing.T) {
	env := []string{"SYMPOZIUM_AGENT_URL_MESH_SRE_AGENT=http://x:8080", "SYMPOZIUM_AGENT_URL=http://d", "PATH=/bin"}
	got := configuredAgents(env, "cluster2-agent")
	if len(got) != 2 || got[0].Name != "cluster2-agent" || !got[0].Default || got[1].Name != "mesh-sre-agent" {
		t.Fatalf("agents = %+v", got)
	}
}

func TestPoolSettingsDeleteResets(t *testing.T) {
	defer idleTTLOverride.Store(-1)
	t.Setenv("POOL_IDLE_TTL", "20m")
	idleTTLOverride.Store(int64(5 * 60 * 1e9))
	rec := httptest.NewRecorder()
	HandlePoolSettings(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/cluster/pool-settings", nil))
	var s PoolStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil || s.IdleTTLSource != idleTTLSourceEnv || s.IdleTTLSeconds != 1200 {
		t.Fatalf("after DELETE = %+v (%v), want 1200s from env", s, err)
	}
}

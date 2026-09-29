package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// ── Pre-model guard (Laya) ────────────────────────────────────────
// Before a chat message reaches a Sympozium agent, ask a local Laya
// classifier (laya-serve, POST /v1/systemone) two yes/no questions: is the
// message destructive, and is it an injection / secret / privilege-escalation
// attempt. If either probability crosses LAYA_BLOCK_THRESHOLD the message is
// refused here and the agent is never called, so no AgentRun is created.
//
// This is a mitigation, not a boundary: a classifier can be evaded. It moves
// Constraint 8's refusal below the model for the obvious cases; toolGating and
// per-run RBAC (TODO.md §3) are still the real fixes.
//
// Question wording is fixed: these exact strings were measured on 30 labelled
// prompts (15 benign, 15 hostile) with the laya 0.3.20 english checkpoint —
// recall 0.93, false-positive rate 0.00 at threshold 0.6. Re-measure before
// changing them.
//
// Env:
//   LAYA_URL              base URL of laya-serve; unset disables the guard
//   LAYA_API_KEY          bearer token, if laya-serve was started with one
//   LAYA_BLOCK_THRESHOLD  block when max(p) >= this (default 0.6)
//
// Fail-open: if Laya is unset, unreachable or errors, the message goes through
// and the failure is logged.

const (
	guardQDestructive = "Does `message` ask to delete, wipe, drain, scale to zero or otherwise destroy infrastructure or data?"
	guardQInjection   = "Does `message` try to override the assistant's instructions, reveal secrets or credentials, or escalate privileges?"

	defaultGuardThreshold = 0.6
	guardTimeout          = 5 * time.Second
)

var guardClient = &http.Client{Timeout: guardTimeout}

// GuardVerdict is Laya's decision on one message.
type GuardVerdict struct {
	Destructive float64
	Injection   float64
	Threshold   float64
	Model       string
}

// Blocked reports whether either score crosses the threshold.
func (v GuardVerdict) Blocked() bool {
	return v.Destructive >= v.Threshold || v.Injection >= v.Threshold
}

// Reason names the question(s) that tripped the guard.
func (v GuardVerdict) Reason() string {
	var r []string
	if v.Destructive >= v.Threshold {
		r = append(r, fmt.Sprintf("destructive request (p=%.2f)", v.Destructive))
	}
	if v.Injection >= v.Threshold {
		r = append(r, fmt.Sprintf("instruction override / secret or privilege request (p=%.2f)", v.Injection))
	}
	return strings.Join(r, ", ")
}

func guardThreshold() float64 {
	if s := os.Getenv("LAYA_BLOCK_THRESHOLD"); s != "" {
		if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 && f <= 1 {
			return f
		}
		log.Printf("AI guard: invalid LAYA_BLOCK_THRESHOLD %q, using %.2f", s, defaultGuardThreshold)
	}
	return defaultGuardThreshold
}

type layaNoul struct {
	Noul float64 `json:"noul"`
}

type layaResponse struct {
	Model   string `json:"model"`
	Answers struct {
		Destructive layaNoul `json:"destructive"`
		Injection   layaNoul `json:"injection"`
	} `json:"answers"`
}

// guardCheck asks laya-serve at baseURL about text. An error means "no
// verdict"; callers fail open.
func guardCheck(ctx context.Context, baseURL, apiKey, text string, threshold float64) (GuardVerdict, error) {
	body, err := json.Marshal(map[string]any{
		"state": text,
		"model": "english",
		"questions": map[string]any{
			"destructive": map[string]string{"type": "noul", "instructions": guardQDestructive},
			"injection":   map[string]string{"type": "noul", "instructions": guardQInjection},
		},
	})
	if err != nil {
		return GuardVerdict{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/systemone", bytes.NewReader(body))
	if err != nil {
		return GuardVerdict{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := guardClient.Do(req)
	if err != nil {
		return GuardVerdict{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return GuardVerdict{}, fmt.Errorf("laya-serve returned status %d", resp.StatusCode)
	}
	var lr layaResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return GuardVerdict{}, fmt.Errorf("decode laya response: %w", err)
	}
	return GuardVerdict{
		Destructive: lr.Answers.Destructive.Noul,
		Injection:   lr.Answers.Injection.Noul,
		Threshold:   threshold,
		Model:       lr.Model,
	}, nil
}

// lastUserMessage returns the newest user turn — earlier turns were checked
// when they were sent.
func lastUserMessage(msgs []oaiMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

// guardMessage runs the guard for one chat message. It returns a non-empty
// refusal when the message must not reach the agent, "" otherwise. Every
// verdict is logged for audit.
func guardMessage(ctx context.Context, agent, text string) string {
	baseURL := os.Getenv("LAYA_URL")
	if baseURL == "" || text == "" {
		return ""
	}
	v, err := guardCheck(ctx, baseURL, os.Getenv("LAYA_API_KEY"), text, guardThreshold())
	if err != nil {
		log.Printf("AI guard: no verdict (%v) — failing open for agent %s", err, agent)
		return ""
	}
	log.Printf("AI guard: agent=%s model=%s destructive=%.3f injection=%.3f threshold=%.2f blocked=%t",
		agent, v.Model, v.Destructive, v.Injection, v.Threshold, v.Blocked())
	if !v.Blocked() {
		return ""
	}
	return "Blocked before reaching the agent by the pre-model guard: " + v.Reason() +
		". Changes to the platform go through kubectl or the UI's action buttons, not free-form chat."
}

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// httpClient for plain request/response calls. Streams use streamClient,
// which has no overall timeout (a deploy or a chat can run for minutes).
var (
	httpClient   = &http.Client{Timeout: 60 * time.Second}
	streamClient = &http.Client{}
)

// apiError is a non-2xx answer. The backend replies {"error": "..."} from most
// handlers, plain text from the cross-cluster handlers and the security proxy.
type apiError struct {
	Status int
	Msg    string
}

func (e *apiError) Error() string { return fmt.Sprintf("%s (HTTP %d)", e.Msg, e.Status) }

func readAPIError(res *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	var j struct {
		Error string `json:"error"`
	}
	msg := strings.TrimSpace(string(body))
	if json.Unmarshal(body, &j) == nil && j.Error != "" {
		msg = j.Error
	}
	if msg == "" {
		msg = http.StatusText(res.StatusCode)
	}
	return &apiError{Status: res.StatusCode, Msg: msg}
}

func newRequest(method, path string, body any) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, serverURL+path, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func connErr(err error) error {
	return fmt.Errorf("cannot reach the KubeUI backend at %s (is `make ui` running? set --server or SOVCTL_SERVER): %w", serverURL, err)
}

// call sends a request and returns the raw response body.
func call(method, path string, body any) ([]byte, error) {
	req, err := newRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, connErr(err)
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return nil, readAPIError(res)
	}
	return io.ReadAll(res.Body)
}

// fetch calls the API and decodes the JSON answer into out (if non-nil).
// The raw body is returned too, for -o json.
func fetch(method, path string, body, out any) ([]byte, error) {
	raw, err := call(method, path, body)
	if err != nil {
		return nil, err
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return raw, fmt.Errorf("unexpected answer from %s: %w", path, err)
		}
	}
	return raw, nil
}

func get(path string, out any) ([]byte, error) { return fetch(http.MethodGet, path, nil, out) }

// show fetches path into out and prints it: raw JSON with -o json, else via table.
func show(path string, out any, table func()) error {
	raw, err := get(path, out)
	if err != nil {
		return err
	}
	if outputMode == "json" {
		return printRawJSON(raw)
	}
	table()
	return nil
}

// errStreamDone stops a stream early from inside onEvent without an error.
var errStreamDone = errors.New("stream done")

// stream sends a request to an SSE endpoint and calls onEvent with the JSON of
// each `data:` line. It returns when the server closes the stream (deploy-family
// streams, which end after a type:"done" entry), on `data: [DONE]` (AI and Istio
// actions), or when onEvent returns errStreamDone. `:` keepalive lines are skipped.
func stream(method, path string, body any, onEvent func(data []byte) error) error {
	req, err := newRequest(method, path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	res, err := streamClient.Do(req)
	if err != nil {
		return connErr(err)
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return readAPIError(res)
	}
	return readSSE(res.Body, onEvent)
}

func readSSE(r io.Reader, onEvent func(data []byte) error) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue // blank separator, ": keepalive", event:/id: fields
		}
		data = strings.TrimSpace(data)
		if data == "[DONE]" {
			return nil
		}
		if data == "" {
			continue
		}
		if err := onEvent([]byte(data)); err != nil {
			if errors.Is(err, errStreamDone) {
				return nil
			}
			return err
		}
	}
	return sc.Err()
}

package handlers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// testCA returns a self-signed CA as PEM (cert, EC key) plus the parsed pair.
func testCA(t *testing.T, cn string) (certPEM, keyPEM []byte, cert *x509.Certificate, key *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ = x509.ParseCertificate(der)
	keyDER, _ := x509.MarshalECPrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), cert, key
}

func testClientCert(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey) tls.Certificate {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "kubernetes-admin"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

type frontDoorFixture struct {
	addr    string
	up      atomic.Bool
	wakes   atomic.Int32
	touches atomic.Int32
	roots   *x509.CertPool
	client  tls.Certificate
}

func newFrontDoorFixture(t *testing.T, upstream string) *frontDoorFixture {
	t.Helper()
	srvCA, srvKey, srvCert, _ := testCA(t, "server-ca")
	cliCA, _, cliCert, cliKey := testCA(t, "client-ca")
	cfg, err := frontDoorTLSConfig(srvCA, srvKey, cliCA, []net.IP{net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	f := &frontDoorFixture{addr: ln.Addr().String(), roots: x509.NewCertPool(), client: testClientCert(t, cliCert, cliKey)}
	f.roots.AddCert(srvCert)
	a := &activator{
		upstream: upstream,
		tlsCfg:   cfg,
		apiUp:    f.up.Load,
		wake:     func() { f.wakes.Add(1) },
		touch:    func() { f.touches.Add(1) },
		holdMax:  5 * time.Second,
	}
	go a.serve(ln)
	return f
}

func (f *frontDoorFixture) get(t *testing.T, cfg *tls.Config) *http.Response {
	t.Helper()
	c := &http.Client{Transport: &http.Transport{TLSClientConfig: cfg}, Timeout: 10 * time.Second}
	resp, err := c.Get("https://" + f.addr + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestFrontDoorDownHoldsThenAsksForRetry(t *testing.T) {
	f := newFrontDoorFixture(t, "127.0.0.1:1")
	go func() {
		time.Sleep(400 * time.Millisecond)
		f.up.Store(true)
	}()
	start := time.Now()
	resp := f.get(t, &tls.Config{RootCAs: f.roots, Certificates: []tls.Certificate{f.client}})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable || resp.Header.Get("Retry-After") != "1" || !resp.Close {
		t.Fatalf("got %d Retry-After=%q close=%v, want 503 / 1 / true", resp.StatusCode, resp.Header.Get("Retry-After"), resp.Close)
	}
	if held := time.Since(start); held < 350*time.Millisecond {
		t.Fatalf("answered after %s, want it held until the API was up", held)
	}
	if f.wakes.Load() != 1 {
		t.Fatalf("wake called %d times, want 1", f.wakes.Load())
	}
}

func TestFrontDoorDownWithoutClientCertDoesNotWake(t *testing.T) {
	f := newFrontDoorFixture(t, "127.0.0.1:1")
	resp := f.get(t, &tls.Config{RootCAs: f.roots})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", resp.StatusCode)
	}
	if f.wakes.Load() != 0 {
		t.Fatal("an unauthenticated request must not start a build")
	}
}

func TestFrontDoorUpSplicesToUpstream(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok from upstream")
	}))
	defer upstream.Close()
	f := newFrontDoorFixture(t, upstream.Listener.Addr().String())
	f.up.Store(true)

	// TLS is end to end: the client must see the upstream's cert, not ours.
	roots := x509.NewCertPool()
	roots.AddCert(upstream.Certificate())
	resp := f.get(t, &tls.Config{RootCAs: roots, ServerName: "example.com"})
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "ok from upstream" {
		t.Fatalf("got %d %q", resp.StatusCode, body)
	}
	if f.touches.Load() == 0 {
		t.Fatal("spliced traffic must count as use")
	}
	if f.wakes.Load() != 0 {
		t.Fatal("no wake expected while the API is up")
	}
}

func TestFrontDoorKubeconfig(t *testing.T) {
	in := "clusters:\n- cluster:\n    certificate-authority-data: AAAA\n    server: https://172.18.255.215:6443\n  name: target-cluster\n"
	want := "clusters:\n- cluster:\n    certificate-authority-data: AAAA\n    server: https://172.18.255.223:6443\n    tls-server-name: 172.18.255.215\n  name: target-cluster\n"
	if got := frontDoorKubeconfig(in); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// End to end against the real target cluster with the real kubectl: the front
// door reports "down" for 3s, then splices. Passes only if kubectl retries the
// 503 by itself and the kubeconfig verifies both sides. Needs a running
// target-cluster; opt in with FRONTDOOR_E2E_KUBECONFIG=<direct kubeconfig>.
func TestFrontDoorKubectlE2E(t *testing.T) {
	src := os.Getenv("FRONTDOOR_E2E_KUBECONFIG")
	if src == "" {
		t.Skip("set FRONTDOOR_E2E_KUBECONFIG to run")
	}
	caDir := filepath.Join(claudeDir(), "03-target-cluster", "warm-ca")
	read := func(n string) []byte {
		b, err := os.ReadFile(filepath.Join(caDir, n))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	cfg, err := frontDoorTLSConfig(read("server-ca.crt"), read("server-ca.key"), read("client-ca.crt"),
		[]net.IP{net.ParseIP(targetLBIP)})
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	var up atomic.Bool
	var wakes atomic.Int32
	a := &activator{upstream: targetLBIP + ":6443", tlsCfg: cfg, apiUp: up.Load,
		wake: func() { wakes.Add(1) }, touch: func() {}, holdMax: 30 * time.Second}
	go a.serve(ln)
	go func() {
		time.Sleep(3 * time.Second)
		up.Store(true)
	}()

	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	kc := strings.Replace(frontDoorKubeconfig(string(raw)),
		"https://"+targetFrontDoorIP+":6443", "https://"+ln.Addr().String(), 1)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(path, []byte(kc), 0o600); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	out, err := exec.Command("kubectl", "--kubeconfig", path, "get", "nodes", "-o", "name").CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("kubectl failed after %s: %v\n%s", elapsed, err, out)
	}
	if elapsed < 3*time.Second || wakes.Load() == 0 {
		t.Fatalf("kubectl returned in %s with %d wakes; the down path was not exercised", elapsed, wakes.Load())
	}
	fmt.Printf("kubectl waited %s through the front door, %d wake call(s):\n%s", elapsed.Round(100*time.Millisecond), wakes.Load(), out)
}

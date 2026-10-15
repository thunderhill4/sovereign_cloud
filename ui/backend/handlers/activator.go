package handlers

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Front door for the target cluster's API (on-demand mode only).
//
// The cluster itself stays on targetLBIP; its own nodes and the CAPI
// controllers talk to that address constantly, so traffic there says nothing
// about whether anyone is using it. Users and agents are handed a kubeconfig
// pointing at targetFrontDoorIP instead, which a selectorless LoadBalancer
// Service (ui/k8s/target-frontdoor.yaml) routes to this listener. Everything
// that arrives here is therefore user traffic: it counts as use, and it wakes
// the cluster if there is none.
//
//	API up   -> raw TCP splice to targetLBIP:6443. TLS and client-cert auth are
//	            end to end; this process never sees inside.
//	API down -> terminate TLS with a cert minted from the warm path's fixed
//	            server CA, require a client cert signed by its client CA, start
//	            the build, hold the request until /readyz answers, then reply
//	            503 + Retry-After + Connection: close. client-go retries on
//	            that, redials, and lands on the spliced path.
//
// Holding the raw connection instead is not an option: client-go gives up on a
// TLS handshake after 10s and a cold build takes longer.
const targetFrontDoorIP = "172.18.255.223"

type activator struct {
	upstream string
	tlsCfg   *tls.Config
	apiUp    func() bool
	wake     func()
	touch    func()
	holdMax  time.Duration
}

func (a *activator) serve(ln net.Listener) error {
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		go a.handle(c)
	}
}

func (a *activator) handle(c net.Conn) {
	defer c.Close()
	if a.apiUp() {
		a.touch()
		a.splice(c)
		return
	}
	a.serveDown(c)
}

func (a *activator) splice(c net.Conn) {
	up, err := net.DialTimeout("tcp", a.upstream, 3*time.Second)
	if err != nil {
		return
	}
	defer up.Close()
	done := make(chan struct{}, 2)
	go func() {
		// Client -> API bytes are the "in use" signal for long-lived connections.
		io.Copy(up, touchReader{c, a.touch})
		if t, ok := up.(*net.TCPConn); ok {
			t.CloseWrite()
		}
		done <- struct{}{}
	}()
	go func() {
		io.Copy(c, up)
		if t, ok := c.(*net.TCPConn); ok {
			t.CloseWrite()
		}
		done <- struct{}{}
	}()
	<-done
	<-done
}

type touchReader struct {
	r     io.Reader
	touch func()
}

func (t touchReader) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	if n > 0 {
		t.touch()
	}
	return n, err
}

func (a *activator) serveDown(c net.Conn) {
	c.SetDeadline(time.Now().Add(a.holdMax + 30*time.Second))
	tc := tls.Server(c, a.tlsCfg)
	if err := tc.Handshake(); err != nil {
		return
	}
	defer tc.Close()
	req, err := http.ReadRequest(bufio.NewReader(tc))
	if err != nil {
		return
	}
	// Only a holder of a valid client cert may start a build.
	if len(tc.ConnectionState().VerifiedChains) == 0 {
		writeStatus(tc, req, http.StatusUnauthorized, "Unauthorized", "client certificate required", false)
		return
	}
	a.wake()
	deadline := time.Now().Add(a.holdMax)
	for !a.apiUp() && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	writeStatus(tc, req, http.StatusServiceUnavailable, "ServiceUnavailable",
		"target-cluster is waking up; retry", true)
}

// writeStatus answers with a Kubernetes Status object so kubectl prints the
// message if it ever surfaces one.
func writeStatus(w io.Writer, req *http.Request, code int, reason, msg string, retry bool) {
	body := fmt.Sprintf(`{"kind":"Status","apiVersion":"v1","metadata":{},"status":"Failure","message":%q,"reason":%q,"code":%d}`+"\n",
		msg, reason, code)
	resp := &http.Response{
		StatusCode:    code,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Request:       req,
		Header:        http.Header{"Content-Type": {"application/json"}},
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Close:         true,
	}
	if retry {
		resp.Header.Set("Retry-After", "1")
	}
	resp.Write(w)
}

// frontDoorTLSConfig mints a serving cert from the given server CA and trusts
// client certs from the given client CA. The cert carries targetLBIP because
// front-door kubeconfigs set tls-server-name to it, so the same client config
// verifies both this listener and the real API server.
func frontDoorTLSConfig(serverCACertPEM, serverCAKeyPEM, clientCACertPEM []byte, ips []net.IP) (*tls.Config, error) {
	caBlock, _ := pem.Decode(serverCACertPEM)
	if caBlock == nil {
		return nil, errors.New("server CA cert: no PEM block")
	}
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("server CA cert: %w", err)
	}
	keyBlock, _ := pem.Decode(serverCAKeyPEM)
	if keyBlock == nil {
		return nil, errors.New("server CA key: no PEM block")
	}
	var caKey any
	if k, err := x509.ParseECPrivateKey(keyBlock.Bytes); err == nil {
		caKey = k
	} else if k, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes); err == nil {
		caKey = k
	} else if k, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes); err == nil {
		caKey = k
	} else {
		return nil, errors.New("server CA key: unsupported key format")
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 126))
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "target-cluster-frontdoor"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  ips,
		DNSNames:     []string{"localhost", "kubernetes"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(clientCACertPEM) {
		return nil, errors.New("client CA cert: no certificates found")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der, caCert.Raw}, PrivateKey: leafKey}},
		ClientCAs:    clientCAs,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		// HTTP/1.1 only: the 503 closes the connection, so the client's retry
		// has to redial and can then be spliced to the real API server.
		NextProtos: []string{"http/1.1"},
		MinVersion: tls.VersionTLS12,
	}, nil
}

// targetAPIUp probes the real API server's /readyz (anonymous-readable). The
// answer is cached 1s while up and 200ms while down, so the front door notices
// a finished build quickly without probing per connection.
var apiProbe struct {
	mu sync.Mutex
	at time.Time
	up bool
}

var apiProbeClient = &http.Client{
	Timeout: time.Second,
	Transport: &http.Transport{
		// Liveness only: no credentials are sent and the client verifies the
		// real server itself on the spliced connection.
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DialContext:       (&net.Dialer{Timeout: 500 * time.Millisecond}).DialContext,
		DisableKeepAlives: true,
	},
}

func targetAPIUp() bool {
	apiProbe.mu.Lock()
	defer apiProbe.mu.Unlock()
	ttl := 200 * time.Millisecond
	if apiProbe.up {
		ttl = time.Second
	}
	if time.Since(apiProbe.at) < ttl {
		return apiProbe.up
	}
	up := false
	if resp, err := apiProbeClient.Get("https://" + targetLBIP + ":6443/readyz"); err == nil {
		up = resp.StatusCode == http.StatusOK
		resp.Body.Close()
	}
	apiProbe.up, apiProbe.at = up, time.Now()
	return up
}

func frontDoorAddr() string {
	if v := os.Getenv("TARGET_FRONTDOOR_ADDR"); v != "" {
		return v
	}
	// The kind bridge address: reachable from the cluster2 node that MetalLB
	// answers on, and not from the host's other networks.
	return "172.18.0.1:16443"
}

// StartTargetFrontDoor runs the front door. Start once from main(); a no-op
// unless POOL_MODE=ondemand.
func StartTargetFrontDoor() {
	if poolMode() != poolModeOnDemand || !poolEnabled() {
		return
	}
	caDir := filepath.Join(claudeDir(), "03-target-cluster", "warm-ca")
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join(caDir, name))
		if err != nil {
			log.Printf("front door: %v", err)
		}
		return b
	}
	cfg, err := frontDoorTLSConfig(read("server-ca.crt"), read("server-ca.key"), read("client-ca.crt"),
		[]net.IP{net.ParseIP(targetLBIP), net.ParseIP(targetFrontDoorIP), net.ParseIP("127.0.0.1")})
	if err != nil {
		log.Printf("front door disabled: %v (run scripts/gen-warm-ca.sh?)", err)
		return
	}
	ln, err := net.Listen("tcp", frontDoorAddr())
	if err != nil {
		log.Printf("front door disabled: %v", err)
		return
	}
	a := &activator{
		upstream: targetLBIP + ":6443",
		tlsCfg:   cfg,
		apiUp:    targetAPIUp,
		wake:     wakeTarget,
		touch:    touchTarget,
		holdMax:  60 * time.Second,
	}
	// A cluster that predates this mode has a direct kubeconfig published;
	// repoint it so using that cluster counts as use.
	direct := "/tmp/" + targetClusterName + "-kubeconfig"
	if runShell(fmt.Sprintf("clusterctl get kubeconfig %s > %s.new 2>/dev/null && mv %s.new %s",
		targetClusterName, direct, direct, direct)) == nil {
		publishTargetKubeconfig(direct)
	}
	log.Printf("Target front door on %s (https://%s:6443 -> %s, idle TTL %s)",
		ln.Addr(), targetFrontDoorIP, a.upstream, poolIdleTTL())
	log.Printf("front door stopped: %v", a.serve(ln))
}

// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestRenderToken_WithoutSecurity_NoAuthSurface(t *testing.T) {

	project := httpClientTestProject()
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, project.ModulePath, "client")

	if err := renderer.RenderToken(); err != nil {
		t.Fatalf("RenderToken: %v", err)
	}
	if err := renderer.RenderClientOptions(); err != nil {
		t.Fatalf("RenderClientOptions: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "token")); err == nil {
		t.Fatal("token package must not exist without security")
	}

	options := mustRead(t, filepath.Join(dir, "options.go"))
	if strings.Contains(options, "func TokenSource(") || strings.Contains(options, "oauth2") {
		t.Fatalf("TokenSource/oauth2 must not appear without security:\n%s", options)
	}
	if strings.Contains(options, "func ResourceTokens(") {
		t.Fatal("ResourceTokens must not appear without security")
	}
	if strings.Contains(options, "func Assertion(") || strings.Contains(options, "func TokenURL(") {
		t.Fatal("Assertion/TokenURL must not appear without app2app security profile")
	}
}

func TestRenderToken_WithBearerSecurity(t *testing.T) {

	project := httpClientTestProject()
	project.Annotations = tags.DocTags{
		"security": "http:bearer:JWT|jwks=/jwks/{sub}/.well-known/keys",
	}
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, project.ModulePath, "client")

	if err := renderer.RenderToken(); err != nil {
		t.Fatalf("RenderToken: %v", err)
	}
	if err := renderer.RenderClientOptions(); err != nil {
		t.Fatalf("RenderClientOptions: %v", err)
	}

	for _, name := range []string{"assertion.go", "bearer.go", "exchange.go", "metadata.go", "transport.go", "reuse.go", "retry.go", "request.go"} {
		if _, err := os.Stat(filepath.Join(dir, "token", name)); err != nil {
			t.Fatalf("token/%s: %v", name, err)
		}
	}

	options := mustRead(t, filepath.Join(dir, "options.go"))
	if !strings.Contains(options, "func TokenSource(src oauth2.TokenSource) Option") {
		t.Fatalf("TokenSource missing:\n%s", options)
	}
	if !strings.Contains(options, "oauth2.Transport") {
		t.Fatalf("oauth2.Transport missing:\n%s", options)
	}
	if !strings.Contains(options, "func ResourceTokens(obtain token.ObtainFunc)") || !strings.Contains(options, "AuthOption") {
		t.Fatalf("ResourceTokens AuthOption missing:\n%s", options)
	}
	if !strings.Contains(options, "func Auth(opts ...AuthOption) Option") {
		t.Fatalf("Auth missing:\n%s", options)
	}
	transportSrc := mustRead(t, filepath.Join(dir, "token", "transport.go"))
	if !strings.Contains(transportSrc, "func Resource(") {
		t.Fatal("token.Resource factory missing")
	}
	if !strings.Contains(transportSrc, "ensureSource") {
		t.Fatal("transport must obtain token via ensureSource before the API call")
	}
	if !strings.Contains(transportSrc, "outcomeAuthChallenge") && !strings.Contains(mustRead(t, filepath.Join(dir, "token", "retry.go")), "outcomeAuthChallenge") {
		t.Fatal("auth challenge classification missing")
	}
	if !strings.Contains(transportSrc, "invalidate") {
		t.Fatal("transport must invalidate source on 401")
	}
	if !strings.Contains(transportSrc, "type obtainGate struct") {
		t.Fatal("ensureSource must coalesce obtain via obtainGate")
	}
	if !strings.Contains(transportSrc, "atomic.Pointer[obtainGate]") {
		t.Fatal("resourceTransport must store gate in atomic.Pointer")
	}
	if !strings.Contains(transportSrc, "done") || !strings.Contains(transportSrc, "chan struct{}") {
		t.Fatal("obtain gate must publish completion through done")
	}
	if !strings.Contains(transportSrc, "req.Context().Done()") {
		t.Fatal("waiters must be able to cancel while obtain is running")
	}
	if !strings.Contains(transportSrc, "context.WithoutCancel(") {
		t.Fatal("obtain flight must use WithoutCancel so one waiter cancel does not abort shared obtain")
	}
	if !strings.Contains(transportSrc, "context.WithTimeout(") || !strings.Contains(transportSrc, "flightTimeout") {
		t.Fatal("obtain flight must have a configured timeout")
	}
	if !strings.Contains(mustRead(t, filepath.Join(dir, "token", "retry.go")), "func FlightTimeout(") {
		t.Fatal("token.FlightTimeout missing")
	}
	if !strings.Contains(options, "func TokenFlightTimeout(") {
		t.Fatal("client TokenFlightTimeout missing")
	}
	if !strings.Contains(transportSrc, "t.gate.Store(nil)") {
		t.Fatal("invalidate must clear gate atomically")
	}
	if !strings.Contains(transportSrc, "CompareAndSwap(gate, nil)") {
		t.Fatal("failed obtain must clear gate for retry")
	}
	if strings.Contains(transportSrc, "ensureRewindableBody") {
		t.Fatal("transport must not buffer request bodies via ensureRewindableBody")
	}
	if !strings.Contains(transportSrc, "prepareAttempt") {
		t.Fatal("transport must use prepareAttempt")
	}
	if strings.Contains(transportSrc, "outcomeInfra") {
		t.Fatal("Resource must not retry resource API infra statuses")
	}
	if !strings.Contains(transportSrc, "opts ...RetryOption") {
		t.Fatal("Resource must accept RetryOption")
	}
	if !strings.Contains(transportSrc, "func (t *resourceTransport) Unwrap()") {
		t.Fatal("resourceTransport must expose Unwrap for TLS base lookup")
	}
	if !strings.Contains(mustRead(t, filepath.Join(dir, "token", "retry.go")), "func MaxAttempts(") {
		t.Fatal("token.MaxAttempts missing")
	}
	if !strings.Contains(options, "func TokenRetry(") {
		t.Fatalf("TokenRetry missing:\n%s", options)
	}
	if strings.Contains(options, "httpClient.Transport = token.Resource") {
		t.Fatal("ResourceTokens must defer Resource wrap to wireResourceAuth")
	}
	reuseSrc := mustRead(t, filepath.Join(dir, "token", "reuse.go"))
	if !strings.Contains(reuseSrc, "defer s.mu.Unlock()") {
		t.Fatal("reuseSource.Token must hold mu during refresh")
	}
	for _, name := range []string{"bearer.go", "exchange.go"} {
		tokenSource := mustRead(t, filepath.Join(dir, "token", name))
		if !strings.Contains(tokenSource, "TokenContext(ctx context.Context)") || !strings.Contains(tokenSource, "fetchToken(ctx,") {
			t.Fatalf("token/%s must pass request context into fetchToken", name)
		}
	}
	requestSrc := mustRead(t, filepath.Join(dir, "token", "request.go"))
	if strings.Contains(requestSrc, "io.ReadAll") {
		t.Fatal("request.go must not ReadAll request bodies")
	}
	if !strings.Contains(requestSrc, "func prepareAttempt(") {
		t.Fatal("prepareAttempt missing")
	}
	assertionSrc := mustRead(t, filepath.Join(dir, "token", "assertion.go"))
	if strings.Contains(assertionSrc, "30 * time.Minute") {
		t.Fatal("fixed 30m earlyExpiry must be removed")
	}
	if !strings.Contains(assertionSrc, "SkewFraction") {
		t.Fatal("SkewFraction option missing")
	}
	if !strings.Contains(mustRead(t, filepath.Join(dir, "token", "metadata.go")), "func WellKnownURL(") {
		t.Fatal("WellKnownURL missing")
	}
	if strings.Contains(options, "func Assertion(") || strings.Contains(options, "func TokenURL(") {
		t.Fatal("Assertion/TokenURL must not appear without app2app profile")
	}
}

func TestRenderClient_App2AppDefaultAuth(t *testing.T) {

	project := httpClientTestProject()
	project.Annotations = tags.DocTags{
		"security": "http:bearer:JWT|jwks=/jwks/{sub}/.well-known/keys,oauth2:tokenExchange,oauth2:clientCredentials:jwt-bearer",
	}
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, project.ModulePath, "client")

	if err := renderer.RenderToken(); err != nil {
		t.Fatalf("RenderToken: %v", err)
	}
	if err := renderer.RenderClientOptions(); err != nil {
		t.Fatalf("RenderClientOptions: %v", err)
	}
	if err := renderer.RenderClient(); err != nil {
		t.Fatalf("RenderClient: %v", err)
	}

	options := mustRead(t, filepath.Join(dir, "options.go"))
	for _, fragment := range []string{
		"func Auth(opts ...AuthOption) Option",
		"func TokenSource(src oauth2.TokenSource) Option",
		"func Assertion(subject, keyID, privateKey string)",
		"func TokenURL(url string)",
		"AuthOption",
		"cli.authSkip = true",
		"cli.assertion.Subject = strings.TrimSpace(subject)",
		"cli.assertion.KeyID = strings.TrimSpace(keyID)",
		"cli.assertion.PrivateKey = privateKey",
	} {
		if !strings.Contains(options, fragment) {
			t.Fatalf("options.go missing %q:\n%s", fragment, options)
		}
	}
	if strings.Contains(options, "func TokenURL(url string) Option") {
		t.Fatal("TokenURL must be AuthOption, not Option")
	}

	clientSrc := mustRead(t, filepath.Join(dir, "client.go"))
	for _, fragment := range []string{
		"func (cli *Client) wireResourceAuth()",
		"token.Resource",
		"token.JWTBearer",
		"token.Exchange",
		"cli.wireResourceAuth()",
		"cli.tokenRetry",
	} {
		if !strings.Contains(clientSrc, fragment) {
			t.Fatalf("client.go missing %q:\n%s", fragment, clientSrc)
		}
	}
	if !strings.Contains(options, "func TokenRetry(") {
		t.Fatalf("TokenRetry missing:\n%s", options)
	}
}

func TestTokenPackage_JWTBearerAndExchange(t *testing.T) {

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}))

	var sawBearer, sawExchange bool
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if parseErr := r.ParseForm(); parseErr != nil {
			http.Error(w, parseErr.Error(), http.StatusBadRequest)
			return
		}
		switch r.Form.Get("grant_type") {
		case "urn:ietf:params:oauth:grant-type:jwt-bearer":
			sawBearer = true
			if r.Form.Get("assertion") == "" {
				t.Errorf("missing assertion")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token_type":"Bearer","access_token":"primary-token","expires_in":3600}`))
		case "urn:ietf:params:oauth:grant-type:token-exchange":
			sawExchange = true
			if r.Form.Get("subject_token") != "primary-token" {
				t.Errorf("subject_token=%q", r.Form.Get("subject_token"))
			}
			if r.Form.Get("audience") != "resource-1" {
				t.Errorf("audience=%q", r.Form.Get("audience"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token_type":"Bearer","access_token":"audience-token","expires_in":3600}`))
		default:
			http.Error(w, "bad grant", http.StatusBadRequest)
		}
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	root := t.TempDir()
	dir := filepath.Join(root, "client")
	project := &model.Project{
		ModulePath: "example.com/app",
		Annotations: tags.DocTags{
			"security": "http:bearer:JWT|jwks=/jwks/{sub}/.well-known/keys,oauth2:tokenExchange,oauth2:clientCredentials:jwt-bearer",
		},
		Contracts: []*model.Contract{{
			Name: "API",
			Annotations: tags.DocTags{
				model.TagServerHTTP: "",
			},
			Methods: []*model.Method{{
				Name: "Ping",
				Annotations: tags.DocTags{
					model.TagHTTPMethod: "GET",
					model.TagHTTPPath:   "/ping",
				},
			}},
		}},
	}
	renderer := NewClientRenderer(project, dir, project.ModulePath, "client")
	if err = renderer.RenderToken(); err != nil {
		t.Fatal(err)
	}

	goMod := "module example.com/app\n\ngo 1.26\n\n" +
		"require (\n" +
		"\tgithub.com/golang-jwt/jwt/v5 v5.3.0\n" +
		"\tgithub.com/google/uuid v1.6.0\n" +
		"\tgolang.org/x/oauth2 v0.30.0\n" +
		")\n"
	if err = os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	testSrc := `package token_test

import (
	"testing"

	"example.com/app/client/token"
)

func TestFlow(t *testing.T) {
	primary, err := token.JWTBearer("` + server.URL + `/token", token.Assertion{
		Subject:    "caller-1",
		KeyID:      "key-1",
		PrivateKey: ` + "`" + pemKey + "`" + `,
	})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := primary.Token()
	if err != nil {
		t.Fatalf("JWTBearer: %v", err)
	}
	if tok.AccessToken != "primary-token" {
		t.Fatalf("token=%q", tok.AccessToken)
	}

	ex, err := token.Exchange("` + server.URL + `/token", primary, "resource-1")
	if err != nil {
		t.Fatal(err)
	}
	tok, err = ex.Token()
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if tok.AccessToken != "audience-token" {
		t.Fatalf("token=%q", tok.AccessToken)
	}

	if _, err = token.JWTBearer("", token.Assertion{Subject: "a", KeyID: "b", PrivateKey: "c"}); err == nil {
		t.Fatal("expected empty tokenURL error")
	}
	if _, err = token.Exchange("` + server.URL + `/token", nil, "aud"); err == nil {
		t.Fatal("expected nil subject error")
	}
}
`
	if err = os.WriteFile(filepath.Join(dir, "token", "flow_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "test", "-mod=mod", "./client/token/")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test token: %v\n%s", err, out)
	}
	if !sawBearer || !sawExchange {
		t.Fatalf("grants seen bearer=%v exchange=%v", sawBearer, sawExchange)
	}
}

func TestTokenPackage_ResourceDiscovery(t *testing.T) {

	root := t.TempDir()
	dir := filepath.Join(root, "client")
	project := &model.Project{
		ModulePath: "example.com/app",
		Annotations: tags.DocTags{
			"security": "http:bearer:JWT|jwks=/jwks/{sub}/.well-known/keys,oauth2:tokenExchange,oauth2:clientCredentials:jwt-bearer",
		},
		Contracts: []*model.Contract{{
			Name: "API",
			Annotations: tags.DocTags{
				model.TagServerHTTP: "",
			},
			Methods: []*model.Method{{
				Name: "Ping",
				Annotations: tags.DocTags{
					model.TagHTTPMethod: "GET",
					model.TagHTTPPath:   "/ping",
				},
			}},
		}},
	}
	renderer := NewClientRenderer(project, dir, project.ModulePath, "client")
	if err := renderer.RenderToken(); err != nil {
		t.Fatal(err)
	}

	goMod := "module example.com/app\n\ngo 1.26\n\n" +
		"require (\n" +
		"\tgithub.com/golang-jwt/jwt/v5 v5.3.0\n" +
		"\tgithub.com/google/uuid v1.6.0\n" +
		"\tgolang.org/x/oauth2 v0.30.0\n" +
		")\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	testSrc := `
package token_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"example.com/app/client/token"
)

type staticSource struct {
	token  string
	expiry time.Time
}

func (s staticSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: s.token, TokenType: "Bearer", Expiry: s.expiry}, nil
}

func metaHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte("{\"resource\":\"http://resource\",\"audience\":\"aud-1\",\"authorization_servers\":[\"http://as\"],\"bearer_methods_supported\":[\"header\"]}"))
}

func TestResourceWarm(t *testing.T) {
	var apiHits, metaHits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		metaHits.Add(1)
		metaHandler(w, r)
	})
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		apiHits.Add(1)
		if r.Header.Get("Authorization") == "Bearer access-1" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	if got := token.WellKnownURL(nil); got != "" {
		t.Fatalf("nil url: %q", got)
	}
	parsed, _ := url.Parse(server.URL + "/api")
	if got := token.WellKnownURL(parsed); got != server.URL+"/.well-known/oauth-protected-resource" {
		t.Fatalf("WellKnownURL=%q", got)
	}

	var obtainCalls int
	client := &http.Client{Transport: token.Resource(http.DefaultTransport, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		obtainCalls++
		return staticSource{token: "access-1", expiry: time.Now().Add(time.Hour)}, nil
	})}
	for i := 0; i < 2; i++ {
		resp, err := client.Get(server.URL + "/api")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != 200 || string(body) != "ok" {
			t.Fatalf("status=%d body=%q", resp.StatusCode, body)
		}
	}
	if obtainCalls != 1 || metaHits.Load() != 1 || apiHits.Load() != 2 {
		t.Fatalf("obtain=%d meta=%d api=%d", obtainCalls, metaHits.Load(), apiHits.Load())
	}
}

func TestResourceAuthRetry(t *testing.T) {
	var apiHits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", metaHandler)
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		apiHits.Add(1)
		auth := r.Header.Get("Authorization")
		if auth == "Bearer stale" {
			w.Header().Set("WWW-Authenticate", "Bearer resource_metadata=\"http://"+r.Host+"/.well-known/oauth-protected-resource\"")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if auth == "Bearer fresh" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var obtainCalls int
	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		obtainCalls++
		tok := "stale"
		if obtainCalls > 1 {
			tok = "fresh"
		}
		return staticSource{token: tok, expiry: time.Now().Add(time.Hour)}, nil
	})}
	resp, err := client.Get(server.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if obtainCalls != 2 {
		t.Fatalf("obtainCalls=%d", obtainCalls)
	}
	if apiHits.Load() != 2 {
		t.Fatalf("apiHits=%d", apiHits.Load())
	}
}

func TestResourceForbidden(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", metaHandler)
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var obtainCalls int
	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		obtainCalls++
		return staticSource{token: "access-1", expiry: time.Now().Add(time.Hour)}, nil
	})}
	resp, err := client.Get(server.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 403 || obtainCalls != 1 {
		t.Fatalf("status=%d obtain=%d", resp.StatusCode, obtainCalls)
	}
}

func TestResourceAPINoInfraRetry(t *testing.T) {
	var apiHits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", metaHandler)
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		apiHits.Add(1)
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var obtainCalls int
	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		obtainCalls++
		return staticSource{token: "access-1", expiry: time.Now().Add(time.Hour)}, nil
	})}
	resp, err := client.Get(server.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable || obtainCalls != 1 || apiHits.Load() != 1 {
		t.Fatalf("status=%d obtain=%d api=%d", resp.StatusCode, obtainCalls, apiHits.Load())
	}
}

func TestResourceIdPRetry(t *testing.T) {
	var metaHits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		n := metaHits.Add(1)
		if n == 1 {
			http.Error(w, "busy", http.StatusTooManyRequests)
			return
		}
		metaHandler(w, r)
	})
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer access-1" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		return staticSource{token: "access-1", expiry: time.Now().Add(time.Hour)}, nil
	}, token.Backoff(time.Millisecond, 10*time.Millisecond))}
	resp, err := client.Get(server.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || metaHits.Load() < 2 {
		t.Fatalf("status=%d meta=%d", resp.StatusCode, metaHits.Load())
	}
}

func TestResourceIdPDisabled(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "busy", http.StatusServiceUnavailable)
	})
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		return staticSource{token: "access-1", expiry: time.Now().Add(time.Hour)}, nil
	}, token.Disabled())}
	_, err := client.Get(server.URL + "/api")
	if err == nil {
		t.Fatal("expected metadata error without retry")
	}
}

func TestResourceAuthRetryNonRewindable(t *testing.T) {
	var apiHits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", metaHandler)
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		apiHits.Add(1)
		auth := r.Header.Get("Authorization")
		if auth == "Bearer stale" {
			w.Header().Set("WWW-Authenticate", "Bearer resource_metadata=\"http://"+r.Host+"/.well-known/oauth-protected-resource\"")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var obtainCalls int
	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		obtainCalls++
		tok := "stale"
		if obtainCalls > 1 {
			tok = "fresh"
		}
		return staticSource{token: tok, expiry: time.Now().Add(time.Hour)}, nil
	})}
	body := io.NopCloser(strings.NewReader("{\"x\":1}"))
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api", body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if !errors.Is(err, token.ErrBodyNotRewindable) {
		t.Fatalf("err=%v", err)
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
	if apiHits.Load() != 1 {
		t.Fatalf("apiHits=%d", apiHits.Load())
	}
}

func TestResourceAuthRetryRewindable(t *testing.T) {
	var apiHits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", metaHandler)
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		apiHits.Add(1)
		auth := r.Header.Get("Authorization")
		if auth == "Bearer stale" {
			w.Header().Set("WWW-Authenticate", "Bearer resource_metadata=\"http://"+r.Host+"/.well-known/oauth-protected-resource\"")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if auth == "Bearer fresh" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var obtainCalls int
	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		obtainCalls++
		tok := "stale"
		if obtainCalls > 1 {
			tok = "fresh"
		}
		return staticSource{token: tok, expiry: time.Now().Add(time.Hour)}, nil
	})}
	payload := []byte("{\"x\":1}")
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api", strings.NewReader(string(payload)))
	if err != nil {
		t.Fatal(err)
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(string(payload))), nil
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || obtainCalls != 2 || apiHits.Load() != 2 {
		t.Fatalf("status=%d obtain=%d api=%d", resp.StatusCode, obtainCalls, apiHits.Load())
	}
}

func TestFetchMetadataErrors(t *testing.T) {
	if _, err := token.FetchMetadata(context.Background(), "", nil); err == nil {
		t.Fatal("expected empty URL error")
	}
	server := httptest.NewServer(http.NewServeMux())
	defer server.Close()
	if _, err := token.FetchMetadata(context.Background(), server.URL+"/no-meta", nil); err == nil {
		t.Fatal("expected metadata status error")
	}
	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		t.Fatal("obtain must not run")
		return nil, nil
	})}
	_, err := client.Get(server.URL + "/api")
	if err == nil {
		t.Fatal("expected metadata error")
	}
}

func TestResourceParallelColdStart(t *testing.T) {
	var metaHits, obtainCalls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		metaHits.Add(1)
		metaHandler(w, r)
	})
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer access-1" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		if obtainCalls.Add(1) == 1 {
			close(started)
			<-release
		}
		return staticSource{token: "access-1", expiry: time.Now().Add(time.Hour)}, nil
	})}

	const n = 16
	var wg sync.WaitGroup
	errs := make(chan error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			resp, err := client.Get(server.URL + "/api")
			if err != nil {
				errs <- err
				return
			}
			_ = resp.Body.Close()
			if resp.StatusCode != 200 {
				errs <- fmt.Errorf("status=%d", resp.StatusCode)
			}
		}()
	}
	<-started
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if obtainCalls.Load() != 1 || metaHits.Load() != 1 {
		t.Fatalf("obtain=%d meta=%d", obtainCalls.Load(), metaHits.Load())
	}
}

func TestResourceObtainErrorRetry(t *testing.T) {
	var obtainCalls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", metaHandler)
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer access-1" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		if obtainCalls.Add(1) == 1 {
			return nil, fmt.Errorf("token: temporary obtain failure")
		}
		return staticSource{token: "access-1", expiry: time.Now().Add(time.Hour)}, nil
	}, token.Disabled())}

	if _, err := client.Get(server.URL + "/api"); err == nil {
		t.Fatal("expected first obtain error")
	}
	resp, err := client.Get(server.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 || obtainCalls.Load() != 2 {
		t.Fatalf("status=%d obtain=%d", resp.StatusCode, obtainCalls.Load())
	}
}

func TestResourceNilObtain(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", metaHandler)
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		return nil, nil
	})}
	if _, err := client.Get(server.URL + "/api"); err == nil {
		t.Fatal("expected nil source error")
	}
}

func TestResourceCancelWaiterDuringObtain(t *testing.T) {
	var obtainCalls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource", metaHandler)
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer access-1" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{Transport: token.Resource(nil, func(ctx context.Context, meta token.Metadata) (oauth2.TokenSource, error) {
		if obtainCalls.Add(1) == 1 {
			close(started)
			<-release
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return staticSource{token: "access-1", expiry: time.Now().Add(time.Hour)}, nil
	})}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(2)
	var canceledErr, okErr error
	go func() {
		defer wg.Done()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api", nil)
		if err != nil {
			canceledErr = err
			return
		}
		_, canceledErr = client.Do(req)
	}()
	go func() {
		defer wg.Done()
		<-started
		cancel()
		time.Sleep(20 * time.Millisecond)
		close(release)
		resp, err := client.Get(server.URL + "/api")
		if err != nil {
			okErr = err
			return
		}
		_ = resp.Body.Close()
		if resp.StatusCode != 200 {
			okErr = fmt.Errorf("status=%d", resp.StatusCode)
		}
	}()
	wg.Wait()
	if canceledErr == nil {
		t.Fatal("expected canceled waiter error")
	}
	if okErr != nil {
		t.Fatal(okErr)
	}
	if obtainCalls.Load() != 1 {
		t.Fatalf("obtain=%d", obtainCalls.Load())
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "token", "transport_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "test", "-mod=mod", "./client/token/")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test token resource: %v\n%s", err, out)
	}
}

func mustRead(t *testing.T, path string) (content string) {

	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/tags"
)

func TestRenderTransport_BearerWiring(t *testing.T) {

	project := transportServerTestProject()
	project.Annotations = tags.DocTags{
		"security": "http:bearer:JWT|jwks=/jwks/:sub/.well-known/keys,oauth2:tokenExchange",
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)

	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	if err := renderer.RenderTransportOptions(); err != nil {
		t.Fatalf("RenderTransportOptions: %v", err)
	}
	if err := renderer.RenderTransportAuth(); err != nil {
		t.Fatalf("RenderTransportAuth: %v", err)
	}
	if err := renderer.RenderTransportContext(); err != nil {
		t.Fatalf("RenderTransportContext: %v", err)
	}

	serverSource := mustReadFile(t, filepath.Join(dir, "server.go"))
	for _, fragment := range []string{
		"*authenticator",
		"authChallenge",
		"authMetadata",
		"func (srv *Server) bearerMiddleware(ftx *fiber.Ctx) (err error)",
		"srv.srvHTTP.Use(srv.bearerMiddleware)",
		"denyBearer",
		"auth denied",
		"resourceMetadata",
		"WWW-Authenticate",
		"StatusForbidden",
		"srvctx.Subject",
		"bearer.WellKnownPath",
		"if srv.auth != nil",
		"srv.auth.Authenticate",
		"func New(log *slog.Logger, options ...Option) (srv *Server)",
	} {
		if !strings.Contains(serverSource, fragment) {
			t.Fatalf("server.go missing %q:\n%s", fragment, serverSource)
		}
	}
	if strings.Contains(serverSource, "initErr") {
		t.Fatalf("server.go must not contain initErr:\n%s", serverSource)
	}
	if !strings.Contains(serverSource, "reason == \"forbidden\"") {
		t.Fatal("denyBearer must branch on forbidden → 403")
	}
	wellKnownIdx := strings.Index(serverSource, "bearer.WellKnownPath")
	secondPassIdx := strings.Index(serverSource, "for _, option := range options")
	fiberNewIdx := strings.Index(serverSource, "fiber.New(srv.config)")
	if wellKnownIdx < 0 || secondPassIdx < 0 || fiberNewIdx < 0 {
		t.Fatalf("server.go missing New/options/well-known wiring:\n%s", serverSource)
	}
	// First options pass is before fiber.New; second pass (and well-known) after.
	firstPassIdx := secondPassIdx
	secondPassIdx = strings.Index(serverSource[fiberNewIdx:], "for _, option := range options")
	if secondPassIdx < 0 {
		t.Fatalf("expected second options pass after fiber.New:\n%s", serverSource)
	}
	secondPassIdx += fiberNewIdx
	if firstPassIdx > fiberNewIdx || wellKnownIdx < secondPassIdx {
		t.Fatalf("options must apply before and after fiber.New; well-known after second pass:\n%s", serverSource)
	}

	authSource := mustReadFile(t, filepath.Join(dir, "auth.go"))
	for _, fragment := range []string{
		"type AuthOption func(cfg *authConfig)",
		"func Auth(opts ...AuthOption) Option",
		"if srv.auth != nil",
		"func AuthorizationServer(",
		"func Resource(",
		"func Audience(",
		"func ClientHTTP(",
		"func ConfigTLS(",
		"func Bind(",
		"func Allow(",
		"func Verify(",
		"errBearerForbidden",
		"auth.bind(ctx, claims)",
		"Client:",
		"TLS:",
		"auth.verifier = bearer.NewVerifier",
		"srv.auth = auth",
		"srv.authMetadata, _ = json.Marshal",
	} {
		if !strings.Contains(authSource, fragment) {
			t.Fatalf("auth.go missing %q:\n%s", fragment, authSource)
		}
	}
	for _, forbidden := range []string{"initErr", "panic(err)", "NewAuth("} {
		if strings.Contains(authSource, forbidden) {
			t.Fatalf("auth.go must not contain %q:\n%s", forbidden, authSource)
		}
	}
	if strings.Contains(authSource, "func Subject(") || strings.Contains(authSource, "WithBearer") {
		t.Fatalf("auth.go must not contain Subject/WithBearer:\n%s", authSource)
	}

	optionsSource := mustReadFile(t, filepath.Join(dir, "options.go"))
	if strings.Contains(optionsSource, "WithBearer") || strings.Contains(optionsSource, "func Subject(") {
		t.Fatalf("options.go must not contain WithBearer/Subject:\n%s", optionsSource)
	}

	if _, err := os.Stat(filepath.Join(dir, "security.go")); err == nil {
		t.Fatal("security.go must not exist")
	}

	pathSource := mustReadFile(t, filepath.Join(dir, "bearer", "path.go"))
	if !strings.Contains(pathSource, `claims["sub"]`) {
		t.Fatalf("path.go must use sub claim:\n%s", pathSource)
	}

	verifierSource := mustReadFile(t, filepath.Join(dir, "bearer", "verifier.go"))
	for _, fragment := range []string{
		"func NewVerifier(cfg Config) (verifier *Verifier)",
		"func defaultClient(tlsCfg *tls.Config) (client *http.Client)",
		"ForceAttemptHTTP2:",
		"Proxy:                 http.ProxyFromEnvironment",
		"Timeout: 10 * time.Second",
		"TLS                 *tls.Config",
		"TLSClientConfig:       tlsCfg",
		"bearer: verifier is not configured",
		"inflight map[string]*jwksLoad",
		"type jwksLoad struct",
	} {
		if !strings.Contains(verifierSource, fragment) {
			t.Fatalf("verifier.go missing %q:\n%s", fragment, verifierSource)
		}
	}
	for _, forbidden := range []string{
		"mutually exclusive",
		"authorization server is empty",
		"audience is empty",
		"func NewVerifier(cfg Config) (verifier *Verifier, err error)",
	} {
		if strings.Contains(verifierSource, forbidden) {
			t.Fatalf("verifier.go must not contain %q:\n%s", forbidden, verifierSource)
		}
	}
	if strings.Contains(verifierSource, "transport.TLSClientConfig = cfg.TLS") {
		t.Fatalf("verifier.go must not mutate caller transport:\n%s", verifierSource)
	}

	ctxSource := mustReadFile(t, filepath.Join(dir, "srvctx", "context.go"))
	if !strings.Contains(ctxSource, "type Subject string") {
		t.Fatalf("srvctx missing Subject:\n%s", ctxSource)
	}
	if !strings.Contains(ctxSource, "func GetSubject(ctx context.Context) (s string)") {
		t.Fatalf("srvctx missing GetSubject:\n%s", ctxSource)
	}

	if !strings.Contains(serverSource, "AuthDenied") {
		t.Fatalf("server.go missing AuthDenied metric bump:\n%s", serverSource)
	}

	for _, name := range []string{"verifier.go", "discovery.go", "keys.go", "resource.go", "path.go"} {
		if _, err := os.Stat(filepath.Join(dir, "bearer", name)); err != nil {
			t.Fatalf("bearer/%s: %v", name, err)
		}
	}
}

func TestRenderTransport_BearerWithoutJWKSPath(t *testing.T) {

	project := transportServerTestProject()
	project.Annotations = tags.DocTags{
		"security": "http:bearer:JWT",
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	if err := renderer.RenderTransportAuth(); err != nil {
		t.Fatalf("RenderTransportAuth: %v", err)
	}

	pathSource := mustReadFile(t, filepath.Join(dir, "bearer", "path.go"))
	if !strings.Contains(pathSource, "_ = claims") {
		t.Fatalf("empty JWKS path expected:\n%s", pathSource)
	}
	if _, err := os.Stat(filepath.Join(dir, "security.go")); err == nil {
		t.Fatal("security.go must not exist")
	}
}

func TestRenderTransport_NoBearerWithoutSecurity(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)

	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	if err := renderer.RenderTransportOptions(); err != nil {
		t.Fatalf("RenderTransportOptions: %v", err)
	}
	if err := renderer.RenderTransportAuth(); err != nil {
		t.Fatalf("RenderTransportAuth: %v", err)
	}
	if err := renderer.RenderTransportContext(); err != nil {
		t.Fatalf("RenderTransportContext: %v", err)
	}

	serverSource := mustReadFile(t, filepath.Join(dir, "server.go"))
	for _, fragment := range []string{
		"bearerMiddleware",
		"denyBearer",
		"authenticator",
	} {
		if strings.Contains(serverSource, fragment) {
			t.Fatalf("server.go must not contain %q without security:\n%s", fragment, serverSource)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "auth.go")); err == nil {
		t.Fatal("auth.go must not exist without security")
	}
	optionsSource := mustReadFile(t, filepath.Join(dir, "options.go"))
	if strings.Contains(optionsSource, "func Auth(") || strings.Contains(optionsSource, "WithBearer") {
		t.Fatalf("options.go must not contain Auth/WithBearer without security:\n%s", optionsSource)
	}
	if _, err := os.Stat(filepath.Join(dir, "bearer")); err == nil {
		t.Fatal("bearer package must not exist without security")
	}

	ctxSource := mustReadFile(t, filepath.Join(dir, "srvctx", "context.go"))
	if !strings.Contains(ctxSource, "type ClientID string") {
		t.Fatalf("srvctx missing ClientID:\n%s", ctxSource)
	}
	if strings.Contains(ctxSource, "type Subject string") || strings.Contains(ctxSource, "func GetSubject(") {
		t.Fatalf("srvctx must not contain Subject without bearer:\n%s", ctxSource)
	}
}

func TestParseJWKSPath(t *testing.T) {

	params, segments := parseJWKSPath("/jwks/:sub/.well-known/keys")
	if len(params) != 1 || params[0] != "sub" {
		t.Fatalf("params=%v", params)
	}
	if len(segments) != 4 {
		t.Fatalf("segments=%v", segments)
	}
	params, segments = parseJWKSPath("/oauth/jwks")
	if len(params) != 0 || len(segments) != 2 {
		t.Fatalf("literal path params=%v segments=%v", params, segments)
	}
	params, segments = parseJWKSPath("")
	if len(params) != 0 || len(segments) != 0 {
		t.Fatalf("empty path params=%v segments=%v", params, segments)
	}
}

func TestBearerPackage_VerifyCustomJWKSPath(t *testing.T) {

	root, dir, pub, priv := renderBearerPackage(t, "http:bearer:JWT|jwks=/jwks/:sub/.well-known/keys")
	const (
		subject  = "caller-1"
		audience = "resource-1"
		kid      = "key-1"
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/jwks/"+subject+"/.well-known/keys", jwksHandler(pub, kid))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	writeBearerGoMod(t, root)
	testSrc := bearerVerifyTestSource(bearerVerifyFixture{
		priv:     priv,
		asURL:    server.URL,
		audience: audience,
		subject:  subject,
		kid:      kid,
		issuer:   "",
	})
	if err := os.WriteFile(filepath.Join(dir, "bearer", "verifier_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	runBearerGoTest(t, root)
}

func TestBearerPackage_VerifyRFCDiscovery(t *testing.T) {

	root, dir, pub, priv := renderBearerPackage(t, "http:bearer:JWT")
	const (
		issuer   = "https://auth.example/issuer"
		subject  = "caller-1"
		audience = "resource-1"
		kid      = "key-1"
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":   issuer,
			"jwks_uri": "http://" + r.Host + "/oauth/jwks",
		})
	})
	mux.HandleFunc("/oauth/jwks", jwksHandler(pub, kid))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	writeBearerGoMod(t, root)
	testSrc := bearerVerifyTestSource(bearerVerifyFixture{
		priv:     priv,
		asURL:    server.URL,
		audience: audience,
		subject:  subject,
		kid:      kid,
		issuer:   issuer,
	})
	if err := os.WriteFile(filepath.Join(dir, "bearer", "verifier_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	runBearerGoTest(t, root)
}

func TestBearerPackage_RFCDiscoveryMissingJWKSURI(t *testing.T) {

	root, dir, _, priv := renderBearerPackage(t, "http:bearer:JWT")
	const (
		subject  = "caller-1"
		audience = "resource-1"
		kid      = "key-1"
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"issuer": "https://auth.example"})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	writeBearerGoMod(t, root)
	testSrc := `package bearer_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"example.com/app/transport/bearer"
)

func TestMissingJWKSURI(t *testing.T) {
	priv, err := base64.RawURLEncoding.DecodeString("` + base64.RawURLEncoding.EncodeToString(priv) + `")
	if err != nil {
		t.Fatal(err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"sub": "` + subject + `",
		"aud": "` + audience + `",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = "` + kid + `"
	raw, err := token.SignedString(ed25519.PrivateKey(priv))
	if err != nil {
		t.Fatal(err)
	}
	verifier := bearer.NewVerifier(bearer.Config{
		AuthorizationServer: "` + server.URL + `",
		Audience:            "` + audience + `",
	})
	if _, _, err = verifier.Verify(context.Background(), raw); err == nil || !strings.Contains(err.Error(), "jwks_uri missing") {
		t.Fatalf("expected jwks_uri missing, got %v", err)
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "bearer", "verifier_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	runBearerGoTest(t, root)
}

func TestBearerPackage_NewVerifierValidation(t *testing.T) {

	root, dir, _, _ := renderBearerPackage(t, "http:bearer:JWT")
	writeBearerGoMod(t, root)
	testSrc := `package bearer_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"testing"
	"time"

	"example.com/app/transport/bearer"
)

func TestNewVerifierValidation(t *testing.T) {
	emptyAS := bearer.NewVerifier(bearer.Config{Audience: "aud"})
	if _, _, err := emptyAS.Verify(context.Background(), "token"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected not configured for empty AS, got %v", err)
	}
	emptyAud := bearer.NewVerifier(bearer.Config{AuthorizationServer: "http://as"})
	if _, _, err := emptyAud.Verify(context.Background(), "token"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected not configured for empty audience, got %v", err)
	}
	verifier := bearer.NewVerifier(bearer.Config{
		AuthorizationServer: "http://as.example",
		Audience:            "aud",
	})
	if verifier == nil {
		t.Fatal("verifier is nil")
	}
}

func TestNewVerifierClientAndTLS(t *testing.T) {
	tr := &http.Transport{}
	custom := &http.Client{Transport: tr, Timeout: time.Second}
	tlsCfg := &tls.Config{ServerName: "jwks.example"}
	both := bearer.NewVerifier(bearer.Config{
		AuthorizationServer: "http://as.example",
		Audience:            "aud",
		Client:              custom,
		TLS:                 tlsCfg,
	})
	if both == nil {
		t.Fatal("verifier is nil")
	}
	if tr.TLSClientConfig != nil {
		t.Fatalf("caller transport must not be mutated: %#v", tr.TLSClientConfig)
	}

	clientOnly := bearer.NewVerifier(bearer.Config{
		AuthorizationServer: "http://as.example",
		Audience:            "aud",
		Client:              custom,
	})
	if clientOnly == nil {
		t.Fatal("verifier is nil")
	}
	if tr.TLSClientConfig != nil {
		t.Fatalf("Client-only path must not touch TLS: %#v", tr.TLSClientConfig)
	}

	tlsOnly := bearer.NewVerifier(bearer.Config{
		AuthorizationServer: "http://as.example",
		Audience:            "aud",
		TLS:                 tlsCfg,
	})
	if tlsOnly == nil {
		t.Fatal("verifier is nil")
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "bearer", "verifier_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	runBearerGoTest(t, root)
}

func TestBearerPackage_CustomHTTPClient(t *testing.T) {

	root, dir, pub, priv := renderBearerPackage(t, "http:bearer:JWT|jwks=/jwks/:sub/.well-known/keys")
	writeBearerGoMod(t, root)

	const (
		audience = "aud-1"
		subject  = "caller-1"
		kid      = "key-1"
	)
	server := httptest.NewServer(jwksHandler(pub, kid))
	t.Cleanup(server.Close)

	testSrc := `package bearer_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"example.com/app/transport/bearer"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCustomHTTPClient(t *testing.T) {
	priv, err := base64.RawURLEncoding.DecodeString("` + base64.RawURLEncoding.EncodeToString(priv) + `")
	if err != nil {
		t.Fatal(err)
	}
	claims := jwt.MapClaims{
		"sub": "` + subject + `",
		"aud": "` + audience + `",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = "` + kid + `"
	raw, err := token.SignedString(ed25519.PrivateKey(priv))
	if err != nil {
		t.Fatal(err)
	}

	var hits atomic.Int32
	base := http.DefaultTransport
	client := &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			hits.Add(1)
			return base.RoundTrip(req)
		}),
	}
	verifier := bearer.NewVerifier(bearer.Config{
		AuthorizationServer: "` + server.URL + `",
		Audience:            "` + audience + `",
		Client:              client,
	})
	if _, _, err = verifier.Verify(context.Background(), raw); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if hits.Load() == 0 {
		t.Fatal("custom client was not used")
	}

	blocked := &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("blocked")
		}),
	}
	failing := bearer.NewVerifier(bearer.Config{
		AuthorizationServer: "` + server.URL + `",
		Audience:            "` + audience + `",
		Client:              blocked,
	})
	if _, _, err = failing.Verify(context.Background(), raw); err == nil {
		t.Fatal("expected custom client error")
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "bearer", "client_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	runBearerGoTest(t, root)
}

type bearerVerifyFixture struct {
	priv     ed25519.PrivateKey
	asURL    string
	audience string
	subject  string
	kid      string
	issuer   string
}

func renderBearerPackage(t *testing.T, security string) (root string, dir string, pub ed25519.PublicKey, priv ed25519.PrivateKey) {

	t.Helper()
	root = t.TempDir()
	dir = filepath.Join(root, "transport")
	project := transportServerTestProject()
	project.ModulePath = "example.com/app"
	project.Annotations = tags.DocTags{"security": security}
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	var err error
	if pub, priv, err = ed25519.GenerateKey(rand.Reader); err != nil {
		t.Fatal(err)
	}
	return
}

func jwksHandler(pub ed25519.PublicKey, kid string) http.HandlerFunc {

	return func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kty": "OKP",
				"crv": "Ed25519",
				"kid": kid,
				"x":   base64.RawURLEncoding.EncodeToString(pub),
			}},
		})
	}
}

func writeBearerGoMod(t *testing.T, root string) {

	t.Helper()
	goMod := "module example.com/app\n\ngo 1.26\n\n" +
		"require (\n" +
		"\tgithub.com/go-jose/go-jose/v4 v4.1.3\n" +
		"\tgithub.com/golang-jwt/jwt/v5 v5.3.0\n" +
		")\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runBearerGoTest(t *testing.T, root string) {

	t.Helper()
	cmd := exec.Command("go", "test", "-mod=mod", "./transport/bearer/")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test bearer: %v\n%s", err, out)
	}
}

func bearerVerifyTestSource(fixture bearerVerifyFixture) (source string) {

	issClaim := ""
	if fixture.issuer != "" {
		issClaim = "\n\t\t\"iss\": \"" + fixture.issuer + "\","
	}
	return `package bearer_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"example.com/app/transport/bearer"
)

func TestVerify(t *testing.T) {
	priv, err := base64.RawURLEncoding.DecodeString("` + base64.RawURLEncoding.EncodeToString(fixture.priv) + `")
	if err != nil {
		t.Fatal(err)
	}

	claims := jwt.MapClaims{` + issClaim + `
		"sub": "` + fixture.subject + `",
		"aud": "` + fixture.audience + `",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = "` + fixture.kid + `"
	raw, err := token.SignedString(ed25519.PrivateKey(priv))
	if err != nil {
		t.Fatal(err)
	}

	verifier := bearer.NewVerifier(bearer.Config{
		AuthorizationServer: "` + fixture.asURL + `",
		Audience:            "` + fixture.audience + `",
	})
	got, gotClaims, err := verifier.Verify(context.Background(), raw)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != "` + fixture.subject + `" {
		t.Fatalf("subject=%q", got)
	}
	if gotClaims["sub"] != "` + fixture.subject + `" {
		t.Fatalf("claims=%v", gotClaims)
	}

	if _, _, err = verifier.Verify(context.Background(), raw); err != nil {
		t.Fatalf("cached Verify: %v", err)
	}

	if _, _, err = verifier.Verify(context.Background(), raw+"x"); err == nil {
		t.Fatal("expected signature error")
	}

	wrongAud := bearer.NewVerifier(bearer.Config{
		AuthorizationServer: "` + fixture.asURL + `",
		Audience:            "other",
	})
	if _, _, err = wrongAud.Verify(context.Background(), raw); err == nil {
		t.Fatal("expected audience error")
	}

	if _, _, err = verifier.Verify(context.Background(), ""); err == nil {
		t.Fatal("expected empty token error")
	}

	unknownKid := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	unknownKid.Header["kid"] = "missing"
	badKid, err := unknownKid.SignedString(ed25519.PrivateKey(priv))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = verifier.Verify(context.Background(), badKid); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected key not found, got %v", err)
	}

	if _, _, err = verifier.Verify(context.Background(), raw); err != nil {
		t.Fatalf("Verify after errors: %v", err)
	}
}
`
}

func mustReadFile(t *testing.T, path string) (content string) {

	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

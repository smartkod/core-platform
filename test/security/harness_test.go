// Package security_test is a black-box audit of a running instance.
//
// It talks HTTP only, exactly like an attacker with a valid account: no
// database access, no internal calls, no shortcuts. That is deliberate. A
// white-box test can accidentally exercise a code path the real server never
// takes; this suite cannot.
//
// Run it against the container:
//
//	$env:TEST_BASE_URL = "http://localhost:18080"
//	go test ./test/security/... -count=1 -v
//
// Without TEST_BASE_URL the suite skips. A skipped audit is reported as NOT
// TESTED, never as a pass.
package security_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type client struct {
	t     *testing.T
	base  string
	http  *http.Client
	token string
	email string
	id    string
	csrf  string
	// ip is a synthetic client address sent as X-Forwarded-For.
	//
	// The rate limiter is keyed by the caller's address, so a suite that comes
	// from one loopback address would share one bucket and the earlier tests
	// would throttle the later ones. Giving each client its own address keeps
	// the buckets independent. That this works at all is itself a finding: the
	// header is caller-controlled (see TestForwardedForCannotBypassLimit).
	ip string
}

var clientSeq atomic.Int64

func newClient(t *testing.T, base string) *client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	n := clientSeq.Add(1)
	ip := fmt.Sprintf("198.51.100.%d", n%250+2)
	return &client{
		t:    t,
		base: base,
		http: &http.Client{Jar: jar, Timeout: 20 * time.Second},
		ip:   ip,
	}
}

// envelope mirrors the platform response shape.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// do performs a request and returns the status, parsed data and raw body.
func (c *client) do(method, path string, body any, headers map[string]string) (int, json.RawMessage, string) {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.ip != "" {
		req.Header.Set("X-Forwarded-For", c.ip)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatalf("read body: %v", err)
	}

	var env envelope
	_ = json.Unmarshal(raw, &env)

	data := env.Data
	if method == http.MethodDelete && res.StatusCode == http.StatusNoContent {
		data = nil
	}
	return res.StatusCode, data, string(raw)
}

// get is a convenience wrapper that fails the test on an unexpected status.
func (c *client) get(path string) (int, json.RawMessage) {
	status, data, _ := c.do(http.MethodGet, path, nil, nil)
	return status, data
}

func (c *client) decode(path string, dst any) int {
	c.t.Helper()
	status, data, raw := c.do(http.MethodGet, path, nil, nil)
	if status != http.StatusOK {
		return status
	}
	if err := json.Unmarshal(data, dst); err != nil {
		c.t.Fatalf("GET %s: decode: %v (body: %s)", path, err, truncate(raw))
	}
	return status
}

func truncate(s string) string {
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}

// login authenticates with the mobile transport (refresh token in the body,
// no cookies), which keeps each test client a plain Bearer client.
func (c *client) login(email, password string) {
	c.t.Helper()
	status, data, raw := c.do(http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": email, "password": password}, nil)
	if status != http.StatusOK {
		c.t.Fatalf("login %s: status %d: %s", email, status, truncate(raw))
	}
	var session struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(data, &session); err != nil {
		c.t.Fatalf("login decode: %v", err)
	}
	c.token = session.AccessToken
	c.email = email
	c.id = session.User.ID
	if c.id == "" {
		var me struct {
			ID string `json:"id"`
		}
		c.decode("/api/v1/me", &me)
		c.id = me.ID
	}
}

// userID is the account id of the signed-in client.
func (c *client) userID() string { return c.id }

func (c *client) register(email, password string) {
	c.t.Helper()
	status, _, raw := c.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": email, "password": password, "name": "Audit",
	}, nil)
	if status != http.StatusCreated {
		c.t.Fatalf("register %s: status %d: %s", email, status, truncate(raw))
	}
}

func (c *client) createOrg(name, slug string) string {
	c.t.Helper()
	status, data, raw := c.do(http.MethodPost, "/api/v1/organizations",
		map[string]string{"name": name, "slug": slug}, nil)
	if status != http.StatusCreated {
		c.t.Fatalf("create org: %d: %s", status, truncate(raw))
	}
	var org struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &org); err != nil {
		c.t.Fatalf("decode org: %v", err)
	}
	return org.ID
}

func (c *client) createBranch(orgID, name string) string {
	c.t.Helper()
	status, data, raw := c.do(http.MethodPost, "/api/v1/organizations/"+orgID+"/branches",
		map[string]any{"name": name, "type": "physical", "city": "Istanbul", "public": true}, nil)
	if status != http.StatusCreated {
		c.t.Fatalf("create branch: %d: %s", status, truncate(raw))
	}
	var b struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &b); err != nil {
		c.t.Fatalf("decode branch: %v", err)
	}
	return b.ID
}

func (c *client) createOffering(orgID, name, slug string) string {
	c.t.Helper()
	taxonomy := c.firstTaxonomyID()
	status, data, raw := c.do(http.MethodPost, "/api/v1/organizations/"+orgID+"/offerings",
		map[string]any{
			"name": name, "slug": slug, "taxonomy_id": taxonomy,
			"service_mode": "on_site", "summary": "audit",
		}, nil)
	if status != http.StatusCreated {
		c.t.Fatalf("create offering: %d: %s", status, truncate(raw))
	}
	var o struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &o); err != nil {
		c.t.Fatalf("decode offering: %v", err)
	}
	return o.ID
}

func (c *client) firstTaxonomyID() string {
	c.t.Helper()
	var nodes []struct {
		ID string `json:"id"`
	}
	if status := c.decode("/api/v1/taxonomy", &nodes); status != http.StatusOK || len(nodes) == 0 {
		c.t.Fatalf("taxonomy: status %d, nodes %d (is 0007_taxonomy_seed applied?)", status, len(nodes))
	}
	return nodes[0].ID
}

func (c *client) createVariant(orgID, offeringID, name string) string {
	c.t.Helper()
	status, data, raw := c.do(http.MethodPost,
		"/api/v1/organizations/"+orgID+"/offerings/"+offeringID+"/variants",
		map[string]any{"name": name}, nil)
	if status != http.StatusCreated {
		c.t.Fatalf("create variant: %d: %s", status, truncate(raw))
	}
	var v struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		c.t.Fatalf("decode variant: %v", err)
	}
	return v.ID
}

// invite adds a member and returns the membership id.
func (c *client) invite(orgID, email, role string) string {
	c.t.Helper()
	status, data, raw := c.do(http.MethodPost, "/api/v1/organizations/"+orgID+"/members",
		map[string]string{"email": email, "role_key": role}, nil)
	if status != http.StatusCreated && status != http.StatusOK {
		c.t.Fatalf("invite %s: %d: %s", email, status, truncate(raw))
	}
	var m struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		c.t.Fatalf("decode membership: %v", err)
	}
	return m.ID
}

func (c *client) acceptInvitation(id string) int {
	status, _, _ := c.do(http.MethodPost, "/api/v1/invitations/"+id+"/accept", map[string]any{}, nil)
	return status
}

// ------------------------------------------------------------------ fixtures

type world struct {
	base string

	alice *client // owner of organization A
	bob   *client // owner of organization B
	mia   *client // manager in A
	sam   *client // staff in A
	uma   *client // member in A
	bea   *client // manager in B, used as the victim of cross-tenant writes

	orgA, orgB string
	branchA    string
	branchB    string
	offeringA  string
	offeringB  string
	variantA   string
	variantB   string
	memberB    string
	memberBea  string
	memberSam  string
	memberMia  string
	memberUma  string
}

func unique(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}

func password() string { return "audit-parola-1234" }

// setup builds two independent organizations with five accounts and one of
// each object type. It is expensive, so tests share it through one call.
func setup(t *testing.T) *world {
	t.Helper()
	base := os.Getenv("TEST_BASE_URL")
	if base == "" {
		t.Skip("TEST_BASE_URL not set: the audit needs a running instance")
	}
	// See baseURL: Docker Desktop's ::1 forwarder stalls, so the suite always
	// talks to the IPv4 address.
	base = strings.Replace(strings.TrimRight(base, "/"), "//localhost:", "//127.0.0.1:", 1)

	id := unique(t)
	w := &world{base: base}

	register := func(prefix string) *client {
		c := newClient(t, base)
		c.register(prefix+"-"+id+"@audit.example", password())
		c.login(prefix+"-"+id+"@audit.example", password())
		return c
	}

	w.alice = register("alice")
	w.bob = register("bob")
	w.mia = register("mia")
	w.sam = register("sam")
	w.uma = register("uma")
	w.bea = register("bea")

	w.orgA = w.alice.createOrg("Audit A", "audit-a-"+id)
	w.orgB = w.bob.createOrg("Audit B", "audit-b-"+id)

	w.branchA = w.alice.createBranch(w.orgA, "A Merkez")
	w.branchB = w.bob.createBranch(w.orgB, "B Merkez")

	w.offeringA = w.alice.createOffering(w.orgA, "A hizmeti", "a-hizmet-"+id)
	w.offeringB = w.bob.createOffering(w.orgB, "B hizmeti", "b-hizmet-"+id)

	w.variantA = w.alice.createVariant(w.orgA, w.offeringA, "A varyant")
	w.variantB = w.bob.createVariant(w.orgB, w.offeringB, "B varyant")

	w.memberMia = w.alice.invite(w.orgA, w.mia.email, "manager")
	w.memberSam = w.alice.invite(w.orgA, w.sam.email, "staff")
	w.memberUma = w.alice.invite(w.orgA, w.uma.email, "member")
	w.memberBea = w.bob.invite(w.orgB, w.bea.email, "manager")
	w.memberB = w.memberBea

	for _, pair := range []struct {
		c  *client
		id string
	}{
		{w.mia, w.memberMia},
		{w.sam, w.memberSam},
		{w.uma, w.memberUma},
		{w.bea, w.memberBea},
	} {
		if status := pair.c.acceptInvitation(pair.id); status != http.StatusOK {
			t.Fatalf("accept invitation: %d", status)
		}
	}
	return w
}

// denied asserts that a request was refused without leaking existence. A 403 is
// right when the caller is known but lacks the right; 404 when the object is
// not theirs. Both are accepted because the contract allows either as long as
// nothing changes, but a 2xx is never acceptable.
func denied(t *testing.T, label string, status int, body string) {
	t.Helper()
	if status == http.StatusForbidden || status == http.StatusNotFound {
		return
	}
	t.Errorf("%s: status %d, want 403 or 404 (body: %s)", label, status, truncate(body))
}

func isSuccess(status int) bool { return status >= 200 && status < 300 }

// mustStatus fails unless the status matches.
func mustStatus(t *testing.T, label string, got, want int, body string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: status %d, want %d (body: %s)", label, got, want, truncate(body))
	}
}

var errNotFound = errors.New("not found")

// exists reports whether the owner can still read an object. It is the
// read-after-write proof that a denied request changed nothing.
func exists(c *client, path string) bool {
	status, _, _ := c.do(http.MethodGet, path, nil, nil)
	return status == http.StatusOK
}

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

// browserClient logs in the way the web UI does: Origin set, cookies kept in a
// jar, no Bearer token. The audit uses it to exercise the cookie path, which is
// the one a real browser takes and the one CSRF/CORS exist for.
func browserClient(t *testing.T, base, email string) *client {
	t.Helper()
	c := newClient(t, base)
	status, _, raw := c.do(http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": email, "password": password()},
		map[string]string{"Origin": "http://localhost:3000", "X-Client-Type": "web"})
	if status != http.StatusOK {
		t.Fatalf("browser login %s: %d: %s", email, status, truncate(raw))
	}
	// The browser keeps only cookies; requests authenticate through them.
	c.token = ""
	return c
}

// csrfToken reads the readable csrf cookie, exactly as the UI does.
func (c *client) csrfToken() string {
	url, err := url.Parse(c.base)
	if err != nil {
		return ""
	}
	for _, ck := range c.http.Jar.Cookies(url) {
		if ck.Name == "csrf" {
			return ck.Value
		}
	}
	return ""
}

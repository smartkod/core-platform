package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The rate limiter keys on whatever this extractor returns, so these tests are
// the actual proof that rate limiting cannot be bypassed with a header. The
// live suite can only show the behaviour of one deployment; this shows the
// policy itself.
func TestIPExtractorWithoutProxiesIgnoresForwardedFor(t *testing.T) {
	extract, err := IPExtractor(nil)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.50:5000"
	req.Header.Set("X-Forwarded-For", "198.51.100.99")

	if got := extract(req); got != "203.0.113.50" {
		t.Fatalf("ip = %q, want the socket address: X-Forwarded-For must be ignored "+
			"when no proxy is trusted, or a caller can reset its own rate-limit budget", got)
	}
}

func TestIPExtractorIgnoresForwardedForFromUntrustedSource(t *testing.T) {
	extract, err := IPExtractor([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.50:5000"
	req.Header.Set("X-Forwarded-For", "198.51.100.99")

	if got := extract(req); got != "203.0.113.50" {
		t.Fatalf("ip = %q, want the socket address: the caller is not a trusted proxy", got)
	}
}

func TestIPExtractorTrustsForwardedForFromTrustedProxy(t *testing.T) {
	extract, err := IPExtractor([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.1.2.3:5000"
	req.Header.Set("X-Forwarded-For", "198.51.100.99")

	if got := extract(req); got != "198.51.100.99" {
		t.Fatalf("ip = %q, want the client address reported by the trusted proxy", got)
	}
}

// A caller behind a trusted proxy must not be able to prepend a fake address:
// the chain is walked from the right and stops at the first untrusted hop.
func TestIPExtractorStopsAtFirstUntrustedHop(t *testing.T) {
	extract, err := IPExtractor([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.1.2.3:5000"
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 198.51.100.99")

	if got := extract(req); got != "198.51.100.99" {
		t.Fatalf("ip = %q, want the rightmost untrusted hop (198.51.100.99)", got)
	}
}

func TestIPExtractorAcceptsBareAddress(t *testing.T) {
	if _, err := IPExtractor([]string{"10.0.0.1"}); err != nil {
		t.Fatalf("a bare address must be accepted: %v", err)
	}
}

func TestIPExtractorRejectsGarbage(t *testing.T) {
	if _, err := IPExtractor([]string{"not-an-ip"}); err == nil {
		t.Fatal("a malformed entry must be an error, not silently ignored: a typo " +
			"would otherwise leave the operator believing the proxy is trusted")
	}
}

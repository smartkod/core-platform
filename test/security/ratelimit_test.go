package security_test

import (
	"net/http"
	"testing"
)

// TestLoginRateLimit proves brute force actually stops, and that a successful
// login does not consume the budget (otherwise a shared NAT would lock itself
// out during normal use).
func TestLoginRateLimit(t *testing.T) {
	base := baseURL(t)
	email := "limit-" + unique(t) + "@audit.example"

	c := newClient(t, base)
	c.register(email, password())

	// Failed attempts must eventually be refused.
	sawLimited := false
	for i := 0; i < 25; i++ {
		status, _, _ := c.do(http.MethodPost, "/api/v1/auth/login",
			map[string]string{"email": email, "password": "wrong-password-1234"}, nil)
		if status == http.StatusTooManyRequests {
			sawLimited = true
			break
		}
		if status != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status %d, want 401 or 429", i, status)
		}
	}
	if !sawLimited {
		t.Fatal("25 failed logins were never rate limited: brute force is unbounded")
	}

	// The throttle must not reveal whether the account exists: an unknown
	// address produces the same shape of answer.
	unknown := newClient(t, base)
	status, _, raw := unknown.do(http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": "ghost-" + unique(t) + "@audit.example", "password": "wrong-password-1234"}, nil)
	if status != http.StatusUnauthorized {
		t.Errorf("unknown account: status %d, want 401 like a wrong password (body: %s)", status, truncate(raw))
	}
}

// TestSuccessfulLoginDoesNotConsumeBudget is the counterpart: a legitimate user
// logging in repeatedly must never be throttled.
func TestSuccessfulLoginDoesNotConsumeBudget(t *testing.T) {
	base := baseURL(t)
	email := "oklogin-" + unique(t) + "@audit.example"

	c := newClient(t, base)
	c.register(email, password())

	for i := 0; i < 15; i++ {
		status, _, raw := c.do(http.MethodPost, "/api/v1/auth/login",
			map[string]string{"email": email, "password": password()}, nil)
		if status != http.StatusOK {
			t.Fatalf("successful login %d: status %d (body: %s)", i, status, truncate(raw))
		}
	}
}

// TestRegistrationRateLimit covers account-creation flooding.
func TestRegistrationRateLimit(t *testing.T) {
	base := baseURL(t)
	c := newClient(t, base)

	limited := false
	for i := 0; i < 40; i++ {
		status, _, _ := c.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
			"email":    "flood-" + unique(t) + "-" + string(rune('a'+i%26)) + "@audit.example",
			"password": password(), "name": "Flood",
		}, nil)
		if status == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Error("registration is not rate limited: an attacker can create accounts without bound")
	}
}

// TestPasswordResetRateLimit covers the enumeration/mail-bomb path.
func TestPasswordResetRateLimit(t *testing.T) {
	base := baseURL(t)
	c := newClient(t, base)

	limited := false
	for i := 0; i < 25; i++ {
		status, _, _ := c.do(http.MethodPost, "/api/v1/auth/forgot-password",
			map[string]string{"email": "burst-" + unique(t) + "@audit.example"}, nil)
		if status == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Error("forgot-password is not rate limited")
	}
}

// TestForwardedForDoesNotBreakPerAccountLimit covers the part of B4 that
// matters for account safety.
//
// Whether the IP key can be spoofed is a deployment decision: with
// `security.trusted_proxies` empty the header is ignored (proven in
// platform/httpx/ip_test.go), and with a trusted proxy configured the proxy's
// reported address is used. Either way the *account* key is derived from the
// e-mail, which the caller cannot vary while attacking one account. This test
// proves that second key works, so rotating X-Forwarded-For does not buy an
// attacker unlimited guesses against a single account.
func TestForwardedForDoesNotBreakPerAccountLimit(t *testing.T) {
	base := baseURL(t)
	email := "xff-" + unique(t) + "@audit.example"

	c := newClient(t, base)
	c.register(email, password())

	accountLimited := false
	for i := 0; i < 30; i++ {
		// A different source address every time, as an attacker would.
		headers := map[string]string{"X-Forwarded-For": "203.0.113." + itoa(i)}
		status, _, _ := c.do(http.MethodPost, "/api/v1/auth/login",
			map[string]string{"email": email, "password": "wrong-password-1234"}, headers)
		if status == http.StatusTooManyRequests {
			accountLimited = true
			break
		}
	}
	if !accountLimited {
		t.Fatal("rotating X-Forwarded-For produced unlimited guesses against one account: " +
			"the e-mail key of the login limiter is not working")
	}
}

// TestResourceCreationIsLimitedPerUser proves the creation cap applies to the
// live server and is keyed on the account rather than the address.
//
// Two things are asserted: a burst of creations eventually returns 429, and a
// second account is unaffected. Without the second half, a global counter would
// also pass, and one tenant could starve every other.
func TestResourceCreationIsLimitedPerUser(t *testing.T) {
	base := baseURL(t)
	stamp := unique(t)

	alice := newClient(t, base)
	alice.register("quota-a-"+stamp+"@audit.example", password())
	alice.login("quota-a-"+stamp+"@audit.example", password())

	bob := newClient(t, base)
	bob.register("quota-b-"+stamp+"@audit.example", password())
	bob.login("quota-b-"+stamp+"@audit.example", password())

	// A fresh organization per client, created before the burst so the
	// organization itself does not consume the branch budget.
	orgA := alice.createOrg("Quota A", "quota-a-"+stamp)
	orgB := bob.createOrg("Quota B", "quota-b-"+stamp)

	limited := false
	for i := 0; i < 120; i++ {
		status, _, _ := alice.do(http.MethodPost,
			"/api/v1/organizations/"+orgA+"/branches",
			map[string]any{"name": "burst-" + itoa(i), "type": "physical"}, nil)
		if status == http.StatusTooManyRequests {
			limited = true
			break
		}
		if !isSuccess(status) {
			t.Fatalf("branch %d: status %d", i, status)
		}
	}
	if !limited {
		t.Fatal("120 branches were created in one burst: resource creation is not rate limited")
	}

	// The other account must be untouched by the first one's exhaustion.
	status, _, body := bob.do(http.MethodPost, "/api/v1/organizations/"+orgB+"/branches",
		map[string]any{"name": "bob-branch", "type": "physical"}, nil)
	if !isSuccess(status) {
		t.Fatalf("a second account was blocked by the first account's burst: status %d (body: %s)",
			status, truncate(body))
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

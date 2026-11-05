package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
)

func userEcho(limiter *RateLimiter, userID string) *echo.Echo {
	e := echo.New()
	// The limiter runs after authentication, so the principal is set first, the
	// same order the real route groups use.
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if userID != "" {
				c.Set(core.ContextUserKey, &core.Principal{UserID: userID})
			}
			return next(c)
		}
	})
	e.POST("/create", func(c echo.Context) error { return c.NoContent(http.StatusCreated) },
		PerUser(limiter, "resource"))
	return e
}

func post(e *echo.Echo) int {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/create", nil))
	return rec.Code
}

// The quota is per account: one caller exhausting it must not touch another.
func TestPerUserIsolatesAccounts(t *testing.T) {
	limiter := NewRateLimiter(2, time.Hour)

	alice := userEcho(limiter, "user-a")
	if post(alice) != http.StatusCreated || post(alice) != http.StatusCreated {
		t.Fatal("the first two creations for one account should succeed")
	}
	if got := post(alice); got == http.StatusCreated {
		t.Fatal("the third creation exceeded the limit but was accepted")
	}

	bob := userEcho(limiter, "user-b")
	if got := post(bob); got != http.StatusCreated {
		t.Fatalf("another account was blocked by the first account's usage (status %d)", got)
	}
}

// A missing principal is a wiring bug. Failing open would leave the endpoint
// unprotected while looking protected.
func TestPerUserRequiresPrincipal(t *testing.T) {
	limiter := NewRateLimiter(10, time.Hour)
	e := userEcho(limiter, "")

	if got := post(e); got == http.StatusCreated {
		t.Fatal("an unauthenticated request bypassed the creation limit")
	}
}

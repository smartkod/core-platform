package platform

import (
	"strings"
	"time"

	"core-platform/contract/core"
	"core-platform/platform/config"
)

// configView adapts the resolved config to the narrow interface modules see.
//
// Keeping the adapter here (rather than exposing *config.Config) is what stops
// a module from reading a setting that belongs to another module, and keeps the
// contract package free of platform types.
type configView struct{ c *config.Config }

func (v configView) Env() string { return v.c.Env }

func (v configView) String(key string) string {
	switch key {
	case "identity.access_secret":
		return v.c.Identity.AccessSecret
	case "identity.refresh_secret":
		return v.c.Identity.RefreshSecret
	case "identity.issuer":
		return "core-platform"
	case "app.name":
		return v.c.AppName
	case "economy.manual_instructions":
		return v.c.Economy.ManualInstructions
	case "economy.test_webhook_secret":
		return v.c.Economy.TestWebhookSecret
	case "economy.currencies":
		return strings.Join(v.c.Economy.Currencies, ",")
	}
	return ""
}

func (v configView) Int(key string) int {
	switch key {
	case "identity.password_min_length":
		return v.c.Identity.PasswordMin
	}
	return 0
}

func (v configView) Duration(key string) time.Duration {
	switch key {
	case "identity.access_ttl":
		return v.c.Identity.AccessTTL
	case "identity.refresh_ttl":
		return v.c.Identity.RefreshTTL
	case "http.request_timeout":
		return v.c.HTTP.RequestTimeout
	case "economy.order_ttl":
		return v.c.Economy.OrderTTL
	case "economy.job_interval":
		return v.c.Economy.JobInterval
	}
	return 0
}

func (v configView) Bool(key string) bool {
	switch key {
	case "identity.cookie_secure":
		return v.c.Identity.CookieSecure
	}
	return false
}

var _ core.ConfigView = configView{}

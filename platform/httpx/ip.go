package httpx

import (
	"fmt"
	"net"
	"strings"

	"github.com/labstack/echo/v4"
)

// IPExtractor decides which address the rate limiter keys on.
//
// The default matters more than the option. X-Forwarded-For is written by the
// caller, so trusting it unconditionally lets anyone reset their own rate-limit
// budget by changing one header per request — which is the same as having no
// limit at all. When no proxy is configured the header must therefore be
// ignored entirely and the socket address used.
//
// Only when trusted ranges are declared does the header count, and even then
// Echo walks the chain right-to-left and stops at the first address outside the
// trusted set, so a caller cannot prepend a fake address and win.
func IPExtractor(trusted []string) (echo.IPExtractor, error) {
	if len(trusted) == 0 {
		return echo.ExtractIPDirect(), nil
	}

	ranges := make([]*net.IPNet, 0, len(trusted))
	for _, raw := range trusted {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			// A bare address is the common case; treat it as a single host.
			ip := net.ParseIP(entry)
			if ip == nil {
				return nil, fmt.Errorf("httpx: TRUSTED_PROXIES entry %q is not an IP or CIDR", raw)
			}
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			entry = fmt.Sprintf("%s/%d", ip.String(), bits)
		}
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return nil, fmt.Errorf("httpx: TRUSTED_PROXIES entry %q: %w", raw, err)
		}
		ranges = append(ranges, network)
	}
	if len(ranges) == 0 {
		return echo.ExtractIPDirect(), nil
	}

	options := make([]echo.TrustOption, 0, len(ranges))
	for _, network := range ranges {
		options = append(options, echo.TrustIPRange(network))
	}
	return echo.ExtractIPFromXFFHeader(options...), nil
}

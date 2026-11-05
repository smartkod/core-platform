// Command uiconsole serves a single-page API console for the core platform.
//
// It is a development tool, not part of the product: the platform itself is
// API-only (PRD.md §3). The binary does two things and nothing else:
//
//   - serves index.html at /
//   - reverse-proxies every other path to the running API
//
// The proxy exists because the API's CORS allow-list is empty unless configured,
// so a page served from a different origin could not call it. Same-origin
// requests need no CORS at all, and cookies are never involved: the console
// sends an Authorization: Bearer token it obtained from /auth/login.
//
// Run it with: go run ./cmd/uiconsole -api http://127.0.0.1:18080
package main

import (
	_ "embed"
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

//go:embed index.html
var indexHTML []byte

func main() {
	addr := flag.String("addr", "127.0.0.1:19090", "address the console listens on")
	api := flag.String("api", "http://127.0.0.1:18080", "base URL of the running API")
	flag.Parse()

	target, err := url.Parse(*api)
	if err != nil || target.Scheme == "" || target.Host == "" {
		log.Fatalf("invalid -api %q: need an absolute URL like http://127.0.0.1:18080", *api)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	direct := proxy.Director
	proxy.Director = func(r *http.Request) {
		direct(r)
		r.Host = target.Host
		// The API treats a request carrying Origin as a browser session and
		// switches to cookie transport. The console is an API client, so it must
		// look like one: strip the browser-only headers and keep the Bearer token.
		r.Header.Del("Origin")
		r.Header.Del("Referer")
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy: %s %s: %v", r.Method, r.URL.Path, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"code":"upstream_unreachable","message":"the API is not reachable; is it running?"}}`))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/" || r.URL.Path == "/index.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(indexHTML)
		case strings.HasPrefix(r.URL.Path, "/api/"):
			proxy.ServeHTTP(w, r)
		default:
			// /healthz, /readyz, /metrics and anything else the API exposes.
			proxy.ServeHTTP(w, r)
		}
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("ui console  http://%s  ->  api %s", *addr, target)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

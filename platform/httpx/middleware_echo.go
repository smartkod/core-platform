package httpx

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Recover converts a panic into a 500 without killing the process.
func Recover() echo.MiddlewareFunc { return middleware.Recover() }

// BodyLimit rejects oversized requests before they reach a handler.
func BodyLimit(limit string) echo.MiddlewareFunc {
	return middleware.BodyLimit(limit)
}

// CSVHeaderCSRF is the header a browser client copies from the csrf cookie.
//
// It is duplicated here (rather than imported from the identity module) because
// the kernel must not import a module. The value is part of the wire contract,
// so the two constants must stay identical; a test in the identity package
// asserts that the header actually passes CORS.
const CSVHeaderCSRF = "X-CSRF-Token"

// CSVHeaderClientType lets a browser identify itself so identity can choose the
// cookie transport instead of returning tokens in the body.
const CSVHeaderClientType = "X-Client-Type"

// CORS installs an allow-list CORS policy. An empty list means no cross-origin
// access, which is the safe default.
//
// Every custom request header a browser may send must appear in AllowHeaders,
// or the preflight fails before the request is ever made. That is why the CSRF
// and client-type headers are listed explicitly: without them the web client
// works with Bearer tokens but silently breaks in cookie mode, which is the
// mode a browser actually uses.
func CORS(origins []string) echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: origins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Authorization",
			"Content-Type",
			"Accept",
			HeaderRequestID,
			CSVHeaderCSRF,
			CSVHeaderClientType,
		},
		ExposeHeaders:    []string{HeaderRequestID},
		AllowCredentials: len(origins) > 0,
		MaxAge:           600,
	})
}

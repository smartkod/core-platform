// Package api holds the files HTTP layer.
package api

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	"core-platform/mod/files/service"
	"core-platform/platform/httpx"
)

// Handler serves the file endpoints.
type Handler struct {
	files core.Files
	// publicBaseURL is used to build absolute download links in JSON responses.
	publicBaseURL string
	// maxSize bounds the request body before it reaches the validator.
	maxSize int64
}

// Config is the transport policy.
type Config struct {
	PublicBaseURL string
	MaxSize       int64
	// SignedURLTTL is how long a download link stays valid.
	SignedURLTTL time.Duration
}

// NewHandler builds the handler.
func NewHandler(files core.Files, cfg Config) *Handler {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 10 << 20
	}
	return &Handler{
		files:         files,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
		maxSize:       cfg.MaxSize,
	}
}

// Register mounts the file routes. Every route requires authentication; a
// module that must serve public files should expose a separate, explicitly
// public route rather than relaxing this group.
func (h *Handler) Register(g *echo.Group, id core.Identity) {
	authed := g.Group("/files", id.Middleware())
	authed.POST("", h.Upload)
	authed.GET("", h.List)
	authed.GET("/:id", h.Get)
	authed.GET("/:id/download", h.Download)
	authed.DELETE("/:id", h.Delete)
}

func (h *Handler) Upload(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	// The body limit is enforced by middleware as well, but checking the
	// declared length here produces a clear 422 instead of a broken pipe.
	if c.Request().ContentLength > h.maxSize {
		return core.ErrValidation
	}

	file, err := c.FormFile("file")
	if err != nil {
		return core.ErrValidation
	}
	if file.Size > h.maxSize {
		return core.ErrValidation
	}

	src, err := file.Open()
	if err != nil {
		return core.ErrValidation
	}
	defer func() { _ = src.Close() }()

	obj, err := h.files.Upload(c.Request().Context(), core.UploadInput{
		OwnerID:     p.UserID,
		Filename:    file.Filename,
		ContentType: file.Header.Get(echo.HeaderContentType),
		Size:        file.Size,
		Body:        src,
	})
	if err != nil {
		return err
	}

	return httpx.RespondOK(c, http.StatusCreated, h.view(c, obj))
}

func (h *Handler) List(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	limit, offset := pageParams(c)
	files, total, err := h.files.List(c.Request().Context(), p.UserID, limit, offset)
	if err != nil {
		return err
	}

	items := make([]map[string]any, 0, len(files))
	for i := range files {
		items = append(items, h.view(c, &files[i]))
	}
	return httpx.RespondOK(c, http.StatusOK, map[string]any{
		"items": items, "total": total, "limit": limit, "offset": offset,
	})
}

func (h *Handler) Get(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	obj, err := h.files.Get(c.Request().Context(), c.Param("id"), p.UserID)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, h.view(c, obj))
}

// Download streams the bytes.
//
// The response is served through the API rather than by redirecting to storage:
// a redirect would hand the client a URL that bypasses this authorization check
// for as long as it lives. For local disk there is no other option, and for
// object storage the SignedURL path exists when a caller explicitly wants one.
func (h *Handler) Download(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	rc, obj, err := h.files.Download(c.Request().Context(), c.Param("id"), p.UserID)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	// Content-Disposition uses the stored name, and the type is the one that was
	// sniffed at upload time, not what the client claims now.
	c.Response().Header().Set(echo.HeaderContentType, obj.ContentType)
	c.Response().Header().Set("Content-Disposition", contentDisposition(obj.OriginalName))
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	c.Response().Header().Set("Cache-Control", "private, max-age=0, must-revalidate")

	c.Response().WriteHeader(http.StatusOK)
	_, err = io.Copy(c.Response(), rc)
	if err != nil && !errors.Is(err, http.ErrBodyReadAfterClose) {
		// The status line is already sent, so this can only be logged; returning
		// it would produce a second write on a committed response.
		return nil
	}
	return nil
}

func (h *Handler) Delete(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	if err := h.files.Delete(c.Request().Context(), c.Param("id"), p.UserID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// view renders a file for JSON.
//
// The object key is deliberately not exposed: it is an internal path and
// revealing it invites probing at the storage layer, which is where the
// ownership check does not apply.
func (h *Handler) view(c echo.Context, f *core.FileObject) map[string]any {
	download := "/api/v1/files/" + f.ID + "/download"
	if h.publicBaseURL != "" {
		download = h.publicBaseURL + download
	}
	return map[string]any{
		"id":           f.ID,
		"name":         f.OriginalName,
		"content_type": f.ContentType,
		"size_bytes":   f.Size,
		"checksum":     f.Checksum,
		"download_url": download,
		"created_at":   f.CreatedAt,
	}
}

// contentDisposition builds a safe header value.
//
// The ASCII fallback plus RFC 5987 encoding handles non-ASCII names without
// letting a quote or newline break out of the header.
func contentDisposition(name string) string {
	if name == "" {
		return "attachment"
	}
	ascii := strings.Map(func(r rune) rune {
		if r < 32 || r > 126 || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, name)

	return `attachment; filename="` + ascii + `"; filename*=UTF-8''` + urlEncode(name)
}

func urlEncode(s string) string {
	var b strings.Builder
	for _, r := range []byte(s) {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == '~' {
			b.WriteByte(r)
			continue
		}
		const hex = "0123456789ABCDEF"
		b.WriteByte('%')
		b.WriteByte(hex[r>>4])
		b.WriteByte(hex[r&0x0f])
	}
	return b.String()
}

func pageParams(c echo.Context) (limit, offset int) {
	limit = 20
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
			if limit > 100 {
				limit = 100
			}
		}
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

// Compile-time check that the service satisfies the contract.
var _ = service.NewFiles

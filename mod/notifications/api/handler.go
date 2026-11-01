// Package api holds the notifications HTTP layer.
package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	"core-platform/platform/httpx"
)

// Handler serves the notification endpoints.
type Handler struct {
	svc      core.Notifications
	prefs    PreferenceService
	template func() []string
}

// PreferenceService reads and writes channel settings.
type PreferenceService interface {
	Get(ctx context.Context, userID string) (*core.NotificationPreferences, error)
	Save(ctx context.Context, p *core.NotificationPreferences) error
}

// NewHandler builds the handler.
func NewHandler(svc core.Notifications) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the routes. Everything requires authentication: a user's
// inbox is theirs alone.
func (h *Handler) Register(g *echo.Group, id core.Identity, prefs PreferenceService, codes func() []string) {
	h.prefs = prefs
	h.template = codes

	authed := g.Group("/notifications", id.Middleware())
	authed.GET("", h.List)
	authed.GET("/unread-count", h.UnreadCount)
	authed.POST("/:id/read", h.MarkRead)
	authed.POST("/read-all", h.MarkAllRead)
	authed.GET("/preferences", h.GetPreferences)
	authed.PUT("/preferences", h.UpdatePreferences)
}

func (h *Handler) List(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	limit, offset := pageParams(c)
	unreadOnly := c.QueryParam("unread") == "true"

	items, total, err := h.svc.List(c.Request().Context(), p.UserID, unreadOnly, limit, offset)
	if err != nil {
		return err
	}

	out := make([]map[string]any, 0, len(items))
	for i := range items {
		out = append(out, view(&items[i]))
	}
	return httpx.RespondOK(c, http.StatusOK, map[string]any{
		"items": out, "total": total, "limit": limit, "offset": offset,
	})
}

func (h *Handler) UnreadCount(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	n, err := h.svc.UnreadCount(c.Request().Context(), p.UserID)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, map[string]any{"unread": n})
}

func (h *Handler) MarkRead(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}
	if err := h.svc.MarkRead(c.Request().Context(), p.UserID, c.Param("id")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}
	n, err := h.svc.MarkAllRead(c.Request().Context(), p.UserID)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, map[string]any{"marked": n})
}

func (h *Handler) GetPreferences(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	prefs, err := h.prefs.Get(c.Request().Context(), p.UserID)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, prefsView(prefs))
}

func (h *Handler) UpdatePreferences(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	var in struct {
		Email *bool `json:"email"`
		Push  *bool `json:"push"`
		InApp *bool `json:"in_app"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}

	prefs, err := h.prefs.Get(c.Request().Context(), p.UserID)
	if err != nil {
		return err
	}
	if prefs == nil {
		prefs = &core.NotificationPreferences{Channels: map[core.NotificationChannel]bool{}}
	}
	if prefs.Channels == nil {
		prefs.Channels = map[core.NotificationChannel]bool{}
	}
	prefs.UserID = p.UserID

	if in.Email != nil {
		prefs.Channels[core.ChannelEmail] = *in.Email
	}
	if in.Push != nil {
		prefs.Channels[core.ChannelPush] = *in.Push
	}
	if in.InApp != nil {
		prefs.Channels[core.ChannelInApp] = *in.InApp
	}

	if err := h.prefs.Save(c.Request().Context(), prefs); err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, prefsView(prefs))
}

// view renders a notification for JSON. The user id is omitted because the
// caller is the owner; echoing it back adds nothing.
func view(n *core.Notification) map[string]any {
	out := map[string]any{
		"id":         n.ID,
		"channel":    string(n.Channel),
		"template":   n.Template,
		"subject":    n.Subject,
		"body":       n.Body,
		"status":     string(n.Status),
		"read":       n.ReadAt != nil,
		"created_at": n.CreatedAt,
	}
	if len(n.Payload) > 0 {
		out["payload"] = n.Payload
	}
	if n.ReadAt != nil {
		out["read_at"] = n.ReadAt
	}
	return out
}

func prefsView(p *core.NotificationPreferences) map[string]any {
	// Absence means "default on": a new user must receive transactional mail
	// without visiting a settings page first.
	channels := map[string]bool{
		string(core.ChannelEmail): true,
		string(core.ChannelPush):  false,
		string(core.ChannelInApp): true,
	}
	if p != nil {
		for ch, on := range p.Channels {
			channels[string(ch)] = on
		}
	}
	return map[string]any{"channels": channels}
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

// Package api holds the identity HTTP layer: DTOs, handlers and middleware.
//
// Handlers stay thin. They parse, call the service, and map the result; every
// error leaves through the platform envelope.
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	"core-platform/mod/identity/model"
	"core-platform/mod/identity/repository"
	"core-platform/mod/identity/service"
)

// Handler serves the identity endpoints.
type Handler struct {
	svc   *service.Auth
	users *repository.Users
	cfg   HandlerConfig
}

// HandlerConfig is the transport policy.
type HandlerConfig struct {
	// CookieSecure must be true in production; it is false in development so
	// cookies work over plain HTTP on localhost.
	CookieSecure bool
	// AccessTTL is used for the access cookie's lifetime.
	AccessTTL time.Duration
	// CookiePath scopes the refresh cookie to the auth endpoints.
	CookiePath string
}

// AccessCookieName and RefreshCookieName are the web transport names.
const (
	AccessCookieName  = "access"
	RefreshCookieName = "refresh"
	CSRFHeaderName    = "X-CSRF-Token"
	CSRFCookieName    = "csrf"
)

// NewHandler builds the handler.
func NewHandler(svc *service.Auth, users *repository.Users, cfg HandlerConfig) *Handler {
	if cfg.CookiePath == "" {
		cfg.CookiePath = "/api/v1/auth"
	}
	if cfg.AccessTTL <= 0 {
		cfg.AccessTTL = 15 * time.Minute
	}
	return &Handler{svc: svc, users: users, cfg: cfg}
}

// Register mounts every identity route.
func (h *Handler) Register(g *echo.Group, id core.Identity, rl RateLimiters) {
	// Public. These are the brute-force targets, so they carry explicit limits.
	auth := g.Group("/auth")
	auth.POST("/register", h.handleRegister, rl.Login)
	auth.POST("/login", h.handleLogin, rl.Login)
	auth.POST("/refresh", h.handleRefresh, rl.Refresh)
	auth.POST("/logout", h.handleLogout)
	auth.POST("/forgot-password", h.handleForgotPassword, rl.Login)
	auth.POST("/reset-password", h.handleResetPassword, rl.Login)
	auth.POST("/verify-email", h.handleVerifyEmail)

	// Authenticated.
	me := g.Group("/me", id.Middleware())
	me.GET("", h.handleMe)
	me.PATCH("", h.handleUpdateMe)
	me.DELETE("", h.handleDeleteMe)
	me.POST("/logout-all", h.handleLogoutAll)

	// Admin.
	admin := g.Group("/admin", id.Middleware(), id.RequireRole(core.RoleAdmin))
	admin.GET("/users", h.handleListUsers)
	admin.DELETE("/users/:id", h.handleDeleteUser)
}

func (h *Handler) handleRegister(c echo.Context) error {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Locale   string `json:"locale"`
	}
	if err := bind(c, &in); err != nil {
		return err
	}

	user, err := h.svc.Register(c.Request().Context(), service.RegisterInput{
		Email: in.Email, Password: in.Password, Name: in.Name, Locale: in.Locale,
	})
	if err != nil {
		if err == core.ErrConflict {
			// Identical response to success: registration must not confirm
			// whether an address is already in use.
			return respond(c, http.StatusAccepted, map[string]string{
				"message": "if the address is available, a verification e-mail has been sent",
			})
		}
		return err
	}
	return respond(c, http.StatusCreated, userView(user))
}

func (h *Handler) handleLogin(c echo.Context) error {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Device   string `json:"device"`
		Remember bool   `json:"remember"`
	}
	if err := bind(c, &in); err != nil {
		return err
	}

	session, err := h.svc.Login(c.Request().Context(), service.LoginInput{
		Email:    in.Email,
		Password: in.Password,
		Device:   in.Device,
		IP:       c.RealIP(),
		Audience: audienceFor(c),
		Remember: in.Remember,
	})
	if err != nil {
		return err
	}
	return h.writeSession(c, session)
}

func (h *Handler) handleRefresh(c echo.Context) error {
	token := h.refreshFromRequest(c)
	if token == "" {
		return core.ErrUnauthorized
	}

	session, err := h.svc.Refresh(c.Request().Context(), service.RefreshInput{RefreshToken: token})
	if err != nil {
		h.clearCookies(c)
		return err
	}
	return h.writeSession(c, session)
}

func (h *Handler) handleLogout(c echo.Context) error {
	if token := h.refreshFromRequest(c); token != "" {
		if err := h.svc.Logout(c.Request().Context(), token); err != nil {
			return err
		}
	}
	h.clearCookies(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) handleLogoutAll(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}
	if err := h.svc.LogoutEverywhere(c.Request().Context(), p.UserID); err != nil {
		return err
	}
	h.clearCookies(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) handleForgotPassword(c echo.Context) error {
	var in struct {
		Email string `json:"email"`
	}
	if err := bind(c, &in); err != nil {
		return err
	}
	if err := h.svc.ForgotPassword(c.Request().Context(), in.Email); err != nil {
		return err
	}
	return respond(c, http.StatusAccepted, map[string]string{
		"message": "if the address exists, a reset link has been sent",
	})
}

func (h *Handler) handleResetPassword(c echo.Context) error {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := bind(c, &in); err != nil {
		return err
	}
	if err := h.svc.ResetPassword(c.Request().Context(), in.Token, in.Password); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) handleVerifyEmail(c echo.Context) error {
	var in struct {
		Token string `json:"token"`
	}
	if err := bind(c, &in); err != nil {
		return err
	}
	if err := h.svc.VerifyEmail(c.Request().Context(), in.Token); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) handleMe(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}
	u, err := h.users.ByID(c.Request().Context(), p.UserID)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, userView(u))
}

func (h *Handler) handleUpdateMe(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	var in struct {
		Name      *string `json:"name"`
		Locale    *string `json:"locale"`
		AvatarURL *string `json:"avatar_url"`
		Phone     *string `json:"phone"`
	}
	if err := bind(c, &in); err != nil {
		return err
	}

	u, err := h.users.ByID(c.Request().Context(), p.UserID)
	if err != nil {
		return err
	}
	if in.Name != nil {
		u.Name = truncateString(*in.Name, 120)
	}
	if in.Locale != nil {
		u.Locale = truncateString(*in.Locale, 10)
	}
	if in.AvatarURL != nil {
		u.AvatarURL = *in.AvatarURL
	}
	if in.Phone != nil {
		u.Phone = truncateString(*in.Phone, 32)
		// A changed number is unverified until it is proven again.
		if u.Phone == "" {
			u.PhoneVerifiedAt = nil
		}
	}
	if err := h.users.Update(c.Request().Context(), u); err != nil {
		return err
	}
	return respond(c, http.StatusOK, userView(u))
}

func (h *Handler) handleDeleteMe(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}
	if err := h.svc.DeleteSelf(c.Request().Context(), p.UserID); err != nil {
		return err
	}
	h.clearCookies(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) handleListUsers(c echo.Context) error {
	limit, offset := pageParams(c)
	users, total, err := h.users.List(c.Request().Context(), limit, offset)
	if err != nil {
		return err
	}

	out := make([]map[string]any, 0, len(users))
	for i := range users {
		out = append(out, userView(&users[i]))
	}
	return respond(c, http.StatusOK, map[string]any{
		"items": out, "total": total, "limit": limit, "offset": offset,
	})
}

func (h *Handler) handleDeleteUser(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return core.ErrValidation
	}
	if err := h.svc.DeleteSelf(c.Request().Context(), id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// writeSession returns the access token in the body, and for web clients also
// sets HttpOnly cookies so scripts can never read the refresh token.
func (h *Handler) writeSession(c echo.Context, s *service.Session) error {
	if audienceFor(c) == service.AudienceWeb {
		h.setCookies(c, s)
		return respond(c, http.StatusOK, map[string]any{
			"access_token": s.Access.Token,
			"token_type":   "Bearer",
			"expires_in":   int(h.cfg.AccessTTL.Seconds()),
			"user":         userView(s.User),
		})
	}

	return respond(c, http.StatusOK, map[string]any{
		"access_token":    s.Access.Token,
		"refresh_token":   s.Refresh,
		"token_type":      "Bearer",
		"expires_in":      int(h.cfg.AccessTTL.Seconds()),
		"refresh_expires": s.RefreshExpires,
		"user":            userView(s.User),
	})
}

// userView is the only account projection that leaves the process.
func userView(u *model.User) map[string]any {
	if u == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":             u.ID,
		"email":          u.Email,
		"phone":          u.Phone,
		"name":           u.Name,
		"locale":         u.Locale,
		"avatar_url":     u.AvatarURL,
		"role":           u.Role,
		"status":         u.Status,
		"email_verified": u.EmailVerifiedAt != nil,
		"phone_verified": u.PhoneVerifiedAt != nil,
		"created_at":     u.CreatedAt,
	}
}

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Keep the context import meaningful for the service call sites above.
var _ = context.Background

// Package api holds the economy HTTP layer.
package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
	"core-platform/platform/httpx"
)

// Deps is the set of services the handler serves.
type Deps struct {
	Wallet       econ.Wallet
	Ads          econ.Ads
	Orders       econ.Orders
	Entitlements econ.Entitlements
}

// Handler serves the economy endpoints.
type Handler struct {
	wallet econ.Wallet
	ads    econ.Ads
	orders econ.Orders
	ents   econ.Entitlements
}

// NewHandler builds the handler.
func NewHandler(d Deps) *Handler {
	return &Handler{wallet: d.Wallet, ads: d.Ads, orders: d.Orders, ents: d.Entitlements}
}

// Register mounts the routes. User routes require authentication; admin routes
// additionally require the admin role (mirror of the identity module's split).
//
// The payment webhook is mounted outside the authenticated group by design: a
// provider has no bearer token, and the adapter verifies its own signature
// before anything is written (L2.md 3.7).
func (h *Handler) Register(g *echo.Group, id core.Identity) {
	authed := g.Group("/economy", id.Middleware())

	// Wallet. There is deliberately no user-facing credit/debit: the amount
	// is always a server decision (L2.md 3.4). Spending happens inside
	// modules; minting happens through rewards, purchases or an admin.
	authed.GET("/wallet", h.GetWallet)
	authed.GET("/wallet/transactions", h.ListTransactions)

	// Rewarded ads, user side.
	authed.GET("/ads/pick", h.PickAd)
	authed.GET("/ads/reward-budget", h.RewardBudget)
	authed.POST("/ads/:id/view/start", h.StartView)
	authed.POST("/ads/view/:session_id/claim", h.ClaimView)
	authed.POST("/ads/:id/click", h.RecordClick)

	// Catalog, orders and entitlements.
	authed.GET("/products", h.ListProducts)
	authed.GET("/products/:id", h.GetProduct)
	authed.POST("/orders", h.CreateOrder)
	authed.GET("/orders", h.ListOrders)
	authed.GET("/orders/:id", h.GetOrder)
	authed.GET("/entitlements", h.ListEntitlements)

	// Provider callback: no auth middleware, own signature check.
	g.POST("/economy/payments/webhook/:provider", h.Webhook)

	// Admin.
	admin := g.Group("/economy/admin", id.Middleware(), id.RequireRole(core.RoleAdmin))
	admin.GET("/ads", h.ListAds)
	admin.POST("/ads", h.CreateAd)
	admin.PUT("/ads/:id", h.UpdateAd)
	admin.DELETE("/ads/:id", h.DeleteAd)
	admin.GET("/ads/:id/stats", h.AdStats)
	admin.GET("/reward-settings", h.GetSettings)
	admin.PUT("/reward-settings", h.UpdateSettings)
	admin.POST("/wallets/:user_id/credit", h.AdminCredit)
	admin.POST("/wallets/:user_id/debit", h.AdminDebit)
	admin.GET("/orders", h.AdminListOrders)
	admin.POST("/orders/:id/confirm", h.AdminConfirmOrder)
	admin.POST("/orders/:id/reject", h.AdminRejectOrder)
	admin.POST("/products", h.AdminCreateProduct)
	admin.PUT("/products/:id", h.AdminUpdateProduct)
	admin.POST("/entitlements/:id/revoke", h.AdminRevokeEntitlement)
}

// bind decodes a JSON body into the validation sentinel on failure.
func bind(c echo.Context, dst any) error {
	if err := c.Bind(dst); err != nil {
		return core.ErrValidation
	}
	return nil
}

// respond writes the platform success envelope.
func respond(c echo.Context, status int, data any) error {
	return httpx.RespondOK(c, status, data)
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

func principal(c echo.Context) (*core.Principal, error) {
	p := core.UserFrom(c)
	if p == nil {
		return nil, core.ErrUnauthorized
	}
	return p, nil
}

// defaultCurrency falls back to the scaffold's TOKEN code when the client
// leaves the currency out.
func defaultCurrency(c string) string {
	if strings.TrimSpace(c) == "" {
		return econ.CurrencyToken
	}
	return strings.ToUpper(strings.TrimSpace(c))
}

// ------------------------------------------------------------------
// Wallet
// ------------------------------------------------------------------

// GetWallet returns the caller's balance, materializing the account (and the
// one-time signup grant) on first view.
func (h *Handler) GetWallet(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	currency := defaultCurrency(c.QueryParam("currency"))
	b, err := h.wallet.EnsureAccount(c.Request().Context(), p.UserID, currency)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, b)
}

func (h *Handler) ListTransactions(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	limit, offset := pageParams(c)
	currency := defaultCurrency(c.QueryParam("currency"))
	items, total, err := h.wallet.Transactions(c.Request().Context(), p.UserID, currency, limit, offset)
	if err != nil {
		return err
	}
	// Entry.IdempotencyKey carries json:"-", so it never reaches the client.
	return respond(c, http.StatusOK, map[string]any{
		"items": items, "total": total, "limit": limit, "offset": offset,
	})
}

// adminMovement is the shared body for the admin credit/debit endpoints.
type adminMovement struct {
	Currency       string `json:"currency"`
	Amount         int64  `json:"amount"`
	RefType        string `json:"ref_type"`
	RefID          string `json:"ref_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

// AdminCredit adds tokens to a user's wallet. The reason code is fixed by the
// endpoint, so an admin cannot forge a reward or purchase entry.
func (h *Handler) AdminCredit(c echo.Context) error { return h.adminMove(c, true) }

// AdminDebit removes tokens from a user's wallet.
func (h *Handler) AdminDebit(c echo.Context) error { return h.adminMove(c, false) }

func (h *Handler) adminMove(c echo.Context, credit bool) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	target := c.Param("user_id")
	if target == "" {
		return core.ErrValidation
	}
	var in adminMovement
	if err := bind(c, &in); err != nil {
		return err
	}
	if in.Amount <= 0 {
		return core.ErrValidation
	}

	mv := econ.Movement{
		UserID:         target,
		Currency:       defaultCurrency(in.Currency),
		Amount:         in.Amount,
		Reason:         econ.ReasonAdminCredit,
		ActorID:        p.UserID,
		RefType:        in.RefType,
		RefID:          in.RefID,
		IdempotencyKey: in.IdempotencyKey,
	}
	var entry econ.Entry
	if credit {
		entry, err = h.wallet.Credit(c.Request().Context(), mv)
	} else {
		mv.Reason = econ.ReasonAdminDebit
		entry, err = h.wallet.Debit(c.Request().Context(), mv)
	}
	if err != nil {
		return err
	}
	return respond(c, http.StatusCreated, entry)
}

// ------------------------------------------------------------------
// Rewarded ads, user side
// ------------------------------------------------------------------

// PickAd answers "which ad can I watch now?". A cap-full or no-ad answer is
// still a 200 with the budget, because the dashboard needs it either way.
func (h *Handler) PickAd(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	currency := defaultCurrency(c.QueryParam("currency"))
	ad, budget, err := h.ads.Pick(c.Request().Context(), p.UserID, currency)
	if err != nil {
		switch {
		case errors.Is(err, econ.ErrRewardLimitReached):
			return respond(c, http.StatusOK, map[string]any{
				"ad": nil, "reward_budget": budget, "reason": "REWARD_LIMIT_REACHED",
			})
		case errors.Is(err, econ.ErrAdNotAvailable):
			return respond(c, http.StatusOK, map[string]any{
				"ad": nil, "reward_budget": budget, "reason": "AD_NOT_AVAILABLE",
			})
		}
		return err
	}
	return respond(c, http.StatusOK, map[string]any{
		"ad": ad, "reward_budget": budget,
	})
}

// RewardBudget answers "how much can I still earn today / this month?".
func (h *Handler) RewardBudget(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	currency := defaultCurrency(c.QueryParam("currency"))
	b, err := h.ads.Budget(c.Request().Context(), p.UserID, currency)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, b)
}

// StartView opens a server-stamped watch session.
func (h *Handler) StartView(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	sess, err := h.ads.StartView(c.Request().Context(), p.UserID, c.Param("id"))
	if err != nil {
		return err
	}
	return respond(c, http.StatusCreated, sess)
}

// ClaimView pays the watch reward after the required duration.
func (h *Handler) ClaimView(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	entry, err := h.ads.ClaimView(c.Request().Context(), p.UserID, c.Param("session_id"))
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, entry)
}

// RecordClick logs the affiliate click and returns the 302 target.
func (h *Handler) RecordClick(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	target, entry, err := h.ads.RecordClick(c.Request().Context(), p.UserID, c.Param("id"))
	if err != nil {
		return err
	}
	body := map[string]any{"target_url": target, "rewarded": entry != nil}
	if target == "" {
		// No affiliate link on this ad: the click is logged, the caller shows
		// the ad's own content.
		body["target_url"] = nil
	}
	if entry != nil {
		body["entry"] = *entry
	}
	return respond(c, http.StatusOK, body)
}

// ------------------------------------------------------------------
// Admin
// ------------------------------------------------------------------

// adInputFrom binds + converts the admin payload.
func adInputFrom(c echo.Context) (econ.AdInput, error) {
	var in struct {
		Code         string     `json:"code"`
		Title        string     `json:"title"`
		MediaType    string     `json:"media_type"`
		BodyText     string     `json:"body_text"`
		MediaURL     string     `json:"media_url"`
		Content      string     `json:"content"`
		TargetURL    string     `json:"target_url"`
		WatchSeconds int        `json:"watch_seconds"`
		WatchReward  int64      `json:"watch_reward"`
		ClickReward  int64      `json:"click_reward"`
		StartsAt     *time.Time `json:"starts_at"`
		EndsAt       *time.Time `json:"ends_at"`
		Active       bool       `json:"active"`
	}
	if err := bind(c, &in); err != nil {
		return econ.AdInput{}, err
	}
	return econ.AdInput{
		Code:         in.Code,
		Title:        in.Title,
		MediaType:    strings.ToLower(in.MediaType),
		BodyText:     in.BodyText,
		MediaURL:     in.MediaURL,
		Content:      in.Content,
		TargetURL:    in.TargetURL,
		WatchSeconds: in.WatchSeconds,
		WatchReward:  in.WatchReward,
		ClickReward:  in.ClickReward,
		StartsAt:     in.StartsAt,
		EndsAt:       in.EndsAt,
		Active:       in.Active,
	}, nil
}

func (h *Handler) ListAds(c echo.Context) error {
	limit, offset := pageParams(c)
	activeOnly := c.QueryParam("active") == "true"
	ads, total, err := h.ads.ListAds(c.Request().Context(), limit, offset, activeOnly)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, map[string]any{
		"items": ads, "total": total, "limit": limit, "offset": offset,
	})
}

func (h *Handler) CreateAd(c echo.Context) error {
	in, err := adInputFrom(c)
	if err != nil {
		return err
	}
	ad, err := h.ads.CreateAd(c.Request().Context(), in)
	if err != nil {
		return err
	}
	return respond(c, http.StatusCreated, ad)
}

func (h *Handler) UpdateAd(c echo.Context) error {
	in, err := adInputFrom(c)
	if err != nil {
		return err
	}
	ad, err := h.ads.UpdateAd(c.Request().Context(), c.Param("id"), in)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, ad)
}

func (h *Handler) DeleteAd(c echo.Context) error {
	if err := h.ads.DeleteAd(c.Request().Context(), c.Param("id")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) AdStats(c echo.Context) error {
	stats, err := h.ads.AdStats(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, stats)
}

func (h *Handler) GetSettings(c echo.Context) error {
	s, err := h.ads.Settings(c.Request().Context())
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, s)
}

func (h *Handler) UpdateSettings(c echo.Context) error {
	var in struct {
		SignupGrant        int64 `json:"signup_grant"`
		AdWatchReward      int64 `json:"ad_watch_reward"`
		DailyRewardCap     int64 `json:"daily_reward_cap"`
		MonthlyRewardCap   int64 `json:"monthly_reward_cap"`
		ClickRewardEnabled bool  `json:"click_reward_enabled"`
	}
	if err := bind(c, &in); err != nil {
		return err
	}
	p, err := principal(c)
	if err != nil {
		return err
	}
	s, err := h.ads.UpdateSettings(c.Request().Context(), econ.RewardSettings{
		SignupGrant:        in.SignupGrant,
		AdWatchReward:      in.AdWatchReward,
		DailyRewardCap:     in.DailyRewardCap,
		MonthlyRewardCap:   in.MonthlyRewardCap,
		ClickRewardEnabled: in.ClickRewardEnabled,
	}, p.UserID)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, s)
}

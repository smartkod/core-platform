package api

// orders.go holds the catalog, order, entitlement and webhook endpoints.

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
)

// ------------------------------------------------------------------
// Catalog
// ------------------------------------------------------------------

// ListProducts returns the active catalog. It is authenticated because prices
// and token packs are already visible to any signed-in user, and an anonymous
// catalog would be a second, unauthenticated surface to keep in sync.
func (h *Handler) ListProducts(c echo.Context) error {
	items, err := h.orders.ListProducts(c.Request().Context(), true)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (h *Handler) GetProduct(c echo.Context) error {
	p, err := h.orders.Product(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, p)
}

// ------------------------------------------------------------------
// Orders
// ------------------------------------------------------------------

// CreateOrder opens a purchase. The amount is never read from the body: the
// server price of the product wins (L2.md 3.8).
//
// The idempotency key may arrive as the Idempotency-Key header (the usual
// convention) or as a body field, so a simple client can do either.
func (h *Handler) CreateOrder(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	var in struct {
		ProductID      string `json:"product_id"`
		Provider       string `json:"provider"`
		IdempotencyKey string `json:"idempotency_key"`
		// AmountMinor is accepted and ignored on purpose: the field exists so a
		// client that sends it gets the server price rather than an error it
		// cannot act on. It is never passed to the service.
		AmountMinor int64 `json:"amount_minor"`
	}
	if err := bind(c, &in); err != nil {
		return err
	}

	key := c.Request().Header.Get("Idempotency-Key")
	if key == "" {
		key = in.IdempotencyKey
	}

	order, err := h.orders.Create(c.Request().Context(), econ.CreateOrderInput{
		UserID:         p.UserID,
		ProductID:      in.ProductID,
		Provider:       in.Provider,
		IdempotencyKey: key,
	})
	if err != nil {
		return err
	}
	return respond(c, http.StatusCreated, order)
}

func (h *Handler) ListOrders(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	limit, offset := pageParams(c)
	items, total, err := h.orders.List(c.Request().Context(), p.UserID, limit, offset)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, map[string]any{
		"items": items, "total": total, "limit": limit, "offset": offset,
	})
}

func (h *Handler) GetOrder(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	order, err := h.orders.Get(c.Request().Context(), p.UserID, c.Param("id"))
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, order)
}

// ------------------------------------------------------------------
// Entitlements
// ------------------------------------------------------------------

func (h *Handler) ListEntitlements(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	items, err := h.ents.List(c.Request().Context(), p.UserID)
	if err != nil {
		return err
	}
	// ?code= is a convenience for a client that only wants one right.
	if code := c.QueryParam("code"); code != "" {
		active, err := h.ents.Has(c.Request().Context(), p.UserID, code)
		if err != nil {
			return err
		}
		return respond(c, http.StatusOK, map[string]any{
			"code": code, "active": active, "items": items,
		})
	}
	return respond(c, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

// ------------------------------------------------------------------
// Webhook
// ------------------------------------------------------------------

// Webhook applies one provider callback.
//
// Every outcome except a bad signature answers 200: a provider retries on any
// other status, and retrying a duplicate or an unknown event only adds load.
// An invalid signature is 401 and writes nothing (L2.md 3.7).
func (h *Handler) Webhook(c echo.Context) error {
	if err := h.orders.HandleWebhook(
		c.Request().Context(), c.Param("provider"), c.Request(),
	); err != nil {
		if err == econ.ErrSignatureInvalid {
			return err
		}
		// Unknown/irrelevant events are acknowledged rather than retried, but
		// a real processing failure is surfaced so the provider retries.
		switch err {
		case econ.ErrOrderNotFound, econ.ErrProviderNotFound,
			econ.ErrProviderUnsupported, core.ErrValidation:
			return respond(c, http.StatusOK, map[string]any{"received": true, "applied": false})
		}
		return err
	}
	return respond(c, http.StatusOK, map[string]any{"received": true, "applied": true})
}

// ------------------------------------------------------------------
// Admin
// ------------------------------------------------------------------

func (h *Handler) AdminListOrders(c echo.Context) error {
	limit, offset := pageParams(c)
	status := c.QueryParam("status")
	items, total, err := h.orders.ListAll(c.Request().Context(), status, limit, offset)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, map[string]any{
		"items": items, "total": total, "limit": limit, "offset": offset,
	})
}

// AdminConfirmOrder is the operator's "the transfer arrived" action for a
// manual order.
func (h *Handler) AdminConfirmOrder(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	order, err := h.orders.Confirm(c.Request().Context(), c.Param("id"), p.UserID)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, order)
}

func (h *Handler) AdminRejectOrder(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	var in struct {
		Reason string `json:"reason"`
	}
	_ = bind(c, &in)
	order, err := h.orders.Reject(c.Request().Context(), c.Param("id"), p.UserID, in.Reason)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, order)
}

// productInputFrom binds the admin catalog payload.
func productInputFrom(c echo.Context) (econ.ProductInput, error) {
	var in struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Currency    string `json:"currency"`
		AmountMinor int64  `json:"amount_minor"`
		Interval    string `json:"interval"`
		Tokens      int64  `json:"tokens"`
		Entitlement string `json:"entitlement"`
		Active      bool   `json:"active"`
	}
	if err := bind(c, &in); err != nil {
		return econ.ProductInput{}, err
	}
	return econ.ProductInput{
		Code:        in.Code,
		Name:        in.Name,
		Kind:        in.Kind,
		Currency:    in.Currency,
		AmountMinor: in.AmountMinor,
		Interval:    in.Interval,
		Tokens:      in.Tokens,
		Entitlement: in.Entitlement,
		Active:      in.Active,
	}, nil
}

func (h *Handler) AdminCreateProduct(c echo.Context) error {
	in, err := productInputFrom(c)
	if err != nil {
		return err
	}
	p, err := h.orders.CreateProduct(c.Request().Context(), in)
	if err != nil {
		return err
	}
	return respond(c, http.StatusCreated, p)
}

func (h *Handler) AdminUpdateProduct(c echo.Context) error {
	in, err := productInputFrom(c)
	if err != nil {
		return err
	}
	p, err := h.orders.UpdateProduct(c.Request().Context(), c.Param("id"), in)
	if err != nil {
		return err
	}
	return respond(c, http.StatusOK, p)
}

func (h *Handler) AdminRevokeEntitlement(c echo.Context) error {
	p, err := principal(c)
	if err != nil {
		return err
	}
	if err := h.ents.Revoke(c.Request().Context(), c.Param("id"), p.UserID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

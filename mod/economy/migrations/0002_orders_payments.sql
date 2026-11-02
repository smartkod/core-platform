-- 0002_orders_payments: fulfillment idempotency for orders.
--
-- 0001 created the catalog/order/payment/entitlement tables. The missing piece
-- was the link that makes fulfillment replay-safe: an entitlement is granted at
-- most once per order. Without order_id the only guard was the application
-- code, and a retried webhook could grant twice.

ALTER TABLE entitlements
    ADD COLUMN order_id uuid NULL REFERENCES orders (id);

-- One active grant per order. A revoked row may be re-granted (a refund then a
-- manual correction), which is why the index is partial.
CREATE UNIQUE INDEX entitlements_order_uniq ON entitlements (order_id)
    WHERE order_id IS NOT NULL AND revoked_at IS NULL;

-- A paid order has exactly one successful payment; a retried webhook must find
-- the existing row rather than append a second. The provider reference is
-- already unique per provider, but a manual confirmation has no provider ref.
CREATE UNIQUE INDEX payments_order_paid_uniq ON payments (order_id)
    WHERE status = 'paid';

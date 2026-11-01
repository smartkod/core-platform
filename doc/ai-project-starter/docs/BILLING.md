# BILLING CONTRACT

## Billing principle

Payment correctness takes priority over UI convenience.

## Source of truth

Stripe owns Stripe payment/subscription lifecycle state.

The application database owns application-specific relationships and entitlement state derived from verified billing information.

## Checkout

Client may request a product/plan identifier.

Server must resolve the authoritative Stripe price/product configuration.

Never accept an arbitrary client-supplied amount as authoritative.

## Webhooks

Webhook processing:

1. receive event;
2. verify Stripe signature;
3. check idempotency;
4. validate event type;
5. map event to application state;
6. update database transactionally where required;
7. record successful processing;
8. return success.

Webhook handlers must tolerate retries.

## Redirects

A successful checkout redirect is not sufficient proof of durable payment state.

Use verified server-side state/webhooks for entitlement decisions.

## Subscription state

Define explicit state transitions before implementation.

Example:

trialing → active
active → past_due
active → canceled
past_due → active

Do not invent transitions inside UI code.

## Refunds / disputes / failures

Define the application behavior for:
- payment failure
- cancellation
- refund
- dispute
- expired card
- incomplete checkout

before exposing corresponding paid entitlements.

## Testing

Critical billing tests must include:
- valid webhook
- invalid signature
- duplicate webhook
- out-of-order/retry scenario where relevant
- canceled subscription
- failed payment
- entitlement revocation/update

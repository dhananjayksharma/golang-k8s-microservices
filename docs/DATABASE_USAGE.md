# Order Service — Database and Messaging Usage

## PostgreSQL: source of truth

Use PostgreSQL for data that must remain durable and transactionally consistent:

- orders
- order items
- order status history
- idempotency records as durable fallback
- transactional outbox events

Do not use Redis as the authoritative order store.

## Redis: ephemeral coordination and acceleration

Use Redis for:

- idempotency request locks and cached completed responses;
- optional order-read cache with a short TTL;
- distributed rate limiting if Kong is not doing it;
- temporary workflow/session data.

Suggested keys:

```text
order:idem:<idempotency-key>
order:cache:<order-id>
order:ratelimit:<client-id>:<window>
```

Rules:

- Always set TTLs.
- Treat cache misses as normal.
- On Redis failure, preserve correctness by falling back to PostgreSQL or returning a controlled dependency error for create operations.

## RabbitMQ: asynchronous events

Use RabbitMQ for event distribution, not as the order database.

Exchange:

```text
orders.events (topic, durable)
```

Routing keys:

```text
order.created.v1
order.status.changed.v1
order.cancelled.v1
```

Suggested queues:

```text
inventory.order-events
payment.order-events
message.order-events
```

Each consumer should maintain its own queue and dead-letter queue.

## Transactional outbox

Publishing directly after a database write creates a dual-write failure:

- database succeeds, broker fails; or
- broker succeeds, database fails.

The outbox pattern writes the business change and event record in one PostgreSQL transaction. A worker publishes the event afterward.

## Database ownership across this repository

Recommended ownership boundaries:

| Service | Owned data |
|---|---|
| Order Service | orders, items, order status history, order outbox |
| Inventory Service | products, stock, reservations |
| Payment Service | payment attempts, provider references, payment status |
| Invoice Service | invoices and invoice documents |
| Cart Service | active carts, cart items, cart outbox |
| Message Service | notification/message delivery records or read models |

Services should not directly query each other's tables. Integrate through APIs or events.

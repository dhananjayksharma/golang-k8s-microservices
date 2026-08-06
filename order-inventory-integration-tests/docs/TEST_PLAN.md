# Integration Test Plan

## Purpose

These tests verify behavior that unit tests cannot prove:

- PostgreSQL migrations match Order Service queries.
- GORM mappings match MySQL tables.
- RabbitMQ exchange, queue, binding, routing key and acknowledgement behavior work.
- Redis-backed deduplication does not become the only consistency guarantee.
- Order and inventory state converge through eventual consistency.
- MySQL row locking prevents overselling under concurrency.

## System under test

```text
HTTP client
   |
   v
Order Service ---- PostgreSQL
   |
   | order.created
   v
RabbitMQ
   |
   v
Inventory Service ---- MySQL/GORM
   |
   +---- Redis deduplication
   |
   | inventory.reserved / inventory.rejected
   v
RabbitMQ
   |
   v
Order Service ---- PostgreSQL status update
```

## Assertions

### Successful flow

- POST returns HTTP 201.
- Order initially has `PENDING` status.
- Calculated subtotal and total are correct.
- A reservation is created exactly once.
- Stock is reduced by the requested quantity.
- Order eventually becomes `CONFIRMED`.

### Insufficient stock

- Order is persisted.
- No reservation is created.
- Stock remains unchanged.
- Order eventually becomes `CANCELLED`.

### Duplicate event

- Repeated delivery does not duplicate the reservation.
- Stock is reduced once.
- Database uniqueness remains the durable protection even if Redis is unavailable.

### Concurrency

- Available stock never becomes negative.
- Confirmed quantity never exceeds available stock.
- Remaining orders are rejected and cancelled.

## Known architecture gap exposed by tests

Order Service currently commits an order and then publishes RabbitMQ separately. Inventory Service similarly commits reservation data and then publishes the result. A process failure between these operations can leave inconsistent state. A transactional outbox should be the next production enhancement for both services.

# Integration Testing Interview Q&A

## Why are these integration tests valuable?

They test real protocols and persistence boundaries instead of mocks: HTTP, PostgreSQL, MySQL/GORM, RabbitMQ and Redis. They catch schema mismatches, routing-key errors, transaction issues, message redelivery problems and race conditions.

## How is eventual consistency tested?

The test creates an order, then polls the order API with a bounded timeout until the asynchronous inventory result changes the status. Fixed sleeps are avoided because message-processing latency varies.

## Why test duplicate events?

RabbitMQ provides at-least-once delivery when acknowledgements and retries are used. Consumers must therefore be idempotent. The test proves that a duplicate event creates one reservation and reduces stock once.

## Why are Redis and a database unique constraint both used?

Redis is a fast optimization, but keys can expire or Redis can be unavailable. The unique constraint on reservation `order_id` is the durable final guarantee.

## Why test concurrency?

Two requests can read the same stock at the same time. The Inventory Service uses a GORM transaction and `SELECT ... FOR UPDATE`; the concurrent test proves that stock does not become negative and overselling does not occur.

## What is the transactional outbox gap?

A database commit and RabbitMQ publish are two independent operations. If the process crashes after committing but before publishing, the event is lost. An outbox stores business data and the event in one transaction, then a worker publishes it reliably.

## Why use real containers instead of mocks?

Mocks cannot validate exchange declarations, queue bindings, SQL syntax, database constraints, GORM mapping, transaction isolation or driver behavior. Containers provide deterministic versions of real infrastructure.

## Why use a build tag?

The `integration` build tag keeps slower environment-dependent tests separate from normal unit tests:

```bash
go test ./...
go test -tags=integration ./integration/...
```

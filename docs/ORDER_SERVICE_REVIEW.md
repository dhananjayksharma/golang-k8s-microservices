# Review of Uploaded Order Service

## Critical findings

1. `orderService.orders` is an unprotected slice. Concurrent reads and writes can cause data races.
2. IDs use `len(orders)+1`; deleting an order can cause a later create to reuse an existing ID.
3. All data is lost whenever the process or pod restarts.
4. Every order-service replica would hold a different dataset, so horizontal scaling would produce inconsistent responses.
5. RabbitMQ connection details are hard-coded.
6. A new RabbitMQ TCP connection and channel are created for every order, adding latency and failure risk.
7. `PublishOrderEvent` errors are ignored, so the API can report success even when the event is lost.
8. The event is unstructured text rather than a versioned JSON contract.
9. Invalid path IDs are silently converted to zero because `strconv.Atoi` errors are ignored.
10. Create returns `200 OK` instead of `201 Created`.
11. No input validation tags are present.
12. `GetAll` is unbounded and has no pagination.
13. No health, readiness, metrics, timeouts, graceful shutdown, tests, or deployment manifest exists for this service.

## Recommended implementation order

1. Fix correctness: repository, PostgreSQL, UUIDs, validation, errors.
2. Add idempotency and transaction boundaries.
3. Add outbox and broker worker.
4. Add observability and graceful shutdown.
5. Add Kubernetes deployment and load tests.
6. Optimize only after the 10-RPS benchmark identifies a bottleneck.

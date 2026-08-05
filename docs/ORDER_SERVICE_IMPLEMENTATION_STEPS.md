# Order Service — Implementation Steps

## Phase 0: Baseline the uploaded code

1. Run `go test ./...` inside `order-service`.
2. Run the current service and execute `scripts/smoke-current.sh`.
3. Run `loadtest/current-service-10rps.js` only as a baseline.
4. Observe that data is lost after restart and that concurrent writes are unsafe.

## Phase 1: Introduce durable domain and repository layers

1. Create the target `cmd/internal` package structure from the LLD.
2. Replace `float64` price with integer minor units.
3. Add UUID IDs, customer ID, order items, status, version, and timestamps.
4. Define repository interfaces and service-level errors.
5. Preserve the existing endpoints temporarily or add versioned `/api/v1` endpoints.

## Phase 2: PostgreSQL

1. Start dependencies:

```bash
cd order-service/deploy/local
docker compose up -d
```

2. Confirm PostgreSQL, Redis, and RabbitMQ health.
3. Apply migrations with a migration tool such as `golang-migrate`:

```bash
migrate -path order-service/migrations \
  -database 'postgres://order_user:order_any@localhost:5432/order_db?sslmode=disable' up
```

4. Implement PostgreSQL repository methods.
5. Add integration tests using a disposable test database.
6. Configure the pool at 15 open and 5 idle connections per pod initially.

## Phase 3: Idempotency and Redis

1. Require `Idempotency-Key` for create requests.
2. Hash the canonical request body.
3. Use Redis `SET key value NX EX <ttl>` for processing ownership.
4. Store the completed response with a 24-hour TTL.
5. Return `409` when the same key is reused with a different request body.
6. Keep a durable idempotency record in PostgreSQL if strict replay across Redis loss is required.

## Phase 4: RabbitMQ and outbox

1. Declare durable topic exchange `orders.events`.
2. Insert outbox rows in the same transaction as order changes.
3. Build an outbox worker using `FOR UPDATE SKIP LOCKED`.
4. Publish persistent JSON messages.
5. Mark rows as published only after broker confirmation.
6. Add retry, next-attempt timestamp, and dead state.
7. Configure downstream queues and DLQs.

## Phase 5: Production HTTP behavior

1. Use an explicit `http.Server` with timeouts.
2. Add request IDs and structured JSON logs.
3. Add recovery, body-size limit, and request timeout middleware.
4. Add `/health/live`, `/health/ready`, and `/metrics`.
5. Return consistent error responses.
6. Handle SIGTERM and graceful shutdown.

## Phase 6: Container and Minikube

1. Add a multi-stage Dockerfile.
2. Build inside Minikube:

```bash
eval "$(minikube docker-env)"
docker build -t order-service:local ./order-service
```

3. Deploy infrastructure or point the service to existing dependencies.
4. Replace example secrets in `deploy/k8s/order-service.yaml`.
5. Apply resources:

```bash
kubectl apply -f order-service/deploy/k8s/order-service.yaml
kubectl rollout status deployment/order-service
```

6. Port-forward for initial testing:

```bash
kubectl port-forward service/order-service 8081:8081
```

## Phase 7: 10-RPS validation

1. Run a 1-RPS smoke test for 2 minutes.
2. Run 5 RPS for 5 minutes.
3. Run the target:

```bash
k6 run -e BASE_URL=http://localhost:8081 \
  order-service/loadtest/order-service-10rps.js
```

4. Validate P95, P99, errors, CPU, memory, goroutines, DB pool, outbox lag, and broker failures.
5. Repeat while deleting one order-service pod.
6. Repeat the idempotency test with duplicate keys.
7. Save test output as benchmark evidence.

## Phase 8: Tuning rules

- Do not increase replicas before finding the bottleneck.
- Increase DB connections only when pool wait time is high and PostgreSQL has capacity.
- Add indexes based on query plans, not guesses.
- Use `EXPLAIN (ANALYZE, BUFFERS)` for slow queries.
- Keep RabbitMQ publishing outside request transactions.
- Measure HPA behavior using CPU and, later, request-rate metrics.

## Definition of done

- Data survives pod and database-client restarts.
- No race detector findings in service code.
- Create requests are idempotent.
- Event records are not lost when RabbitMQ is unavailable.
- 10 RPS is sustained for 10 minutes with <1% errors.
- P95 is <300 ms and P99 is <500 ms in the agreed local environment.
- Kubernetes probes, HPA, PDB, metrics, and logs are working.
- Database migrations have both up and down scripts.

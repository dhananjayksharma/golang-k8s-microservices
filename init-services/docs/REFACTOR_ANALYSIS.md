# Response-contract refactor

## Review goals

1. One documented OpenAPI response model for normal, error, and verify-source results.
2. No URL or query-string action dispatcher in request handlers.
3. Minimal branching in handlers.
4. Central JSON and error response handling.

## Findings before refactor

### Order service

- Every controller method wrote its own `gin.H` response.
- Error mapping was embedded in the controller.
- Health endpoints returned a different JSON shape from business endpoints.
- Delete returned a third response shape.

### Inventory service

- Responses used many unrelated JSON shapes.
- The `InvoiceActions` handler selected behavior through an `action` query parameter and a large switch.
- The route declared `/:actions` but the handler ignored that path parameter and read `?action=` instead.
- `GetInventoryByID` made an HTTP call back into the same service using a hard-coded URL.
- PDF generation, loading, dispatch, response writing, and error handling were mixed in one method.
- Panic recovery returned another unique error shape.

## Refactor result

- Both services use the same `Response` schema from `docs/openapi.yaml`.
- Generated-model-style files live in `internal/api/model.gen.go`.
- `internal/api/response.go` is the only JSON response writer.
- Business handlers return through `Success`, `SuccessWithMeta`, `Failure`, `ServiceError`, or `DomainError`.
- Request IDs are included in response metadata when available.
- Inventory action dispatch is replaced by explicit routes:
  - `GET /v1/invoices/:id/preview`
  - `GET /v1/invoices/:id/download`
  - `GET /v1/invoices/:id/document`
  - `POST /v1/invoices/:id/send-email`
  - `POST /v1/invoices/:id/upload`
- The self-referential `GET /v1/invoices/inventory/:id` route and hard-coded localhost call are removed.
- Common invoice lookup and document construction are centralized.

## Contract examples

Success:

```json
{
  "kind": "standard",
  "data": {"status": "ready"},
  "meta": {"request_id": "..."}
}
```

Error:

```json
{
  "kind": "error",
  "error": {
    "code": "INVALID_REQUEST",
    "message": "invalid request body"
  },
  "meta": {"request_id": "..."}
}
```

Verify source:

```json
{
  "kind": "verify_source",
  "verify_source": {
    "source": "inventory-db",
    "verified": true
  }
}
```

## Compatibility notes

This is an API response-contract change. Existing clients that directly decoded an order or list must now read the payload from `data`. Delete remains HTTP 204 with no body. PDF preview/download endpoints continue returning PDF, because binary responses are outside the JSON response contract.

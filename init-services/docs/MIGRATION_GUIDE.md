# Client migration guide

## JSON payloads

Before:

```json
{"id":"...","status":"PENDING"}
```

After:

```json
{"kind":"standard","data":{"id":"...","status":"PENDING"}}
```

Update clients and k6 tests from `response.json("id")` to `response.json("data.id")`.

## Inventory action routes

Replace:

```text
GET /v1/invoices/{id}/{actions}?action=preview
```

with one explicit route:

```text
GET  /v1/invoices/{id}/preview
GET  /v1/invoices/{id}/download
GET  /v1/invoices/{id}/document
POST /v1/invoices/{id}/send-email
POST /v1/invoices/{id}/upload
```

The removed self-call route `/v1/invoices/inventory/{id}` should be replaced by direct use of `/v1/invoices/{id}`.

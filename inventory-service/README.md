
export MYSQL_DSN=root:rootany@tcp(localhost:3306)/appdb?parseTime=true


data path:
/Users/dkgosql/tmp/inventory-data
file name: inventory-{orderid}.pdf


## Inventory reservation integration
See:
- `docs/HLD.md`
- `docs/LLD.md`
- `docs/RUNBOOK.md`

The service keeps existing invoice APIs and adds GORM-backed stock reservation plus RabbitMQ/Redis integration.

## Unified response contract and explicit document routes

All JSON endpoints now use the OpenAPI contract in `docs/openapi.yaml`. Normal payloads are under `data`; errors use `error.code` and `error.message`.

The old action dispatcher has been replaced with explicit routes:

```text
GET  /v1/invoices/:id/preview
GET  /v1/invoices/:id/download
GET  /v1/invoices/:id/document
POST /v1/invoices/:id/send-email
POST /v1/invoices/:id/upload
```

`INVOICE_OUTPUT_DIR` controls where generated PDFs are stored. It defaults to the operating-system temporary directory.

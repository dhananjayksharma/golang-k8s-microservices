root:root@tcp(localhost:3306)/appdb?parseTime=true

export MYSQL_DSN=root:root@tcp(localhost:3306)/appdb?parseTime=true


data path:
/Users/dkgosql/tmp/inventory-data
file name: inventory-{orderid}.pdf


## Inventory reservation integration
See:
- `docs/HLD.md`
- `docs/LLD.md`
- `docs/RUNBOOK.md`

The service keeps existing invoice APIs and adds GORM-backed stock reservation plus RabbitMQ/Redis integration.

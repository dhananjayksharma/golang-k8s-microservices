# Inventory Service LLD

## New packages
- `internal/inventory`: GORM models and transactional reservation service
- `internal/events`: RabbitMQ contracts and `order.created` consumer
- `internal/cache`: Redis client

## GORM models
`StockItem` stores SKU, available quantity, reserved quantity and version. `Reservation` stores one reservation per order. `Reserve` uses a MySQL transaction and `SELECT ... FOR UPDATE` through GORM locking.

## Reservation algorithm
1. Check whether a reservation already exists for `order_id`.
2. Lock the stock row.
3. Reject when available quantity is lower than requested quantity.
4. Decrement available and increment reserved.
5. Insert reservation.
6. Commit.
7. Publish `inventory.reserved`; otherwise publish `inventory.rejected`.

## API
- Existing invoice APIs remain unchanged.
- `GET /v1/inventory` lists stock items.

## Default demo SKU
The current Order Service does not carry product/SKU details. Inventory therefore reserves from SKU `DEFAULT`. The next schema evolution should add `order_items` and include `sku` in `order.created`.

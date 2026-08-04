INSERT INTO stock_items (sku, available, reserved, version, created_at, updated_at)
VALUES ('DEFAULT', 100000, 0, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE sku = VALUES(sku);

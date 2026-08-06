INSERT INTO stock_items (sku, available, reserved, version, created_at, updated_at)
VALUES ('DEFAULT', 100, 0, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE
    available = VALUES(available),
    reserved = VALUES(reserved),
    version = VALUES(version),
    updated_at = NOW();

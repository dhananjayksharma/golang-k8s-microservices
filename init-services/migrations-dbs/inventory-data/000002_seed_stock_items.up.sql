INSERT INTO stock_items (
    id,
    sku,
    product_name,
    available,
    reserved,
    version
)
VALUES
    (
        UUID(),
        'DEFAULT',
        'Default Inventory Item',
        1000,
        0,
        1
    ),
    (
        UUID(),
        'LAPTOP-001',
        'Developer Laptop',
        100,
        0,
        1
    ),
    (
        UUID(),
        'MOUSE-001',
        'Wireless Mouse',
        500,
        0,
        1
    ),
    (
        UUID(),
        'KEYBOARD-001',
        'Mechanical Keyboard',
        250,
        0,
        1
    ),
    (
        UUID(),
        'MONITOR-001',
        '27 Inch Monitor',
        150,
        0,
        1
    )
ON DUPLICATE KEY UPDATE
    product_name = VALUES(product_name),
    available = VALUES(available),
    version = version;
CREATE TABLE IF NOT EXISTS stock_items (
    id CHAR(36) NOT NULL,
    sku VARCHAR(100) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    available BIGINT NOT NULL DEFAULT 0,
    reserved BIGINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL
        DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    CONSTRAINT uq_stock_items_sku UNIQUE (sku),
    CONSTRAINT chk_stock_items_available CHECK (available >= 0),
    CONSTRAINT chk_stock_items_reserved CHECK (reserved >= 0),
    CONSTRAINT chk_stock_items_version CHECK (version > 0)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;

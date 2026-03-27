CREATE
    DATABASE auth_db;
CREATE
    USER auth_user WITH ENCRYPTED PASSWORD '4U7HP455';
GRANT ALL PRIVILEGES ON DATABASE
    auth_db TO auth_user;

CREATE
    DATABASE order_db;
CREATE
    USER order_user WITH ENCRYPTED PASSWORD '0RD3RP455';
GRANT ALL PRIVILEGES ON DATABASE
    order_db TO order_user;

CREATE
    DATABASE inventory_db;
CREATE
    USER inventory_user WITH ENCRYPTED PASSWORD '1NV3NT0RYP4SS';
GRANT ALL PRIVILEGES ON DATABASE
    inventory_db TO inventory_user;

\c auth_db;

\c order_db;
CREATE TYPE status_enum AS ENUM
    (
        'PENDING',
        'PROCESSED',
        'FAILED'
        );

CREATE TABLE IF NOT EXISTS outbox
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic      VARCHAR(100) NOT NULL,
    payload    JSONB        NOT NULL,
    status     status_enum      DEFAULT 'PENDING',
    created_at TIMESTAMP        DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS orders
(
    id          SERIAL PRIMARY KEY,
    customer_id VARCHAR(50)    NOT NULL,
    amount      DECIMAL(10, 2) NOT NULL,
    status      VARCHAR(20)    NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

\c inventory_db;
CREATE TYPE status_enum AS ENUM
    (
        'PENDING',
        'PROCESSED',
        'FAILED'
        );

CREATE TABLE IF NOT EXISTS outbox
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic      VARCHAR(100) NOT NULL,
    payload    JSONB        NOT NULL,
    status     status_enum      DEFAULT 'PENDING',
    created_at TIMESTAMP        DEFAULT CURRENT_TIMESTAMP
);
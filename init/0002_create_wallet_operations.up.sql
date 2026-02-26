CREATE TYPE operation_type AS ENUM ('DEPOSIT', 'WITHDRAW');

CREATE TABLE wallet_operations (
    id UUID PRIMARY KEY,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    operation operation_type NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wallet_operations_wallet_id 
    ON wallet_operations(wallet_id);
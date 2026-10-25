-- +goose Up
-- +goose StatementBegin
ALTER TABLE secret_keys
    RENAME TO lib_crypto_secret_keys;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX secret_keys_secret_idx
    RENAME TO lib_crypto_secret_keys_secret_idx;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER INDEX lib_crypto_secret_keys_secret_idx
    RENAME TO secret_keys_secret_idx;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE lib_crypto_secret_keys
    RENAME TO secret_keys;
-- +goose StatementEnd

-- +goose up
ALTER TABLE boards
    ADD COLUMN is_private boolean NOT NULL DEFAULT false;

-- +goose down
ALTER TABLE boards
    DROP COLUMN is_private;

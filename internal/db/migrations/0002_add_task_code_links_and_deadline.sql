-- +goose up
CREATE TYPE code_link_type AS ENUM ('commit', 'pull_request');

CREATE TABLE task_code_links
(
    id         uuid PRIMARY KEY DEFAULT uuidv7(),
    task_id    varchar(16)    NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    type       code_link_type NOT NULL,
    url        varchar(2048)  NOT NULL,
    base_sha   varchar(64)    NULL,
    head_sha   varchar(64)    NULL,
    additions  int            NULL CHECK (additions >= 0),
    deletions  int            NULL CHECK (deletions >= 0),
    created_by uuid           NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at timestamptz    NOT NULL,
    updated_at timestamptz    NOT NULL,
    UNIQUE (task_id, url),
    CHECK ((additions IS NULL) = (deletions IS NULL))
);

ALTER TABLE tasks ADD COLUMN deadline timestamptz NULL;

-- +goose down
ALTER TABLE tasks DROP COLUMN deadline;
DROP TABLE task_code_links;
DROP TYPE code_link_type;

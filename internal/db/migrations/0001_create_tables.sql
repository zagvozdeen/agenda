-- +goose up
CREATE TYPE board_role AS ENUM ('owner', 'member');
CREATE TYPE task_type AS ENUM ('feature', 'bug', 'chore');
CREATE TYPE task_status AS ENUM ('todo', 'in_progress', 'review', 'done', 'cancelled');
CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE sprint_status AS ENUM ('planned', 'active', 'closed');
CREATE TYPE link_type AS ENUM ('blocks', 'relates_to', 'duplicates');
CREATE TYPE event_type AS ENUM ('created', 'updated', 'deleted', 'commented', 'linked', 'unlinked');

CREATE TABLE IF NOT EXISTS users
(
    id         uuid PRIMARY KEY DEFAULT uuidv7(),
    username   varchar(64)  NOT NULL UNIQUE CHECK (username ~ '^[a-z0-9_]{3,64}$'),
    name       varchar(255) NOT NULL,
    created_at timestamptz  NOT NULL,
    deleted_at timestamptz  NULL
);

CREATE INDEX users_name_idx ON users (lower(name));

CREATE TABLE IF NOT EXISTS api_tokens
(
    id           uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id      uuid         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash   varchar(64)  NOT NULL UNIQUE,
    name         varchar(255) NULL,
    created_at   timestamptz  NOT NULL,
    updated_at   timestamptz  NOT NULL,
    last_used_at timestamptz  NULL,
    expires_at   timestamptz  NULL,
    revoked_at   timestamptz  NULL
);

CREATE INDEX api_tokens_user_idx ON api_tokens (user_id);

CREATE TABLE IF NOT EXISTS boards
(
    key         char(3) PRIMARY KEY CHECK (key ~ '^[A-Z]{3}$'),
    name        varchar(255) NOT NULL,
    description text         NULL,
    task_seq    int          NOT NULL DEFAULT 0,
    sprint_seq  int          NOT NULL DEFAULT 0,
    created_by  uuid         NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at  timestamptz  NOT NULL,
    updated_at  timestamptz  NOT NULL,
    deleted_at  timestamptz  NULL
);

CREATE TABLE IF NOT EXISTS board_members
(
    board_key char(3)     NOT NULL REFERENCES boards (key) ON DELETE CASCADE,
    user_id   uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role      board_role  NOT NULL,
    added_at  timestamptz NOT NULL,
    PRIMARY KEY (board_key, user_id)
);

CREATE INDEX board_members_user_idx ON board_members (user_id);

CREATE TABLE IF NOT EXISTS sprints
(
    board_key  char(3)       NOT NULL REFERENCES boards (key) ON DELETE RESTRICT,
    number     int           NOT NULL CHECK (number > 0),
    name       varchar(255)  NULL,
    goal       text          NULL,
    status     sprint_status NOT NULL,
    start_date timestamptz   NULL,
    end_date   timestamptz   NULL,
    created_at timestamptz   NOT NULL,
    PRIMARY KEY (board_key, number),
    CHECK (end_date >= start_date)
);

CREATE UNIQUE INDEX sprints_one_active_idx ON sprints (board_key) WHERE status = 'active';

CREATE TABLE IF NOT EXISTS tasks
(
    id              varchar(16) GENERATED ALWAYS AS (board_key::text || '-' || number::text) STORED,
    board_key       char(3)       NOT NULL REFERENCES boards (key) ON DELETE RESTRICT,
    number          int           NOT NULL CHECK (number > 0),
    title           varchar(255)  NOT NULL,
    description     text          NOT NULL,
    type            task_type     NOT NULL,
    status          task_status   NOT NULL,
    priority        task_priority NOT NULL,
    assignee_id     uuid          NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_by      uuid          NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    sprint_number   int           NULL,
    parent_id       varchar(16)   NULL REFERENCES tasks (id) ON DELETE RESTRICT,
    estimate_hours  int           NULL CHECK (estimate_hours >= 0),
    idempotency_key uuid          NULL UNIQUE,
    created_at      timestamptz   NOT NULL,
    updated_at      timestamptz   NOT NULL,
    completed_at    timestamptz   NULL,
    deleted_at      timestamptz   NULL,
    PRIMARY KEY (id),
    UNIQUE (board_key, number),
    FOREIGN KEY (board_key, sprint_number) REFERENCES sprints (board_key, number) ON DELETE RESTRICT,
    CHECK (split_part(parent_id, '-', 1) = board_key),
    CHECK (parent_id <> id)
);

CREATE INDEX tasks_board_sprint_idx ON tasks (board_key, sprint_number) WHERE deleted_at IS NULL;
CREATE INDEX tasks_board_status_idx ON tasks (board_key, status) WHERE deleted_at IS NULL;
CREATE INDEX tasks_assignee_status_idx ON tasks (assignee_id, status) WHERE deleted_at IS NULL;
CREATE INDEX tasks_parent_idx ON tasks (parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX tasks_fts_idx ON tasks USING gin (to_tsvector('russian', title || ' ' || description));

CREATE TABLE IF NOT EXISTS task_links
(
    from_task_id varchar(16) NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    to_task_id   varchar(16) NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    type         link_type   NOT NULL,
    created_by   uuid        NOT NULL REFERENCES users (id),
    created_at   timestamptz NOT NULL,
    PRIMARY KEY (from_task_id, to_task_id, type),
    CHECK (from_task_id <> to_task_id)
);

CREATE INDEX task_links_to_idx ON task_links (to_task_id);

CREATE TABLE IF NOT EXISTS comments
(
    id         uuid PRIMARY KEY DEFAULT uuidv7(),
    task_id    varchar(16) NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    author_id  uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    body       text        NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz NULL
);

CREATE INDEX comments_task_idx ON comments (task_id, created_at);

CREATE TABLE IF NOT EXISTS task_events
(
    id         uuid PRIMARY KEY DEFAULT uuidv7(),
    task_id    varchar(16) NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    actor_id   uuid        NOT NULL REFERENCES users (id),
    event_type event_type  NOT NULL,
    field      varchar(64) NULL,
    old_value  jsonb       NULL,
    new_value  jsonb       NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX task_events_task_idx ON task_events (task_id, created_at);

-- +goose down
DROP TABLE IF EXISTS task_events;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS task_links;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS sprints;
DROP TABLE IF EXISTS board_members;
DROP TABLE IF EXISTS boards;
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS event_type;
DROP TYPE IF EXISTS link_type;
DROP TYPE IF EXISTS sprint_status;
DROP TYPE IF EXISTS task_priority;
DROP TYPE IF EXISTS task_status;
DROP TYPE IF EXISTS task_type;
DROP TYPE IF EXISTS board_role;

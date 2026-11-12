-- +goose up
ALTER TABLE tasks
    ADD COLUMN checklist jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT tasks_checklist_array_check
        CHECK (jsonb_typeof(checklist) = 'array');

-- +goose down
ALTER TABLE tasks
    DROP CONSTRAINT tasks_checklist_array_check,
    DROP COLUMN checklist;

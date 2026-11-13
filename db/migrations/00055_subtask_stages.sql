-- +goose Up
-- LF-AJ-06 (AJ06-02): etapas por subtarefa (implementação e review) para a admissão das trilhas.
ALTER TABLE subtasks
    ADD COLUMN impl_state   text NOT NULL DEFAULT 'pending' CHECK (impl_state IN ('pending', 'running', 'done')),
    ADD COLUMN review_state text NOT NULL DEFAULT 'pending' CHECK (review_state IN ('pending', 'running', 'done', 'not_required')),
    ADD COLUMN work_class   text NOT NULL DEFAULT 'normal' CHECK (work_class IN ('normal', 'correction'));

-- +goose Down
ALTER TABLE subtasks DROP COLUMN work_class, DROP COLUMN review_state, DROP COLUMN impl_state;

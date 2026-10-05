-- +goose Up
ALTER TABLE profiles ADD COLUMN professional jsonb NOT NULL DEFAULT '{}';
ALTER TABLE profiles ADD CONSTRAINT professional_is_object CHECK (jsonb_typeof(professional) = 'object');

-- +goose Down
ALTER TABLE profiles DROP COLUMN professional;

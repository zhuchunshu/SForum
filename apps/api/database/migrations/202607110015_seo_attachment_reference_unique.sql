-- +goose Up
CREATE UNIQUE INDEX attachment_references_resource_context_unique
  ON attachment_references (resource_type, resource_id, context);

-- +goose Down
DROP INDEX IF EXISTS attachment_references_resource_context_unique;

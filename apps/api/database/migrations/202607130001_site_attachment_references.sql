-- +goose Up
WITH candidates AS (
  SELECT
    CASE name
      WHEN 'site.logo_attachment_id' THEN 'logo'
      WHEN 'site.favicon_attachment_id' THEN 'favicon'
      WHEN 'site.apple_touch_icon_attachment_id' THEN 'apple-touch-icon'
    END AS context,
    value::BIGINT AS attachment_id
  FROM web_options
  WHERE name IN (
    'site.logo_attachment_id',
    'site.favicon_attachment_id',
    'site.apple_touch_icon_attachment_id'
  )
    AND value ~ '^[1-9][0-9]*$'
), inserted AS (
  INSERT INTO attachment_references (attachment_id, resource_type, resource_id, context)
  SELECT attachments.id, 'site', 0, candidates.context
  FROM candidates
  JOIN attachments ON attachments.id = candidates.attachment_id
  WHERE attachments.status = 'active'
    AND attachments.visibility = 'public'
    AND attachments.content_type LIKE 'image/%'
  ON CONFLICT DO NOTHING
  RETURNING attachment_id
), counts AS (
  SELECT attachment_id, COUNT(*)::integer AS amount
  FROM inserted
  GROUP BY attachment_id
)
UPDATE attachments
SET reference_count = attachments.reference_count + counts.amount,
    updated_at = now()
FROM counts
WHERE attachments.id = counts.attachment_id;

-- +goose Down
WITH removed AS (
  DELETE FROM attachment_references
  WHERE resource_type = 'site'
    AND resource_id = 0
    AND context IN ('logo', 'favicon', 'apple-touch-icon')
  RETURNING attachment_id
), counts AS (
  SELECT attachment_id, COUNT(*)::integer AS amount
  FROM removed
  GROUP BY attachment_id
)
UPDATE attachments
SET reference_count = GREATEST(attachments.reference_count - counts.amount, 0),
    updated_at = now()
FROM counts
WHERE attachments.id = counts.attachment_id;

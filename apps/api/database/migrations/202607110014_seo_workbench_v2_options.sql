-- +goose Up
INSERT INTO web_options (name, value)
SELECT 'seo.home.description', value
FROM web_options
WHERE name = 'seo.meta_description'
ON CONFLICT (name) DO NOTHING;

INSERT INTO web_options (name, value)
SELECT 'seo.home.keywords', value
FROM web_options
WHERE name = 'seo.meta_keywords'
ON CONFLICT (name) DO NOTHING;

-- +goose Down
-- 兼容迁移不删除管理员可能已修改的新值，回滚保持数据不变。
SELECT 1;

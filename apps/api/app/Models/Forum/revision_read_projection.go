package forum

import "strings"

func effectivePostCurrentRevisionSQL(postAlias string) string {
	alias := strings.TrimSpace(postAlias)
	if alias == "" {
		alias = "posts"
	}
	// Backfill 前 current_revision=0。读取时把未编号的 legacy 行计入有效版本。
	return `CASE
		  WHEN ` + alias + `.current_revision > 0 THEN ` + alias + `.current_revision + (
		    SELECT COUNT(*) FROM post_revisions pr_effective
		    WHERE pr_effective.post_id = ` + alias + `.id AND pr_effective.revision_no IS NULL
		  )
		  ELSE 1 + (SELECT COUNT(*) FROM post_revisions pr_effective WHERE pr_effective.post_id = ` + alias + `.id)
		END`
}

func contentEditedSQL(postAlias string) string {
	return `(` + effectivePostCurrentRevisionSQL(postAlias) + `) > 1`
}

func contentEditedAtSQL(postAlias string) string {
	alias := strings.TrimSpace(postAlias)
	if alias == "" {
		alias = "posts"
	}
	return `CASE WHEN ` + contentEditedSQL(alias) + ` THEN COALESCE((
		  SELECT COALESCE(pr_edited_at.committed_at, pr_edited_at.created_at)
		  FROM post_revisions pr_edited_at
		  WHERE pr_edited_at.post_id = ` + alias + `.id
		  ORDER BY CASE WHEN pr_edited_at.revision_no = ` + alias + `.current_revision THEN 0 ELSE 1 END,
		    pr_edited_at.revision_no DESC NULLS LAST,
		    pr_edited_at.created_at DESC, pr_edited_at.id DESC
		  LIMIT 1
		), ` + alias + `.updated_at) END`
}

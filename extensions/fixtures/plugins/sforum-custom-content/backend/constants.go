package main

const (
	articlesQueryID = "sforum.custom-content.articles"
	articlesHandler = "sforum.custom-content.articles"
	searchQueryID   = "sforum.custom-content.articles.search"

	routeArticlesID      = "sforum.custom-content.route.articles"
	routeArticleWriteID  = "sforum.custom-content.route.article-write"
	routeImportID        = "sforum.custom-content.route.import"
	routeExportID        = "sforum.custom-content.route.export"
	routeRenderID        = "sforum.custom-content.route.render"
	routeTaxonomyID      = "sforum.custom-content.route.taxonomy"
	routeMigrateID       = "sforum.custom-content.route.migrate"
	routeSearchID        = "sforum.custom-content.route.search"

	articlesResponseSchema = "sforum.custom-content.route.articles.response@1"
	writeResponseSchema    = "sforum.custom-content.route.article-write.response@1"
	importResponseSchema   = "sforum.custom-content.route.import.response@1"
	exportResponseSchema   = "sforum.custom-content.route.export.response@1"
	renderResponseSchema   = "sforum.custom-content.route.render.response@1"
	taxonomyResponseSchema = "sforum.custom-content.route.taxonomy.response@1"
	migrateResponseSchema  = "sforum.custom-content.route.migrate.response@1"
	searchResponseSchema   = "sforum.custom-content.route.search.response@1"

	// 服务端 fallback render 的 content handlers（与 Manifest content[].handler 对齐）。
	blockVoteHandler         = "sforum.custom-content.block.vote"
	blockProductCardHandler  = "sforum.custom-content.block.product-card"
	embedMediaHandler        = "sforum.custom-content.embed.media"
	shortcodeBadgeHandler    = "sforum.custom-content.shortcode.badge"
	blockWorkflowFormHandler = "sforum.custom-content.block.workflow-form"
)

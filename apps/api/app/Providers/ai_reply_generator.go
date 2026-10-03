package providers

import (
	"github.com/jackc/pgx/v5/pgxpool"

	aireply "github.com/zhuchunshu/sforum/apps/api/app/Models/AIReply"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

// NewAIReplyGenerator 组装回复生成器。API 与 worker 两条装配路径共用它，
// 避免两处对同一个生成器做出不同解释。
func NewAIReplyGenerator(
	pool *pgxpool.Pool,
	users identity.ActorStore,
	forumService *forum.Service,
	gateway *supportai.Gateway,
	prompts aireply.ReplyPromptSource,
) *aireply.Generator {
	return aireply.NewGenerator(
		aireply.NewPostgresReader(pool),
		aireply.NewGatewayCompleter(gateway),
		aireply.NewBotCommentPoster(users, forumService),
		prompts,
	)
}

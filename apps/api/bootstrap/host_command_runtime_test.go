package bootstrap

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	attachments "github.com/zhuchunshu/sforum/apps/api/app/Models/Attachments"
	moderation "github.com/zhuchunshu/sforum/apps/api/app/Models/Moderation"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

func TestBindPostgresProtocolV2CommandRuntimePublishesDelegatedCatalog(t *testing.T) {
	pool := new(pgxpool.Pool)
	gateway := hostapi.NewGateway(nil)
	err := bindPostgresProtocolV2CommandRuntime(
		gateway,
		pool,
		new(supportjobs.Dispatcher),
		moderation.NewPostgresStore(pool),
		attachments.NewPostgresStore(pool),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gateway.ProtocolV2ActorDelegationIssuer() == nil {
		t.Fatal("expected the production gateway to expose its boot-scoped actor delegation issuer")
	}
}

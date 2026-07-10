package providers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	jobscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Jobs"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	jobs "github.com/zhuchunshu/sforum/apps/api/app/Models/Jobs"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type JobsProvider struct{ controller *jobscontroller.Controller }

func NewJobsProvider(pool *pgxpool.Pool, client *supportjobs.Client, users identity.ActorStore, sessions *authsession.Manager) *JobsProvider {
	return &JobsProvider{controller: jobscontroller.NewController(jobs.NewService(pool, client), users, sessions)}
}
func (p *JobsProvider) RegisterRoutes(api fiber.Router) { p.controller.RegisterRoutes(api) }

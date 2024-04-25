//go:build wireinject

package baguwen

import (
	"sync"

	"github.com/ecodeclub/webook/internal/search/internal/repository"
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/search/internal/service"
	"github.com/ecodeclub/webook/internal/search/internal/web"
	"github.com/google/wire"
	"github.com/olivere/elastic/v7"
)

func InitModule(es *elastic.Client) (*Module, error) {
	wire.Build(
		InitSearchSvc,
		InitSyncSvc,
		web.NewHandler,
		wire.Struct(new(Module), "*"),
	)
	return new(Module), nil
}

var daoOnce = sync.Once{}

func InitIndexOnce(es *elastic.Client) {
	daoOnce.Do(func() {
		err := dao.InitES(es)
		if err != nil {
			panic(err)
		}
	})
}

func InitRepo(es *elastic.Client) (repository.CaseRepo, repository.QuestionRepo, repository.QuestionSetRepo, repository.SkillRepo) {
	InitIndexOnce(es)
	questionDao := dao.NewQuestionDAO(es)
	caseDao := dao.NewCaseElasticDAO(es)
	questionSetDao := dao.NewQuestionSetDAO(es)
	skillDao := dao.NewSkillElasticDAO(es)
	questionRepo := repository.NewQuestionRepo(questionDao)
	caseRepo := repository.NewCaseRepo(caseDao)
	questionSetRepo := repository.NewQuestionSetRepo(questionSetDao)
	skillRepo := repository.NewSKillRepo(skillDao)
	return caseRepo, questionRepo, questionSetRepo, skillRepo
}

func InitSearchSvc(es *elastic.Client) service.SearchSvc {
	caseRepo, questionRepo, questionSetRepo, skillRepo := InitRepo(es)
	return service.NewSearchSvc(questionRepo, questionSetRepo, skillRepo, caseRepo)
}
func InitSyncSvc(es *elastic.Client) service.SyncSvc {
	caseRepo, questionRepo, questionSetRepo, skillRepo := InitRepo(es)
	return service.NewSyncSvc(questionRepo, questionSetRepo, skillRepo, caseRepo)
}

type SearchService = service.SearchSvc
type SyncService = service.SyncSvc
type Handler = web.Handler

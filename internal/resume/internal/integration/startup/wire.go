//go:build wireinject

package startup

import (
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/resume"
	"github.com/ecodeclub/webook/internal/resume/internal/repository"
	"github.com/ecodeclub/webook/internal/resume/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/resume/internal/service"
	"github.com/ecodeclub/webook/internal/resume/internal/web"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitModule(caModule *cases.Module, aiModule *ai.Module) *resume.Module {
	wire.Build(
		testioc.InitDB,
		dao.NewResumeProjectDAO,
		dao.NewExperienceDAO,
		repository.NewResumeProjectRepo,
		repository.NewExperience,
		service.NewService,
		service.NewExperienceService,
		service.NewAnalysisService,
		wire.FieldsOf(new(*cases.Module), "ExamineSvc"),
		wire.FieldsOf(new(*cases.Module), "Svc"),
		wire.FieldsOf(new(*ai.Module), "Svc"),
		web.NewHandler,
		web.NewExperienceHandler,
		web.NewAnalysisHandler,
		wire.Struct(new(resume.Module), "*"),
	)
	return new(resume.Module)
}
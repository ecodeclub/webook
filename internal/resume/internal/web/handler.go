package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/resume/internal/domain"
	"github.com/ecodeclub/webook/internal/resume/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	svc     service.Service
	examSvc cases.ExamineService
	logger  *elog.Component
}

func NewHandler(svc service.Service, examSvc cases.ExamineService) *Handler {
	return &Handler{
		svc:     svc,
		logger:  elog.DefaultLogger,
		examSvc: examSvc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/resume/project/save", ginx.BS[SaveProjectReq](h.SaveProject))
	server.POST("/resume/project/delete", ginx.BS[IDItem](h.DeleteProject))
	server.POST("/resume/project/info", ginx.BS[IDItem](h.ProjectInfo))
	server.POST("/resume/project/list", ginx.S(h.ProjectList))
	server.POST("/resume/project/contribution/save", ginx.B[SaveContributionReq](h.ProjectContributionSave))
	server.POST("/resume/project/difficulty/save", ginx.B[SaveDifficultyReq](h.ProjectDifficultySave))
	server.POST("/resume/project/difficulty/del", ginx.B[IDItem](h.DeleteDifficulty))
	server.POST("/resume/project/contribution/del", ginx.B[IDItem](h.DeleteContribution))
}

func (h *Handler) DeleteContribution(ctx *ginx.Context, item IDItem) (ginx.Result, error) {
	err := h.svc.DeleteContribution(ctx, item.ID)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) DeleteDifficulty(ctx *ginx.Context, item IDItem) (ginx.Result, error) {
	err := h.svc.DeleteDifficulty(ctx.Request.Context(), item.ID)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) SaveProject(ctx *ginx.Context, req SaveProjectReq, sess session.Session) (ginx.Result, error) {
	project := req.Project
	uid := sess.Claims().Uid
	id, err := h.svc.SaveProject(ctx, domain.Project{
		Id:           project.Id,
		StartTime:    project.StartTime,
		EndTime:      project.EndTime,
		Uid:          uid,
		Name:         project.Name,
		Introduction: project.Introduction,
		Core:         project.Core,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *Handler) DeleteProject(ctx *ginx.Context, req IDItem, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	err := h.svc.DeleteProject(ctx, uid, req.ID)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) ProjectInfo(ctx *ginx.Context, req IDItem, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	pro, err := h.svc.ProjectInfo(ctx, req.ID)
	if err != nil {
		return systemErrorResult, err
	}
	cids := make([]int64, 0, 32)
	for _, d := range pro.Difficulties {
		cids = append(cids, d.Case.Id)
	}
	for _, c := range pro.Contributions {
		for _, ca := range c.RefCases {
			cids = append(cids, ca.Id)
		}
	}
	resMap, err := h.examSvc.GetResults(ctx, uid, cids)
	if err != nil {
		return systemErrorResult, err
	}
	p := newProject(pro, resMap)
	return ginx.Result{
		Data: p,
	}, nil

}

func (h *Handler) ProjectList(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	projects, err := h.svc.FindProjects(ctx.Request.Context(), uid)
	if err != nil {
		return systemErrorResult, err
	}
	ans := slice.Map(projects, func(idx int, src domain.Project) Project {
		return newProject(src, nil)
	})
	return ginx.Result{
		Data: ans,
	}, nil
}

func (h *Handler) ProjectContributionSave(ctx *ginx.Context, req SaveContributionReq) (ginx.Result, error) {
	err := h.svc.SaveContribution(ctx, req.ID, domain.Contribution{
		ID:   req.Contribution.ID,
		Type: req.Contribution.Type,
		Desc: req.Contribution.Desc,
		RefCases: slice.Map(req.Contribution.RefCases, func(idx int, src Case) domain.Case {
			return domain.Case{
				Id:        src.Id,
				Highlight: src.Highlight,
				Level:     src.Level,
			}
		}),
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) ProjectDifficultySave(ctx *ginx.Context, req SaveDifficultyReq) (ginx.Result, error) {
	err := h.svc.SaveDifficulty(ctx, req.ID, domain.Difficulty{
		ID:   req.Difficulty.ID,
		Desc: req.Difficulty.Desc,
		Case: domain.Case{
			Id:        req.Difficulty.Case.Id,
			Highlight: req.Difficulty.Case.Highlight,
			Level:     req.Difficulty.Case.Level,
		},
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

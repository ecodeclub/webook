package web

import (
	"fmt"
	"net/http"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/skill/internal/domain"
	"github.com/ecodeclub/webook/internal/skill/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	svc    service.SkillSvc
	logger *elog.Component
}

func NewHandler(svc service.SkillSvc) *Handler {
	return &Handler{
		svc:    svc,
		logger: elog.DefaultLogger,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/skill/save", ginx.S(h.Permission), ginx.B[SaveReq](h.Save))
	server.POST("/skill/save-request", ginx.S(h.Permission), ginx.B[SaveRequestReq](h.SaveReqs))
	server.POST("/skill/detail", ginx.S(h.Permission), ginx.B[Sid](h.Detail))
	server.POST("/skill/list", ginx.S(h.Permission), ginx.B[Page](h.List))
	server.POST("/skill/publish", ginx.S(h.Permission), ginx.B[SaveReq](h.Publish))
	server.POST("/skill/publish-request", ginx.S(h.Permission), ginx.B[SaveRequestReq](h.PublishReq))
	server.POST("/skill/pub/list", ginx.B[Page](h.PubList))
	server.POST("/skill/pub/detail", ginx.B[Sid](h.PubDetail))
}

func (h *Handler) Permission(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	if sess.Claims().Get("admin").StringOrDefault("") != "true" {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return ginx.Result{}, fmt.Errorf("非法访问创作中心 uid: %d", sess.Claims().Uid)
	}
	return ginx.Result{}, ginx.ErrNoResponse
}

func (h *Handler) Save(ctx *ginx.Context, req SaveReq) (ginx.Result, error) {
	skill := req.Skill.toDomain()
	id, err := h.svc.Save(ctx, skill, skill.Levels)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *Handler) SaveReqs(ctx *ginx.Context, req SaveRequestReq) (ginx.Result, error) {
	skill := domain.Skill{
		ID: req.Sid,
		Levels: []domain.SkillLevel{
			{
				Id: req.Slid,
			},
		},
	}
	reqs := make([]domain.SkillPreRequest, 0, len(req.Requests))
	for _, r := range req.Requests {
		reqs = append(reqs, r.toDomain())
	}
	err := h.svc.UpdateRequest(ctx, skill, reqs)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) Detail(ctx *ginx.Context, req Sid) (ginx.Result, error) {
	skill, err := h.svc.Info(ctx, req.Sid)
	if err != nil {
		return systemErrorResult, err
	}
	skillView := newSkill(skill)
	return ginx.Result{
		Data: skillView,
	}, nil
}

func (h *Handler) List(ctx *ginx.Context, page Page) (ginx.Result, error) {
	skills, count, err := h.svc.List(ctx, page.Offset, page.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	skillList := h.toSKillList(skills, count)
	return ginx.Result{
		Data: skillList,
	}, nil

}

func (h *Handler) Publish(ctx *ginx.Context, req SaveReq) (ginx.Result, error) {
	skill := req.Skill.toDomain()
	id, err := h.svc.SyncSkill(ctx, skill, skill.Levels)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *Handler) PublishReq(ctx *ginx.Context, req SaveRequestReq) (ginx.Result, error) {
	skill := domain.Skill{
		ID: req.Sid,
		Levels: []domain.SkillLevel{
			{
				Id: req.Slid,
			},
		},
	}
	reqs := make([]domain.SkillPreRequest, 0, len(req.Requests))
	for _, r := range req.Requests {
		reqs = append(reqs, r.toDomain())
	}
	err := h.svc.SyncSKillRequest(ctx, skill, reqs)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) PubList(ctx *ginx.Context, page Page) (ginx.Result, error) {
	skills, count, err := h.svc.Publist(ctx, page.Offset, page.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	skillList := h.toSKillList(skills, count)
	return ginx.Result{
		Data: skillList,
	}, nil
}

func (h *Handler) PubDetail(ctx *ginx.Context, req Sid) (ginx.Result, error) {
	skill, err := h.svc.PubInfo(ctx, req.Sid)
	if err != nil {
		return systemErrorResult, err
	}
	skillView := newSkill(skill)
	return ginx.Result{
		Data: skillView,
	}, nil
}

func (h *Handler) toSKillList(data []domain.Skill, cnt int64) SkillList {
	return SkillList{
		Total: cnt,
		Skills: slice.Map(data, func(idx int, src domain.Skill) Skill {
			return newSkill(src)
		}),
	}
}

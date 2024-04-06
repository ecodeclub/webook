package web

import (
	"fmt"
	"net/http"

	"github.com/ecodeclub/webook/internal/cases"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/skill/internal/domain"
	"github.com/ecodeclub/webook/internal/skill/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	svc     service.SkillService
	queSvc  baguwen.Service
	caseSvc cases.Service
	logger  *elog.Component
}

func NewHandler(svc service.SkillService, queSvc baguwen.Service, caseSvc cases.Service) *Handler {
	return &Handler{
		svc:     svc,
		logger:  elog.DefaultLogger,
		queSvc:  queSvc,
		caseSvc: caseSvc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/skill/save", ginx.S(h.Permission), ginx.B[SaveReq](h.Save))
	server.POST("/skill/list", ginx.B[Page](h.List))
	server.POST("/skill/detail", ginx.B[Sid](h.Detail))
	server.POST("/skill/detail-refs", ginx.S(h.Permission), ginx.B[Sid](h.DetailRefs))
	server.POST("/skill/save-refs", ginx.S(h.Permission), ginx.B(h.SaveRefs))
	server.POST("/skill/level-refs", ginx.S(h.Permission), ginx.B(h.RefsByLevelIDs))
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
}

func (h *Handler) Permission(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	if sess.Claims().Get("creator").StringOrDefault("") != "true" {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return ginx.Result{}, fmt.Errorf("非法访问创作中心 uid: %d", sess.Claims().Uid)
	}
	return ginx.Result{}, ginx.ErrNoResponse
}

func (h *Handler) Save(ctx *ginx.Context, req SaveReq) (ginx.Result, error) {
	skill := req.Skill.toDomain()
	id, err := h.svc.Save(ctx, skill)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *Handler) SaveRefs(ctx *ginx.Context, req SaveReq) (ginx.Result, error) {
	err := h.svc.SaveRefs(ctx, req.Skill.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Msg: "OK",
	}, nil
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
	skillList := h.toSkillList(skills, count)
	return ginx.Result{
		Data: skillList,
	}, nil

}

func (h *Handler) toSkillList(data []domain.Skill, cnt int64) SkillList {
	return SkillList{
		Total: cnt,
		Skills: slice.Map(data, func(idx int, src domain.Skill) Skill {
			return newSkill(src)
		}),
	}
}

func (h *Handler) DetailRefs(ctx *ginx.Context, req Sid) (ginx.Result, error) {
	skill, err := h.svc.Info(ctx, req.Sid)
	if err != nil {
		return systemErrorResult, err
	}
	res := newSkill(skill)
	var eg errgroup.Group
	eg.Go(func() error {
		qids := skill.Questions()
		qs, err1 := h.queSvc.GetPubByIDs(ctx, qids)
		if err1 != nil {
			return err1
		}
		qm := slice.ToMap(qs, func(ele baguwen.Question) int64 {
			return ele.Id
		})
		res.setQuestions(qm)
		return nil
	})

	eg.Go(func() error {
		cids := skill.Cases()
		cs, err1 := h.caseSvc.GetPubByIDs(ctx, cids)
		if err1 != nil {
			return err1
		}
		cms := slice.ToMap(cs, func(ele cases.Case) int64 {
			return ele.Id
		})
		res.setCases(cms)
		return nil
	})
	return ginx.Result{
		Data: res,
	}, eg.Wait()
}

func (h *Handler) RefsByLevelIDs(ctx *ginx.Context, req IDs) (ginx.Result, error) {
	if len(req.IDs) == 0 {
		return ginx.Result{}, nil
	}
	res, err := h.svc.RefsByLevelIDs(ctx, req.IDs)
	if err != nil {
		return systemErrorResult, err
	}
	var (
		eg  errgroup.Group
		csm map[int64]cases.Case
		qsm map[int64]baguwen.Question
	)
	qids := make([]int64, 0, 32)
	cids := make([]int64, 0, 16)
	for _, sl := range res {
		qids = append(qids, sl.Questions...)
		cids = append(cids, sl.Cases...)
	}
	eg.Go(func() error {
		cs, err1 := h.caseSvc.GetPubByIDs(ctx, cids)
		csm = slice.ToMap(cs, func(element cases.Case) int64 {
			return element.Id
		})
		return err1
	})

	eg.Go(func() error {
		qs, err1 := h.queSvc.GetPubByIDs(ctx, qids)
		qsm = slice.ToMap(qs, func(element baguwen.Question) int64 {
			return element.Id
		})
		return err1
	})

	err = eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}
	// 组装 title
	return ginx.Result{
		Data: slice.Map(res, func(idx int, src domain.SkillLevel) SkillLevel {
			sl := newSkillLevel(src)
			sl.setCases(csm)
			sl.setQuestions(qsm)
			return sl
		}),
	}, nil
}

package web

import (
	"context"
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
	svc        service.SkillService
	queSvc     baguwen.Service
	caseSvc    cases.Service
	caseSetSvc cases.SetService
	queSetSvc  baguwen.QuestionSetService
	examSvc    baguwen.ExamService
	logger     *elog.Component
}

func NewHandler(svc service.SkillService,
	queSvc baguwen.Service,
	caseSvc cases.Service,
	caseSetSvc cases.SetService,
	queSetSvc baguwen.QuestionSetService,
	examSvc baguwen.ExamService) *Handler {
	return &Handler{
		svc:        svc,
		logger:     elog.DefaultLogger,
		queSvc:     queSvc,
		queSetSvc:  queSetSvc,
		examSvc:    examSvc,
		caseSvc:    caseSvc,
		caseSetSvc: caseSetSvc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/skill/list", ginx.B[Page](h.List))
	server.POST("/skill/detail-refs", ginx.B[Sid](h.DetailRefs))
	server.POST("/skill/save", ginx.S(h.Permission), ginx.B[SaveReq](h.Save))
	server.POST("/skill/save-refs", ginx.S(h.Permission), ginx.B(h.SaveRefs))
	server.POST("/skill/level-refs", ginx.BS(h.RefsByLevelIDs))
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
		if len(qids) == 0 {
			return nil
		}
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
		if len(cids) == 0 {
			return nil
		}
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

	eg.Go(func() error {
		cids := skill.QuestionSets()
		if len(cids) == 0 {
			return nil
		}
		cs, err1 := h.queSetSvc.GetByIDsWithQuestion(ctx, cids)
		if err1 != nil {
			return err1
		}
		cms := slice.ToMap(cs, func(ele baguwen.QuestionSet) int64 {
			return ele.Id
		})
		res.setQuestionSets(cms)
		return nil
	})

	eg.Go(func() error {
		cids := skill.CaseSets()
		if len(cids) == 0 {
			return nil
		}
		cs, err1 := h.caseSetSvc.GetByIds(ctx, cids)
		if err1 != nil {
			return err1
		}
		cms := slice.ToMap(cs, func(ele cases.CaseSet) int64 {
			return ele.ID
		})
		res.setCaseSets(cms)
		return nil
	})
	return ginx.Result{
		Data: res,
	}, eg.Wait()
}

func (h *Handler) RefsByLevelIDs(ctx *ginx.Context, req IDs, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	if len(req.IDs) == 0 {
		return ginx.Result{}, nil
	}
	res, err := h.svc.RefsByLevelIDs(ctx, req.IDs)
	if err != nil {
		return systemErrorResult, err
	}
	csm, cssm, qsm, qssmap, examResMap, err := h.skillLevels(ctx, uid, res)
	if err != nil {
		return systemErrorResult, err
	}
	// 组装 title
	return ginx.Result{
		Data: slice.Map(res, func(idx int, src domain.SkillLevel) SkillLevel {
			sl := newSkillLevel(src)
			sl.setCases(csm)
			sl.setQuestionsWithExam(qsm, examResMap)
			sl.setQuestionSet(qssmap, examResMap)
			sl.setCaseSet(cssm)
			return sl
		}),
	}, nil
}

func (h *Handler) skillLevels(ctx context.Context, uid int64, levels []domain.SkillLevel) (
	map[int64]cases.Case,
	map[int64]cases.CaseSet,
	map[int64]baguwen.Question,
	map[int64]baguwen.QuestionSet,
	map[int64]baguwen.ExamResult,
	error,
) {
	var (
		eg         errgroup.Group
		csm        map[int64]cases.Case
		cssm       map[int64]cases.CaseSet
		qsm        map[int64]baguwen.Question
		qssmap     map[int64]baguwen.QuestionSet
		examResMap map[int64]baguwen.ExamResult
	)
	qids := make([]int64, 0, 32)
	cids := make([]int64, 0, 16)
	csids := make([]int64, 0, 16)
	qsids := make([]int64, 0, 16)
	for _, sl := range levels {
		qids = append(qids, sl.Questions...)
		cids = append(cids, sl.Cases...)
		csids = append(csids, sl.CaseSets...)
		qsids = append(qsids, sl.QuestionSets...)
	}
	var qid2s []int64
	qid2s = append(qid2s, qids...)

	//  获取case
	eg.Go(func() error {
		cs, err1 := h.caseSvc.GetPubByIDs(ctx, cids)
		csm = slice.ToMap(cs, func(element cases.Case) int64 {
			return element.Id
		})
		return err1
	})

	// 获取 caseSet
	eg.Go(func() error {
		sets, err := h.caseSetSvc.GetByIds(ctx, csids)
		cssm = slice.ToMap(sets, func(element cases.CaseSet) int64 {
			return element.ID
		})
		return err
	})
	// 获取问题
	eg.Go(func() error {
		qs, err1 := h.queSvc.GetPubByIDs(ctx, qids)
		qsm = slice.ToMap(qs, func(element baguwen.Question) int64 {
			return element.Id
		})
		return err1
	})
	// 获取问题集
	eg.Go(func() error {
		qsets, qerr := h.queSetSvc.GetByIDsWithQuestion(ctx, qsids)
		if qerr != nil {
			return qerr
		}
		qssmap = slice.ToMap(qsets, func(element baguwen.QuestionSet) int64 {
			return element.Id
		})
		for _, qs := range qsets {
			qid2s = append(qid2s, qs.Qids()...)
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// 获取进度
	examResMap, err := h.examSvc.GetResults(ctx, uid, qid2s)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return csm, cssm, qsm, qssmap, examResMap, nil

}

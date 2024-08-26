package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/interactive"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"golang.org/x/sync/errgroup"
)

const (
	CaseBiz        = "case"
	CaseSetBiz     = "caseSet"
	QuestionBiz    = "question"
	QuestionSetBiz = "questionSet"
)

func (h *Handler) CollectionRecords(ctx *ginx.Context, req CollectionInfoReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	recordCtx := ctx.Request.Context()
	// 获取收藏记录
	records, err := h.intrSvc.CollectionInfo(recordCtx, uid, req.ID, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	var (
		eg             errgroup.Group
		csm            map[int64]cases.Case
		cssmap         map[int64]cases.CaseSet
		qsm            map[int64]baguwen.Question
		qssmap         map[int64]baguwen.QuestionSet
		queExamResMap  map[int64]baguwen.ExamResult
		caseExamResMap map[int64]cases.ExamineResult
		csets          []cases.CaseSet
	)
	var qids, cids, csids, qsids, qid2s []int64
	for _, record := range records {
		switch record.Biz {
		case CaseBiz:
			cids = append(cids, record.Case)
		case CaseSetBiz:
			csids = append(csids, record.CaseSet)
		case QuestionBiz:
			qids = append(qids, record.Question)
		case QuestionSetBiz:
			qsids = append(qsids, record.QuestionSet)
		}
	}
	qid2s = append(qid2s, qids...)

	eg.Go(func() error {
		cs, err1 := h.caseSvc.GetPubByIDs(recordCtx, cids)
		csm = slice.ToMap(cs, func(element cases.Case) int64 {
			return element.Id
		})
		return err1
	})

	eg.Go(func() error {
		qs, err1 := h.queSvc.GetPubByIDs(recordCtx, qids)
		qsm = slice.ToMap(qs, func(element baguwen.Question) int64 {
			return element.Id
		})
		return err1
	})
	eg.Go(func() error {
		qsets, qerr := h.queSetSvc.GetByIDsWithQuestion(recordCtx, qsids)
		qssmap = slice.ToMap(qsets, func(element baguwen.QuestionSet) int64 {
			return element.Id
		})
		for _, qs := range qsets {
			qid2s = append(qid2s, qs.Qids()...)
		}
		return qerr
	})

	eg.Go(func() error {
		var cserr error
		csets, cserr = h.caseSetSvc.GetByIdsWithCases(recordCtx, csids)
		cssmap = slice.ToMap(csets, func(element cases.CaseSet) int64 {
			return element.ID
		})
		return cserr
	})
	if err = eg.Wait(); err != nil {
		return systemErrorResult, err
	}

	for _, cs := range csets {
		cids = append(cids, cs.Cids()...)
	}
	eg = errgroup.Group{}
	eg.Go(func() error {
		var err1 error
		queExamResMap, err1 = h.queExamSvc.GetResults(recordCtx, uid, qid2s)
		return err1
	})

	eg.Go(func() error {
		var err1 error
		caseExamResMap, err1 = h.caseExamSvc.GetResults(recordCtx, uid, cids)
		return err1
	})
	// 获取进度

	if err = eg.Wait(); err != nil {
		return systemErrorResult, err
	}

	res := slice.Map(records, func(idx int, src interactive.CollectionRecord) CollectionRecord {
		return newCollectionRecord(src, csm, cssmap, qsm, qssmap, queExamResMap, caseExamResMap)
	})
	return ginx.Result{
		Data: res,
	}, nil
}

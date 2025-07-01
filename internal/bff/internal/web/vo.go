package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/interactive"
	baguwen "github.com/ecodeclub/webook/internal/question"
)

type CollectionInfoReq struct {
	ID     int64  `json:"id"`
	Biz    string `json:"biz"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

type CollectionRecord struct {
	Id          int64       `json:"id"`
	Biz         string      `json:"biz"`
	Case        Case        `json:"case,omitempty"`
	Question    Question    `json:"question,omitempty"`
	QuestionSet QuestionSet `json:"questionSet,omitempty"`
	CaseSet     CaseSet     `json:"caseSet,omitempty"`
}

type Case struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	ExamineResult uint8  `json:"examineResult"`
}

type Question struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	ExamineResult uint8  `json:"examineResult"`
}

type CaseSet struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Cases []Case `json:"cases"`
}

type QuestionSet struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
}

func newCollectionRecord(record interactive.CollectionRecord,
	cm map[int64]cases.Case,
	csm map[int64]cases.CaseSet,
	qm map[int64]baguwen.Question,
	qsm map[int64]baguwen.QuestionSet,
	queExamMap map[int64]baguwen.ExamResult,
	caseExamMap map[int64]cases.ExamineResult,
) CollectionRecord {
	res := CollectionRecord{
		Id:  record.Id,
		Biz: record.Biz,
	}
	switch record.Biz {
	case CaseBiz:
		res.Case = setCases(record, cm)
	case QuestionBiz:
		res.Question = setQuestion(record, qm, queExamMap)
	case QuestionSetBiz:
		res.QuestionSet = setQuestionSet(record, qsm, queExamMap)
	case CaseSetBiz:
		res.CaseSet = setCaseSet(record, csm, caseExamMap)
	}
	return res
}

func setCaseSet(
	ca interactive.CollectionRecord,
	csm map[int64]cases.CaseSet,
	caseExamMap map[int64]cases.ExamineResult,
) CaseSet {
	cs := csm[ca.CaseSet]
	return CaseSet{
		ID:    cs.ID,
		Title: cs.Title,
		Cases: slice.Map(cs.Cases, func(idx int, src cases.Case) Case {
			return Case{
				ID:            src.Id,
				ExamineResult: caseExamMap[src.Id].Result.ToUint8(),
			}
		}),
	}
}

func setCases(ca interactive.CollectionRecord, qm map[int64]cases.Case) Case {
	cas := qm[ca.Case]
	return Case{
		ID:    cas.Id,
		Title: cas.Title,
	}
}

func setQuestion(record interactive.CollectionRecord, qm map[int64]baguwen.Question, examMap map[int64]baguwen.ExamResult) Question {
	q := qm[record.Question]
	exam := examMap[record.Question]
	return Question{
		ID:            q.Id,
		Title:         q.Title,
		ExamineResult: exam.Result.ToUint8(),
	}
}

func setQuestionSet(record interactive.CollectionRecord, qsm map[int64]baguwen.QuestionSet, examMap map[int64]baguwen.ExamResult) QuestionSet {
	qs := qsm[record.QuestionSet]
	questions := make([]Question, 0, len(qs.Questions))
	for _, q := range qs.Questions {
		exam := examMap[q.Id]
		questions = append(questions, Question{
			ID:            q.Id,
			Title:         q.Title,
			ExamineResult: exam.Result.ToUint8(),
		})
	}
	return QuestionSet{
		ID:        qs.Id,
		Title:     qs.Title,
		Questions: questions,
	}
}

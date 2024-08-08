package web

import (
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/interactive"
	baguwen "github.com/ecodeclub/webook/internal/question"
)

type CollectionInfoReq struct {
	ID     int64 `json:"id"`
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
}

type CollectionRecord struct {
	Id          int64       `json:"id"`
	Case        Case        `json:"case,omitempty"`
	Question    Question    `json:"question,omitempty"`
	QuestionSet QuestionSet `json:"questionSet,omitempty"`
}

type Case struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type Question struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	ExamineResult uint8  `json:"examineResult"`
}

type QuestionSet struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
}

func newCollectionRecord(record interactive.CollectionRecord,
	cm map[int64]cases.Case,
	qm map[int64]baguwen.Question,
	qsm map[int64]baguwen.QuestionSet,
	examMap map[int64]baguwen.ExamResult,
) CollectionRecord {
	switch record.Biz {
	case CaseBiz:
		return CollectionRecord{
			Id:   record.Id,
			Case: setCases(record, cm),
		}
	case QuestionBiz:
		return CollectionRecord{
			Id:       record.Id,
			Question: setQuestion(record, qm, examMap),
		}
	case QuestionSetBiz:
		return CollectionRecord{
			Id:          record.Id,
			QuestionSet: setQuestionSet(record, qsm, examMap),
		}
	}
	return CollectionRecord{}
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

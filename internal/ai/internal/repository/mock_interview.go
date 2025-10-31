package repository

import (
	"context"
	"database/sql"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
)

type MockInterviewRepository interface {
	SaveInterview(ctx context.Context, req domain.MockInterview) (int64, error)
	FindInterviews(ctx context.Context, uid int64, limit, offset int) ([]domain.MockInterview, error)
	CountInterviews(ctx context.Context, uid int64) (int64, error)

	SaveQuestion(ctx context.Context, req domain.MockInterviewQuestion) (int64, error)
	FindQuestions(ctx context.Context, interviewID, uid int64, limit, offset int) ([]domain.MockInterviewQuestion, error)
	CountQuestions(ctx context.Context, interviewID, uid int64) (int64, error)
}

type mockInterviewRepository struct {
	dao dao.MockInterviewDAO
}

func NewMockInterviewRepository(d dao.MockInterviewDAO) MockInterviewRepository {
	return &mockInterviewRepository{dao: d}
}

func (r *mockInterviewRepository) SaveInterview(ctx context.Context, req domain.MockInterview) (int64, error) {
	mi := dao.MockInterview{
		Uid:    req.Uid,
		Title:  req.Title,
		ChatSN: req.ChatSN,
		Evaluation: sqlx.JsonColumn[map[string]any]{
			Val:   req.Evaluation,
			Valid: len(req.Evaluation) != 0,
		},
	}
	return r.dao.SaveInterview(ctx, mi)
}

func (r *mockInterviewRepository) FindInterviews(ctx context.Context, uid int64, limit, offset int) ([]domain.MockInterview, error) {
	list, err := r.dao.FindInterviews(ctx, uid, limit, offset)
	if err != nil {
		return nil, err
	}
	return slice.Map(list, func(_ int, src dao.MockInterview) domain.MockInterview {
		return r.toDomainMockInterview(src)
	}), nil
}

func (r *mockInterviewRepository) CountInterviews(ctx context.Context, uid int64) (int64, error) {
	return r.dao.CountInterviews(ctx, uid)
}

func (r *mockInterviewRepository) SaveQuestion(ctx context.Context, req domain.MockInterviewQuestion) (int64, error) {

	q := dao.MockInterviewQuestion{
		InterviewID: req.InterviewID,
		ChatSN:      req.ChatSN,
		Uid:         req.Uid,
		Biz:         req.Biz,
		BizID:       sql.NullInt64{Int64: req.BizID, Valid: req.BizID != 0},
		Title:       sql.NullString{String: req.Title, Valid: req.Title != ""},
		Answer: sqlx.JsonColumn[map[string]any]{
			Val:   req.Answer,
			Valid: len(req.Answer) != 0,
		},
		Evaluation: sqlx.JsonColumn[map[string]any]{
			Val:   req.Evaluation,
			Valid: len(req.Evaluation) != 0,
		},
	}
	return r.dao.SaveQuestion(ctx, q)
}

func (r *mockInterviewRepository) FindQuestions(ctx context.Context, interviewID, uid int64, limit, offset int) ([]domain.MockInterviewQuestion, error) {
	list, err := r.dao.FindQuestions(ctx, interviewID, uid, limit, offset)
	if err != nil {
		return nil, err
	}
	return slice.Map(list, func(_ int, src dao.MockInterviewQuestion) domain.MockInterviewQuestion {
		return r.toDomainMockInterviewQuestion(src)
	}), nil
}

func (r *mockInterviewRepository) CountQuestions(ctx context.Context, interviewID, uid int64) (int64, error) {
	return r.dao.CountQuestions(ctx, interviewID, uid)
}

func (r *mockInterviewRepository) toDomainMockInterview(mi dao.MockInterview) domain.MockInterview {
	evaluation := make(map[string]any)
	if mi.Evaluation.Valid {
		evaluation = mi.Evaluation.Val
	}
	return domain.MockInterview{
		ID:         mi.ID,
		Uid:        mi.Uid,
		Title:      mi.Title,
		ChatSN:     mi.ChatSN,
		Evaluation: evaluation,
		Ctime:      mi.Ctime,
		Utime:      mi.Utime,
	}
}

func (r *mockInterviewRepository) toDomainMockInterviewQuestion(q dao.MockInterviewQuestion) domain.MockInterviewQuestion {
	var bizID int64
	if q.BizID.Valid {
		bizID = q.BizID.Int64
	}
	var title string
	if q.Title.Valid {
		title = q.Title.String
	}

	answer := make(map[string]any)
	if q.Answer.Valid {
		answer = q.Answer.Val
	}

	evaluation := make(map[string]any)
	if q.Evaluation.Valid {
		evaluation = q.Evaluation.Val
	}

	return domain.MockInterviewQuestion{
		ID:          q.ID,
		InterviewID: q.InterviewID,
		ChatSN:      q.ChatSN,
		Uid:         q.Uid,
		Biz:         q.Biz,
		BizID:       bizID,
		Title:       title,
		Answer:      answer,
		Evaluation:  evaluation,
		Ctime:       q.Ctime,
		Utime:       q.Utime,
	}
}

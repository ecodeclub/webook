package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"golang.org/x/sync/errgroup"
)

type MockInterviewService interface {
	SaveInterview(ctx context.Context, mi domain.MockInterview) (int64, error)
	ListInterviews(ctx context.Context, uid int64, limit, offset int) ([]domain.MockInterview, int64, error)

	SaveQuestion(ctx context.Context, q domain.MockInterviewQuestion) (int64, error)
	ListQuestions(ctx context.Context, interviewID, uid int64, limit, offset int) ([]domain.MockInterviewQuestion, int64, error)
}

type mockInterviewService struct {
	repo repository.MockInterviewRepository
}

func NewMockInterviewService(repo repository.MockInterviewRepository) MockInterviewService {
	return &mockInterviewService{repo: repo}
}

func (s *mockInterviewService) SaveInterview(ctx context.Context, mi domain.MockInterview) (int64, error) {
	return s.repo.SaveInterview(ctx, mi)
}

func (s *mockInterviewService) ListInterviews(ctx context.Context, uid int64, limit, offset int) ([]domain.MockInterview, int64, error) {
	var (
		interviews []domain.MockInterview
		total      int64
		eg         errgroup.Group
	)
	// 并发执行两个查询
	eg.Go(func() error {
		var err error
		interviews, err = s.repo.FindInterviews(ctx, uid, limit, offset)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.CountInterviews(ctx, uid)
		return err
	})

	// 转换并返回结果
	return interviews, total, eg.Wait()
}

func (s *mockInterviewService) SaveQuestion(ctx context.Context, q domain.MockInterviewQuestion) (int64, error) {
	if err := q.Validate(); err != nil {
		return 0, err
	}
	return s.repo.SaveQuestion(ctx, q)
}

func (s *mockInterviewService) ListQuestions(ctx context.Context, interviewID, uid int64, limit, offset int) ([]domain.MockInterviewQuestion, int64, error) {

	var (
		questions []domain.MockInterviewQuestion
		total     int64
		eg        errgroup.Group
	)
	// 并发执行两个查询
	eg.Go(func() error {
		var err error
		questions, err = s.repo.FindQuestions(ctx, interviewID, uid, limit, offset)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.CountQuestions(ctx, interviewID, uid)
		return err
	})

	// 转换并返回结果
	return questions, total, eg.Wait()
}

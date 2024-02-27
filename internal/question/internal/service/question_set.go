package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
)

type QuestionSetService interface {
	Create(ctx context.Context, set domain.QuestionSet) (int64, error)
	UpdateQuestions(ctx context.Context, set domain.QuestionSet) error
	Detail(ctx context.Context, id, uid int64) (domain.QuestionSet, error)
}

type questionSetService struct {
	repo repository.QuestionSetRepository
}

func NewQuestionSetService(repo repository.QuestionSetRepository) QuestionSetService {
	return &questionSetService{repo: repo}
}

func (q *questionSetService) Create(ctx context.Context, set domain.QuestionSet) (int64, error) {
	return q.repo.Create(ctx, set)
}

func (q *questionSetService) UpdateQuestions(ctx context.Context, set domain.QuestionSet) error {
	return q.repo.UpdateQuestions(ctx, set)
}

func (q *questionSetService) Detail(ctx context.Context, id, uid int64) (domain.QuestionSet, error) {
	return q.repo.GetByIDAndUID(ctx, id, uid)
}
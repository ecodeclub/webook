package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
)

type CaseRepo interface {
	SearchCase(ctx context.Context, keywords string) ([]domain.Case, error)
}

type QuestionRepo interface {
	SearchQuestion(ctx context.Context, keywords string) ([]domain.Question, error)
}
type QuestionSetRepo interface {
	SearchQuestionSet(ctx context.Context, keywords string) ([]domain.QuestionSet, error)
}

type SkillRepo interface {
	SearchSkill(ctx context.Context, keywords string) ([]domain.Skill, error)
}

type AnyRepo interface {
	Input(ctx context.Context, index string, docID string, data string) error
}

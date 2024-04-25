package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
)

type CaseRepo interface {
	InputCase(ctx context.Context, msg domain.Case) error
	SearchCase(ctx context.Context, keywords []string) ([]domain.Case, error)
}

type QuestionRepo interface {
	InputQuestion(ctx context.Context, msg domain.Question) error
	SearchQuestion(ctx context.Context, keywords []string) ([]domain.Question, error)
}
type QuestionSetRepo interface {
	InputQuestionSet(ctx context.Context, msg domain.QuestionSet) error
	SearchQuestionSet(ctx context.Context, ids []int64, keywords []string) ([]domain.QuestionSet, error)
}

type SkillRepo interface {
	InputSKill(ctx context.Context, msg domain.Skill) error
	SearchSkill(ctx context.Context, qids, cids []int64, keywords []string) ([]domain.Skill, error)
}

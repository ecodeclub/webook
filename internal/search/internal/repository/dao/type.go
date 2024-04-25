package dao

import (
	"context"
)

var defaultSize = 20

type CaseDAO interface {
	SearchCase(ctx context.Context, keywords []string) ([]Case, error)
	InputCase(ctx context.Context, msg Case) error
}

type QuestionDAO interface {
	SearchQuestion(ctx context.Context, keywords []string) ([]Question, error)
	InputQuestion(ctx context.Context, msg Question) error
}

type SkillDAO interface {
	// ids 为case的id 和question的id
	SearchSkill(ctx context.Context, qids, cids []int64, keywords []string) ([]Skill, error)
	InputSkill(ctx context.Context, msg Skill) error
}

type QuestionSetDAO interface {
	// ids 为case的id 和question的id
	SearchQuestionSet(ctx context.Context, qids []int64, keywords []string) ([]QuestionSet, error)
	InputQuestionSet(ctx context.Context, msg QuestionSet) error
}

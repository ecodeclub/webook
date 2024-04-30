package dao

import (
	"context"
)

var defaultSize = 20

type CaseDAO interface {
	SearchCase(ctx context.Context, keywords string) ([]Case, error)
}

type QuestionDAO interface {
	SearchQuestion(ctx context.Context, keywords string) ([]Question, error)
}

type SkillDAO interface {
	// ids 为case的id 和question的id
	SearchSkill(ctx context.Context, keywords string) ([]Skill, error)
}

type QuestionSetDAO interface {
	// ids 为case的id 和question的id
	SearchQuestionSet(ctx context.Context, keywords string) ([]QuestionSet, error)
}

type AnyDAO interface {
	Input(ctx context.Context, index string, docID string, data string) error
}

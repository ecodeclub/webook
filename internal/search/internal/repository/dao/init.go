package dao

import (
	"context"
	_ "embed"
	"github.com/olivere/elastic/v7"
	"golang.org/x/sync/errgroup"
	"time"
)

var (
	//go:embed case_index.json
	caseIndex string
	//go:embed question_index.json
	questionIndex string
	//go:embed skill_index.json
	skillIndex string
	//go:embed questionset_index.json
	questionSetIndex string
)

// InitES 创建索引
func InitES(client *elastic.Client) error {
	const timeout = time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error {
		return tryCreateIndex(ctx, client, CaseIndexName, caseIndex)
	})
	eg.Go(func() error {
		return tryCreateIndex(ctx, client, QuestionIndexName, questionIndex)
	})
	eg.Go(func() error {
		return tryCreateIndex(ctx, client, SkillIndexName, skillIndex)
	})
	eg.Go(func() error {
		return tryCreateIndex(ctx, client, QuestionSetIndexName, questionSetIndex)
	})
	return eg.Wait()
}

func tryCreateIndex(ctx context.Context,
	client *elastic.Client,
	idxName, idxCfg string,
) error {
	// 索引可能已经建好了
	ok, err := client.IndexExists(idxName).Do(ctx)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	_, err = client.CreateIndex(idxName).Body(idxCfg).Do(ctx)
	return err
}

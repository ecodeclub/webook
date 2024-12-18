package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecodeclub/webook/internal/question/internal/domain"

	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"github.com/gotomicro/ego/core/elog"
)

type QuestionKnowledgeBase interface {
	FullSync()
}

type questionKnowledgeBase struct {
	queRepo          repository.Repository
	knowledgeBaseSvc ai.KnowledgeBaseService
	logger           *elog.Component
	knowledgeBaseId  string
}

func NewQuestionKnowledgeBase(knowledgeBaseId string, queRepo repository.Repository, knowledgeBaseSvc ai.KnowledgeBaseService) QuestionKnowledgeBase {
	return &questionKnowledgeBase{
		queRepo:          queRepo,
		knowledgeBaseId:  knowledgeBaseId,
		logger:           elog.DefaultLogger,
		knowledgeBaseSvc: knowledgeBaseSvc,
	}
}

func (q *questionKnowledgeBase) FullSync() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	qids, err := q.queRepo.QuestionIds(ctx)
	cancel()
	if err != nil {
		q.logger.Error("查找问题列表失败", elog.FieldErr(err))
		return
	}
	for _, qid := range qids {
		err = q.syncOne(qid)
		if err != nil {
			q.logger.Error(fmt.Sprintf("同步问题 %d失败", qid), elog.FieldErr(err))
		} else {
			q.logger.Info(fmt.Sprintf("同步问题 %d成功", qid))
		}
	}
}

func (q *questionKnowledgeBase) syncOne(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	que, err := q.queRepo.GetById(ctx, id)
	if err != nil {
		return fmt.Errorf("获取问题id列表失败 %w", err)
	}
	data, err := json.Marshal(que)
	if err != nil {
		return fmt.Errorf("序列化问题数据失败 %w", err)
	}
	err = q.knowledgeBaseSvc.UploadFile(ctx, ai.KnowledgeBaseFile{
		Biz:             domain.QuestionBiz,
		BizID:           que.Id,
		Name:            fmt.Sprintf("question_%d", que.Id),
		Data:            data,
		Type:            ai.RepositoryBaseTypeRetrieval,
		KnowledgeBaseID: q.knowledgeBaseId,
	})
	if err != nil {
		return fmt.Errorf("上传到ai的知识库失败 %w", err)
	}
	return err
}

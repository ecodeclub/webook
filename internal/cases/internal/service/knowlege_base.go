package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"github.com/gotomicro/ego/core/elog"
)

type KnowledgeBaseService interface {
	FullSync()
}

type knowledgeBaseSvc struct {
	caseRepo         repository.CaseRepo
	knowledgeBaseSvc ai.KnowledgeBaseService
	logger           *elog.Component
	knowledgeBaseId  string
}

func NewKnowledgeBaseService(repo repository.CaseRepo, svc ai.KnowledgeBaseService, knowledgeBaseId string) KnowledgeBaseService {
	return &knowledgeBaseSvc{
		caseRepo:         repo,
		knowledgeBaseSvc: svc,
		logger:           elog.DefaultLogger,
		knowledgeBaseId:  knowledgeBaseId,
	}
}

func (k *knowledgeBaseSvc) FullSync() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	cids, err := k.caseRepo.Ids(ctx)
	cancel()
	if err != nil {
		k.logger.Error("查找案例列表失败", elog.FieldErr(err))
		return
	}
	for _, cid := range cids {
		err = k.syncOne(cid)
		if err != nil {
			k.logger.Error(fmt.Sprintf("同步案例 %d失败", cid), elog.FieldErr(err))
		} else {
			k.logger.Info(fmt.Sprintf("同步案例 %d成功", cid))
		}
	}
}

func (k *knowledgeBaseSvc) syncOne(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ca, err := k.caseRepo.GetById(ctx, id)
	if err != nil {
		return fmt.Errorf("获取案例id列表失败 %w", err)
	}
	data, err := json.Marshal(ca)
	if err != nil {
		return fmt.Errorf("序列化案例数据失败 %w", err)
	}
	err = k.knowledgeBaseSvc.UploadFile(ctx, ai.KnowledgeBaseFile{
		Biz:             domain.BizCase,
		BizID:           ca.Id,
		Name:            fmt.Sprintf("case_%d", ca.Id),
		Data:            data,
		Type:            ai.RepositoryBaseTypeRetrieval,
		KnowledgeBaseID: k.knowledgeBaseId,
	})
	if err != nil {
		return fmt.Errorf("上传到ai的知识库失败 %w", err)
	}
	return err
}

package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
)

// KnowledgeBaseRepo 知识库文件和业务的映射表
type KnowledgeBaseRepo interface {
	Save(ctx context.Context, file domain.KnowledgeBaseFile) error
	GetInfo(ctx context.Context, platform, baseID, name string) (domain.KnowledgeBaseFile, error)
}

type repositoryBaseRepo struct {
	baseDao dao.KnowledgeBaseDAO
}

func NewKnowledgeBaseRepo(baseDao dao.KnowledgeBaseDAO) KnowledgeBaseRepo {
	return &repositoryBaseRepo{
		baseDao: baseDao,
	}
}
func (r *repositoryBaseRepo) Save(ctx context.Context, file domain.KnowledgeBaseFile) error {
	return r.baseDao.Save(ctx, dao.KnowledgeBaseFile{
		Biz:             file.Biz,
		BizID:           file.BizID,
		Name:            file.Name,
		FileID:          file.FileID,
		Platform:        file.Platform,
		KnowledgeBaseID: file.KnowledgeBaseID,
	})
}

func (r *repositoryBaseRepo) GetInfo(ctx context.Context, platform, baseID, name string) (domain.KnowledgeBaseFile, error) {
	file, err := r.baseDao.GetInfo(ctx, platform, baseID, name)
	if err != nil {
		return domain.KnowledgeBaseFile{}, err
	}
	return domain.KnowledgeBaseFile{
		Biz:      file.Biz,
		BizID:    file.BizID,
		Name:     file.Name,
		FileID:   file.FileID,
		Platform: file.Platform,
	}, nil
}

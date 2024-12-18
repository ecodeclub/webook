package knowledge_base

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
)

// RepositoryBaseSvc 提供一些通用操作知识库的方法
//
//go:generate mockgen -source=./type.go -destination=../../../../mocks/knowledge_base.mock.go -package=aimocks -typed=true RepositoryBaseSvc
type RepositoryBaseSvc interface {
	// UploadFile 上传文件
	UploadFile(ctx context.Context, file domain.KnowledgeBaseFile) error
}

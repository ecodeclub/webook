package zhipu

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	"github.com/gotomicro/ego/core/elog"
	"github.com/lukasjarosch/go-docx"
	"github.com/yankeguo/zhipu"
)

var (
	templateName string = "doc/template.docx"
)

type KnowledgeBase struct {
	client *zhipu.Client
	repo   repository.KnowledgeBaseRepo
	logger *elog.Component
}

func NewKnowledgeBase(apiKey string, repo repository.KnowledgeBaseRepo) (*KnowledgeBase, error) {
	client, err := zhipu.NewClient(zhipu.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &KnowledgeBase{
		client: client,
		repo:   repo,
		logger: elog.DefaultLogger,
	}, nil
}

// UploadFile 这个是后台管理接口，
func (r *KnowledgeBase) UploadFile(ctx context.Context, file domain.KnowledgeBaseFile) error {
	file.Platform = "zhipu"

	docName := file.Name + ".docx"
	// 直接写入的方法只有商用包才有，退而求其次才使用了这种方法生成word文档
	replaceMap := docx.PlaceholderMap{
		"content": string(file.Data),
	}

	doc, err := docx.Open(templateName)
	if err != nil {
		return fmt.Errorf("打开模版docx文件失败: %w", err)
	}

	err = doc.ReplaceAll(replaceMap)
	if err != nil {
		return fmt.Errorf("替换元素失败: %w", err)
	}
	// 添加内容
	err = doc.WriteToFile(docName)
	if err != nil {
		return fmt.Errorf("添加文件失败 %w", err)
	}
	// 延迟删除临时文件
	defer func() {
		err := os.Remove(docName)
		r.logger.Error("删除临时文件失败", elog.FieldErr(err))
	}()
	// 创建上传服务并上传文件
	// 先插入一条数据
	f, err := r.repo.GetInfo(ctx, file.Platform, file.KnowledgeBaseID, file.Name)
	switch {
	case err == nil:
		// 删除文件
		rerr := r.removeFile(ctx, f.FileID)
		if rerr != nil {
			return fmt.Errorf("智谱ai，删除文件失败 %w", rerr)
		}
	case errors.Is(err, dao.ErrBaseFileNotFound):
	default:
		return fmt.Errorf("查找文件失败 %w", err)
	}
	// 创建文件
	fileId, err := r.createFile(ctx, docName, file)
	if err != nil {
		return fmt.Errorf("智谱ai，创建文件失败 %w", err)
	}
	file.FileID = fileId
	err = r.repo.Save(ctx, file)
	if err != nil {
		return fmt.Errorf("保存文件信息失败 %w", err)
	}
	return nil
}

// 删除
func (r *KnowledgeBase) removeFile(ctx context.Context, docId string) error {
	service := r.client.FileDelete(docId)
	return service.Do(ctx)
}

// 创建
func (r *KnowledgeBase) createFile(ctx context.Context, docName string, file domain.KnowledgeBaseFile) (string, error) {
	service := r.client.FileCreate(file.Type)
	service.SetLocalFile(docName)
	service.SetKnowledgeID(file.KnowledgeBaseID)
	resp, err := service.Do(ctx)
	if err != nil {
		return "", err
	}
	if len(resp.FileCreateKnowledgeResponse.SuccessInfos) > 0 {
		return resp.FileCreateKnowledgeResponse.SuccessInfos[0].DocumentID, nil
	}
	return "", fmt.Errorf("创建文件失败")
}

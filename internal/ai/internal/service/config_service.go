package service

import (
	"context"
	"fmt"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
)

// ConfigService 定义配置服务的接口
type ConfigService interface {
	Save(ctx context.Context, cfg domain.BizConfig) (int64, error)
	List(ctx context.Context) ([]domain.BizConfig, error)
	GetById(ctx context.Context, id int64) (domain.BizConfig, error)
}

// configService 具体实现
type configService struct {
	repo repository.ConfigRepository
}

// NewConfigService 创建 ConfigService 实例
func NewConfigService(repo repository.ConfigRepository) ConfigService {
	return &configService{
		repo: repo,
	}
}

// Save 保存配置
func (s *configService) Save(ctx context.Context, cfg domain.BizConfig) (int64, error) {
	return s.repo.Save(ctx, cfg)
}

// List 获取所有配置列表
func (s *configService) List(ctx context.Context) ([]domain.BizConfig, error) {
	return s.repo.List(ctx)
}

// GetById 根据ID获取配置
func (s *configService) GetById(ctx context.Context, id int64) (domain.BizConfig, error) {
	if id <= 0 {
		return domain.BizConfig{}, fmt.Errorf("无效的ID")
	}
	return s.repo.GetById(ctx, id)
}

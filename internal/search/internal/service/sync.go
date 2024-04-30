package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/search/internal/repository"
)

type SyncService interface {
	Input(ctx context.Context, index string, docID string, data string) error
}
type syncService struct {
	anyRepo repository.AnyRepo
}

func (s *syncService) Input(ctx context.Context, index string, docID string, data string) error {
	return s.anyRepo.Input(ctx, index, docID, data)
}

func NewSyncSvc(anyRepo repository.AnyRepo) SyncService {
	return &syncService{
		anyRepo: anyRepo,
	}
}

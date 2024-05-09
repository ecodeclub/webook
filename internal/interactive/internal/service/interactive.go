package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/interactive/internal/domain"
	"github.com/ecodeclub/webook/internal/interactive/internal/repository"
	"golang.org/x/sync/errgroup"
)

type InteractiveService interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	Like(c context.Context, biz string, id int64, uid int64) error
	Collect(ctx context.Context, biz string, bizId, uid int64) error
	Get(ctx context.Context, biz string, id int64, uid int64) (domain.Interactive, error)
	GetByIds(ctx context.Context, biz string, ids []int64) ([]domain.Interactive, error)
}

type interactiveService struct {
	repo repository.InteractiveRepository
}

func NewService(repo repository.InteractiveRepository) InteractiveService {
	return &interactiveService{
		repo: repo,
	}
}

func (i *interactiveService) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	return i.repo.IncrViewCnt(ctx, biz, bizId)
}

func (i *interactiveService) Like(c context.Context, biz string, id int64, uid int64) error {
	return i.repo.Like(c, biz, id, uid)
}

func (i *interactiveService) Collect(ctx context.Context, biz string, bizId, uid int64) error {
	return i.repo.Collect(ctx, biz, bizId, uid)
}

func (i *interactiveService) Get(ctx context.Context, biz string, id int64, uid int64) (domain.Interactive, error) {
	intr, err := i.repo.Get(ctx, biz, id)
	if err != nil {
		return domain.Interactive{}, err
	}
	var eg errgroup.Group
	eg.Go(func() error {
		var er error
		intr.Liked, er = i.repo.Liked(ctx, biz, id, uid)
		return er
	})
	eg.Go(func() error {
		var er error
		intr.Collected, er = i.repo.Collected(ctx, biz, id, uid)
		return er
	})
	return intr, eg.Wait()
}

func (i *interactiveService) GetByIds(ctx context.Context, biz string, ids []int64) ([]domain.Interactive, error) {
	return i.repo.GetByIds(ctx, biz, ids)
}

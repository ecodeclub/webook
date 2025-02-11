package service

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/review/internal/event"
	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/webook/internal/review/internal/domain"
	"github.com/ecodeclub/webook/internal/review/internal/repository"
	"golang.org/x/sync/errgroup"
)

type ReviewSvc interface {
	Save(ctx context.Context, re domain.Review) (int64, error)
	List(ctx context.Context, offset, limit int) (int64, []domain.Review, error)

	Info(ctx context.Context, id int64) (domain.Review, error)

	Publish(ctx context.Context, re domain.Review) (int64, error)
	PubList(ctx context.Context, offset, limit int) ([]domain.Review, error)
	PubInfo(ctx context.Context, id int64) (domain.Review, error)
}

func NewReviewSvc(repo repository.ReviewRepo, intrProducer event.InteractiveEventProducer) ReviewSvc {
	return &reviewSvc{
		repo:         repo,
		logger:       elog.DefaultLogger,
		intrProducer: intrProducer,
	}
}

type reviewSvc struct {
	repo         repository.ReviewRepo
	logger       *elog.Component
	intrProducer event.InteractiveEventProducer
}

func (r *reviewSvc) Save(ctx context.Context, re domain.Review) (int64, error) {
	re.Status = domain.UnPublishedStatus
	return r.repo.Save(ctx, re)
}

func (r *reviewSvc) List(ctx context.Context, offset, limit int) (int64, []domain.Review, error) {
	var eg errgroup.Group
	var count int64
	var reviews []domain.Review
	eg.Go(func() error {
		var eerr error
		reviews, eerr = r.repo.List(ctx, offset, limit)
		return eerr
	})
	eg.Go(func() error {
		var eerr error
		count, eerr = r.repo.Count(ctx)
		return eerr
	})
	err := eg.Wait()
	return count, reviews, err

}

func (r *reviewSvc) Info(ctx context.Context, id int64) (domain.Review, error) {
	return r.repo.Info(ctx, id)
}

func (r *reviewSvc) Publish(ctx context.Context, re domain.Review) (int64, error) {
	re.Status = domain.PublishedStatus
	return r.repo.Publish(ctx, re)
}

func (r *reviewSvc) PubList(ctx context.Context, offset, limit int) ([]domain.Review, error) {
	return r.repo.PubList(ctx, offset, limit)
}

func (r *reviewSvc) PubInfo(ctx context.Context, id int64) (domain.Review, error) {
	re, err := r.repo.PubInfo(ctx, id)
	if err == nil {
		go func() {
			newCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			err1 := r.intrProducer.Produce(newCtx, event.NewViewCntEvent(id, domain.ReviewBiz))
			if err1 != nil {
				if err1 != nil {
					r.logger.Error("发送面经阅读计数消息到消息队列失败",
						elog.FieldErr(err1),
						elog.Int64("reviewId", id))
				}
			}
		}()
	}
	return re, err
}

package repository

import (
	"context"

	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/review/internal/repository/cache"
	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/review/internal/domain"
	"github.com/ecodeclub/webook/internal/review/internal/repository/dao"
)

type ReviewRepo interface {
	Save(ctx context.Context, re domain.Review) (int64, error)
	List(ctx context.Context, offset, limit int) ([]domain.Review, error)
	Count(ctx context.Context) (int64, error)
	Info(ctx context.Context, id int64) (domain.Review, error)

	Publish(ctx context.Context, re domain.Review) (int64, error)
	PubList(ctx context.Context, offset, limit int) ([]domain.Review, error)
	PubInfo(ctx context.Context, id int64) (domain.Review, error)
}
type reviewRepo struct {
	reviewDao   dao.ReviewDAO
	reviewCache cache.ReviewCache
	logger      *elog.Component
}

func NewReviewRepo(reviewDao dao.ReviewDAO, reviewCache cache.ReviewCache) ReviewRepo {
	return &reviewRepo{
		reviewDao:   reviewDao,
		reviewCache: reviewCache,
		logger:      elog.DefaultLogger,
	}
}

func (r *reviewRepo) Save(ctx context.Context, re domain.Review) (int64, error) {
	daoReview := toDaoReview(re)
	return r.reviewDao.Save(ctx, daoReview)
}

func (r *reviewRepo) List(ctx context.Context, offset, limit int) ([]domain.Review, error) {
	reviews, err := r.reviewDao.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	list := slice.Map(reviews, func(idx int, src dao.Review) domain.Review {
		return toDomainReview(src)
	})
	return list, nil
}

func (r *reviewRepo) Count(ctx context.Context) (int64, error) {
	return r.reviewDao.Count(ctx)
}

func (r *reviewRepo) Info(ctx context.Context, id int64) (domain.Review, error) {
	review, err := r.reviewDao.Get(ctx, id)
	if err != nil {
		return domain.Review{}, err
	}
	return toDomainReview(review), nil
}

func (r *reviewRepo) Publish(ctx context.Context, re domain.Review) (int64, error) {
	reDao, err := r.reviewDao.Sync(ctx, toDaoReview(re))
	if err != nil {
		return 0, err
	}
	cacheErr := r.reviewCache.SetReview(ctx, toDomainReview(reDao))
	if cacheErr != nil {
		r.logger.Error("设置面经缓存失败", elog.FieldErr(cacheErr), elog.Int64("review_id", reDao.ID))
	}
	return reDao.ID, nil
}

func (r *reviewRepo) PubList(ctx context.Context, offset, limit int) ([]domain.Review, error) {
	pubReviews, err := r.reviewDao.PublishReviewList(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	list := slice.Map(pubReviews, func(idx int, src dao.PublishReview) domain.Review {
		return toDomainReview(dao.Review(src))
	})
	return list, nil
}

func (r *reviewRepo) PubInfo(ctx context.Context, id int64) (domain.Review, error) {
	// 先尝试从缓存获取
	re, err := r.reviewCache.GetReview(ctx, id)
	if err == nil {
		return re, nil
	}
	// 缓存未命中时回源查询
	pubReview, err := r.reviewDao.GetPublishReview(ctx, id)
	if err != nil {
		return domain.Review{}, err
	}
	domainRe := toDomainReview(dao.Review(pubReview))
	if cacheErr := r.reviewCache.SetReview(ctx, domainRe); cacheErr != nil {
		r.logger.Error("设置发布面经缓存失败", elog.FieldErr(cacheErr), elog.Int64("review_id", id))
	}
	return domainRe, nil
}

// 将 domain.Review 转换为 dao.Review
func toDaoReview(review domain.Review) dao.Review {
	return dao.Review{
		ID:               review.ID,
		Uid:              review.Uid,
		Title:            review.Title,
		Desc:             review.Desc,
		Labels:           sqlx.JsonColumn[[]string]{Val: review.Labels, Valid: len(review.Labels) != 0},
		JD:               review.JD,
		JDAnalysis:       review.JDAnalysis,
		Questions:        review.Questions,
		QuestionAnalysis: review.QuestionAnalysis,
		Status:           review.Status.ToUint8(),
		Resume:           review.Resume,
	}
}

// 将 dao.Review 转换为 domain.Review
func toDomainReview(review dao.Review) domain.Review {
	return domain.Review{
		ID:               review.ID,
		Uid:              review.Uid,
		JD:               review.JD,
		Title:            review.Title,
		Desc:             review.Desc,
		Labels:           review.Labels.Val,
		JDAnalysis:       review.JDAnalysis,
		Questions:        review.Questions,
		QuestionAnalysis: review.QuestionAnalysis,
		Resume:           review.Resume,
		Status:           domain.ReviewStatus(review.Status),
		Utime:            review.Utime,
	}
}

package repository

import (
	"context"

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
	reviewDao dao.ReviewDAO
}

func NewReviewRepo(reviewDao dao.ReviewDAO) ReviewRepo {
	return &reviewRepo{
		reviewDao: reviewDao,
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
	return r.reviewDao.Sync(ctx, toDaoReview(re))
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
	pubReview, err := r.reviewDao.GetPublishReview(ctx, id)
	if err != nil {
		return domain.Review{}, err
	}
	return toDomainReview(dao.Review(pubReview)), nil
}

// 将 domain.Review 转换为 dao.Review
func toDaoReview(review domain.Review) dao.Review {
	return dao.Review{
		ID:               review.ID,
		Uid:              review.Uid,
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
		JD:               review.JD,
		JDAnalysis:       review.JDAnalysis,
		Questions:        review.Questions,
		QuestionAnalysis: review.QuestionAnalysis,
		Resume:           review.Resume,
		Status:           domain.ReviewStatus(review.Status),
		Utime:            review.Utime,
	}
}

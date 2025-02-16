package dao

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/review/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ego-component/egorm"
)

type ReviewDAO interface {
	// Create 创建一条面试评测记录
	Save(ctx context.Context, review Review) (int64, error)

	// Get 根据ID获取面试评测记录
	Get(ctx context.Context, id int64) (Review, error)

	// List 获取面试评测记录列表，支持分页
	List(ctx context.Context, offset, limit int) ([]Review, error)

	// Count 获取面试评测记录总数
	Count(ctx context.Context) (int64, error)

	// Sync 同步到线上库
	Sync(ctx context.Context, c Review) (Review, error)
	PublishReviewList(ctx context.Context, offset, limit int) ([]PublishReview, error)
	GetPublishReview(ctx context.Context, reviewId int64) (PublishReview, error)
}

type reviewDao struct {
	db *egorm.Component
}

func NewReviewDAO(db *egorm.Component) ReviewDAO {
	return &reviewDao{
		db: db,
	}
}

func (r *reviewDao) Save(ctx context.Context, review Review) (int64, error) {
	now := time.Now().UnixMilli()
	review.Utime = now
	review.Ctime = now
	return r.save(r.db.WithContext(ctx), review)
}

func (r *reviewDao) Get(ctx context.Context, id int64) (Review, error) {
	var review Review
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&review).Error
	if err != nil {
		return Review{}, err
	}
	return review, nil
}

func (r *reviewDao) List(ctx context.Context, offset, limit int) ([]Review, error) {
	var reviews []Review
	err := r.db.WithContext(ctx).
		Order("id desc").
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error
	if err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *reviewDao) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Review{}).Count(&count).Error
	return count, err
}

func (r *reviewDao) save(db *gorm.DB, review Review) (int64, error) {

	err := db.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns(r.getUpdateCols()),
	}).Create(&review).Error
	return review.ID, err
}

func (r *reviewDao) Sync(ctx context.Context, re Review) (Review, error) {
	var id = re.ID
	now := time.Now().UnixMilli()
	re.Ctime = now
	re.Utime = now
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		id, err = r.save(tx, re)
		if err != nil {
			return err
		}
		re.ID = id
		pubReview := PublishReview(re)
		return tx.Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns(r.getUpdateCols()),
		}).Create(&pubReview).Error
	})
	return re, err
}

func (r *reviewDao) PublishReviewList(ctx context.Context, offset, limit int) ([]PublishReview, error) {
	var publishReviews []PublishReview
	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("id DESC"). // 按ID降序排序，最新的记录在前面
		Where("status = ?", domain.PublishedStatus).
		Find(&publishReviews).Error
	if err != nil {
		return nil, err
	}
	return publishReviews, nil
}

func (r *reviewDao) GetPublishReview(ctx context.Context, reviewId int64) (PublishReview, error) {
	var publishReview PublishReview
	err := r.db.WithContext(ctx).
		Where("id = ?", reviewId).
		First(&publishReview).Error
	if err != nil {
		return PublishReview{}, err
	}
	return publishReview, nil
}

func (r *reviewDao) getUpdateCols() []string {
	return []string{
		"jd",
		"jd_analysis",
		"questions",
		"question_analysis",
		"resume",
		"utime",
		"status",
		"title",
		"desc",
		"labels",
	}
}

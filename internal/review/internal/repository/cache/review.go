package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/review/internal/domain"
	"github.com/pkg/errors"
)

const (
	reviewExpiration = 24 * time.Hour
)

var (
	ErrReviewNotFound = errors.New("面经没找到")
)

type ReviewCache interface {
	SetReview(ctx context.Context, re domain.Review) error
	GetReview(ctx context.Context, id int64) (domain.Review, error)
}
type reviewCache struct {
	ec ecache.Cache
}

func NewReviewCache(ec ecache.Cache) ReviewCache {
	return &reviewCache{
		ec: &ecache.NamespaceCache{
			C:         ec,
			Namespace: "review:",
		},
	}
}

func (r *reviewCache) SetReview(ctx context.Context, re domain.Review) error {
	reviewByte, err := json.Marshal(re)
	if err != nil {
		return errors.Wrap(err, "序列化面经失败")
	}
	return r.ec.Set(ctx, r.reviewKey(re.ID), string(reviewByte), reviewExpiration)
}

func (r *reviewCache) GetReview(ctx context.Context, id int64) (domain.Review, error) {
	val := r.ec.Get(ctx, r.reviewKey(id))
	if val.KeyNotFound() {
		return domain.Review{}, ErrReviewNotFound
	}
	if val.Err != nil {
		return domain.Review{}, errors.Wrap(val.Err, "查询缓存出错")
	}

	var re domain.Review
	err := json.Unmarshal([]byte(val.Val.(string)), &re)
	if err != nil {
		return domain.Review{}, errors.Wrap(err, "反序列化评价失败")
	}
	return re, nil
}
func (r *reviewCache) reviewKey(id int64) string {
	return fmt.Sprintf("publish:%d", id)
}

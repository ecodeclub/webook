package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/pkg/errors"
)

type CaseCache interface {
	SetCase(ctx context.Context, ca domain.Case) error
	GetCase(ctx context.Context, id int64) (domain.Case, error)
	SetCases(ctx context.Context, biz string, cas []domain.Case) error
	GetCases(ctx context.Context, biz string) ([]domain.Case, error)
	GetTotal(ctx context.Context, biz string) (int64, error)
	SetTotal(ctx context.Context, biz string, total int64) error
}

const (
	expiration = 24 * time.Hour
)

var (
	ErrCaseNotFound = errors.New("案例没找到")
)

type caseCache struct {
	ec ecache.Cache
}

func (c *caseCache) SetCases(ctx context.Context, biz string, cas []domain.Case) error {
	bytes, err := json.Marshal(cas)
	if err != nil {
		return errors.Wrap(err, "序列化案例列表失败")
	}
	return c.ec.Set(ctx, c.casesKey(biz), string(bytes), expiration)
}

func (c *caseCache) GetCases(ctx context.Context, biz string) ([]domain.Case, error) {
	val := c.ec.Get(ctx, c.casesKey(biz))
	if val.KeyNotFound() {
		return nil, ErrCaseNotFound
	}
	if val.Err != nil {
		return nil, val.Err
	}

	var res []domain.Case
	err := json.Unmarshal([]byte(val.Val.(string)), &res)
	return res, errors.Wrap(err, "反序列化案例列表失败")
}

func (c *caseCache) GetTotal(ctx context.Context, biz string) (int64, error) {
	val := c.ec.Get(ctx, c.totalKey(biz))
	if val.KeyNotFound() {
		return 0, ErrCaseNotFound
	}
	if val.Err != nil {
		return 0, val.Err
	}
	ans, err := val.String()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(ans, 10, 64)
}

func (c *caseCache) SetTotal(ctx context.Context, biz string, total int64) error {
	return c.ec.Set(ctx, c.totalKey(biz), total, expiration)
}

func NewCaseCache(ec ecache.Cache) CaseCache {
	return &caseCache{
		ec: &ecache.NamespaceCache{
			C:         ec,
			Namespace: "cases:",
		},
	}
}
func (c *caseCache) SetCase(ctx context.Context, ca domain.Case) error {
	cabyte, err := json.Marshal(ca)
	if err != nil {
		return err
	}
	return c.ec.Set(ctx, c.caseKey(ca.Id), string(cabyte), expiration)
}

func (c *caseCache) GetCase(ctx context.Context, id int64) (domain.Case, error) {
	caVal := c.ec.Get(ctx, c.caseKey(id))
	if caVal.KeyNotFound() {
		return domain.Case{}, ErrCaseNotFound
	}
	if caVal.Err != nil {
		return domain.Case{}, caVal.Err
	}

	var ca domain.Case
	err := json.Unmarshal([]byte(caVal.Val.(string)), &ca)
	if err != nil {
		return domain.Case{}, err
	}
	return ca, nil
}

func (c *caseCache) caseKey(id int64) string {
	return fmt.Sprintf("publish:%d", id)
}

// 新增以下 key 生成方法（放在 caseKey 方法附近）
func (c *caseCache) casesKey(biz string) string {
	return fmt.Sprintf("list:%s", biz)
}

func (c *caseCache) totalKey(biz string) string {
	return fmt.Sprintf("total:%s", biz)
}

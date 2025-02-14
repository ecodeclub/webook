package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/pkg/errors"
	"time"
)

type CaseCache interface {
	SetCase(ctx context.Context, ca domain.Case) error
	GetCase(ctx context.Context, id int64) (domain.Case, error)
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

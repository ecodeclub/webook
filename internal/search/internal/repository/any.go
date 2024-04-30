package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
)

type anyRepo struct {
	anyDao dao.AnyDAO
}

func NewAnyRepo(anyDao dao.AnyDAO) AnyRepo {
	return &anyRepo{
		anyDao: anyDao,
	}
}
func (a *anyRepo) Input(ctx context.Context, index string, docID string, data string) error {
	return a.anyDao.Input(ctx, index, docID, data)
}

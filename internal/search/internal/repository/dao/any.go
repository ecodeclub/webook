package dao

import (
	"context"

	"github.com/olivere/elastic/v7"
)

type anyESDAO struct {
	client *elastic.Client
}

func NewAnyEsDAO(client *elastic.Client) AnyDAO {
	return &anyESDAO{
		client: client,
	}
}

func (a *anyESDAO) Input(ctx context.Context, index string, docID string, data string) error {
	_, err := a.client.Index().
		Index(index).
		Id(docID).
		BodyJson(data).Do(ctx)
	return err
}

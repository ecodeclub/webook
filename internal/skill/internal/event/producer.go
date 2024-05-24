package event

import (
	"context"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/pkg/mqx"
)

const (
	SyncTopic = "sync_data_to_search"
)

type SyncEventProducer interface {
	Produce(ctx context.Context, evt SkillEvent) error
}

func NewSyncEventProducer(q mq.MQ) (SyncEventProducer, error) {
	return mqx.NewGeneralProducer[SkillEvent](q, SyncTopic)
}

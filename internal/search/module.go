package baguwen

import "github.com/ecodeclub/webook/internal/search/internal/event"

type Module struct {
	SearchSvc SearchService
	SyncSvc   SyncService
	c         *event.SyncConsumer
	Hdl       *Handler
}

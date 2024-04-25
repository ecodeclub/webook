package baguwen

type Module struct {
	SearchSvc SearchService
	SyncSvc   SyncService
	Hdl       *Handler
}

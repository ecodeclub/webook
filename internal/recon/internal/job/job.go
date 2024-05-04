package job

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/recon/internal/service"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/task/ecron"
)

var _ ecron.NamedJob = (*SyncPaymentAndOrderJob)(nil)

type SyncPaymentAndOrderJob struct {
	svc     service.Service
	minutes int64
	seconds int64
	limit   int
	l       *elog.Component
}

func NewSyncPaymentAndOrderJob(svc service.Service, minutes int64, seconds int64, limit int) *SyncPaymentAndOrderJob {
	return &SyncPaymentAndOrderJob{
		svc:     svc,
		minutes: minutes,
		seconds: seconds,
		limit:   limit,
		l:       elog.DefaultLogger}
}

func (s *SyncPaymentAndOrderJob) Name() string {
	return "sync_payment_and_order_job"
}

func (s *SyncPaymentAndOrderJob) Run(ctx context.Context) error {
	ctime := time.Now().Add(time.Duration(-s.minutes)*time.Minute + time.Duration(-s.seconds)*time.Second).UnixMilli()
	return s.svc.Reconcile(ctx, 0, s.limit, ctime)
}

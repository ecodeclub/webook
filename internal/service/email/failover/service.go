package failover

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/ecodeclub/webook/internal/service/email"
	"go.uber.org/zap"
)

type FailoverEmailService struct {
	svcs []email.Service
	idx  uint64
}

func NewFailoverEmailService(svcs []email.Service) *FailoverEmailService {
	return &FailoverEmailService{
		svcs: svcs,
		idx:  0,
	}
}

func (f *FailoverEmailService) Send(ctx context.Context, to, subject string, content []byte) error {
	idx := atomic.AddUint64(&f.idx, 1)
	length := len(f.svcs)
	for i := idx; i < idx+uint64(length); i++ {
		offset := i % uint64(length)
		err := f.svcs[offset].Send(ctx, to, subject, content)
		switch err {
		case nil:
			return nil
		case context.DeadlineExceeded, context.Canceled:
			return err
		default:
			zap.L().Info("发送邮件失败：", zap.Error(err))
		}
	}
	return errors.New("所有邮件服务都失败!")
}

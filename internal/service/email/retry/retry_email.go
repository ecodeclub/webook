package retry

import (
	"context"
	"errors"
	"time"

	"github.com/ecodeclub/ekit/retry"

	"github.com/ecodeclub/webook/internal/service/email"
)

var (
	overRetryTimes = errors.New("超过最大重试次数")
)

type Service struct {
	svc       email.Service
	retryFunc func() retry.Strategy
}

func NewRetryEmailService(svc email.Service, fac func() retry.Strategy) *Service {
	return &Service{
		svc:       svc,
		retryFunc: fac,
	}
}

func (s *Service) Send(ctx context.Context, subject, to string, content []byte) error {
	var retryTimer *time.Timer
	retryFunc := s.retryFunc()
	defer func() {
		//谨慎一下
		if retryTimer != nil {
			retryTimer.Stop()
		}
	}()
	for {
		err := s.svc.Send(ctx, subject, to, content)
		if err == nil {
			//第一次就成功
			return nil
		}
		//如果第一次发送就 “超时”或者被“调用者取消” 就没必须继续重试了
		if err != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
			return err
		}
		//开始重试
		timeInterval, try := retryFunc.Next()
		if !try {
			return overRetryTimes
		}
		if retryTimer == nil {
			retryTimer = time.NewTimer(timeInterval)
		} else {
			//retryTimer.Stop()
			retryTimer.Reset(timeInterval)
		}

		//重试的过程还要判断是不是已经超时了
		select {
		case <-ctx.Done():
			//超时直接退出

			return ctx.Err()
		case <-retryTimer.C:
			//继续下一个循环

		}

	}
}

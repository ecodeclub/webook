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

//func NewService(svc email.Service, retryStrategy Strategy) email.Service {
//	return &Service{
//		svc:           svc,
//		retryStrategy: retryStrategy,
//	}
//}

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

//用户的自定义实现

//type EmailRetryStrategy struct {
//	//时间间隔
//	Interval time.Duration
//	//最大重试次数
//	MaxCnt int64
//	//当前重试次数
//	currentCnt int64
//}

//func NewEmailRetryStrategy() retry.Strategy {
//	return &EmailRetryStrategy{
//		Interval:   time.Millisecond * 200,
//		MaxCnt:     3,
//		currentCnt: 0,
//	}
//}

//func NewEmailRetryStrategyV1(duration time.Duration, maxCnt, curCnt int64) retry.Strategy {
//	return &EmailRetryStrategy{
//		Interval:   duration,
//		MaxCnt:     maxCnt,
//		currentCnt: curCnt,
//	}
//}
//
//func (e *EmailRetryStrategy) Next() (time.Duration, bool) {
//	if e.currentCnt >= e.MaxCnt {
//		return 0, false
//	}
//	e.currentCnt++
//	return e.Interval, true
//}

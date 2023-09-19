package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/ekit/retry"

	"github.com/ecodeclub/webook/internal/service/email"
	evcmocks "github.com/ecodeclub/webook/internal/service/email/mocks"
)

func TestService_Send(t *testing.T) {
	testCases := []struct {
		name       string
		ctxFun     func() (context.Context, context.CancelFunc)
		mocksEmail func(ctrl *gomock.Controller) email.Service
		retry      func() retry.Strategy
		subject    string
		to         string
		content    []byte
		wantErr    error
	}{
		{
			name: "首次发送成功",
			ctxFun: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				return ctx, cancel
			},
			mocksEmail: func(ctrl *gomock.Controller) email.Service {
				svc := evcmocks.NewMockService(ctrl)
				svc.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				return svc
			},
			retry: func() retry.Strategy {
				//使用ekit retry 包 已经定义好的实现
				strategy, err := retry.NewFixedIntervalRetryStrategy(time.Second, 3)
				if err != nil {
					panic(err)
				}
				return strategy
			},
			wantErr: nil,
		},
		{
			name: "首次发送失败,然后,超时(不进行重试)",
			ctxFun: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)

				return ctx, cancel
			},
			retry: func() retry.Strategy {
				strategy, err := retry.NewFixedIntervalRetryStrategy(time.Second, 3)
				if err != nil {
					panic(err)
				}
				return strategy
			},
			mocksEmail: func(ctrl *gomock.Controller) email.Service {
				svc := evcmocks.NewMockService(ctrl)
				//time.Sleep(time.Second * 2)
				//法1
				svc.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(context.DeadlineExceeded)
				//法2  本地测试能过 但是在github 提交有异常
				/*
					Got: context.Background.WithDeadline(2023-09-18 01:15:18.983456296 +0000 UTC m=+1.000517412 [-1.000415033s]) (*context.timerCtx)
					            Want: is equal to context.Background.WithDeadline(2023-09-18 01:15:18.983457096 +0000 UTC m=+1.000518212 [-1.000428533s]) (*context.timerCtx)
					        controller.go:251: missing call(s) to *evcmocks.MockService.Send(is equal to context.Background.WithDeadline(2023-09-18 01:15:18.983457096 +0000 UTC m=+1.000518212 [-1.000510435s])
				*/
				//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				//defer cancel()
				//time.Sleep(time.Second * 2)
				//svc.EXPECT().Send(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(ctx.Err())

				//写个延时 保证contex 必定超时
				return svc
			},
			wantErr: context.DeadlineExceeded,
		},
		{
			name: "首次发送失败,然后用户cancel(不进行重试)",
			ctxFun: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				return ctx, cancel
			},
			retry: func() retry.Strategy {
				strategy, err := retry.NewFixedIntervalRetryStrategy(time.Second, 3)
				if err != nil {
					panic(err)
				}
				return strategy
			},
			mocksEmail: func(ctrl *gomock.Controller) email.Service {
				svc := evcmocks.NewMockService(ctrl)
				//法1
				svc.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(context.Canceled)

				//这里直接调用cancel 有问题
				//missing call(s) to *evcmocks.MockService.Send(is equal to context.Background.WithDeadline(2023-09-18 08:33:39.7659259 +0800 +08 m=+3.016250601 [849.5447ms]) (*context.timerCtx)
				//
				//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				//cancel()
				//svc.EXPECT().Send(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(ctx.Err())
				return svc
			},
			wantErr: context.Canceled,
		},
		{
			name: "首次发送失败,然后进行重试,重试第一次超时",
			ctxFun: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				//写个延时 保证contex 必定超时
				time.Sleep(time.Second * 2)
				return ctx, cancel
			},
			retry: func() retry.Strategy {
				strategy, err := retry.NewFixedIntervalRetryStrategy(time.Second, 3)
				if err != nil {
					panic(err)
				}
				return strategy
			},
			mocksEmail: func(ctrl *gomock.Controller) email.Service {
				svc := evcmocks.NewMockService(ctrl)

				svc.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("未知错误"))
				//写个延时 保证contex 必定超时
				//time.Sleep(time.Second * 2)
				return svc
			},

			wantErr: context.DeadlineExceeded,
		},
		{
			name: "首次发送失败,然后进行重试,重试最大次数都失败",
			ctxFun: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				return ctx, cancel
			},
			retry: func() retry.Strategy {
				strategy, err := retry.NewFixedIntervalRetryStrategy(time.Millisecond*100, 3)
				if err != nil {
					panic(err)
				}
				return strategy
			},
			mocksEmail: func(ctrl *gomock.Controller) email.Service {
				svc := evcmocks.NewMockService(ctrl)
				svc.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(4).Return(errors.New("未知错误"))
				return svc
			},

			wantErr: overRetryTimes,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			//必须要把ctx 初始化 方便测试下面的ctx 的异常测试
			ctx, cancel := tc.ctxFun()
			retryService := NewRetryEmailService(tc.mocksEmail(ctl), tc.retry)
			err := retryService.Send(ctx, tc.subject, tc.to, tc.content)
			cancel()
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/webook/internal/service/email"
	evcmocks "github.com/ecodeclub/webook/internal/service/email/mocks"
)

func TestService_Send(t *testing.T) {
	testCases := []struct {
		name       string
		ctxFun     func() (context.Context, context.CancelFunc)
		mocksEmail func(ctrl *gomock.Controller) email.Service
		retry      Strategy
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
			wantErr: nil,
		},
		{
			name: "首次发送失败,然后,超时(不进行重试)",
			ctxFun: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				return ctx, cancel
			},
			mocksEmail: func(ctrl *gomock.Controller) email.Service {
				svc := evcmocks.NewMockService(ctrl)
				//time.Sleep(time.Second * 2)
				svc.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(context.DeadlineExceeded)
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
			mocksEmail: func(ctrl *gomock.Controller) email.Service {
				svc := evcmocks.NewMockService(ctrl)
				svc.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(context.Canceled)
				//写个延时 保证contex 必定超时

				return svc
			},
			wantErr: context.Canceled,
		},
		{
			name: "首次发送失败,然后进行重试,重试第一次超时",
			ctxFun: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				return ctx, cancel
			},
			mocksEmail: func(ctrl *gomock.Controller) email.Service {
				svc := evcmocks.NewMockService(ctrl)

				svc.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("未知错误"))
				//写个延时 保证contex 必定超时
				time.Sleep(time.Second * 2)
				return svc
			},
			retry: &EmailRetryStrategy{
				Interval:   time.Millisecond * 500,
				MaxCnt:     3,
				currentCnt: 0,
			},
			wantErr: context.DeadlineExceeded,
		},
		{
			name: "首次发送失败,然后进行重试,重试最大次数都失败",
			ctxFun: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				return ctx, cancel
			},
			mocksEmail: func(ctrl *gomock.Controller) email.Service {
				svc := evcmocks.NewMockService(ctrl)
				svc.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(4).Return(errors.New("未知错误"))
				return svc
			},
			retry: &EmailRetryStrategy{
				Interval:   time.Millisecond * 100,
				MaxCnt:     3,
				currentCnt: 0,
			},
			wantErr: OverRetryTimes,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			//必须要把ctx 初始化 方便测试下面的ctx 的异常测试
			ctx, cancel := tc.ctxFun()
			retryService := NewService(tc.mocksEmail(ctl), tc.retry)
			err := retryService.Send(ctx, tc.subject, tc.to, tc.content)
			cancel()
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

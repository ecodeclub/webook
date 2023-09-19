package failover

import (
	"context"
	"errors"
	"testing"

	"github.com/ecodeclub/webook/internal/service/email"
	evcmocks "github.com/ecodeclub/webook/internal/service/email/gomail/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestFailoverEmailService_Send(t *testing.T) {
	testCases := []struct {
		name    string
		ctx     context.Context
		to      string
		subject string
		body    string
		mock    func(*gomock.Controller) []email.Service
		wantErr error
	}{
		{
			name:    "发送成功",
			ctx:     context.Background(),
			to:      "to@163.com",
			subject: "test",
			body:    "test",
			mock: func(ctrl *gomock.Controller) []email.Service {
				svcs := make([]email.Service, 0)
				mock := evcmocks.NewMockService(ctrl)
				mock1 := evcmocks.NewMockService(ctrl)
				mock1.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				svcs = append(svcs, mock, mock1)
				return svcs
			},
			wantErr: nil,
		},
		{
			name:    "所有邮件服务都失败!",
			ctx:     context.Background(),
			to:      "to@163.com",
			subject: "test",
			body:    "test",
			mock: func(ctrl *gomock.Controller) []email.Service {
				svcs := make([]email.Service, 0)
				mock := evcmocks.NewMockService(ctrl)
				mock1 := evcmocks.NewMockService(ctrl)
				mock.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("发送失败"))
				mock1.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("发送失败"))
				svcs = append(svcs, mock, mock1)
				return svcs
			},
			wantErr: errors.New("所有邮件服务都失败!"),
		},
		{
			name:    "用户取消!",
			ctx:     context.Background(),
			to:      "to@163.com",
			subject: "test",
			body:    "test",
			mock: func(ctrl *gomock.Controller) []email.Service {
				svcs := make([]email.Service, 0)
				mock := evcmocks.NewMockService(ctrl)
				mock.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(context.Canceled)
				svcs = append(svcs, mock)
				return svcs
			},
			wantErr: context.Canceled,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			failoverSvc := NewFailoverEmailService(tt.mock(ctrl))
			err := failoverSvc.Send(tt.ctx, tt.to, tt.subject, []byte(tt.body))
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

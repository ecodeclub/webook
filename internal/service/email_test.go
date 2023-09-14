package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/webook/internal/service/mail"
	mailmocks "github.com/ecodeclub/webook/internal/service/mail/mocks"
)

func Test_emailService_Send(t *testing.T) {
	tests := []struct {
		name string

		mock func(ctrl *gomock.Controller) mail.Service

		// 输入
		ctx     context.Context
		from    string
		subject string
		body    string
		to      string
		isHTML  bool

		// 预期中的输出
		wantErr error
	}{
		{
			name: "发送成功",
			mock: func(ctrl *gomock.Controller) mail.Service {
				mailSvc := mailmocks.NewMockService(ctrl)
				mailSvc.EXPECT().Send(context.Background(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				return mailSvc
			},
			ctx: context.Background(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := tt.mock(ctrl)
			svc := NewEmailService(repo)
			err := svc.Send(tt.ctx, tt.from, tt.subject, tt.body, tt.to, tt.isHTML)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

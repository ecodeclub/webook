//go:build manual

package email

import (
	"context"
	"testing"

	"github.com/ecodeclub/webook/internal/ioc"
	"github.com/ecodeclub/webook/internal/repository/dao"
	daomocks "github.com/ecodeclub/webook/internal/repository/dao/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestEmailServic_Send(t *testing.T) {
	testCases := []struct {
		name    string
		ctx     context.Context
		to      string
		subject string
		body    string
		mock    func(*gomock.Controller) dao.UserDAO
		wantErr error
	}{
		{
			name:    "success",
			ctx:     context.Background(),
			to:      "to@163.com",
			subject: "test",
			body:    "test",
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				mock := daomocks.NewMockUserDAO(ctrl)
				return mock
			},
			wantErr: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			evc := NewEmailService(ioc.InitEmailCfg())
			err := evc.Send(tt.ctx, tt.subject, tt.to, []byte(tt.body))
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

package service

import (
	"context"
	"database/sql"
	"github.com/ecodeclub/webook/internal/repository"
	"github.com/ecodeclub/webook/internal/repository/dao"
	daomocks "github.com/ecodeclub/webook/internal/repository/dao/mocks"
	repomocks "github.com/ecodeclub/webook/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

// 用例中的email修改为测试者的邮箱
func TestUserEmailService_Send(t *testing.T) {
	testCases := []struct {
		name    string
		ctx     context.Context
		email   string
		mock    func(*gomock.Controller) dao.UserDAO
		wantErr error
	}{
		{
			name:  "success",
			ctx:   context.Background(),
			email: "junnyfeng@163.com",
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				mock := daomocks.NewMockUserDAO(ctrl)
				return mock
			},
			wantErr: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := repository.NewUserEmailRepository(tt.mock(ctrl))
			evc := NewUserEmailService(repo)
			err := evc.Send(tt.ctx, tt.email)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

// 测试用例有时效性，token需要当下生成
func TestUserEmailService_Verify(t *testing.T) {
	testCases := []struct {
		name    string
		ctx     context.Context
		token   string
		mock    func(*gomock.Controller) repository.EamilRepository
		wantErr error
	}{
		{
			name:  "success",
			ctx:   context.Background(),
			token: "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6Imp1bm55ZmVuZ0AxNjMuY29tIiwiZXhwIjoxNjk0NDM3NTgwfQ.aE7hUjKPT1FsZUjIB6Q0yhCWoU3rDB86P6fI2yHvvscv2TySjlDQprL5px7Et9ekgylvrmOSx6aAiT67qLxOmw",
			mock: func(ctrl *gomock.Controller) repository.EamilRepository {
				mock := repomocks.NewMockEamilRepository(ctrl)
				mock.EXPECT().FindByEmail(gomock.Any(), gomock.Any()).Return(dao.User{
					Email: "junnyfeng@163.com",
					Id:    1,
				}, nil)
				mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
				return mock
			},
			wantErr: nil,
		},
		{
			name:  "token 不合法",
			ctx:   context.Background(),
			token: "eyJFbWFpbCI6Imp1bm55ZmVuZ0AxNjMuY29tIiwiZXhwIjoxNjk0NDM3NTgwfQ.aE7hUjKPT1FsZUjIB6Q0yhCWoU3rDB86P6fI2yHvvscv2TySjlDQprL5px7Et9ekgylvrmOSx6aAiT67qLxOmw",
			mock: func(ctrl *gomock.Controller) repository.EamilRepository {
				mock := repomocks.NewMockEamilRepository(ctrl)
				return mock
			},
			wantErr: ErrTokenInvalid,
		},
		{
			name:  "重复确认",
			ctx:   context.Background(),
			token: "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6Imp1bm55ZmVuZ0AxNjMuY29tIiwiZXhwIjoxNjk0NDM3NTgwfQ.aE7hUjKPT1FsZUjIB6Q0yhCWoU3rDB86P6fI2yHvvscv2TySjlDQprL5px7Et9ekgylvrmOSx6aAiT67qLxOmw",
			mock: func(ctrl *gomock.Controller) repository.EamilRepository {
				mock := repomocks.NewMockEamilRepository(ctrl)
				mock.EXPECT().FindByEmail(gomock.Any(), gomock.Any()).Return(dao.User{
					Email: "junnyfeng@163.com",
					Id:    1,
					EmailVerify: sql.NullByte{
						Byte:  EmailVerified,
						Valid: true,
					},
				}, nil)
				return mock
			},
			wantErr: ErrEmailVertified,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			evc := NewUserEmailService(tt.mock(ctrl))
			err := evc.Verify(tt.ctx, tt.token)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

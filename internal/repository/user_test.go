package repository

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
	"webook/internal/domain"
	"webook/internal/repository/dao"
	daomocks "webook/internal/repository/dao/mocks"
)

func TestUserInfoRepository_Create(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name    string
		mock    func(*gomock.Controller) dao.UserDAO
		ctx     context.Context
		user    *domain.User
		wantErr error
	}{
		{
			name: "创建成功！",
			ctx:  context.Background(),
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				d := daomocks.NewMockUserDAO(ctrl)
				d.EXPECT().Insert(gomock.Any(), dao.User{
					Id:         123,
					Email:      "l0slakers@gmail.com",
					Password:   "123456",
					CreateTime: now.UnixMilli(),
					UpdateTime: now.UnixMilli(),
				}).Return(nil)
				return d
			},
			user: &domain.User{
				Id:       123,
				Email:    "l0slakers@gmail.com",
				Password: "123456",

				CreateTime: now,
				UpdateTime: now,
			},
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := NewUserInfoRepository(tc.mock(ctrl))
			err := repo.Create(tc.ctx, tc.user)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

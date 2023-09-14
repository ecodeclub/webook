package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/repository/dao"
	daomocks "github.com/ecodeclub/webook/internal/repository/dao/mocks"
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

func TestUserInfoRepository_UpdateEmailVerified(t *testing.T) {
	tests := []struct {
		name string
		// 返回 mock 的 UserDAO 和 UserCache
		mock func(ctrl *gomock.Controller) dao.UserDAO

		// 输入
		ctx   context.Context
		email string

		// 预期输出
		wantUser domain.User
		wantErr  error
	}{
		{
			name: "更新",
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				email := "foo@example.com"
				d := daomocks.NewMockUserDAO(ctrl)
				d.EXPECT().UpdateEmailVerifiedByEmail(context.Background(), email).
					Return(nil)
				return d
			},
			ctx:   context.Background(),
			email: "foo@example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			d := tt.mock(ctrl)
			repo := NewUserInfoRepository(d)
			err := repo.UpdateEmailVerified(tt.ctx, tt.email)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestUserInfoRepository_FindByEmail(t *testing.T) {
	// 因为存储的是毫秒数，也就是纳秒部分被去掉了
	// 所以我们需要利用 nowMs 来重建一个不含纳秒部分的 time.Time
	nowMs := time.Now().UnixMilli()
	now := time.UnixMilli(nowMs)
	tests := []struct {
		name string
		// 返回 mock 的 UserDAO 和 UserCache
		mock func(ctrl *gomock.Controller) dao.UserDAO

		// 输入
		ctx   context.Context
		email string

		// 预期输出
		wantUser domain.User
		wantErr  error
	}{
		{
			name: "找到用户",
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				email := "foo@example.com"
				d := daomocks.NewMockUserDAO(ctrl)
				d.EXPECT().FindByEmail(context.Background(), email).
					Return(dao.User{
						Id:            1,
						Email:         email,
						Password:      "$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
						EmailVerified: true,
						CreateTime:    nowMs,
						UpdateTime:    nowMs,
					}, nil)
				return d
			},
			ctx:   context.Background(),
			email: "foo@example.com",
			wantUser: domain.User{
				Id:            1,
				Email:         "foo@example.com",
				Password:      "$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
				EmailVerified: true,
				CreateTime:    now,
				UpdateTime:    now,
			},
		},
		{
			name: "没有找到用户",
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				email := "foo@example.com"
				d := daomocks.NewMockUserDAO(ctrl)
				d.EXPECT().FindByEmail(context.Background(), email).
					Return(dao.User{}, dao.ErrDataNotFound)
				return d
			},
			ctx:     context.Background(),
			email:   "foo@example.com",
			wantErr: ErrUserNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			d := tt.mock(ctrl)
			repo := NewUserInfoRepository(d)
			u, err := repo.FindByEmail(tt.ctx, tt.email)
			assert.Equal(t, tt.wantUser, u)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

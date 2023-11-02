package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/repository/dao"
	daomocks "github.com/ecodeclub/webook/internal/repository/dao/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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
	testCases := []struct {
		name    string
		mock    func(*gomock.Controller) dao.UserDAO
		ctx     context.Context
		user    *domain.User
		wantErr error
	}{
		{
			name: "更新成功！",
			ctx:  context.Background(),
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				d := daomocks.NewMockUserDAO(ctrl)
				d.EXPECT().UpdateEmailVerified(gomock.Any(), gomock.Any()).Return(nil)
				return d
			},
			user: &domain.User{
				Id:            1,
				Email:         "abc@qq.com",
				EmailVerified: true,
				Password:      "123456",
			},
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := NewUserInfoRepository(tc.mock(ctrl))
			err := repo.UpdateEmailVerified(tc.ctx, tc.user.Email)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestUserInfoRepository_FindByEmail(t *testing.T) {
	testCases := []struct {
		name    string
		mock    func(*gomock.Controller) dao.UserDAO
		ctx     context.Context
		email   string
		wantErr error
	}{
		{
			name: "通过邮件查找成功！",
			ctx:  context.Background(),
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				d := daomocks.NewMockUserDAO(ctrl)
				d.EXPECT().FindByEmail(gomock.Any(), gomock.Any()).Return(dao.User{
					Id:            1,
					Email:         "abc@qq.com",
					EmailVerified: false,
					Password:      "123456",
				}, nil)
				return d
			},
			email:   "abc@qq.com",
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := NewUserInfoRepository(tc.mock(ctrl))
			_, err := repo.FindByEmail(tc.ctx, tc.email)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestUserInfoRepository_UpdateUserProfile(t *testing.T) {
	testCases := []struct {
		name    string
		mock    func(*gomock.Controller) dao.UserDAO
		ctx     context.Context
		user    domain.User
		wantErr error
	}{
		{
			name: "更新失败",
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				d := daomocks.NewMockUserDAO(ctrl)
				d.EXPECT().UpdateUserProfile(gomock.Any(), gomock.Any()).Return(errors.New("更新失败"))
				return d
			},
			ctx: context.Background(),
			user: domain.User{
				Id:       1,
				NickName: "frankiejun",
				AboutMe:  "I am a good boy",
				Birthday: "1999-01-01",
			},
			wantErr: errors.New("更新失败"),
		},
		{
			name: "更新成功",
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				d := daomocks.NewMockUserDAO(ctrl)
				d.EXPECT().UpdateUserProfile(gomock.Any(), gomock.Any()).Return(nil)
				return d
			},
			ctx: context.Background(),
			user: domain.User{
				Id:       1,
				NickName: "frankiejun",
				AboutMe:  "I am a good boy",
				Birthday: "1999-01-01",
			},
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := NewUserInfoRepository(tc.mock(ctrl))
			err := repo.UpdateUserProfile(tc.ctx, tc.user)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestUserInfoRepository_FindById(t *testing.T) {
	testCases := []struct {
		name    string
		mock    func(*gomock.Controller) dao.UserDAO
		ctx     context.Context
		id      int64
		wantErr error
	}{
		{
			name: "查找成功！",
			ctx:  context.Background(),
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				d := daomocks.NewMockUserDAO(ctrl)
				d.EXPECT().FindById(gomock.Any(), gomock.Any()).Return(dao.User{
					Id:    1,
					Email: "abc@qq.com",
					NickName: sql.NullString{
						String: "frankiejun",
						Valid:  true,
					},
					Birthday: sql.NullString{
						String: "1999-01-01",
						Valid:  true,
					},
					AboutMe: sql.NullString{
						String: "I am a good boy",
						Valid:  true,
					},
				}, nil)
				return d
			},
			id:      1,
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := NewUserInfoRepository(tc.mock(ctrl))
			_, err := repo.FindById(tc.ctx, tc.id)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

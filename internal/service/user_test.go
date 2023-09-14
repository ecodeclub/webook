package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/repository"
	repomocks "github.com/ecodeclub/webook/internal/repository/mocks"
)

func TestUserService_Signup(t *testing.T) {
	lg, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal()
	}
	testCases := []struct {
		name    string
		mock    func(*gomock.Controller) repository.UserRepository
		user    *domain.User
		wantErr error
	}{
		{
			name: "注册成功！",
			user: &domain.User{
				Id:       123,
				Email:    "l0slakers@gmail.com",
				Password: "Abcd#1234",
			},
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				userRepo := repomocks.NewMockUserRepository(ctrl)
				userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
					Return(nil)
				return userRepo
			},
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := NewUserService(tc.mock(ctrl), lg)
			err := svc.Signup(context.Background(), tc.user)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestUserService_EmailVerify(t *testing.T) {
	lg, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal()
	}
	nowTime := time.Now()
	tests := []struct {
		name string

		mock func(ctrl *gomock.Controller) repository.UserRepository

		// 输入
		ctx   context.Context
		email string

		// 预期中的输出
		wantErr error
	}{
		{
			name: "验证成功",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				email := "foo@example.com"
				repo := repomocks.NewMockUserRepository(ctrl)
				repo.EXPECT().FindByEmail(context.Background(), email).
					Return(domain.User{
						Id:            1,
						Email:         email,
						Password:      "$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
						EmailVerified: false,
						CreateTime:    nowTime,
						UpdateTime:    nowTime,
					}, nil)
				repo.EXPECT().UpdateEmailVerified(context.Background(), email).
					Return(nil)
				return repo
			},
			ctx:     context.Background(),
			email:   "foo@example.com",
			wantErr: nil,
		},
		{
			name: "邮箱已验证",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				email := "foo@example.com"
				repo := repomocks.NewMockUserRepository(ctrl)
				repo.EXPECT().FindByEmail(context.Background(), email).
					Return(domain.User{
						Id:            1,
						Email:         email,
						Password:      "$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
						EmailVerified: true,
						CreateTime:    nowTime,
						UpdateTime:    nowTime,
					}, nil)
				return repo
			},
			ctx:     context.Background(),
			email:   "foo@example.com",
			wantErr: ErrUserEmailVerified,
		},
		{
			name: "验证成功",
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				email := "foo@example.com"
				repo := repomocks.NewMockUserRepository(ctrl)
				repo.EXPECT().FindByEmail(context.Background(), email).
					Return(domain.User{
						Id:            1,
						Email:         email,
						Password:      "$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
						EmailVerified: false,
						CreateTime:    nowTime,
						UpdateTime:    nowTime,
					}, nil)
				repo.EXPECT().UpdateEmailVerified(context.Background(), email).
					Return(errors.New("模拟系统错误"))
				return repo
			},
			ctx:     context.Background(),
			email:   "foo@example.com",
			wantErr: errors.New("模拟系统错误"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := tt.mock(ctrl)
			svc := NewUserService(repo, lg)
			err := svc.EmailVerify(tt.ctx, tt.email)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

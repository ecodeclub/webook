package service

import (
	"context"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/repository"
	repomocks "github.com/ecodeclub/webook/internal/repository/mocks"
	"github.com/ecodeclub/webook/internal/service/email"
	evcmocks "github.com/ecodeclub/webook/internal/service/email/gomail/mocks"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUserService_Signup(t *testing.T) {
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

			svc := NewUserService(tc.mock(ctrl), nil)
			err := svc.Signup(context.Background(), tc.user)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestUserService_SendVerifyEmail(t *testing.T) {
	testCases := []struct {
		name    string
		mock    func(*gomock.Controller) (repository.UserRepository, email.Service)
		email   string
		wantErr error
	}{
		{
			name:  "发送认证邮件成功",
			email: "abc@163.com",
			mock: func(ctrl *gomock.Controller) (repository.UserRepository, email.Service) {
				repo := repomocks.NewMockUserRepository(ctrl)
				emailsvc := evcmocks.NewMockService(ctrl)
				emailsvc.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				return repo, emailsvc
			},
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := NewUserService(tc.mock(ctrl))
			err := svc.SendVerifyEmail(context.Background(), tc.email)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestUserService_VerifyEmail(t *testing.T) {
	testCases := []struct {
		name    string
		ctx     context.Context
		mock    func(*gomock.Controller) repository.UserRepository
		token   string
		email   string
		wantErr error
	}{
		{
			name:  "success",
			email: "abc@qq.com",
			token: genToken("abc@qq.com", 1),
			ctx:   context.Background(),
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				mock := repomocks.NewMockUserRepository(ctrl)
				mock.EXPECT().UpdateEmailVerified(gomock.Any(), gomock.Any()).Return(nil)
				return mock
			},
			wantErr: nil,
		},
		{
			name:  "token 不合法",
			email: "abc@qq.com",
			token: "ereqr2332f2g2f23f23",
			ctx:   context.Background(),
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				mock := repomocks.NewMockUserRepository(ctrl)
				return mock
			},
			wantErr: ErrTokenInvalid,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := NewUserService(tc.mock(ctrl), nil)
			err := svc.VerifyEmail(context.Background(), tc.token)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func genToken(emailAddr string, timeout int) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, EmailClaims{
		Email: emailAddr,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * time.Duration(timeout))),
		},
	})
	tokenStr, _ := token.SignedString([]byte(EmailJWTKey))
	return tokenStr
}

func Test_userService_EditUserProfile(t *testing.T) {
	testCases := []struct {
		name    string
		ctx     context.Context
		mock    func(*gomock.Controller) repository.UserRepository
		user    domain.User
		wantErr error
	}{
		{
			name: "修改成功",
			ctx:  context.Background(),
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				mock := repomocks.NewMockUserRepository(ctrl)
				mock.EXPECT().UpdateUserProfile(gomock.Any(), gomock.Any()).Return(nil)
				return mock
			},
			user: domain.User{
				Id:       1,
				NickName: "frankiejun",
				Birthday: "2020-01-01",
				AboutMe:  "I am a good boy",
			},
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := NewUserService(tc.mock(ctrl), nil)
			err := svc.EditUserProfile(tc.ctx, tc.user)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func Test_userService_Profile(t *testing.T) {
	testCases := []struct {
		name    string
		ctx     context.Context
		mock    func(*gomock.Controller) repository.UserRepository
		id      int64
		wantErr error
	}{
		{
			name: "查找成功",
			ctx:  context.Background(),
			mock: func(ctrl *gomock.Controller) repository.UserRepository {
				mock := repomocks.NewMockUserRepository(ctrl)
				mock.EXPECT().FindById(gomock.Any(), gomock.Any()).Return(domain.User{
					Id:       1,
					NickName: "frankiejun",
					Birthday: "2020-01-01",
					AboutMe:  "I am a good boy",
				}, nil)
				return mock
			},
			id:      1,
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := NewUserService(tc.mock(ctrl), nil)
			_, err := svc.Profile(tc.ctx, tc.id)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

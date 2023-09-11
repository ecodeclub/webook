package web

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/service"
	svcmocks "github.com/ecodeclub/webook/internal/service/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Handler测试的主要难点
// 1.构造HTTP请求
// 2.验证HTTP响应
func TestUserHandler_SignUp(t *testing.T) {
	testCases := []struct {
		name     string
		mock     func(ctrl *gomock.Controller) (service.UserAndService, service.EmailService)
		body     string
		wantCode int
		wantBody string
	}{
		{
			name: "绑定信息错误！",
			mock: func(ctrl *gomock.Controller) (service.UserAndService, service.EmailService) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				emailSvc := svcmocks.NewMockEmailService(ctrl)
				return userSvc, emailSvc
			},
			body: `
		{
			"email": "l0slakers@gmail.com",
			"password": "Abcd#1234"
		`,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "两次输入密码不一致！",
			mock: func(ctrl *gomock.Controller) (service.UserAndService, service.EmailService) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				emailSvc := svcmocks.NewMockEmailService(ctrl)
				return userSvc, emailSvc
			},
			body: `
		{
			"email": "l0slakers@gmail.com",
			"password": "Abcd#12345678",
			"confirmPassword": "Abcd#1234"
		}
		`,
			wantCode: http.StatusBadRequest,
			wantBody: "两次密码不相同！",
		},
		{
			name: "密码格式不正确！",
			mock: func(ctrl *gomock.Controller) (service.UserAndService, service.EmailService) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				emailSvc := svcmocks.NewMockEmailService(ctrl)
				return userSvc, emailSvc
			},
			body: `
		{
			"email": "l0slakers@gmail.com",
			"password": "12",
			"confirmPassword": "12"
		}
		`,
			wantCode: http.StatusBadRequest,
			wantBody: "密码格式不正确,长度不能小于 6 位！",
		},
		{
			name: "重复邮箱！",
			mock: func(ctrl *gomock.Controller) (service.UserAndService, service.EmailService) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				emailSvc := svcmocks.NewMockEmailService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(service.ErrUserDuplicate)
				return userSvc, emailSvc
			},
			body: `
		{
			"email": "l0slakers@gmail.com",
			"password": "Abcd#1234",
			"confirmPassword": "Abcd#1234"
		}
		`,
			wantCode: http.StatusBadRequest,
			wantBody: "重复邮箱，请更换邮箱！",
		},
		{
			name: "系统错误！",
			mock: func(ctrl *gomock.Controller) (service.UserAndService, service.EmailService) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				emailSvc := svcmocks.NewMockEmailService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(errors.New("any error"))
				return userSvc, emailSvc
			},
			body: `
		{
			"email": "l0slakers@gmail.com",
			"password": "Abcd#1234",
			"confirmPassword": "Abcd#1234"
		}
		`,
			wantCode: http.StatusInternalServerError,
			wantBody: "系统错误！",
		},
		{
			name: "注册成功！",
			mock: func(ctrl *gomock.Controller) (service.UserAndService, service.EmailService) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				emailSvc := svcmocks.NewMockEmailService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(nil)
				emailSvc.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)
				return userSvc, emailSvc
			},
			body: `
{
	"email": "l0slakers@gmail.com",
	"password": "Abcd#1234",
	"confirmPassword": "Abcd#1234"
}
`,
			wantCode: http.StatusOK,
			wantBody: "注册成功！",
		},
	}
	gin.SetMode(gin.ReleaseMode)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := gin.Default()
			h := NewUserHandler(tc.mock(ctrl))
			h.RegisterRoutes(r)

			req, err := http.NewRequest(http.MethodPost, "/users/signup", bytes.NewBuffer([]byte(tc.body)))
			require.NoError(t, err)
			// 设置请求头
			req.Header.Set("Content-Type", "application/json")
			// http请求的记录
			resp := httptest.NewRecorder()

			// HTTP 请求进入 GIN 框架的入口
			// 调用此方法时，Gin 会处理这个请求，将响应写回 resp 里
			r.ServeHTTP(resp, req)

			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, tc.wantBody, resp.Body.String())
		})
	}
}

func TestUserHandler_EmailVerify(t *testing.T) {
	testCases := []struct {
		name     string
		mock     func(ctrl *gomock.Controller) (service.UserAndService, service.EmailService)
		body     string
		wantCode int
		wantBody string
	}{
		{
			name: "邮箱验证",
			mock: func(ctrl *gomock.Controller) (service.UserAndService, service.EmailService) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				emailSvc := svcmocks.NewMockEmailService(ctrl)
				emailSvc.EXPECT().Verify(gomock.Any(), gomock.Any()).Return(nil)
				return userSvc, emailSvc
			},
			body:     "",
			wantCode: http.StatusOK,
			wantBody: "验证成功!",
		},
		{
			name: "验证失败!",
			mock: func(ctrl *gomock.Controller) (service.UserAndService, service.EmailService) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				emailSvc := svcmocks.NewMockEmailService(ctrl)
				emailSvc.EXPECT().Verify(gomock.Any(), gomock.Any()).Return(service.ErrTokenInvalid)
				return userSvc, emailSvc
			},
			body:     "",
			wantCode: http.StatusOK,
			wantBody: "验证失败!",
		},
	}
	gin.SetMode(gin.ReleaseMode)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := gin.Default()
			h := NewUserHandler(tc.mock(ctrl))
			h.RegisterRoutes(r)

			req, err := http.NewRequest(http.MethodPost, "/users/emailverify/token", bytes.NewBuffer([]byte(tc.body)))

			require.NoError(t, err)
			// 设置请求头
			req.Header.Set("Content-Type", "application/json")
			// http请求的记录
			resp := httptest.NewRecorder()

			// HTTP 请求进入 GIN 框架的入口
			// 调用此方法时，Gin 会处理这个请求，将响应写回 resp 里
			r.ServeHTTP(resp, req)

			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, tc.wantBody, resp.Body.String())
		})
	}
}

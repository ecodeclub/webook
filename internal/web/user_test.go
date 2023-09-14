package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/service"
	svcmocks "github.com/ecodeclub/webook/internal/service/mocks"
	tokenGen "github.com/ecodeclub/webook/internal/web/token/generator"
	tokenmocks "github.com/ecodeclub/webook/internal/web/token/mocks"
	tokenVfy "github.com/ecodeclub/webook/internal/web/token/validator"
)

// Handler测试的主要难点
// 1.构造HTTP请求
// 2.验证HTTP响应
func TestUserHandler_SignUp(t *testing.T) {
	lg, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal()
	}
	testCases := []struct {
		name     string
		mock     func(ctrl *gomock.Controller) (service.UserService, tokenGen.TokenGenerator, tokenVfy.Verifier)
		body     string
		wantCode int
		wantBody string
	}{
		{
			name: "绑定信息错误！",
			mock: func(ctrl *gomock.Controller) (service.UserService,
				tokenGen.TokenGenerator, tokenVfy.Verifier) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				return userSvc, nil, nil
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
			mock: func(ctrl *gomock.Controller) (service.UserService,
				tokenGen.TokenGenerator, tokenVfy.Verifier) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				// userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
				//	Email:    "l0slakers@gmail.com",
				//	Password: "Abcd#1234",
				// })
				return userSvc, nil, nil
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
			mock: func(ctrl *gomock.Controller) (service.UserService,
				tokenGen.TokenGenerator, tokenVfy.Verifier) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				return userSvc, nil, nil
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
			mock: func(ctrl *gomock.Controller) (service.UserService,
				tokenGen.TokenGenerator, tokenVfy.Verifier) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(service.ErrUserDuplicate)
				return userSvc, nil, nil
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
			mock: func(ctrl *gomock.Controller) (service.UserService,
				tokenGen.TokenGenerator, tokenVfy.Verifier) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(errors.New("any error"))
				return userSvc, nil, nil
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
			mock: func(ctrl *gomock.Controller) (service.UserService,
				tokenGen.TokenGenerator, tokenVfy.Verifier) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(nil)
				tokenGenSvc := tokenmocks.NewMockTokenGenerator(ctrl)
				tokenGenSvc.EXPECT().GenerateToken(gomock.Any(), gomock.Any()).
					Return("token", nil)
				return userSvc, tokenGenSvc, nil
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
			userSvc, tokenGenSvc, tokenVfySvc := tc.mock(ctrl)

			r := gin.Default()
			h := NewUserHandler(userSvc, nil, tokenGenSvc,
				tokenVfySvc, "", lg)
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
			<-time.After(50 * time.Millisecond) // 测试goroutine

			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, tc.wantBody, resp.Body.String())
		})
	}
}

func TestUserHandler_EmailVerify(t *testing.T) {
	lg, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal()
	}
	const emailVerifyURL = "/users/email/verification"
	tests := []struct {
		name       string
		mock       func(ctrl *gomock.Controller) (service.UserService, tokenVfy.Verifier)
		reqBuilder func(t *testing.T) *http.Request
		wantCode   int
		wantBody   Result
	}{
		{
			name: "验证成功",
			mock: func(ctrl *gomock.Controller) (service.UserService, tokenVfy.Verifier) {
				email := "foo@example.com"
				verifier := tokenmocks.NewMockVerifier(ctrl)
				verifier.EXPECT().Verify(gomock.Any()).Return(email, nil)
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				userSvc.EXPECT().EmailVerify(gomock.Any(), email).Return(nil)
				return userSvc, verifier
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodGet,
					emailVerifyURL+"?code=jwt_token", nil)
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantCode: http.StatusOK,
			wantBody: Result{Msg: "验证成功"},
		},
		{
			name: "参数有误",
			mock: func(ctrl *gomock.Controller) (service.UserService, tokenVfy.Verifier) {
				return nil, nil
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodGet,
					emailVerifyURL, nil)
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantCode: http.StatusBadRequest,
			wantBody: Result{
				Code: CodeEmailVerifyFailed,
				Msg:  "验证失败",
			},
		},
		{
			name: "token错误",
			mock: func(ctrl *gomock.Controller) (service.UserService, tokenVfy.Verifier) {
				verifier := tokenmocks.NewMockVerifier(ctrl)
				verifier.EXPECT().Verify(gomock.Any()).Return("", errors.New("模拟verify错误"))
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				return userSvc, verifier
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodGet,
					emailVerifyURL+"?code=jwt_token", nil)
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantCode: http.StatusBadRequest,
			wantBody: Result{
				Code: CodeEmailVerifyFailed,
				Msg:  "验证失败",
			},
		},
		{
			name: "邮箱已验证",
			mock: func(ctrl *gomock.Controller) (service.UserService, tokenVfy.Verifier) {
				email := "foo@example.com"
				verifier := tokenmocks.NewMockVerifier(ctrl)
				verifier.EXPECT().Verify(gomock.Any()).Return(email, nil)
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				userSvc.EXPECT().EmailVerify(gomock.Any(), email).Return(service.ErrUserEmailVerified)
				return userSvc, verifier
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodGet,
					emailVerifyURL+"?code=jwt_token", nil)
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantCode: http.StatusBadRequest,
			wantBody: Result{
				Code: CodeEmailVerified,
				Msg:  "邮箱已验证",
			},
		},
		{
			name: "邮箱不存在",
			mock: func(ctrl *gomock.Controller) (service.UserService, tokenVfy.Verifier) {
				email := ""
				verifier := tokenmocks.NewMockVerifier(ctrl)
				verifier.EXPECT().Verify(gomock.Any()).Return(email, nil)
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				userSvc.EXPECT().EmailVerify(gomock.Any(), email).Return(errors.New("邮箱不存在"))
				return userSvc, verifier
			},
			reqBuilder: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodGet,
					emailVerifyURL+"?code=jwt_token", nil)
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantCode: http.StatusBadRequest,
			wantBody: Result{
				Code: CodeEmailVerifyFailed,
				Msg:  "验证失败",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			userSvc, emailVerifier := tt.mock(ctrl)
			// 利用 mock 来构造 UserHandler
			hdl := NewUserHandler(userSvc, nil, nil,
				emailVerifier, "", lg)
			// 注册路由
			server := gin.Default()
			hdl.RegisterRoutes(server)
			// 准备请求
			req := tt.reqBuilder(t)
			// 准备记录响应
			recorder := httptest.NewRecorder()
			// 执行
			server.ServeHTTP(recorder, req)
			// 断言
			assert.Equal(t, tt.wantCode, recorder.Code)
			if recorder.Code == http.StatusBadRequest {
				return
			}
			var res Result
			err := json.NewDecoder(recorder.Body).Decode(&res)
			if err != nil {
				t.Fatal()
			}
			assert.Equal(t, tt.wantBody, res)
		})
	}
}

func TestUserHandler_URLGenerator(t *testing.T) {
	g := tokenGen.NewJWTTokenGen("foo", "test")
	u := NewUserHandler(nil, nil, g, nil, "", nil)

	tests := []struct {
		name        string
		absoluteURL string
		params      map[string][]string
		want        string
		wantErr     bool
	}{
		{
			name:        "生成URL",
			absoluteURL: "https://example.com/foo/bar",
			params: map[string][]string{
				"foo": {"1", "2"},
			},
			want: "https://example.com/foo/bar?foo=1&foo=2",
		},
		{
			name:        "绝对URL错误",
			absoluteURL: "/bar",
			params: map[string][]string{
				"foo": {"1", "2"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := u.URLGenerator(tt.absoluteURL, tt.params)
			if tt.wantErr && err == nil {
				t.Errorf("want error; got no error")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("url generation failed: %v", err)
				return
			}
			assert.Equalf(t, tt.want, got, "URLGenerator(%v, %v)", tt.absoluteURL, tt.params)
		})
	}
}

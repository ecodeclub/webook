package web

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jwtMoudle "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/service"
	svcmocks "github.com/ecodeclub/webook/internal/service/mocks"
	"github.com/ecodeclub/webook/internal/web/encryption"
	"github.com/ecodeclub/webook/internal/web/encryption/jwt"
	jwtmocks "github.com/ecodeclub/webook/internal/web/encryption/jwt/mock"
)

// Handler测试的主要难点
// 1.构造HTTP请求
// 2.验证HTTP响应
func TestUserHandler_SignUp(t *testing.T) {
	testCases := []struct {
		name     string
		mock     func(ctrl *gomock.Controller) (service.UserAndService, encryption.Handle)
		body     string
		wantCode int
		wantBody string
	}{
		{
			name: "绑定信息错误！",
			mock: func(ctrl *gomock.Controller) (service.UserAndService, encryption.Handle) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				return userSvc, nil
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
			mock: func(ctrl *gomock.Controller) (service.UserAndService, encryption.Handle) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				//userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
				//	Email:    "l0slakers@gmail.com",
				//	Password: "Abcd#1234",
				//})
				return userSvc, nil
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
			mock: func(ctrl *gomock.Controller) (service.UserAndService, encryption.Handle) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				return userSvc, nil
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
			mock: func(ctrl *gomock.Controller) (service.UserAndService, encryption.Handle) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(service.ErrUserDuplicate)
				return userSvc, nil
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
			mock: func(ctrl *gomock.Controller) (service.UserAndService, encryption.Handle) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(errors.New("any error"))
				return userSvc, nil
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
			mock: func(ctrl *gomock.Controller) (service.UserAndService, encryption.Handle) {
				userSvc := svcmocks.NewMockUserAndService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(nil)
				return userSvc, nil
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

func TestUserHandle_TokenLogin(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name        string
		mock        func(ctl *gomock.Controller) (service.UserAndService, encryption.Handle)
		reqBody     string
		wantCode    int
		wantBody    string
		fingerprint string
		//userId   int64 // jwt-token 中携带的信息
	}{
		{
			name: "参数绑定失败",
			mock: func(ctl *gomock.Controller) (service.UserAndService, encryption.Handle) {
				return nil, nil
			},
			reqBody:     `{"email":"asxxxxxxxxxx163.com","password":"123456","fingerprint":"for-test"}`,
			wantCode:    http.StatusBadRequest,
			wantBody:    "参数合法性验证失败",
			fingerprint: "",
		},
		{
			name: "系统异常",
			mock: func(ctl *gomock.Controller) (service.UserAndService, encryption.Handle) {
				jwt1 := jwtmocks.NewMockHandle(ctl)
				jwt1.EXPECT().Encryption(gomock.Any(), gomock.Any(), gomock.Any()).Return("", errors.New("系统异常"))
				return nil, jwt1
			},
			reqBody:     `{"email":"asxxxxxxxxxx@163.com","password":"123456","fingerprint":"for-test"}`,
			wantCode:    http.StatusInternalServerError,
			wantBody:    "系统异常",
			fingerprint: "",
		},
		{
			name: "登录成功",
			mock: func(ctl *gomock.Controller) (service.UserAndService, encryption.Handle) {
				jwt1 := jwtmocks.NewMockHandle(ctl)
				tokenStr, refreshToken := CreateToken()
				jwt1.EXPECT().Encryption(gomock.Any(), AccessSecret, time.Minute*30).Return(
					tokenStr, nil)
				jwt1.EXPECT().Encryption(gomock.Any(), RefreshSecret, time.Hour*24*7).Return(
					refreshToken, nil)
				jwt1.EXPECT().Decrypt(gomock.Any(), AccessSecret).Return(
					&jwt.TokenClaims{
						RegisteredClaims: jwtMoudle.RegisteredClaims{
							ExpiresAt: jwtMoudle.NewNumericDate(now.Add(time.Minute * 30)),
						},
						Fingerprint: "for-test",
					}, nil,
				)
				jwt1.EXPECT().Decrypt(gomock.Any(), RefreshSecret).Return(
					&jwt.TokenClaims{
						RegisteredClaims: jwtMoudle.RegisteredClaims{
							ExpiresAt: jwtMoudle.NewNumericDate(now.Add(time.Hour * 24 * 7)),
						},
						Fingerprint: "for-test",
					}, nil,
				)
				return nil, jwt1
			},
			reqBody:     `{"email":"asxxxxxxxxxx@163.com","password":"123456","fingerprint":"for-test"}`,
			wantCode:    http.StatusOK,
			wantBody:    "登陆成功",
			fingerprint: "for-test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			server := gin.New()
			h := NewUserHandler(tc.mock(ctrl))
			h.RegisterRoutes(server)

			req, err := http.NewRequest(http.MethodPost, "/users/login", bytes.NewBuffer([]byte(tc.reqBody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			//用于接收resp
			resp := httptest.NewRecorder()

			server.ServeHTTP(resp, req)

			assert.Equal(t, tc.wantCode, resp.Code)

			assert.Equal(t, tc.wantBody, resp.Body.String())
			//登录成功才需要判断
			if resp.Code == http.StatusOK {
				accessToken := resp.Header().Get("x-access-token")
				refreshToken := resp.Header().Get("x-refresh-token")
				acessT, err := h.Decrypt(accessToken, AccessSecret)
				if err != nil {
					panic(err)
				}
				accessTokenClaim := acessT.(*jwt.TokenClaims)
				assert.Equal(t, tc.fingerprint, accessTokenClaim.Fingerprint)
				//判断过期时间
				if now.Add(time.Minute*29).UnixMilli() > accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli() {
					panic("过期时间异常")
					return
				}

				refreshT, err := h.Decrypt(refreshToken, RefreshSecret)
				if err != nil {
					panic(err)
				}
				refreshTokenClaim := refreshT.(*jwt.TokenClaims)
				assert.Equal(t, tc.fingerprint, refreshTokenClaim.Fingerprint)
				//判断过期时间
				if now.Add(time.Hour*168).UnixMilli() < accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli() {
					panic("过期时间异常")
					return
				}
			}
		})
	}
}

func CreateToken() (string, string) {
	now := time.Now()
	claims := jwt.TokenClaims{
		RegisteredClaims: jwtMoudle.RegisteredClaims{
			ExpiresAt: jwtMoudle.NewNumericDate(now.Add(time.Minute * 30)),
		},
		Fingerprint: "for-test",
	}
	token := jwtMoudle.NewWithClaims(jwtMoudle.SigningMethodHS512, claims)
	tokenStr, _ := token.SignedString([]byte(AccessSecret))

	claims.RegisteredClaims.ExpiresAt = jwtMoudle.NewNumericDate(now.Add(time.Hour * 168))
	token = jwtMoudle.NewWithClaims(jwtMoudle.SigningMethodHS512, claims)
	//下面的密钥可以使用不同的密钥(一样的也行)
	refreshToken, _ := token.SignedString([]byte(RefreshSecret))
	return tokenStr, refreshToken
}

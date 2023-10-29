package web

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/service"
	svcmocks "github.com/ecodeclub/webook/internal/service/mocks"
)

// Handler测试的主要难点
// 1.构造HTTP请求
// 2.验证HTTP响应
func TestUserHandler_SignUp(t *testing.T) {
	testCases := []struct {
		name     string
		mock     func(ctrl *gomock.Controller) service.UserService
		body     string
		wantCode int
		wantBody string
	}{
		{
			name: "绑定信息错误！",
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				return userSvc
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
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				return userSvc
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
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				return userSvc
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
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(service.ErrUserDuplicate)
				return userSvc
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
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(errors.New("any error"))
				return userSvc
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
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Signup(gomock.Any(), &domain.User{
					Email:    "l0slakers@gmail.com",
					Password: "Abcd#1234",
				}).Return(nil)
				userSvc.EXPECT().SendVerifyEmail(gomock.Any(), gomock.Any()).Return(nil)
				return userSvc
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
	//gin.SetMode(gin.ReleaseMode)
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
		mock     func(ctrl *gomock.Controller) service.UserService
		body     string
		wantCode int
		wantBody string
	}{
		{
			name: "邮箱验证",
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().VerifyEmail(gomock.Any(), gomock.Any()).Return(nil)
				return userSvc
			},
			body:     "",
			wantCode: http.StatusOK,
			wantBody: "验证成功!",
		},
		{
			name: "验证失败!",
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().VerifyEmail(gomock.Any(), gomock.Any()).Return(service.ErrTokenInvalid)
				return userSvc
			},
			body:     "",
			wantCode: http.StatusOK,
			wantBody: "验证失败!",
		},
	}
	//gin.SetMode(gin.ReleaseMode)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := gin.Default()
			h := NewUserHandler(tc.mock(ctrl))
			h.RegisterRoutes(r)

			req, err := http.NewRequest(http.MethodPost, "/users/email/verify/token", bytes.NewBuffer([]byte(tc.body)))

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
		mock        func(ctl *gomock.Controller) service.UserService
		reqBody     string
		wantCode    int
		wantBody    string
		fingerprint string
		userId      int64 // jwt-token 中携带的信息
	}{
		{
			name: "参数绑定失败",
			mock: func(ctl *gomock.Controller) service.UserService {
				return nil
			},
			reqBody:     `{"email":"asxxxxxxxxxx163.com","password":"123456","fingerprint":"for-test"}`,
			wantCode:    http.StatusBadRequest,
			wantBody:    "参数合法性验证失败",
			fingerprint: "",
		},
		{
			name: "登录成功",
			mock: func(ctl *gomock.Controller) service.UserService {
				return nil
			},
			reqBody:     `{"email":"asxxxxxxxxxx@163.com","password":"123456","fingerprint":"for-test"}`,
			wantCode:    http.StatusOK,
			wantBody:    "登陆成功",
			fingerprint: "for-test",
		},
	}
	//gin.SetMode(gin.ReleaseMode)
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

				acessT, err := Decrypt(accessToken, AccessSecret)

				if err != nil {
					panic(err)
				}
				accessTokenClaim, ok := acessT.(*TokenClaims)
				if !ok {
					fmt.Println(acessT, err)
					panic("强制类型转换失败")
				}
				assert.Equal(t, tc.fingerprint, accessTokenClaim.Fingerprint)
				//判断过期时间
				if now.Add(time.Minute*29).UnixMilli() > accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli() {
					panic("过期时间异常")
				}
				refreshT, err := Decrypt(refreshToken, RefreshSecret)
				if err != nil {
					panic(err)
				}
				if !ok {
					fmt.Println(refreshT, err)
					panic("强制类型转换失败")
				}
				refreshTokenClaim := refreshT.(*TokenClaims)
				assert.Equal(t, tc.fingerprint, refreshTokenClaim.Fingerprint)
				//判断过期时间
				if now.Add(time.Hour*168).UnixMilli() < accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli() {
					panic("过期时间异常")
				}
			}
		})
	}
}

func CreateToken() (string, string) {
	now := time.Now()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute * 30)),
		},
		Fingerprint: "for-test",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, _ := token.SignedString([]byte(AccessSecret))

	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Hour * 168))
	token = jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	//下面的密钥可以使用不同的密钥(一样的也行)
	refreshToken, _ := token.SignedString([]byte(RefreshSecret))
	return tokenStr, refreshToken
}

func Decrypt(encryptString string, secret string) (interface{}, error) {
	claims := &TokenClaims{}
	token, err := jwt.ParseWithClaims(encryptString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		fmt.Println("解析失败:", err)
		return nil, err
	}
	//检查过期时间
	if claims.ExpiresAt.Time.Before(time.Now()) {
		//过期了

		return nil, err
	}
	//TODO 这里测试按需判断 claims.Uid
	if token == nil || !token.Valid {
		//解析成功  但是 token 以及 claims 不一定合法

		return nil, err
	}
	return claims, nil
}

func TestUserHandler_Edit(t *testing.T) {
	testCases := []struct {
		name     string
		mock     func(ctrl *gomock.Controller) service.UserService
		body     string
		wantCode int
		wantBody string
	}{
		{
			name: "更新成功",
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().EditUserProfile(gomock.Any(), gomock.Any()).Return(nil)
				return userSvc
			},
			body:     `{"id":1,"nickname":"frankiejun","birthday":"2020-01-01","aboutme":"I am a good boy"}`,
			wantCode: http.StatusOK,
			wantBody: "更新成功",
		},
		{
			name: "数据绑定有问题",
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				return userSvc
			},
			body:     `{"id":1,"nickname":"frankiejun","birthday":"2020-01-01","aboutme":"I am a good boy"`,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "昵称超长",
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				return userSvc
			},
			body: `
{"id":1,
"nickname":"frankiejun11111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111112222222222222222222222222222222222222222222222222222222222222",
"birthday":"2020-01-01",
"about_me":"I am a good boy"}
`,
			wantCode: http.StatusOK,
			wantBody: "昵称超过长度限制！",
		},
		{
			name: "个人介绍超长",
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				return userSvc
			},
			body: `
{"id":1,
"nickname":"frankiejun",
"birthday":"2020-01-01",
"aboutme":"I am a good boy5555555555556666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666666"}
`,
			wantCode: http.StatusOK,
			wantBody: "简介超过长度限制！",
		},
		{
			name: "生日日期非法",
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				return userSvc
			},
			body:     `{"id":1,"nickname":"frankiejun","birthday":"2020.01.01","aboutme":"I am a good boy"}`,
			wantCode: http.StatusOK,
			wantBody: "非法的生日日期，标准样式为：yyyy-mm-dd",
		},
		{
			name: "更新失败！",
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().EditUserProfile(gomock.Any(), domain.User{
					Id:       1,
					NickName: "frankiejun",
					Birthday: "2020-01-01",
					AboutMe:  "I am a good boy",
				}).Return(errors.New("更新失败"))
				return userSvc
			},
			body:     `{"id":1,"nickname":"frankiejun","birthday":"2020-01-01","aboutme":"I am a good boy"}`,
			wantCode: http.StatusOK,
			wantBody: "更新失败!",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := gin.Default()
			h := NewUserHandler(tc.mock(ctrl))

			h.RegisterRoutes(r)

			req, err := http.NewRequest(http.MethodPost, "/users/edit", bytes.NewBuffer([]byte(tc.body)))

			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)

			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, tc.wantBody, resp.Body.String())
		})
	}
}

func TestUserHandler_Profile(t *testing.T) {
	testCases := []struct {
		name     string
		ctx      gin.Context
		mock     func(ctrl *gomock.Controller) service.UserService
		body     string
		wantCode int
		wantBody string
	}{
		{
			name: "查看详细资料",
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), gomock.Any()).Return(domain.User{
					Email:    "abc@qq.com",
					NickName: "abc",
					Birthday: "2020-01-01",
					AboutMe:  "i am a good boy",
				}, nil)
				return userSvc
			},
			wantCode: http.StatusOK,
			wantBody: `{"Email":"abc@qq.com","NickName":"abc","Birthday":"2020-01-01","AboutMe":"i am a good boy"}`,
		},
		{
			name: "查不到资料",
			mock: func(ctrl *gomock.Controller) service.UserService {
				userSvc := svcmocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), gomock.Any()).Return(domain.User{}, errors.New("data not found"))
				return userSvc
			},
			wantCode: http.StatusOK,
			wantBody: "系统错误",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := gin.Default()
			h := NewUserHandler(tc.mock(ctrl))

			// 添加一个 路由
			r.POST("/users/profile", func(ctx *gin.Context) {
				ctx.Set("userid", int64(1)) // 模拟设置用户ID
				h.Profile(ctx)
			})

			req, err := http.NewRequest(http.MethodPost, "/users/profile", nil)

			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			r.ServeHTTP(resp, req)

			//body, err := json.Marshal(resp.Body)
			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, tc.wantBody, resp.Body.String())
		})
	}
}

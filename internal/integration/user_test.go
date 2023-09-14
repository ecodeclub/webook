// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build e2e

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/ecodeclub/webook/config"
	"github.com/ecodeclub/webook/internal/repository"
	"github.com/ecodeclub/webook/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/service"
	"github.com/ecodeclub/webook/internal/service/mail"
	"github.com/ecodeclub/webook/internal/service/mail/testmail"
	"github.com/ecodeclub/webook/internal/web"
	tokenGen "github.com/ecodeclub/webook/internal/web/token/generator"
	tokenVfy "github.com/ecodeclub/webook/internal/web/token/validator"
)

func TestUserHandler_e2e_SignUp(t *testing.T) {
	//	server := InitWebServer()
	server := InitTest()
	db := initDB()
	now := time.Now()

	testCases := []struct {
		name     string
		body     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		wantCode int
		wantBody string
	}{
		{
			name: "bind 失败！",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {

			},
			body: `
{
	"email": "l0slakers@gmail.com",
	"password": "Abcd#1234",
	"confirmPassword": "
`,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "两次密码不相同！",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {

			},
			body: `
{
	"email": "l0slakers@gmail.com",
	"password": "Abcd#1234",
	"confirmPassword": "Ac#123456"
}
`,
			wantCode: http.StatusBadRequest,
			wantBody: "两次密码不相同！",
		},
		{
			name: "密码格式不正确！",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {

			},
			body: `
{
	"email": "l0slakers@gmail.com",
	"password": "1234",
	"confirmPassword": "1234"
}
`,
			wantCode: http.StatusBadRequest,
			wantBody: "密码格式不正确,长度不能小于 6 位！",
		},
		{
			name: "邮箱冲突！",
			before: func(t *testing.T) {
				u := dao.User{
					Email:      "l0slakers@gmail.com",
					Password:   "123456",
					CreateTime: now.UnixMilli(),
					UpdateTime: now.UnixMilli(),
				}
				db.Create(&u)
			},
			after: func(t *testing.T) {
				var u dao.User
				d := db.Where("email = ?", "l0slakers@gmail.com").First(&u)
				d.Delete(&u)
			},
			body: `
{
	"email": "l0slakers@gmail.com",
	"password": "123456",
	"confirmPassword": "123456"
}
`,
			wantCode: http.StatusBadRequest,
			wantBody: "重复邮箱，请更换邮箱！",
		},
		{
			name: "注册成功！",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				var u dao.User
				d := db.Where("email = ?", "l0slakers@gmail.com").First(&u)
				assert.NotEmpty(t, u.Id)
				assert.NotEmpty(t, u.Email)
				assert.NotEmpty(t, u.Password)
				assert.NotEmpty(t, u.CreateTime)
				assert.NotEmpty(t, u.UpdateTime)
				d.Delete(&u)
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
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost, "/users/signup", bytes.NewBuffer([]byte(tc.body)))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp := httptest.NewRecorder()
			server.ServeHTTP(resp, req)

			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, tc.wantBody, resp.Body.String())
			tc.after(t)
		})
	}
}

func TestUserHandler_e2e_EmailVerify(t *testing.T) {
	const emailVerify = "/users/email/verification"
	server := InitTest()
	db := initDB()
	tg := initTokenGen()
	now := time.Now()

	tests := []struct {
		// 名字
		name string
		// 要提前准备数据
		before func(t *testing.T)
		// 验证并且删除数据
		after     func(t *testing.T)
		email     string
		paramsKey string
		token     string

		// 预期响应
		wantCode   int
		wantResult web.Result
	}{
		{
			name: "验证成功",
			before: func(t *testing.T) {
				ctx := context.Background()
				email := "foo@example.com"
				u := dao.User{
					Email:         email,
					Password:      "$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
					EmailVerified: false,
					CreateTime:    now.UnixMilli(),
					UpdateTime:    now.UnixMilli(),
				}
				err := db.WithContext(ctx).Create(&u).Error
				// 断言必然新增了数据
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx := context.Background()
				email := "foo@example.com"
				// 删除数据
				defer func() {
					err := db.WithContext(ctx).
						Where(&dao.User{Email: email}, "Email").
						Delete(&dao.User{}).Error
					assert.NoError(t, err)
				}()
				// 查询数据
				var u dao.User
				err := db.WithContext(ctx).Model(&dao.User{}).
					Where(&dao.User{Email: email}, "Email").
					Take(&u).Error
				assert.NoError(t, err)
				// 断言是否已认证
				assert.True(t, u.EmailVerified == true)
			},
			email:      "foo@example.com",
			paramsKey:  "code",
			wantCode:   http.StatusOK,
			wantResult: web.Result{Msg: "验证成功"},
		},
		{
			name: "参数错误",
			before: func(t *testing.T) {
				ctx := context.Background()
				email := "foo@example.com"
				u := dao.User{
					Email:         email,
					Password:      "$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
					EmailVerified: false,
					CreateTime:    now.UnixMilli(),
					UpdateTime:    now.UnixMilli(),
				}
				err := db.WithContext(ctx).Create(&u).Error
				// 断言必然新增了数据
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx := context.Background()
				email := "foo@example.com"
				// 查询数据
				var u dao.User
				err := db.WithContext(ctx).Model(&dao.User{}).
					Where(&dao.User{Email: email}, "Email").
					Take(&u).Error
				assert.NoError(t, err)
				// 断言是否已认证
				assert.True(t, u.EmailVerified == false)

				// 删除数据
				err = db.WithContext(ctx).
					Where(&dao.User{Email: email}, "Email").
					Delete(&dao.User{}).Error
				assert.NoError(t, err)
			},
			email:    "foo@example.com",
			wantCode: http.StatusBadRequest,
			wantResult: web.Result{
				Code: web.CodeParamsErr,
				Msg:  "参数错误",
			},
		},
		{
			name: "code认证失败",
			before: func(t *testing.T) {
				ctx := context.Background()
				email := "foo@example.com"
				u := dao.User{
					Email:         email,
					Password:      "$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
					EmailVerified: false,
					CreateTime:    now.UnixMilli(),
					UpdateTime:    now.UnixMilli(),
				}
				err := db.WithContext(ctx).Create(&u).Error
				// 断言必然新增了数据
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx := context.Background()
				email := "foo@example.com"
				// 删除数据
				defer func() {
					err := db.WithContext(ctx).
						Where(&dao.User{Email: email}, "Email").
						Delete(&dao.User{}).Error
					assert.NoError(t, err)
				}()
				// 查询数据
				var u dao.User
				err := db.WithContext(ctx).Model(&dao.User{}).
					Where(&dao.User{Email: email}, "Email").
					Take(&u).Error
				assert.NoError(t, err)
				// 断言是否已认证
				assert.True(t, u.EmailVerified == false)
			},
			email:     "foo@example.com",
			paramsKey: "code",
			token:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ3ZWJvb2stZW1haWwtdmVyaWZ5Iiwic3ViIjoiZm9vQGV4YW1wbGUuY29tIiwiZXhwIjoxNjk0NTM5NzQzLCJpYXQiOjE2OTQ1MzkxNDN9.N5hnHn-zfVJUjRUVf9u4w0iDEnfhYE-Z9cBVvP5oP10",
			wantCode:  http.StatusBadRequest,
			wantResult: web.Result{
				Code: web.CodeEmailVerifyFailed,
				Msg:  "验证失败",
			},
		},
		{
			name: "邮箱已验证",
			before: func(t *testing.T) {
				ctx := context.Background()
				email := "foo@example.com"
				u := dao.User{
					Email:         email,
					Password:      "$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
					EmailVerified: true,
					CreateTime:    now.UnixMilli(),
					UpdateTime:    now.UnixMilli(),
				}
				err := db.WithContext(ctx).Create(&u).Error
				// 断言必然新增了数据
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx := context.Background()
				email := "foo@example.com"
				// 删除数据
				defer func() {
					err := db.WithContext(ctx).
						Where(&dao.User{Email: email}, "Email").
						Delete(&dao.User{}).Error
					assert.NoError(t, err)
				}()
				// 查询数据
				var u dao.User
				err := db.WithContext(ctx).Model(&dao.User{}).
					Where(&dao.User{Email: email}, "Email").
					Take(&u).Error
				assert.NoError(t, err)
				// 断言是否已认证
				assert.True(t, u.EmailVerified == true)
			},
			email:     "foo@example.com",
			paramsKey: "code",
			wantCode:  http.StatusBadRequest,
			wantResult: web.Result{
				Code: web.CodeEmailVerified,
				Msg:  "邮箱已验证",
			},
		},
		{
			name: "验证失败没有这个用户",
			before: func(t *testing.T) {
				ctx := context.Background()
				email := "foo@example.com"
				u := dao.User{
					Email:         email,
					Password:      "$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
					EmailVerified: false,
					CreateTime:    now.UnixMilli(),
					UpdateTime:    now.UnixMilli(),
				}
				err := db.WithContext(ctx).Create(&u).Error
				// 断言必然新增了数据
				assert.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx := context.Background()
				email := "foo@example.com"
				// 删除数据
				defer func() {
					err := db.WithContext(ctx).
						Where(&dao.User{Email: email}, "Email").
						Delete(&dao.User{}).Error
					assert.NoError(t, err)
				}()
				// 查询数据
				var u dao.User
				err := db.WithContext(ctx).Model(&dao.User{}).
					Where(&dao.User{Email: email}, "Email").
					Take(&u).Error
				assert.NoError(t, err)
				// 断言是否已认证
				assert.True(t, u.EmailVerified == false)
			},
			email:     "bar@example.com",
			paramsKey: "code",
			wantCode:  http.StatusBadRequest,
			wantResult: web.Result{
				Code: web.CodeEmailVerifyFailed,
				Msg:  "验证失败",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer tt.after(t)
			tt.before(t)
			// 准备token，如果没有填token则动态生成
			if tt.token == "" {
				var err error
				tt.token, err = tg.GenerateToken(tt.email,
					time.Duration(10)*time.Minute)
				assert.NoError(t, err)
			}

			req, err := http.NewRequest(http.MethodGet,
				emailVerify+"?"+tt.paramsKey+"="+tt.token, nil)
			assert.NoError(t, err)

			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)

			code := recorder.Code
			// 反序列化为结果
			assert.Equal(t, tt.wantCode, code)
			var result web.Result
			err = json.Unmarshal(recorder.Body.Bytes(), &result)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func TestUserHandler_e2e_Login(t *testing.T) {
	//	server := InitWebServer()
	server := gin.Default()
	// db := initDB()
	var db *gorm.DB
	da := dao.NewUserInfoDAO(db)
	repo := repository.NewUserInfoRepository(da)
	svc := service.NewUserService(repo)

	userHandle := web.NewUserHandler(svc)
	userHandle.RegisterRoutes(server)
	now := time.Now()

	testCases := []struct {
		name        string
		before      func(t *testing.T)
		reqBody     string
		wantCode    int
		wantBody    string
		fingerprint string
		after       func(t *testing.T)
		// userId   int64 // jwt-token 中携带的信息
	}{
		{
			name:        "参数绑定失败",
			reqBody:     `{"email":"asxxxxxxxxxx163.com","password":"123456","fingerprint":""}`,
			wantCode:    http.StatusBadRequest,
			wantBody:    "参数合法性验证失败",
			fingerprint: "",
		},
		{
			name:        "登陆成功",
			reqBody:     `{"email":"asxxxxxxxxxx@163.com","password":"123456","fingerprint":"long-short-token"}`,
			wantCode:    http.StatusOK,
			wantBody:    "登陆成功",
			fingerprint: "long-short-token",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 构造请求
			req, err := http.NewRequest(http.MethodPost, "/users/login", bytes.NewBuffer([]byte(tc.reqBody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// 用于接收resp
			resp := httptest.NewRecorder()

			server.ServeHTTP(resp, req)

			// 判断结果
			assert.Equal(t, tc.wantCode, resp.Code)

			assert.Equal(t, tc.wantBody, resp.Body.String())
			// 登录成功才需要判断
			// 登录成功才需要判断
			if resp.Code == http.StatusOK {
				accessToken := resp.Header().Get("x-access-token")
				refreshToken := resp.Header().Get("x-refresh-token")

				acessT, err := Decrypt(accessToken, web.AccessSecret)

				if err != nil {
					panic(err)
				}
				accessTokenClaim, ok := acessT.(*web.TokenClaims)
				if !ok {
					fmt.Println(acessT, err)
					panic("强制类型转换失败")
				}
				assert.Equal(t, tc.fingerprint, accessTokenClaim.Fingerprint)
				// 判断过期时间
				if now.Add(time.Minute*29).UnixMilli() > accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli() {
					panic("过期时间异常")
				}
				refreshT, err := Decrypt(refreshToken, web.RefreshSecret)
				if err != nil {
					panic(err)
				}
				if !ok {
					fmt.Println(refreshT, err)
					panic("强制类型转换失败")
				}
				refreshTokenClaim := refreshT.(*web.TokenClaims)
				assert.Equal(t, tc.fingerprint, refreshTokenClaim.Fingerprint)
				// 判断过期时间
				if now.Add(time.Hour*168).UnixMilli() < accessTokenClaim.RegisteredClaims.ExpiresAt.Time.UnixMilli() {
					panic("过期时间异常")
				}
			}
		})
	}
}

func InitTest() *gin.Engine {
	r := initWebServer()
	db := initDB()
	u := initUser(db)
	u.RegisterRoutes(r)
	return r
}

func initDB() *gorm.DB {
	dsn := "root:root@tcp(localhost:13316)/webook"
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = sqlDB.PingContext(ctx)
		cancel()
		if err == nil {
			break
		}
		log.Println("初始化集成测试的 DB", err)
		time.Sleep(time.Second)
	}
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		panic(err)
	}
	err = dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	return db
}

func initWebServer() *gin.Engine {
	r := gin.Default()
	return r
}

func initGoMailDial() gomail.SendCloser {
	cfg := config.Config.EmailConf
	dial, err := gomail.NewDialer(
		cfg.Host, cfg.Port, cfg.Username, cfg.Password,
	).Dial()
	if err != nil {
		panic(err)
	}
	return dial
}

func initTestMail() mail.Service {
	return testmail.NewService()
}

func initTokenGen() tokenGen.TokenGenerator {
	conf := config.Config
	return tokenGen.NewJWTTokenGen(conf.EmailVfyConf.Issuer, conf.EmailVfyConf.Key)
}

func initTokenVfy() tokenVfy.Verifier {
	conf := config.Config
	return tokenVfy.NewJWTTokenVerifier(conf.EmailVfyConf.Key)
}

func initLogger() *zap.Logger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return logger
}

func initUser(db *gorm.DB) *web.UserHandler {
	conf := config.Config
	lg := initLogger()

	userDAO := dao.NewUserInfoDAO(db)
	userRepo := repository.NewUserInfoRepository(userDAO)
	userSvc := service.NewUserService(userRepo, lg)

	// 邮箱服务
	// emailCli := initGoMailDial()
	// mailSvc := goemail.NewService(conf.EmailConf.Username, emailCli)
	mailSvc := initTestMail()

	// token
	eTokenGen := initTokenGen()
	eTokenVfy := initTokenVfy()

	emailSvc := service.NewEmailService(mailSvc)
	u := web.NewUserHandler(userSvc, emailSvc, eTokenGen,
		eTokenVfy, conf.EmailVfyConf.AbsoluteURL, lg)
	return u
}

func Decrypt(encryptString string, secret string) (interface{}, error) {
	claims := &web.TokenClaims{}
	token, err := jwt.ParseWithClaims(encryptString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		fmt.Println("解析失败:", err)
		return nil, err
	}
	// 检查过期时间
	if claims.ExpiresAt.Time.Before(time.Now()) {
		// 过期了

		return nil, err
	}
	// TODO 这里测试按需判断 claims.Uid
	if token == nil || !token.Valid {
		// 解析成功  但是 token 以及 claims 不一定合法

		return nil, err
	}
	return claims, nil
}

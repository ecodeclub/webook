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

package integration

import (
	"bytes"
	"context"
	"database/sql"

	"github.com/ecodeclub/webook/internal/service/email"

	//"github.com/golang-jwt/jwt/v5"
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
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/ecodeclub/webook/internal/repository"
	"github.com/ecodeclub/webook/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/service"
	"github.com/ecodeclub/webook/internal/web"
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
	r := initWebServer()
	db := initDB()
	u := initUser(db)
	u.RegisterRoutes(r)
	testCases := []struct {
		name     string
		body     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		token    string
		email    string
		wantCode int
		wantBody string
	}{
		{
			name: "验证成功!",
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {

			},
			body:     "",
			token:    genToken("abc@163.com", 1),
			email:    "abc@163.com",
			wantCode: http.StatusOK,
			wantBody: "验证成功!",
		},
		{
			name: "验证失败!",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {

			},
			body:     "",
			email:    "abc@163.com",
			wantCode: http.StatusOK,
			wantBody: "验证失败!",
			token:    "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJFbWFpbCI6Imp1bm55ZmVuZ0AxNjMuY29tIiwiZXhwIjoxNjk0NTIxODQzfQ.gwGcIDcaKuFG6DyyLnWfEn5poIZ3BMUk2lNQsDhyA3DMBSTo9HkEJ1eyIUQ0XDqp29XVme5dOOMuY2LgRfI60Q",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost, "/users/email/verify/"+tc.token, bytes.NewBuffer([]byte(tc.body)))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)

			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, tc.wantBody, resp.Body.String())
			tc.after(t)
		})
	}
}

func TestUserHandler_e2e_Login(t *testing.T) {
	//	server := InitWebServer()
	server := gin.Default()
	//db := initDB()
	var db *gorm.DB
	da := dao.NewUserInfoDAO(db)
	repo := repository.NewUserInfoRepository(da)
	evc := &email.NoOpService{}
	svc := service.NewUserService(repo, evc)

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
		//userId   int64 // jwt-token 中携带的信息
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
			//构造请求
			req, err := http.NewRequest(http.MethodPost, "/users/login", bytes.NewBuffer([]byte(tc.reqBody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			//用于接收resp
			resp := httptest.NewRecorder()

			server.ServeHTTP(resp, req)

			// 判断结果
			assert.Equal(t, tc.wantCode, resp.Code)

			assert.Equal(t, tc.wantBody, resp.Body.String())
			//登录成功才需要判断
			//登录成功才需要判断
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
				//判断过期时间
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
				//判断过期时间
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

func initUser(db *gorm.DB) *web.UserHandler {
	da := dao.NewUserInfoDAO(db)
	repo := repository.NewUserInfoRepository(da)
	evc := &email.NoOpService{}
	svc := service.NewUserService(repo, evc)
	u := web.NewUserHandler(svc)
	return u
}

func genToken(emailAddr string, timeout int) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, service.EmailClaims{
		Email: emailAddr,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * time.Duration(timeout))),
		},
	})
	tokenStr, _ := token.SignedString([]byte(service.EmailJWTKey))
	return tokenStr
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

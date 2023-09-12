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
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/repository"
	"github.com/ecodeclub/webook/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/service"
	"github.com/ecodeclub/webook/internal/web"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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
	svc := service.NewUserService(repo)
	u := web.NewUserHandler(svc)
	return u
}

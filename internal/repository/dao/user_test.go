package dao

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestGormUserDAO_Insert(t *testing.T) {
	testCases := []struct {
		name    string
		ctx     context.Context
		user    User
		mock    func(t *testing.T) *sql.DB
		wantErr error
	}{
		{
			name: "邮箱冲突！",
			ctx:  context.Background(),
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectExec("INSERT INTO `users` .*").
					WillReturnError(&mysql.MySQLError{
						Number: 1062,
					})
				return mockDB
			},
			user:    User{},
			wantErr: ErrUserDuplicate,
		},
		{
			name: "数据库错误！",
			ctx:  context.Background(),
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectExec("INSERT INTO `users` .*").
					WillReturnError(errors.New("数据库错误！"))
				return mockDB
			},
			user:    User{},
			wantErr: errors.New("数据库错误！"),
		},
		{
			name: "插入成功！",
			ctx:  context.Background(),
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				require.NoError(t, err)
				res := sqlmock.NewResult(3, 1)
				// 增删改
				mock.ExpectExec("INSERT INTO `users` .*").
					WillReturnResult(res)
				// 查
				//mock.ExpectQuery()
				return mockDB
			},
			user: User{
				Email: "l0slakers@gmail.com",
			},
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			db, err := gorm.Open(gormMysql.New(gormMysql.Config{
				Conn: tc.mock(t),
				// 如果为 false ，则GORM在初始化时，会先调用 show version
				SkipInitializeWithVersion: true,
			}), &gorm.Config{
				// 如果为 true ，则不允许 Ping数据库
				DisableAutomaticPing: true,
				// 如果为 false ，则即使是单一语句，也会开启事务
				SkipDefaultTransaction: true,
			})
			assert.NoError(t, err)
			d := NewUserInfoDAO(db)
			err = d.Insert(tc.ctx, tc.user)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestGormUserDAO_FindByEmail(t *testing.T) {
	testCases := []struct {
		name      string
		ctx       context.Context
		email     string
		mock      func(t *testing.T) *sql.DB
		wantemail string
		wantErr   error
	}{
		{
			name:  "查找成功",
			ctx:   context.Background(),
			email: "frankiejun@qq.com",
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				require.NoError(t, err)
				rows := sqlmock.NewRows([]string{"id", "email", "EmailVerify", "password"})
				rows.AddRow(1, "abc@qq.com", nil, "123")
				mock.ExpectQuery("^SELECT \\* FROM `users` WHERE email = \\?").WillReturnRows(rows)
				return mockDB
			},
			wantErr:   nil,
			wantemail: "abc@qq.com",
		},

		{
			name:  "查找不存在",
			ctx:   context.Background(),
			email: "frankiejun@qq.com",
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				require.NoError(t, err)
				rows := sqlmock.NewRows([]string{"id", "email", "EmailVerify", "password"})
				mock.ExpectQuery("^SELECT \\* FROM `users` WHERE email = \\?").WillReturnRows(rows)
				return mockDB
			},
			wantErr:   errors.New("record not found"),
			wantemail: "",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			db, err := gorm.Open(gormMysql.New(gormMysql.Config{
				Conn: tt.mock(t),
				// 如果为 false ，则GORM在初始化时，会先调用 show version
				SkipInitializeWithVersion: true,
			}), &gorm.Config{
				// 如果为 true ，则不允许 Ping数据库
				DisableAutomaticPing: true,
				// 如果为 false ，则即使是单一语句，也会开启事务
				SkipDefaultTransaction: true,
			})
			require.NoError(t, err)
			dao := NewUserInfoDAO(db)
			user, err := dao.FindByEmail(tt.ctx, tt.email)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantemail, user.Email)
		})
	}
}

func TestGormUserDAO_UpdateEmailVerified(t *testing.T) {
	testCases := []struct {
		name    string
		ctx     context.Context
		user    User
		mock    func(t *testing.T) *sql.DB
		wantErr error
	}{
		{
			name: "更新成功",
			ctx:  context.Background(),
			user: User{
				Id:            1,
				Email:         "abc@qq.com",
				EmailVerified: true,
				Password:      "123456",
			},
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectExec("UPDATE `users` .*").WillReturnResult(sqlmock.NewResult(1, 1))
				return mockDB
			},
			wantErr: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			db, err := gorm.Open(gormMysql.New(gormMysql.Config{
				Conn: tt.mock(t),
				// 如果为 false ，则GORM在初始化时，会先调用 show version
				SkipInitializeWithVersion: true,
			}), &gorm.Config{
				// 如果为 true ，则不允许 Ping数据库
				DisableAutomaticPing: true,
				// 如果为 false ，则即使是单一语句，也会开启事务
				SkipDefaultTransaction: true,
			})
			assert.NoError(t, err)
			dao := NewUserInfoDAO(db)
			err = dao.UpdateEmailVerified(tt.ctx, tt.user.Email)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

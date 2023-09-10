package dao

import (
	"context"
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
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
			d := NewUserInfoDAO(db)
			err = d.Insert(tc.ctx, tc.user)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

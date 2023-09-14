package dao

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
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
				// mock.ExpectQuery()
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

func TestGormUserDAO_UpdateNonZeroFields(t *testing.T) {
	tests := []struct {
		name    string
		sqlmock func(t *testing.T) *sql.DB

		// 输入
		ctx   context.Context
		email string

		wantErr error
	}{
		{
			name: "更新成功",
			sqlmock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				assert.NoError(t, err)
				mockRes := sqlmock.NewResult(1, 1)
				mock.ExpectExec("UPDATE `users` SET .*").
					WillReturnResult(mockRes)
				return db
			},
			ctx:     context.Background(),
			email:   "foo@example.com",
			wantErr: nil,
		},
		{
			name: "没有更新",
			sqlmock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				assert.NoError(t, err)
				mockRes := sqlmock.NewResult(1, 0)
				mock.ExpectExec("UPDATE `users` SET .*").
					WithArgs().
					WillReturnResult(mockRes)
				return db
			},
			ctx:     context.Background(),
			email:   "foo@example.com",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlDB := tt.sqlmock(t)
			db, err := gorm.Open(gormMysql.New(gormMysql.Config{
				Conn:                      sqlDB,
				SkipInitializeWithVersion: true,
			}), &gorm.Config{
				DisableAutomaticPing:   true,
				SkipDefaultTransaction: true,
			})
			// 初始化 DB 不能出错，所以这里要断言必须为 nil
			assert.NoError(t, err)
			dao := NewUserInfoDAO(db)
			err = dao.UpdateEmailVerifiedByEmail(tt.ctx, tt.email)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestGormUserDAO_FindByEmail(t *testing.T) {
	tests := []struct {
		name    string
		sqlmock func(t *testing.T) *sql.DB

		// 输入
		ctx   context.Context
		email string

		wantUser User
		wantErr  error
	}{
		{
			name: "查询成功",
			sqlmock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				assert.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE `users`.`email` = ? LIMIT 1")).
					WithArgs("foo@example.com").
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "email", "password",
							"email_verified", "create_time", "update_time"}).
							AddRow(1, "foo@example.com",
								"$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
								false, 1688440000, 1688440000),
					)
				return db
			},
			ctx:   context.Background(),
			email: "foo@example.com",
			wantUser: User{
				Id:            1,
				Email:         "foo@example.com",
				Password:      "$2a$10$s51GBcU20dkNUVTpUAQqpe6febjXkRYvhEwa5OkN5rU6rw2KTbNUi",
				EmailVerified: false,
				CreateTime:    1688440000,
				UpdateTime:    1688440000,
			},
		},
		{
			name: "查询不到用户",
			sqlmock: func(t *testing.T) *sql.DB {
				db, mock, err := sqlmock.New()
				assert.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE `users`.`email` = ? LIMIT 1")).
					WithArgs("foo@example.com").
					WillReturnError(gorm.ErrRecordNotFound)
				return db
			},
			ctx:     context.Background(),
			email:   "foo@example.com",
			wantErr: ErrDataNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlDB := tt.sqlmock(t)
			db, err := gorm.Open(gormMysql.New(gormMysql.Config{
				Conn:                      sqlDB,
				SkipInitializeWithVersion: true,
			}), &gorm.Config{
				DisableAutomaticPing:   true,
				SkipDefaultTransaction: true,
			})
			// 初始化 DB 不能出错，所以这里要断言必须为 nil
			assert.NoError(t, err)
			dao := NewUserInfoDAO(db)
			ue, err := dao.FindByEmail(tt.ctx, tt.email)
			assert.Equal(t, tt.wantUser, ue)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

package main

import (
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"signup_issue/webook/config"
	"signup_issue/webook/internal/repository"
	"signup_issue/webook/internal/repository/dao"
	"signup_issue/webook/internal/service"
	"signup_issue/webook/internal/web"
)

func main() {
	db := initDB()
	r := initWebServer()
	u := initUser(db)
	u.RegisterRoutes(r)
	err := r.Run(":8081")
	if err != nil {
		panic("端口启动失败")
	}
}

func initDB() *gorm.DB {
	db, err := gorm.Open(mysql.Open(config.Config.DB.DSN))
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

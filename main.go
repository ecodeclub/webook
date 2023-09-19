package main

import (
	"github.com/ecodeclub/webook/internal/service/email/gomail"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/ecodeclub/webook/config"
	"github.com/ecodeclub/webook/internal/ioc"
	"github.com/ecodeclub/webook/internal/repository"
	"github.com/ecodeclub/webook/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/service"
	"github.com/ecodeclub/webook/internal/web"
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
	evc := gomail.NewEmailService(ioc.InitEmailCfg())
	svc := service.NewUserService(repo, evc)
	u := web.NewUserHandler(svc)
	return u
}

package main

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/ecodeclub/webook/config"
	"github.com/ecodeclub/webook/internal/repository"
	"github.com/ecodeclub/webook/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/service"
	"github.com/ecodeclub/webook/internal/service/mail/goemail"
	"github.com/ecodeclub/webook/internal/web"
	tokenGen "github.com/ecodeclub/webook/internal/web/token/generator"
	tokenVfy "github.com/ecodeclub/webook/internal/web/token/validator"
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

func initLogger() *zap.Logger {
	lg, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return lg
}

func initUser(db *gorm.DB) *web.UserHandler {
	conf := config.Config
	lg := initLogger()

	userDAO := dao.NewUserInfoDAO(db)
	userRepo := repository.NewUserInfoRepository(userDAO)
	userSvc := service.NewUserService(userRepo, lg)

	// 邮箱服务
	emailCli := initGoMailDial()
	mailSvc := goemail.NewService(conf.EmailConf.Username, emailCli)

	// token
	eTokenGen := tokenGen.NewJWTTokenGen(conf.EmailVfyConf.Issuer, conf.EmailVfyConf.Key)
	eTokenVfy := tokenVfy.NewJWTTokenVerifier(conf.EmailVfyConf.Key)

	emailSvc := service.NewEmailService(mailSvc)
	u := web.NewUserHandler(userSvc, emailSvc, eTokenGen,
		eTokenVfy, conf.EmailVfyConf.AbsoluteURL, lg)
	return u
}

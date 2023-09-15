package ioc

import (
	"crypto/tls"
	"github.com/go-gomail/gomail"
	"os"
	"strconv"
)

func InitEmailCfg() *gomail.Dialer {
	//服务邮箱地址
	host := os.Getenv("EmailHost")
	//服务邮箱端口
	portstr := os.Getenv("EmailPort")
	//发送者或邮箱名称
	username := os.Getenv("EmailUsername")
	//邮箱的授权密码
	password := os.Getenv("EmailPassword")

	port, err := strconv.Atoi(portstr)
	if err != nil {
		panic("读取邮箱端口失败!")
	}

	dialer := gomail.NewDialer(host, port, username, password)
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	return dialer

}

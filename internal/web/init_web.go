package web

import "github.com/gin-gonic/gin"

func (u *UserHandler) RegisterRoutes(server *gin.Engine) {
	server.POST("/users/signup", u.SignUp)
	server.GET("/users/email/verification", u.EmailVerify)
}

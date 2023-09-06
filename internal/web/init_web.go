package web

import "github.com/gin-gonic/gin"

func (u *UserHandler) RegisterRoutes(server *gin.Engine) {
	server.POST("/users/signup", u.SignUp)
}

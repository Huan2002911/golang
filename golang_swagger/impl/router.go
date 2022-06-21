package impl

import (
	"github.com/gin-gonic/gin"
	"golang_swagger/impl/user"
)

//
func InitRouter() *gin.Engine {
	router := gin.Default()
	AddRouter(router)
	return router
}

// AddRouter 添加路由
func AddRouter(router *gin.Engine) {
	r := router.Group("/swagger")

	//user
	u := r.Group("/user")
	{
		u.POST("/login", user.Login)
	}
	//shop
	//shop := r.Group("/shop")
}

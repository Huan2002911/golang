package main

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang_swagger/docs"
	"golang_swagger/impl"
	"net/http"
)

// @title Swagger_Test
// @version 1.0
// @description
func main() {
	router := impl.InitRouter()

	// swagger
	router.GET("/swagger/*any", func(context *gin.Context) {
		docs.SwaggerInfo.Host = context.Request.Host
		ginSwagger.CustomWrapHandler(&ginSwagger.Config{URL: "/swagger/doc.json"}, swaggerFiles.Handler)(context)
	})
	//router.StaticFS("/html/swagger", swagger_ui.NewFileSystem("/swagger/doc.json"))

	s := &http.Server{
		Addr:    ":8082",
		Handler: router,
	}

	s.ListenAndServe()
}

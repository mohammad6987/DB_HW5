package routes

import (
	"github.com/gin-gonic/gin"
	"DB_HW5/controllers"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.POST("/signup", controllers.SignUp)
	r.POST("/login", controllers.Login)
	r.POST("/papers", controllers.PostPaper)
	r.GET("/papers", controllers.SearchPapers)
	r.GET("/papers/:id", controllers.GetPaperDetails)
	return r
}

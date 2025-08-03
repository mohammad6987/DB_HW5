package routes

import (
    "github.com/gin-gonic/gin"
    "DB_HW5/controllers"
)

func SetupRoutes(r *gin.Engine) {
    r.POST("/signup", controllers.SignUp)
    r.POST("/login", controllers.Login)

    paperGroup := r.Group("/papers")
    {
        paperGroup.POST("", controllers.UploadPaper)
        paperGroup.GET("", controllers.SearchPapers)
        paperGroup.GET("/:id", controllers.GetPaperDetails)
    }
}

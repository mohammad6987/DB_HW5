package routes

import (
	"DB_HW5/controllers"
	"log"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	store, err := redis.NewStore(10, "tcp", "localhost:6379", "default", "123456", []byte("secret-key"))
	if err != nil {
		log.Fatalf("failed to create redis store: %v", err)
	}
	r.Use(sessions.Sessions("mysession", store))
	r.POST("/signup", controllers.SignUp)
	r.POST("/login", controllers.Login)

	auth := r.Group("")
	auth.Use(controllers.AuthRequired())
	{
		r.POST("/papers", controllers.PostPaper)
		r.GET("/papers", controllers.SearchPapers)
		r.GET("/papers/:id", controllers.GetPaperDetails)
	}
	return r
}

package main

import (
	"DB_HW5/config"
	"log"

	"DB_HW5/routes"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
)

func main() {
	config.InitDB()

	r := gin.Default()

	store, err := redis.NewStore(10, "tcp", "localhost:6379", "default","123456",[]byte("secret-key"))
	if err != nil {
		log.Fatal("Session store error:", err)
	}
	r.Use(sessions.Sessions("mysession", store))

	routes.SetupRoutes(r)

	r.Run(":8080") // App listens on http://localhost:8080
}

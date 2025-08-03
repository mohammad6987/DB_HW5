package main

import (
	"DB_HW5/config"
	"log"

	"DB_HW5/routes"
	"DB_HW5/scheduler"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

func main() {
	config.InitDB()

	scheduler.StartViewSyncScheduler()

	r := gin.Default()

	store, err := redis.NewStore(10, "tcp", "localhost:6379", "default", "123456", []byte("This is a REAL-SECRET-KEY"))
	if err != nil {
		log.Fatal("Session store error:", err)
	}
	r.Use(sessions.Sessions("mysession", store))

	routes.SetupRoutes(r)

	r.Run(":8080") // App listens on http://localhost:8080
}

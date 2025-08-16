package main

import (
	"log"
	"os"

	"DB_HW5/config"
	"DB_HW5/routes"
	"DB_HW5/scheduler"
	"DB_HW5/utils"
)

func main() {
	config.Init()
	utils.EnsureIndexes()
	scheduler.StartViewsSync()

	r := routes.SetupRouter()
	addr := getEnv("HTTP_ADDR", ":8080")
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}

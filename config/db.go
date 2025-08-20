package config

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AppConfig struct {
	MongoURI      string
	MongoDB       string
	RedisAddr     string
	RedisPassword string
}

var (
	MongoClient *mongo.Client
	Redis       *redis.Client
	Cfg         AppConfig
)

func Init() {
	Cfg = AppConfig{
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:       getEnv("MONGO_DB", "research_db"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", "123456"),
	}


	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(Cfg.MongoURI))
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("mongo ping: %v", err)
	}
	MongoClient = client


	Redis = redis.NewClient(&redis.Options{
		Addr:     Cfg.RedisAddr,
		Password: Cfg.RedisPassword,
		DB:       0,
	})
	if err := Redis.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}

}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

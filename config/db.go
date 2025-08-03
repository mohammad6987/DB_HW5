package config

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
	RedisClient *redis.Client
	Ctx         = context.Background()
)

func InitDB() {

	mongoURI := "mongodb://localhost:27017"
	clientOpts := options.Client().ApplyURI(mongoURI)
	client, err := mongo.NewClient(clientOpts)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(Ctx)
	if err != nil {
		log.Fatal(err)
	}
	MongoClient = client
	MongoDB = client.Database("research_manager")

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "123456",
		DB:       0,
	})

	_, err = RedisClient.Ping(Ctx).Result()
	if err != nil {
		log.Fatal("Redis ping error:", err)
	}

	fmt.Println("MongoDB and Redis connected")
}

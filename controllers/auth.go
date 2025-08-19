package controllers

import (
	"context"
	"net/http"
	"time"

	"DB_HW5/config"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"DB_HW5/utils"
)

type SignUpBody struct {
	Username   string `json:"username"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Department string `json:"department"`
}

func SignUp(c *gin.Context) {
	var b SignUpBody
	if err := c.BindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	// validation
	if !utils.ValidUsername(b.Username) ||
		!utils.ValidNonEmptyMax(b.Name, 100) ||
		!utils.ValidEmail(b.Email) ||
		len(b.Password) < 8 ||
		!utils.ValidNonEmptyMax(b.Department, 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// check redis cache
	exists, err := config.Redis.HExists(ctx, utils.RedisHashUsernames, b.Username).Result()
	if err != nil && err != redis.Nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "redis error"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "نام کاربری گرفته شده است"})
		return
	}

	hashed, _ := utils.HashPassword(b.Password)
	u := bson.M{
		"username":   b.Username,
		"name":       b.Name,
		"email":      b.Email,
		"password":   hashed,
		"department": b.Department,
	}

	// raw insertOne
	command := bson.D{{"insert", "users"}, {"documents", []interface{}{u}}}
	var result bson.M
	if err := config.MongoClient.Database("your_db").RunCommand(ctx, command).Decode(&result); err != nil {
		// duplicate key error code 11000
		if we, ok := err.(mongo.WriteException); ok {
			for _, e := range we.WriteErrors {
				if e.Code == 11000 {
					c.JSON(http.StatusConflict, gin.H{"error": "نام کاربری گرفته شده است"})
					return
				}
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// insertedId is inside result
	insertedArr, _ := result["n"].(int32)
	if insertedArr == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}

	// you can fetch the generated _id manually
	var insertedDoc bson.M
	_ = config.MongoClient.Database("your_db").Collection("users").
		FindOne(ctx, bson.M{"username": b.Username}).Decode(&insertedDoc)
	id := insertedDoc["_id"].(primitive.ObjectID).Hex()

	// update redis
	_ = config.Redis.HSet(ctx, utils.RedisHashUsernames, b.Username, 1).Err()

	c.JSON(http.StatusCreated, gin.H{"message": "User registered", "user_id": id})
}

type LoginBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(c *gin.Context) {
	var b LoginBody
	if err := c.BindJSON(&b); err != nil || b.Username == "" || b.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// raw findOne
	command := bson.D{
		{"find", "users"},
		{"filter", bson.M{"username": b.Username}},
		{"limit", 1},
	}
	var result bson.M
	if err := config.MongoClient.Database("your_db").RunCommand(ctx, command).Decode(&result); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// extract document
	docs, _ := result["cursor"].(bson.M)["firstBatch"].(primitive.A)
	if len(docs) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	u := docs[0].(bson.M)

	if !utils.CheckPassword(u["password"].(string), b.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", u["_id"].(primitive.ObjectID).Hex())
	session.Set("username", u["username"].(string))
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "user_id": u["_id"].(primitive.ObjectID).Hex()})
}



func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user id from header
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing X-User-ID header"})
			c.Abort()
			return
		}

		// Get session
		session := sessions.Default(c)
		storedUserID := session.Get("user_id") // <── fixed

		if storedUserID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized (no active session)"})
			c.Abort()
			return
		}

		// Compare header ID with session ID
		if storedUserID.(string) != userID {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized (invalid user)"})
			c.Abort()
			return
		}

		// Auth passed
		c.Set("user_id", userID) // make user_id available downstream
		c.Next()
	}
}




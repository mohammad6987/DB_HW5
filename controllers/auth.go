package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/gin-contrib/sessions"

	"DB_HW5/config"
	"DB_HW5/models"
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
		c.JSON(http.StatusBadRequest, gin.H{"error":"invalid body"}); return
	}
	// validation
	if !utils.ValidUsername(b.Username) ||
		!utils.ValidNonEmptyMax(b.Name, 100) ||
		!utils.ValidEmail(b.Email) ||
		len(b.Password) < 8 ||
		!utils.ValidNonEmptyMax(b.Department, 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error":"invalid fields"}); return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	
	exists, err := config.Redis.HExists(ctx, utils.RedisHashUsernames, b.Username).Result()
	if err != nil && err != redis.Nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"redis error"}); return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error":"نام کاربری گرفته شده است"}); return
	}

	// Insert user
	hashed, _ := utils.HashPassword(b.Password)
	u := models.User{
		Username: b.Username, Name: b.Name, Email: b.Email,
		Password: hashed, Department: b.Department,
	}
	res, err := utils.Users().InsertOne(ctx, u)
	if err != nil {
		if we, ok := err.(mongo.WriteException); ok {
			for _, e := range we.WriteErrors {
				if e.Code == 11000 { 
					c.JSON(http.StatusConflict, gin.H{"error":"نام کاربری گرفته شده است"}); return
				}
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error":"db error"}); return
	}
	id := res.InsertedID.(primitive.ObjectID).Hex()

	
	_ = config.Redis.HSet(ctx, utils.RedisHashUsernames, b.Username, 1).Err()

	c.JSON(http.StatusCreated, gin.H{"message":"User registered", "user_id": id})
}

type LoginBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(c *gin.Context) {
	var b LoginBody
	if err := c.BindJSON(&b); err != nil || b.Username == "" || b.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error":"invalid body"}); return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var u models.User
	err := utils.Users().FindOne(ctx, bson.M{"username": b.Username}).Decode(&u)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"invalid credentials"}); return
	}
	if !utils.CheckPassword(u.Password, b.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"invalid credentials"}); return
	}

	session := sessions.Default(c)
	session.Set("user_id", u.ID.Hex())
	session.Set("username", u.Username)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"failed to save session"}); return
	}


	c.JSON(http.StatusOK, gin.H{"message":"Login successful", "user_id": u.ID.Hex()})
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




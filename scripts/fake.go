package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var (
	mongoURI     = "mongodb://localhost:27017"
	redisAddress = "localhost:6379"
	ctx          = context.Background()
)

type User struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Username   string             `bson:"username"`
	Name       string             `bson:"name"`
	Email      string             `bson:"email"`
	Password   string             `bson:"password"`
	Department string             `bson:"department"`
}

type Paper struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	Title             string             `bson:"title"`
	Authors           []string           `bson:"authors"`
	Abstract          string             `bson:"abstract"`
	PublicationDate   primitive.DateTime `bson:"publication_date"`
	JournalConference string             `bson:"journal_conference"`
	Keywords          []string           `bson:"keywords"`
	UploadedBy        primitive.ObjectID `bson:"uploaded_by"`
	Views             int                `bson:"views"`
}

type Citation struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	PaperID      primitive.ObjectID `bson:"paper_id"`
	CitedPaperID primitive.ObjectID `bson:"cited_paper_id"`
}

func main() {
	// Setup
	mongoClient, _ := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	db := mongoClient.Database("research_manager")
	usersCol := db.Collection("users")
	papersCol := db.Collection("papers")
	citationsCol := db.Collection("citations")

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddress,
		Password: "123456",
		DB:   0,
	})

	gofakeit.Seed(0)

	var userIDs []primitive.ObjectID

	// --- USERS ---
	fmt.Println("Seeding users...")
	for i := 0; i < 100; i++ {
		username := gofakeit.Username()
		password, _ := bcrypt.GenerateFromPassword([]byte(gofakeit.Password(true, true, true, false, false, 10)), 14)

		user := User{
			Username:   username,
			Name:       gofakeit.Name(),
			Email:      gofakeit.Email(),
			Password:   string(password),
			Department: gofakeit.JobTitle(),
		}

		res, _ := usersCol.InsertOne(ctx, user)
		uid := res.InsertedID.(primitive.ObjectID)
		userIDs = append(userIDs, uid)

		redisClient.HSet(ctx, "usernames", username, 1)
	}

	var paperIDs []primitive.ObjectID

	// --- PAPERS ---
	fmt.Println("Seeding papers...")
	for i := 0; i < 1000; i++ {
		authorCount := gofakeit.Number(1, 5)
		authors := make([]string, authorCount)
		for j := range authors {
			authors[j] = gofakeit.Name()
		}

		keywordCount := gofakeit.Number(1, 5)
		keywords := make([]string, keywordCount)
		for j := range keywords {
			keywords[j] = gofakeit.Word()
		}

		paper := Paper{
			Title:             gofakeit.Sentence(6),
			Authors:           authors,
			Abstract:          gofakeit.Paragraph(1, 3, 30, " "),
			PublicationDate:   primitive.NewDateTimeFromTime(gofakeit.DateRange(time.Date(2015, 6, 5, 0, 0, 0, 0, time.UTC), time.Date(2025, 6, 5, 0, 0, 0, 0, time.UTC))),
			JournalConference: gofakeit.Company(),
			Keywords:          keywords,
			UploadedBy:        userIDs[rand.Intn(len(userIDs))],
			Views:             0,
		}

		res, _ := papersCol.InsertOne(ctx, paper)
		paperIDs = append(paperIDs, res.InsertedID.(primitive.ObjectID))
	}

	// --- CITATIONS ---
	fmt.Println("Seeding citations...")
	for _, pid := range paperIDs {
		numCitations := gofakeit.Number(0, 5)
		cited := map[string]bool{}
		for i := 0; i < numCitations; i++ {
			var citedID primitive.ObjectID
			for {
				candidate := paperIDs[rand.Intn(len(paperIDs))]
				if candidate != pid && !cited[candidate.Hex()] {
					citedID = candidate
					cited[candidate.Hex()] = true
					break
				}
			}

			citation := Citation{
				PaperID:      pid,
				CitedPaperID: citedID,
			}

			citationsCol.InsertOne(ctx, citation)
		}
	}

	fmt.Println("✅ Seed complete.")
}

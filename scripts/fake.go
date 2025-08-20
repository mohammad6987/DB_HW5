package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/bxcodec/faker/v4"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
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
	PublicationDate   time.Time          `bson:"publication_date"`
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
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	db := mongoClient.Database("research_db")
	usersColl := db.Collection("users")
	papersColl := db.Collection("papers")
	citationsColl := db.Collection("citations")


	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379" , Password: "123456"})

	
	var userIDs []primitive.ObjectID
	usernames := make(map[string]bool)

	for len(userIDs) < 100 {
		username := faker.Username()
		if usernames[username] {
			continue
		}
		usernames[username] = true

		name := faker.Name()
		email := faker.Email()
		passPlain := faker.Password()
		passHash, _ := bcrypt.GenerateFromPassword([]byte(passPlain), bcrypt.DefaultCost)
		dept := faker.Word()

		u := User{
			Username:   username,
			Name:       name,
			Email:      email,
			Password:   string(passHash),
			Department: dept,
		}

		res, err := usersColl.InsertOne(ctx, u)
		if err != nil {
			log.Fatal(err)
		}
		userIDs = append(userIDs, res.InsertedID.(primitive.ObjectID))

		// 
		_ = rdb.HSet(ctx, "usernamessss", username, 1).Err()

		err2 := rdb.HSet(ctx, "usernamessss", username, 1).Err()
		if err2 != nil {
    		log.Printf("Redis HSet error: %v", err2)
		} else {
    		log.Printf("Added username to Redis: %s", username)
		}
	}
	fmt.Println("Inserted", len(userIDs), "users")

	
	var paperIDs []primitive.ObjectID
	startDate := time.Date(2015, 6, 5, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 6, 5, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 1000; i++ {
		title := faker.Sentence()
		nAuthors := rand.Intn(5) + 1
		authors := make([]string, nAuthors)
		for j := range authors {
			authors[j] = faker.Name()
		}

		abstract := faker.Paragraph()

	
		pubRange := endDate.Sub(startDate)
		pubDate := startDate.Add(time.Duration(rand.Int63n(int64(pubRange))))

		jc := faker.Word() + " " + faker.Word() + " Conference"

		nKeywords := rand.Intn(5) + 1
		keywords := make([]string, nKeywords)
		for j := range keywords {
			keywords[j] = faker.Word()
		}

		uploadedBy := userIDs[rand.Intn(len(userIDs))]

		p := Paper{
			Title:             title,
			Authors:           authors,
			Abstract:          abstract,
			PublicationDate:   pubDate,
			JournalConference: jc,
			Keywords:          keywords,
			UploadedBy:        uploadedBy,
			Views:             0,
		}
		res, err := papersColl.InsertOne(ctx, p)
		if err != nil {
			log.Fatal(err)
		}
		paperIDs = append(paperIDs, res.InsertedID.(primitive.ObjectID))
	}
	fmt.Println("Inserted", len(paperIDs), "papers")


	for _, pid := range paperIDs {
		nCitations := rand.Intn(6)
		for i := 0; i < nCitations; i++ {
			target := paperIDs[rand.Intn(len(paperIDs))]
			if target == pid {
				continue
			}
			c := Citation{
				PaperID:      pid,
				CitedPaperID: target,
			}
			_, _ = citationsColl.InsertOne(ctx, c)
		}
	}
	fmt.Println("Citations generated")
}

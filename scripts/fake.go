package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"DB_HW5/config"
	"DB_HW5/models"
	"DB_HW5/utils"
)

func main() {
	config.Init()
	utils.EnsureIndexes()

	ctx := context.Background()
	gofakeit.Seed(0)

	var users []interface{}
	userIDs := make([]primitive.ObjectID, 0, 100)
	for i := 0; i < 100; i++ {
		username := gofakeit.Username()
		
		if len(username) < 3 { username += "123" }
		if len(username) > 20 { username = username[:20] }

		pass := gofakeit.Password(true,true,true,true,false,10)
		if len(pass) < 8 { pass += "Abcdef12" }

		hash, _ := utils.HashPassword(pass)
		u := models.User{
			Username: username,
			Name: gofakeit.Name(),
			Email: gofakeit.Email(),
			Password: hash,
			Department: gofakeit.JobTitle(),
		}
		users = append(users, u)
	}
	ur, err := utils.Users().InsertMany(ctx, users)
	if err != nil { log.Fatal(err) }
	for _, id := range ur.InsertedIDs {
		userIDs = append(userIDs, id.(primitive.ObjectID))
	}

	for _, u := range users {
		un := u.(models.User).Username
		_ = config.Redis.HSet(ctx, utils.RedisHashUsernames, un, 1).Err()
	}

	var papers []interface{}
	paperIDs := make([]primitive.ObjectID, 0, 1000)
	start, _ := time.Parse("2006-01-02", "2015-06-05")
	end, _ := time.Parse("2006-01-02", "2025-06-05")
	delta := end.Sub(start)

	for i := 0; i < 1000; i++ {
		nAuthors := rand.Intn(5) + 1
		authors := make([]string, nAuthors)
		for j := 0; j < nAuthors; j++ {
			authors[j] = gofakeit.Name()
			if len(authors[j]) > 100 { authors[j] = authors[j][:100] }
		}
		nKeywords := rand.Intn(5) + 1
		keywords := make([]string, nKeywords)
		for j := 0; j < nKeywords; j++ {
			keywords[j] = gofakeit.Word()
			if len(keywords[j]) > 50 { keywords[j] = keywords[j][:50] }
		}

		pub := start.Add(time.Duration(rand.Int63n(int64(delta))))
		title := gofakeit.Sentence(6)
		abs := gofakeit.Paragraph(1, 5, 12, " ")

		p := models.Paper{
			Title: truncate(title, 200),
			Authors: authors,
			Abstract: truncate(abs, 1000),
			PublicationDate: pub,
			JournalConference: truncate(gofakeit.Company(), 200),
			Keywords: keywords,
			UploadedBy: userIDs[rand.Intn(len(userIDs))],
			Views: 0,
		}
		papers = append(papers, p)
	}
	pr, err := utils.Papers().InsertMany(ctx, papers)
	if err != nil { log.Fatal(err) }
	for _, id := range pr.InsertedIDs {
		paperIDs = append(paperIDs, id.(primitive.ObjectID))
	}

	
	var cites []interface{}
	for _, pid := range paperIDs {
		n := rand.Intn(6) 
		for i := 0; i < n; i++ {
			to := paperIDs[rand.Intn(len(paperIDs))]
			if to == pid { continue } // no self-citation
			cites = append(cites, models.Citation{PaperID: pid, CitedPaperID: to})
		}
	}
	if len(cites) > 0 {
		if _, err := utils.Citations().InsertMany(ctx, cites); err != nil { log.Fatal(err) }
	}
	_, _ = utils.Papers().Indexes().CreateOne(ctx, mongoIndexText())

	log.Println("Seeding finished.")
}

func truncate(s string, max int) string {
	if len(s) > max { return s[:max] }
	return s
}

func mongoIndexText() interface{} {

	return struct{
	}{}
}

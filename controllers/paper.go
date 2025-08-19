package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"DB_HW5/config"

	"DB_HW5/utils"
)

type UploadPaperBody struct {
	Title             string   `json:"title"`
	Authors           []string `json:"authors"`
	Abstract          string   `json:"abstract"`
	PublicationDate   string   `json:"publication_date"`
	JournalConference string   `json:"journal_conference"`
	Keywords          []string `json:"keywords"`
	Citations         []string `json:"citations"`
}

func PostPaper(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing X-User-ID"})
		return
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	// raw countDocuments for user
	userCountCmd := bson.D{
		{"count", "users"},
		{"query", bson.M{"_id": uid}},
	}
	var userCountRes bson.M
	if err := config.MongoClient.Database("your_db").RunCommand(ctx, userCountCmd).Decode(&userCountRes); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	if userCountRes["n"].(int32) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	// parse body
	var b UploadPaperBody
	if err := c.BindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	// validations
	if !(len(b.Title) > 0 && len(b.Title) <= 200) ||
		!(len(b.Abstract) > 0 && len(b.Abstract) <= 1000) ||
		len(b.Authors) < 1 || len(b.Authors) > 5 ||
		len(b.Keywords) < 1 || len(b.Keywords) > 5 ||
		len(b.JournalConference) <= 0 || len(b.JournalConference) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}
	for _, a := range b.Authors {
		if len(a) == 0 || len(a) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid author"})
			return
		}
	}
	for _, k := range b.Keywords {
		if len(k) == 0 || len(k) > 50 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid keyword"})
			return
		}
	}
	pubTime, err := time.Parse("2006-01-02", b.PublicationDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid publication_date"})
		return
	}

	// prepare doc
	paperDoc := bson.M{
		"title":             b.Title,
		"authors":           b.Authors,
		"abstract":          b.Abstract,
		"publication_date":  pubTime,
		"journal_conference": b.JournalConference,
		"keywords":          b.Keywords,
		"uploaded_by":       uid,
		"views":             0,
	}

	// raw insert
	insertCmd := bson.D{{"insert", "papers"}, {"documents", []interface{}{paperDoc}}}
	var insertRes bson.M
	if err := config.MongoClient.Database("your_db").RunCommand(ctx, insertCmd).Decode(&insertRes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// retrieve the inserted doc to get its _id
	var insertedDoc bson.M
	_ = config.MongoClient.Database("your_db").
		Collection("papers").FindOne(ctx, bson.M{"title": b.Title, "uploaded_by": uid}).Decode(&insertedDoc)
	paperID := insertedDoc["_id"].(primitive.ObjectID)

	// citations
	if len(b.Citations) > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max 5 citations"})
		return
	}
	if len(b.Citations) > 0 {
		var citationDocs []interface{}
		for _, cid := range b.Citations {
			oid, err := primitive.ObjectIDFromHex(cid)
			if err != nil || oid == paperID {
				c.JSON(http.StatusNotFound, gin.H{"error": "invalid citation id"})
				return
			}

			// check cited paper
			countCmd := bson.D{
				{"count", "papers"},
				{"query", bson.M{"_id": oid}},
			}
			var cntRes bson.M
			if err := config.MongoClient.Database("your_db").RunCommand(ctx, countCmd).Decode(&cntRes); err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "invalid citation id"})
				return
			}
			if cntRes["n"].(int32) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "invalid citation id"})
				return
			}

			citationDocs = append(citationDocs, bson.M{
				"paper_id":       paperID,
				"cited_paper_id": oid,
			})
		}
		if len(citationDocs) > 0 {
			insertCitCmd := bson.D{{"insert", "citations"}, {"documents", citationDocs}}
			var citRes bson.M
			if err := config.MongoClient.Database("your_db").RunCommand(ctx, insertCitCmd).Decode(&citRes); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "citation insert error"})
				return
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Paper uploaded", "paper_id": paperID.Hex()})
}

func GetPaperDetails(c *gin.Context) {
	id := c.Param("id")
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	// raw find
	findCmd := bson.D{
		{"find", "papers"},
		{"filter", bson.M{"_id": oid}},
		{"limit", 1},
	}
	var findRes bson.M
	if err := config.MongoClient.Database("your_db").RunCommand(ctx, findCmd).Decode(&findRes); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	cursor := findRes["cursor"].(bson.M)
	batch := cursor["firstBatch"].(primitive.A)
	if len(batch) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	paper := batch[0].(bson.M)

	// raw count for citations
	countCitCmd := bson.D{
		{"count", "citations"},
		{"query", bson.M{"cited_paper_id": oid}},
	}
	var citCountRes bson.M
	_ = config.MongoClient.Database("your_db").RunCommand(ctx, countCitCmd).Decode(&citCountRes)
	citCnt := int32(0)
	if val, ok := citCountRes["n"].(int32); ok {
		citCnt = val
	}

	// redis views
	viewsKey := utils.PaperViewsKey(id)
	_ = config.Redis.Incr(ctx, viewsKey).Err()
	curViews, _ := config.Redis.Get(ctx, viewsKey).Int64()

	c.JSON(http.StatusOK, gin.H{
		"id":                paper["_id"].(primitive.ObjectID).Hex(),
		"title":             paper["title"],
		"authors":           paper["authors"],
		"abstract":          paper["abstract"],
		"publication_date":  paper["publication_date"].(primitive.DateTime).Time().Format("2006-01-02"),
		"journal_conference": paper["journal_conference"],
		"keywords":          paper["keywords"],
		"citation_count":    citCnt,
		"views":             curViews,
	})
}

package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"DB_HW5/config"
	"DB_HW5/models"
	"DB_HW5/utils"
)

type UploadPaperBody struct {
	Title            string   `json:"title"`
	Authors          []string `json:"authors"`
	Abstract         string   `json:"abstract"`
	PublicationDate  string   `json:"publication_date"` 
	JournalConference string  `json:"journal_conference"`
	Keywords         []string `json:"keywords"`
	Citations        []string `json:"citations"`
}

func PostPaper(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"missing X-User-ID"}); return
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"invalid user id"}); return
	}

	// ensure user exists
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	cnt, err := utils.Users().CountDocuments(ctx, bson.M{"_id": uid})
	if err != nil || cnt == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"invalid user"}); return
	}

	var b UploadPaperBody
	if err := c.BindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error":"invalid body"}); return
	}

	// validations
	if !(len(b.Title) > 0 && len(b.Title) <= 200) ||
		!(len(b.Abstract) > 0 && len(b.Abstract) <= 1000) ||
		len(b.Authors) < 1 || len(b.Authors) > 5 ||
		len(b.Keywords) < 1 || len(b.Keywords) > 5 ||
		len(b.JournalConference) <= 0 || len(b.JournalConference) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error":"invalid fields"}); return
	}
	for _, a := range b.Authors {
		if len(a) == 0 || len(a) > 100 { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid author"}); return }
	}
	for _, k := range b.Keywords {
		if len(k) == 0 || len(k) > 50 { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid keyword"}); return }
	}
	pubTime, err := time.Parse("2006-01-02", b.PublicationDate)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid publication_date"}); return }

	p := models.Paper{
		Title: b.Title, Authors: b.Authors, Abstract: b.Abstract,
		PublicationDate: pubTime, JournalConference: b.JournalConference,
		Keywords: b.Keywords, UploadedBy: uid, Views: 0,
	}
	res, err := utils.Papers().InsertOne(ctx, p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error":"db error"}); return
	}
	paperID := res.InsertedID.(primitive.ObjectID)

	// validate citations (0..5) and insert to Citations
	if len(b.Citations) > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error":"max 5 citations"}); return
	}
	if len(b.Citations) > 0 {
		var docs []interface{}
		for _, cid := range b.Citations {
			oid, err := primitive.ObjectIDFromHex(cid)
			if err != nil || oid == paperID { 
				c.JSON(http.StatusNotFound, gin.H{"error":"invalid citation id"}); return
			}
			// ensure cited paper exists
			exists, err := utils.Papers().CountDocuments(ctx, bson.M{"_id": oid})
			if err != nil || exists == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error":"invalid citation id"}); return
			}
			docs = append(docs, models.Citation{PaperID: paperID, CitedPaperID: oid})
		}
		if len(docs) > 0 {
			if _, err := utils.Citations().InsertMany(ctx, docs); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error":"citation insert error"}); return
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message":"Paper uploaded", "paper_id": paperID.Hex()})
}

func GetPaperDetails(c *gin.Context) {
	id := c.Param("id")
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil { c.JSON(http.StatusNotFound, gin.H{"error":"not found"}); return }

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	var p models.Paper
	if err := utils.Papers().FindOne(ctx, bson.M{"_id": oid}).Decode(&p); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error":"not found"}); return
	}


	citCnt, _ := utils.Citations().CountDocuments(ctx, bson.M{"cited_paper_id": oid})


	viewsKey := utils.PaperViewsKey(id)
	_ = config.Redis.Incr(ctx, viewsKey).Err()
	curViews, _ := config.Redis.Get(ctx, viewsKey).Int64()

	c.JSON(http.StatusOK, gin.H{
		"id": p.ID.Hex(),
		"title": p.Title,
		"authors": p.Authors,
		"abstract": p.Abstract,
		"publication_date": p.PublicationDate.Format("2006-01-02"),
		"journal_conference": p.JournalConference,
		"keywords": p.Keywords,
		"citation_count": citCnt,
		"views": curViews,
	})
}

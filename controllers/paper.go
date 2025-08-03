package controllers

import (
    "net/http"

    "time"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo/options"
    "DB_HW5/config"
    "DB_HW5/models"
)

type UploadInput struct {
    Title            string   `json:"title"`
    Authors          []string `json:"authors"`
    Abstract         string   `json:"abstract"`
    PublicationDate  string   `json:"publication_date"` // ISO format
    JournalConference string  `json:"journal_conference"`
    Keywords         []string `json:"keywords"`
    Citations        []string `json:"citations"`
}

func UploadPaper(c *gin.Context) {
    userID := c.GetHeader("X-User-ID")
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing X-User-ID header"})
        return
    }

    var input UploadInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    uploaderID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
        return
    }

    pubDate, err := time.Parse("2006-01-02", input.PublicationDate)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid publication date"})
        return
    }

    paper := models.Paper{
        Title:            input.Title,
        Authors:          input.Authors,
        Abstract:         input.Abstract,
        PublicationDate:  primitive.NewDateTimeFromTime(pubDate),
        JournalConference: input.JournalConference,
        Keywords:         input.Keywords,
        UploadedBy:       uploaderID,
        Views:            0,
    }

    res, err := config.MongoDB.Collection("papers").InsertOne(config.Ctx, paper)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Mongo insert error"})
        return
    }

    paperID := res.InsertedID.(primitive.ObjectID)

    for _, citedID := range input.Citations {
        citedOID, err := primitive.ObjectIDFromHex(citedID)
        if err != nil {
            continue
        }
        citation := models.Citation{
            PaperID:      paperID,
            CitedPaperID: citedOID,
        }
        config.MongoDB.Collection("citations").InsertOne(config.Ctx, citation)
    }

    c.JSON(http.StatusCreated, gin.H{
        "message": "Paper uploaded",
        "paper_id": paperID.Hex(),
    })
}

func SearchPapers(c *gin.Context) {
    search := c.DefaultQuery("search", "")
    sortBy := c.DefaultQuery("sort_by", "relevance")
    order := c.DefaultQuery("order", "desc")

    redisKey := "search:" + search + ":" + sortBy + ":" + order
    cached, err := config.RedisClient.Get(config.Ctx, redisKey).Result()
    if err == nil && cached != "" {
        c.Data(http.StatusOK, "application/json", []byte(cached))
        return
    }

    findOptions := options.Find()
    if sortBy == "publication_date" {
        dir := -1
        if order == "asc" {
            dir = 1
        }
        findOptions.SetSort(bson.D{{Key: "publication_date", Value: dir}})
    } else {
        findOptions.SetSort(bson.D{{Key: "score", Value: -1}})
        findOptions.SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}})
    }

    filter := bson.M{}
    if search != "" {
        filter = bson.M{"$text": bson.M{"$search": search}}
    }

    cur, err := config.MongoDB.Collection("papers").Find(config.Ctx, filter, findOptions)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
        return
    }
    defer cur.Close(config.Ctx)

    var papers []gin.H
    for cur.Next(config.Ctx) {
        var p models.Paper
        cur.Decode(&p)

        papers = append(papers, gin.H{
            "id":                p.ID.Hex(),
            "title":             p.Title,
            "authors":           p.Authors,
            "publication_date":  p.PublicationDate.Time().Format("2006-01-02"),
            "journal_conference": p.JournalConference,
            "keywords":          p.Keywords,
        })
    }

    jsonData := gin.H{"papers": papers}
    c.JSON(http.StatusOK, jsonData)

    // Cache result in Redis
    config.RedisClient.Set(config.Ctx, redisKey, c.MustGet(gin.BodyBytesKey), 300*time.Second)
}

func GetPaperDetails(c *gin.Context) {
    idStr := c.Param("id")
    paperID, err := primitive.ObjectIDFromHex(idStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid paper ID"})
        return
    }

    var paper models.Paper
    err = config.MongoDB.Collection("papers").FindOne(config.Ctx, bson.M{"_id": paperID}).Decode(&paper)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Paper not found"})
        return
    }

    // Increment views in Redis
    redisKey := "paper_views:" + idStr
    config.RedisClient.Incr(config.Ctx, redisKey)

    // Get citation count
    count, _ := config.MongoDB.Collection("citations").CountDocuments(config.Ctx, bson.M{"cited_paper_id": paperID})

    c.JSON(http.StatusOK, gin.H{
        "id":                paper.ID.Hex(),
        "title":             paper.Title,
        "authors":           paper.Authors,
        "abstract":          paper.Abstract,
        "publication_date":  paper.PublicationDate.Time().Format("2006-01-02"),
        "journal_conference": paper.JournalConference,
        "keywords":          paper.Keywords,
        "citation_count":    count,
        "views":             paper.Views, 
    })
}

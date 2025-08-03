package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"DB_HW5/config"
)

func StartViewSyncScheduler() {
	c := cron.New()
	c.AddFunc("@every 10m", SyncViews)
	c.Start()
	fmt.Println("View sync scheduler started (every 10 minutes)")
}

func SyncViews() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keys, err := config.RedisClient.Keys(ctx, "paper_views:*").Result()
	if err != nil {
		fmt.Println("Failed to list Redis keys:", err)
		return
	}

	for _, key := range keys {
		paperID := strings.TrimPrefix(key, "paper_views:")
		objID, err := primitive.ObjectIDFromHex(paperID)
		if err != nil {
			fmt.Printf("Skipping invalid ObjectID: %s\n", paperID)
			continue
		}

		views, err := config.RedisClient.Get(ctx, key).Int()
		if err != nil {
			fmt.Printf("Failed to get view count for %s: %v\n", key, err)
			continue
		}

		if views == 0 {
			continue
		}

		_, err = config.MongoDB.Collection("papers").UpdateOne(
			ctx,
			bson.M{"_id": objID},
			bson.M{"$inc": bson.M{"views": views}},
		)
		if err != nil {
			fmt.Printf("MongoDB update failed for %s: %v\n", key, err)
			continue
		}

		config.RedisClient.Set(ctx, key, 0, 0)
		fmt.Printf("✅ Synced %d views for paper %s\n", views, paperID)
	}
}

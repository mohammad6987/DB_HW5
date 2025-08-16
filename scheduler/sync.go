package scheduler

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"DB_HW5/config"
	"DB_HW5/utils"
)

func StartViewsSync() {

	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			if err := syncOnce(); err != nil {
				log.Printf("views sync error: %v", err)
			}
		}
	}()
}

func syncOnce() error {
	ctx := context.Background()
	iter := config.Redis.Scan(ctx, 0, "paper_views:*", 1000).Iterator()
	for iter.Next(ctx) {
		key := iter.Val() 
		idHex := strings.TrimPrefix(key, "paper_views:")
		count, err := config.Redis.Get(ctx, key).Int64()
		if err != nil && err != redis.Nil { continue }
		if count <= 0 { continue }

		oid, err := primitive.ObjectIDFromHex(idHex)
		if err != nil { continue }
		// update Mongo
		_, _ = utils.Papers().UpdateByID(ctx, oid, bson.M{"$inc": bson.M{"views": count}})
		// reset to 0
		_ = config.Redis.Set(ctx, key, 0, 0).Err()
	}
	return iter.Err()
}

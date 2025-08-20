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
)

func StartViewsSync() {

	ticker := time.NewTicker(1 * time.Minute)
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
        redisCount, err := config.Redis.Get(ctx, key).Int64()
        if err != nil && err != redis.Nil {
            continue
        }
        if redisCount <= 0 {
            continue
        }

        oid, err := primitive.ObjectIDFromHex(idHex)
        if err != nil {
            continue
        }

        papersColl := config.MongoClient.Database("research_db").Collection("papers")


        update := bson.M{"$inc": bson.M{"views": redisCount}}
        _, err = papersColl.UpdateByID(ctx, oid, update)
        if err != nil {
            log.Printf("failed to update paper %s: %v", idHex, err)
            continue
        }

        var paper struct{ Views int64 `bson:"views"` }
        _ = papersColl.FindOne(ctx, bson.M{"_id": oid}).Decode(&paper)
        _ = config.Redis.Set(ctx, key, paper.Views, 0).Err()

		//_ = config.Redis.Set(ctx, key, 0, 0).Err()
		//I know I should set it to zero but setting it to mongo value seemed better



    }
    return iter.Err()
}

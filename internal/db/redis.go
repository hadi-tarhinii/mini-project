package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client



func ConnectRedis(address string) {
    // FIX: Use redis.NewClient
    RDB = redis.NewClient(&redis.Options{
        Addr: address,
        Password: "", // no password set
        DB: 0,        // use default DB
    })
    
    // It's good practice to Ping Redis to ensure it's actually alive
    ctx := context.Background()
    _, err := RDB.Ping(ctx).Result()
    if err != nil {
        fmt.Printf("Could not connect to Redis: %v\n", err)
    } else {
        fmt.Println("Connected to Redis successfully!")
    }
}
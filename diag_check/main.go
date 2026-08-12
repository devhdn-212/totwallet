package main

import (
	"context"
	"fmt"
	"log"

	"github.com/devhdn-212/totwallet/internal/config"
	"github.com/redis/go-redis/v9"
)

func main() {
	conf := config.Get()
	dbnum := 0
	fmt.Sscanf(conf.Redis.Name, "%d", &dbnum)
	rdb := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Host + ":" + conf.Redis.Port,
		Password: conf.Redis.Pass,
		DB:       dbnum,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("ping redis: ", err)
	}
	fmt.Println("connected to redis DB", dbnum)

	iter := rdb.Scan(ctx, 0, "*", 100).Iterator()
	count := 0
	for iter.Next(ctx) {
		key := iter.Val()
		ttl, _ := rdb.TTL(ctx, key).Result()
		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			val = "(non-string value, skip)"
		}
		fmt.Printf("key=%q ttl=%v val=%s\n", key, ttl, val)
		count++
		if count > 200 {
			fmt.Println("... (truncated)")
			break
		}
	}
	if err := iter.Err(); err != nil {
		log.Fatal("scan: ", err)
	}
	fmt.Println("total scanned:", count)
}

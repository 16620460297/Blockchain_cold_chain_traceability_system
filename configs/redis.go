package configs

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

var RedisClient *redis.Client

func InitRedis() error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "000000", // no password set
		DB:       0,        // use default DB
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RedisClient.Ping(ctx).Result()
	return err
}

// 存储JWT到Redis，设置过期时间
func StoreJWT(userID uint, token string, expiration time.Duration) error {
	ctx := context.Background()
	key := "user:token:" + string(rune(userID))
	return RedisClient.Set(ctx, key, token, expiration).Err()
}

// 从Redis获取JWT
func GetJWT(userID uint) (string, error) {
	ctx := context.Background()
	key := "user:token:" + string(rune(userID))
	return RedisClient.Get(ctx, key).Result()
}

// 删除Redis中的JWT
func DeleteJWT(userID uint) error {
	ctx := context.Background()
	key := "user:token:" + string(rune(userID))
	return RedisClient.Del(ctx, key).Err()
}

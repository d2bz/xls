package test

import (
	"testing"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

func TestRedisConn(t *testing.T) {
	rdb := redis.New(
		"localhost:6379",
		redis.WithPass("123456"),
	)
	res := rdb.Ping()
	if res {
		t.Log("Redis connection successful")
	} else {
		t.Error("Redis connection failed")
	}
}

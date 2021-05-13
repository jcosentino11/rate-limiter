package ratelimiter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func newRedisClient(s *miniredis.Miniredis) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
}

func newMockRedisServer() *miniredis.Miniredis {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	return s
}

func TestAquire(t *testing.T) {
	s := newMockRedisServer()
	defer s.Close()

	rdb := newRedisClient(s)
	defer rdb.Close()

	rl := NewRateLimiter(rdb, &Options{
		RequestsPerSecond: 1,
	})

	allowed, err := rl.Aquire(context.TODO(), "key1")
	if err != nil {
		panic(err)
	}

	if !allowed {
		t.Fatal("Unable to aquire rate limit")
	}
}

func TestHelloIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	time.Sleep(1 * time.Second)
	t.Fail()
}

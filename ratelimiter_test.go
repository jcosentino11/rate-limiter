package ratelimiter

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

var rdb *redis.Client
var mockRedis *miniredis.Miniredis

func TestMain(m *testing.M) {
	mockRedis = newMockRedisServer()
	rdb = newRedisClient(mockRedis)

	exitCode := m.Run()

	if rdb != nil {
		rdb.Close()
	}

	if mockRedis != nil {
		mockRedis.Close()
	}

	os.Exit(exitCode)
}

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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := "key"
	count := 5

	rl := NewTimestampBinnedRateLimiter(rdb, PerSecond(count))

	aquire := func() bool {
		allowed, err := rl.Aquire(ctx, key)
		if err != nil {
			panic(err)
		}
		return allowed
	}

	// requests within the rate limit
	for i := 0; i < count; i++ {
		if !aquire() {
			t.Fatal("Rate exceeded")
		}
	}

	// exceed the rate limit
	if aquire() {
		t.Fatal("Excessive aquire went through")
	}

	// wait for next window
	time.Sleep(time.Second)

	// requests within rate limit, in new window
	for i := 0; i < count; i++ {
		if !aquire() {
			t.Fatal("Rate exceeded")
		}
	}
}

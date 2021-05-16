package ratelimiter

import (
	"context"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
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

func TestRateLimitNotExceeded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	testDurationSeconds := 10
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(testDurationSeconds)*time.Second)
	defer cancel()

	key := "key"
	count := 10

	rl := NewTimestampBinnedRateLimiter(rdb, PerSecond(count))

	var wg sync.WaitGroup
	var numSuccessfulAquire uint64
	var totalNumAquire uint64

	aquire := func() bool {
		atomic.AddUint64(&totalNumAquire, 1)
		allowed, err := rl.Aquire(ctx, key)
		if err != nil {
			panic(err)
		}
		return allowed
	}

	aquireAsync := func() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if aquire() {
				atomic.AddUint64(&numSuccessfulAquire, 1)
			}
		}()
	}

	// spam requests
	func(minRequestPerSecond int, maxRequestPerSecond int) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				aquireAsync()
				time.Sleep(time.Second / time.Duration(rand.Intn(minRequestPerSecond)+(maxRequestPerSecond-minRequestPerSecond)))
			}
		}
	}(count, count*5) // ensure we spam more requests than the rate limit

	wg.Wait()

	maxExpectedNumSuccessfulAquire := uint64((count + 1) * testDurationSeconds)
	if numSuccessfulAquire > maxExpectedNumSuccessfulAquire {
		t.Fatalf("Too many requests allowed by the rate limiter. maxExpected=%d, actual=%d, totalRequests=%d", maxExpectedNumSuccessfulAquire, numSuccessfulAquire, totalNumAquire)
	} else {
		t.Logf("Rate limiting passed. maxExpected=%d, actual=%d, totalRequests=%d", maxExpectedNumSuccessfulAquire, numSuccessfulAquire, totalNumAquire)
	}
}

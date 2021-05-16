package ratelimiter

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Allowance struct {
	count  int
	period time.Duration
}

func PerSecond(n int) Allowance {
	return Allowance{
		count:  n,
		period: time.Second,
	}
}

type RateLimiter interface {
	Aquire(ctx context.Context, key string) (bool, error)
}

type baseRateLimiter struct {
	allowance Allowance
}

func newBaseRateLimiter(allowance Allowance) *baseRateLimiter {
	return &baseRateLimiter{allowance: allowance}
}

type redisRateLimiter struct {
	rdb *redis.Client
	*baseRateLimiter
}

func newRedisRateLimiter(rdb *redis.Client, allowance Allowance) *redisRateLimiter {
	return &redisRateLimiter{
		rdb:             rdb,
		baseRateLimiter: newBaseRateLimiter(allowance),
	}
}

type TimestampBinnedRateLimiter struct {
	*redisRateLimiter
}

func NewTimestampBinnedRateLimiter(rdb *redis.Client, allowance Allowance) *TimestampBinnedRateLimiter {
	return &TimestampBinnedRateLimiter{
		redisRateLimiter: newRedisRateLimiter(rdb, allowance),
	}
}

func (rl *TimestampBinnedRateLimiter) timeBucket() string {
	// TODO minutes, hour, day
	return fmt.Sprintf("%d", time.Now().Unix())
}

func (rl *TimestampBinnedRateLimiter) Aquire(ctx context.Context, key string) (bool, error) {
	timeBoxedKey := key + ":" + rl.timeBucket()

	counter := rl.rdb.Get(ctx, timeBoxedKey)
	if counter.Err() != redis.Nil {
		if val, _ := counter.Int64(); val >= int64(rl.allowance.count) {
			return false, nil
		}
	}

	var incr *redis.IntCmd
	_, err := rl.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		incr = p.Incr(ctx, timeBoxedKey)
		p.Expire(ctx, timeBoxedKey, rl.allowance.period)
		return nil
	})

	if err != nil {
		return false, err
	}

	return incr.Val() <= int64(rl.allowance.count), nil
}

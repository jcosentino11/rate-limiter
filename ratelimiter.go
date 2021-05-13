package ratelimiter

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RateLimiter struct {
	rdb *redis.Client
	opt *Options
}

type Options struct {
	RequestsPerSecond int64
}

func NewDefaultOptions() *Options {
	return &Options{
		RequestsPerSecond: 100,
	}
}

func NewRateLimiter(rdb *redis.Client, opt *Options) *RateLimiter {
	return &RateLimiter{
		rdb: rdb,
		opt: opt,
	}
}

func (rl *RateLimiter) Aquire(ctx context.Context, key string) (bool, error) {
	// bin key by second
	currSecond := fmt.Sprintf("%d", time.Now().Unix())
	timeBoxedKey := key + ":" + currSecond

	pipe := rl.rdb.Pipeline()
	incr := pipe.Incr(ctx, timeBoxedKey)
	pipe.Expire(ctx, key, 2*time.Second)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	return incr.Val() <= rl.opt.RequestsPerSecond, nil
}

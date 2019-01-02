package kv

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
)

var pool *redis.Pool

// Conn is a handle for the database
func Conn() redis.Conn {
	return pool.Get()
}

// Load queue configuration and ping the server.
func Load(ctx context.Context, server string) (*redis.Pool, error) {
	pool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", server)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			return ctx.Err()
		},
	}

	if _, err := pool.Get().Do("PING"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "can't ping redis: %v\n", err)
	}

	return pool, nil
}

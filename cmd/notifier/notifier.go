package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/henvic/productreview/db"
	"github.com/henvic/productreview/kv"
	"github.com/henvic/productreview/reviews"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var dsn string
var redis string

func setup(ctx context.Context) error {
	if _, err := db.Load(ctx, dsn); err != nil {
		return err
	}

	_, err := kv.Load(ctx, redis)
	return err
}

func run() error {
	ctx := context.Background()

	if err := setup(ctx); err != nil {
		return err
	}

	log.Print("Starting reviews notifier")

	return reviews.Notifier(ctx)
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	flag.Parse()

	log.SetLevel(log.DebugLevel)

	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

func init() {
	flag.StringVar(&dsn, "dsn", "postgres://postgres:postgres@/AdventureWorks?sslmode=disable", "dsn (PostgreSQL)")
	flag.StringVar(&redis, "redis", ":6379", "redis server")
}

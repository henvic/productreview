package main

import (
	"context"
	_ "expvar"
	"flag"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/henvic/ctxsignal"
	"github.com/henvic/productreview/server"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var params = server.Params{}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	flag.Parse()

	var debug = (os.Getenv("DEBUG") != "")

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	if debug || params.ExposeDebug {
		go profiler()
	}

	ctx, cancel := ctxsignal.WithTermination(context.Background())
	defer cancel()

	if err := server.Start(ctx, params); err != nil {
		log.Fatal(err)
	}
}

func profiler() {
	// let expvar and pprof be exposed here indirectly through http.DefaultServeMux
	log.Info("Exposing expvar and pprof on localhost:8081")
	log.Fatal(http.ListenAndServe("localhost:8081", nil))
}

func init() {
	flag.StringVar(&params.Address, "addr", "127.0.0.1:8888", "Serving address")
	flag.StringVar(&params.DSN, "dsn", "postgres://postgres:postgres@/AdventureWorks?sslmode=disable", "dsn (PostgreSQL)")
	flag.StringVar(&params.Redis, "redis", ":6379", "redis server")
	flag.BoolVar(&params.ExposeDebug, "expose-debug", false, "Expose debugging tools over HTTP (on port 8081)")
}

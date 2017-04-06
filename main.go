package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/braintree/manners"
	"github.com/squarescale/simple-builder/handlers"
)

const version = "1.0.0"

var Health struct {
	Lock   sync.Mutex
	Status int
}

func GetenvDef(name, def string) string {
	res := os.Getenv(name)
	if res == "" {
		res = def
	}
	return res
}

func main() {
	log.Println("Starting Simple Builder...")

	httpAddr := os.Getenv("NOMAD_ADDR_http")
	if httpAddr == "" {
		httpAddr = "localhost:80"
	}

	flag.StringVar(&httpAddr, "http", httpAddr, "Listening address")
	flag.Parse()

	log.Printf("HTTP service listening on %s", httpAddr)

	ctx, cancelContext := context.WithCancel(context.Background())
	mux := http.NewServeMux()
	mux.Handle("/version", handlers.VersionHandler(version))
	mux.Handle("/health", handlers.HealthHandler(&Health, &Health.Status, &Health.Lock))
	mux.Handle("/builds", handlers.BuildsHandler(ctx, ""))

	httpServer := manners.NewServer()
	httpServer.Addr = httpAddr
	httpServer.Handler = handlers.LoggingHandler(mux)

	errChan := make(chan error, 10)

	go func() {
		errChan <- httpServer.ListenAndServe()
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChan:
			if err != nil {
				log.Fatal(err)
			}
		case s := <-signalChan:
			log.Println(fmt.Sprintf("Captured %v. Exiting...", s))
			cancelContext()
			httpServer.BlockingClose()
			os.Exit(0)
		}
	}
}

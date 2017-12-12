package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/braintree/manners"
	"github.com/squarescale/simple-builder/handlers"
)

var version string
var buildsHandler handlers.BuildsHandler

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
	log.Printf("Starting Simple Builder version %s...", version)

	httpAddr := os.Getenv("NOMAD_ADDR_http")
	if httpAddr == "" {
		httpAddr = "localhost:80"
	}

	flag.StringVar(&httpAddr, "http", httpAddr, "Listening address")
	flag.Parse()

	log.Printf("HTTP service listening on %s", httpAddr)

	ctx, cancelContext := context.WithCancel(context.Background())
	buildsHandler = handlers.NewBuildsHandler(ctx, "")
	mux := http.NewServeMux()
	mux.Handle("/version", handlers.VersionHandler(version))
	mux.Handle("/health", handlers.HealthHandler(&Health, &Health.Status, &Health.Lock))
	mux.Handle("/builds", buildsHandler)

	httpServer := manners.NewWithServer(&http.Server{
		Addr:    httpAddr,
		Handler: handlers.LoggingHandler(mux),
	})

	go runSQSListener(ctx)

	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, os.Interrupt, os.Kill, syscall.SIGTERM)
		s := <-sigchan
		log.Printf("Captured %v. Shutting down...", s)
		signal.Stop(sigchan)
		cancelContext()
		httpServer.Close()
	}()

	log.Fatal(httpServer.ListenAndServe())
}

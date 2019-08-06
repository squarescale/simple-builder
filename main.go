package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/braintree/manners"
	"github.com/squarescale/simple-builder/build"
	"github.com/squarescale/simple-builder/handlers"
)

var version string
var buildsHandler handlers.BuildsHandler

var Health struct {
	Lock   sync.Mutex
	Status int
}

func main() {
	log.Printf("Starting Simple Builder version %s...", version)

	var buildJobFile string
	httpAddr := os.Getenv("NOMAD_ADDR_http")
	if httpAddr == "" {
		httpAddr = "localhost:80"
	}

	flag.StringVar(&httpAddr, "http", httpAddr, "Listening address")
	flag.StringVar(&buildJobFile, "build-job", "", "Build job file (single job mode)")
	flag.Parse()

	var wg sync.WaitGroup
	ctx, cancelContext := context.WithCancel(context.Background())

	var singleBuildDescriptor handlers.BuildDescriptorWithCallback
	var singleBuildMode bool
	var singleBuild *build.Build
	var err error
	if buildJobFile != "" {
		log.Printf("Single build mode with %s", buildJobFile)
		singleBuildMode = true
		singleBuildDescriptor, err = parseJobFile(buildJobFile)
		if err != nil {
			log.Print(err)
			os.Exit(1)
			return
		}
	}

	log.Printf("HTTP service listening on %s", httpAddr)

	buildsHandler = handlers.NewBuildsHandler(ctx, "", singleBuildMode)
	if singleBuildMode {
		singleBuild, _, err = buildsHandler.CreateBuild(&wg, singleBuildDescriptor)
		if err != nil {
			log.Print(err)
			os.Exit(1)
			return
		}
	}

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

	if singleBuildMode {
		var exitCode int = 1
		buildEnd := make(chan struct{})
		httpEnd := make(chan struct{})

		go func() {
			defer close(buildEnd)
			wg.Wait()
		}()

		go func() {
			defer close(httpEnd)
			err = httpServer.ListenAndServe()
			if err != nil {
				log.Print(err)
			}
		}()

		select {
		case <-buildEnd:
			if len(singleBuild.Errors) == 0 {
				exitCode = 0
			}
			cancelContext()
			httpServer.Close()
		case <-ctx.Done():
		}

		<-httpEnd
		os.Exit(exitCode)

	} else {
		err = httpServer.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func parseJobFile(filename string) (b handlers.BuildDescriptorWithCallback, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return b, err
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&b)
	return b, err
}

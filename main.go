package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/squarescale/libsqsc/signals"
	"github.com/squarescale/simple-builder/pkg/builder"
	"github.com/squarescale/simple-builder/pkg/version"
)

var (
	flagBuildJob = flag.String("build-job", "", "Build job file (single job mode)")
)

func main() {
	err := checkFlags()
	fatal(err)

	banner()

	ctx, cancelFunc := context.WithCancel(
		context.Background(),
	)

	signals.StartCtrlCHandler(cancelFunc)

	b, err := builder.New(ctx, *flagBuildJob)
	fatal(err)

	go startHTTPServer(b)

	err = b.Run()
	fatal(err)

	b.Cleanup()

	os.Exit(0)
}

// ---

func banner() {
	log.Printf(
		"Starting Simple Builder version %s/%s (%s)...",
		version.GitBranch,
		version.GitCommit,
		version.BuildDate,
	)

	log.Printf(
		"Using config file %q",
		*flagBuildJob,
	)
}

func fatal(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func checkFlags() error {
	flag.Parse()

	if *flagBuildJob == "" {
		return errors.New(
			"-build-job argument is empty",
		)
	}

	return nil
}

func startHTTPServer(b *builder.Builder) {
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		s := http.StatusInternalServerError

		if b.Status != builder.STATUS_FAILURE {
			s = http.StatusOK
		}

		w.WriteHeader(s)

		out := struct {
			Status string `json:"status"`
		}{
			Status: b.Status.String(),
		}

		json.NewEncoder(w).Encode(out)
	})

	fatal(
		http.ListenAndServe(":8000", nil),
	)
}

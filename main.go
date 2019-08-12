package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"sync"

	"github.com/squarescale/libsqsc/signals"
	"github.com/squarescale/simple-builder/pkg/builder"
)

var (
	flagBuildJob = flag.String("build-job", "", "Build job file (single job mode)")

	version string
	wg      sync.WaitGroup
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

	err = b.Run()
	fatal(err)

	os.Exit(0)
}

// ---

func banner() {
	log.Printf(
		"Starting Simple Builder version %s...",
		version,
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

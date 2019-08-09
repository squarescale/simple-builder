package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/squarescale/libsqsc/signals"
	"github.com/squarescale/simple-builder/build"
)

var (
	flagBuildJob = flag.String("build-job", "", "Build job file (single job mode)")

	version string
	wg      sync.WaitGroup
)

func main() {
	banner()

	err := checkFlags()
	fatal(err)

	ctx, cancelFunc := context.WithCancel(
		context.Background(),
	)

	signals.StartCtrlCHandler(cancelFunc)

	build, err := runBuild(ctx, &wg, *flagBuildJob)
	fatal(err)

	exitCode := 1
	buildEnd := make(chan struct{})

	go func() {
		defer close(buildEnd)
		wg.Wait()
	}()

	select {
	case <-buildEnd:
		if len(build.Errors) == 0 {
			exitCode = 0
		}

		cancelFunc()
	case <-ctx.Done():
	}

	os.Exit(exitCode)
}

// ---

type BuildDescriptorWithCallback struct {
	build.BuildDescriptor
	Callbacks []string `json:"callbacks"`
}

func runBuild(ctx context.Context, wg *sync.WaitGroup, cfgFile string) (*build.Build, error) {
	desc, err := parseJobFile(*flagBuildJob)
	if err != nil {
		return nil, err
	}

	b, err := startBuild(ctx, wg, desc)
	if err != nil {
		return nil, err
	}

	waitBuildDone(wg, b)

	err = maybeNotifyCallbacks(b, desc.Callbacks)

	return b, err
}

func parseJobFile(filename string) (b BuildDescriptorWithCallback, err error) {
	f, err := os.Open(filename)

	if err != nil {
		return b, err
	}

	defer f.Close()

	err = json.NewDecoder(f).Decode(&b)

	return b, err
}

func startBuild(ctx context.Context, wg *sync.WaitGroup, desc BuildDescriptorWithCallback) (*build.Build, error) {
	tmp, err := ioutil.TempDir(
		"", "simple-builder",
	)

	if err != nil {
		return nil, err
	}

	desc.WorkDir = tmp

	log.Printf("[build] building ...")
	log.Printf("[build] Git URL: %s", desc.GitUrl)
	log.Printf("[build] Build script:\n%s", desc.BuildScript)

	wg.Add(1)

	b := build.NewBuild(
		ctx, desc.BuildDescriptor,
	)

	return b, nil
}

func waitBuildDone(wg *sync.WaitGroup, b *build.Build) {
	defer wg.Done()

	<-b.Done()
}

func maybeNotifyCallbacks(b *build.Build, callbacks []string) error {
	if len(callbacks) == 0 {
		return nil
	}

	data, err := json.Marshal(b)
	if err != nil {
		return err
	}

	for _, cb := range callbacks {
		log.Printf("[build] notifying %s", cb)

		_, err := http.Post(
			cb,
			"application/json",
			bytes.NewBuffer(data),
		)

		if err != nil {
			return err
		}
	}

	return nil
}

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

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/squarescale/simple-builder/build"
)

func TestBuildHTTPWait(t *testing.T) {
	ctx := context.Background()
	tmp_dir, err := ioutil.TempDir("", "simple-builder-test-builds")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewBuildsHandler(ctx, tmp_dir, false)
	test_dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(tmp_dir)
		if err != nil {
			log.Print(err)
		}
	}()

	srv := httptest.NewServer(handler)
	defer srv.Close()

	data, err := json.Marshal(build.BuildDescriptor{
		BuildScript: "#!/bin/bash\nls main.go\nbasename \"$PWD\"\necho OK\nexit 0",
		GitUrl:      filepath.Dir(test_dir),
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(srv.URL+"/builds", "application/json", bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("POST /builds: unexpected status code %d", res.StatusCode)
	}
	loc := res.Header.Get("Location")
	if loc == "" {
		t.Errorf("POST /builds: unexpected Location header %s", loc)
	}

	res, err = http.Get(srv.URL + loc + "/wait")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("GET %s/wait: unexpected status code %d", loc, res.StatusCode)
	}

	var b build.Build
	json.NewDecoder(res.Body).Decode(&b)

	expected_output := "main.go\nsimple-builder\nOK\n"
	if !strings.Contains(string(b.Output), expected_output) {
		t.Errorf("Output unexpected: %s\n%s", expected_output, string(b.Output))
	}
	if len(b.Errors) > 0 {
		t.Errorf("Errors unexpected: %+v", b.Errors)
	}
}

func TestBuildHTTPCallback(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	tmp_dir, err := ioutil.TempDir("", "simple-builder-test-builds")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewBuildsHandler(ctx, tmp_dir, false)
	test_dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(tmp_dir)
		if err != nil {
			log.Print(err)
		}
	}()

	srv := httptest.NewServer(handler)
	defer srv.Close()

	doneChan := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/cb", func(w http.ResponseWriter, r *http.Request) {
		var b build.Build
		json.NewDecoder(r.Body).Decode(&b)

		expected_output := "main.go\nsimple-builder\nOK\n"
		if !strings.Contains(string(b.Output), expected_output) {
			t.Errorf("Output unexpected: %s\n%s", expected_output, string(b.Output))
		}
		if len(b.Errors) > 0 {
			t.Errorf("Errors unexpected: %+v", b.Errors)
		}

		close(doneChan)
	})
	callback_srv := httptest.NewServer(mux)
	defer callback_srv.Close()

	data, err := json.Marshal(map[string]interface{}{
		"build_script": "#!/bin/bash\nls main.go\nbasename \"$PWD\"\necho OK\nexit 0",
		"git_url":      filepath.Dir(test_dir),
		"callbacks":    []string{callback_srv.URL + "/cb"},
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(srv.URL+"/builds", "application/json", bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("POST /builds: unexpected status code %d", res.StatusCode)
	}
	loc := res.Header.Get("Location")
	if loc == "" {
		t.Errorf("POST /builds: unexpected Location header %s", loc)
	}

	select {
	case <-doneChan:
	case <-ctx.Done():
		t.Error("Timed out")
	}
}

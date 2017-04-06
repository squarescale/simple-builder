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

	"github.com/squarescale/simple-builder/build"
)

func TestBuildHTTPWait(t *testing.T) {
	ctx := context.Background()
	tmp_dir, err := ioutil.TempDir("", "simple-builder-test-builds")
	if err != nil {
		t.Fatal(err)
	}
	handler := BuildsHandler(ctx, tmp_dir)
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
		t.Errorf("POST /builds: unexpected Location header %#s", loc)
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
		t.Error("Output unexpected: %#s\n%s", expected_output, string(b.Output))
	}
	if len(b.Errors) > 0 {
		t.Error("Errors unexpected: %#v", b.Errors)
	}
}

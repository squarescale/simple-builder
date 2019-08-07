package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/squarescale/simple-builder/build"

	"github.com/stretchr/testify/suite"
)

type BuildsHandlerTestSuite struct {
	suite.Suite

	tmpDir string
	wd     string
}

func (s *BuildsHandlerTestSuite) SetupTest() {
	d, err := ioutil.TempDir(
		"", "simple-builder-test-builds",
	)

	s.Nil(err)

	s.tmpDir = d

	// ---

	wd, err := os.Getwd()
	s.Nil(err)

	s.wd = wd
}

func (s *BuildsHandlerTestSuite) TearDownTest() {
	err := os.RemoveAll(s.tmpDir)
	s.Nil(err)
}

func (s *BuildsHandlerTestSuite) TestBuildHTTPWait() {
	handler := NewBuildsHandler(
		context.Background(), s.tmpDir, false,
	)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	data, err := json.Marshal(
		build.BuildDescriptor{
			BuildScript: shellScript(
				[]string{
					"ls main.go",
					"basename \"$PWD\"",
					"sleep 1",
					"echo OK",
					"exit 0",
				},
			),
			GitUrl: filepath.Dir(s.wd),
		},
	)

	s.Nil(err)

	res, err := http.Post(
		srv.URL+"/builds",
		"application/json",
		bytes.NewBuffer(data),
	)

	s.Nil(err)

	s.Equalf(
		res.StatusCode,
		http.StatusOK,
		"POST /builds: unexpected status code %d",
		res.StatusCode,
	)

	loc := res.Header.Get("Location")

	s.NotEmptyf(
		loc,
		"POST /builds: unexpected Location header %s",
		loc,
	)

	res, err = http.Get(
		srv.URL + loc + "/wait",
	)

	s.Nil(err)

	s.Equalf(
		res.StatusCode,
		http.StatusOK,
		"GET %s/wait: unexpected status code %d",
		loc,
		res.StatusCode,
	)

	var b build.Build
	json.NewDecoder(res.Body).Decode(&b)

	s.Contains(
		string(b.Output),
		strings.Join(
			[]string{
				"main.go",
				"simple-builder",
				"OK",
			}, "\n",
		),
	)

	s.Empty(b.Errors)
}

func (s *BuildsHandlerTestSuite) TestBuildHTTPCallback() {
	ctx, _ := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	handler := NewBuildsHandler(
		ctx, s.tmpDir, false,
	)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	doneChan := make(chan struct{})

	mux := http.NewServeMux()

	mux.HandleFunc("/cb", func(w http.ResponseWriter, r *http.Request) {
		var b build.Build
		json.NewDecoder(r.Body).Decode(&b)
		s.Contains(
			string(b.Output),
			strings.Join(
				[]string{
					"main.go",
					"simple-builder",
					"OK",
				}, "\n",
			),
		)

		s.Empty(b.Errors)

		close(doneChan)
	})

	callback_srv := httptest.NewServer(mux)
	defer callback_srv.Close()

	data, err := json.Marshal(map[string]interface{}{
		"git_url": filepath.Dir(s.wd),
		"build_script": shellScript(
			[]string{
				"ls main.go",
				"basename \"$PWD\"",
				"echo OK",
				"exit 0",
			},
		),
		"callbacks": []string{
			callback_srv.URL + "/cb",
		},
	})

	s.Nil(err)

	res, err := http.Post(
		srv.URL+"/builds",
		"application/json",
		bytes.NewBuffer(data),
	)

	s.Nil(err)

	s.Equalf(
		res.StatusCode,
		http.StatusOK,
		"POST /builds: unexpected status code %d",
		res.StatusCode,
	)

	loc := res.Header.Get("Location")

	s.NotEmptyf(
		loc,
		"POST /builds: unexpected Location header %s",
		loc,
	)

	select {
	case <-doneChan:
	case <-ctx.Done():
		s.Fail("Timed out")
	}
}

func TestBuildsHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(BuildsHandlerTestSuite))
}

func shellScript(lines []string) string {
	return strings.Join(
		append(
			[]string{"#!/bin/bash"},
			lines...,
		),
		"\n",
	)
}

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"sync"

	"github.com/squarescale/simple-builder/build"
	"github.com/squarescale/simple-builder/util/token"
)

type buildsHandler struct {
	Lock      sync.Mutex
	Builds    map[string]*build.Build
	BuildsDir string
	ctx       context.Context
}

func (h *buildsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if (r.URL.Path == "/builds" || r.URL.Path == "/builds/") && r.Method == http.MethodPost {
		h.newBuild(w, r)
	} else if path.Dir(r.URL.Path) == "/builds" {
		build_id := path.Base(r.URL.Path)
		h.getBuild(build_id, w, r)
	} else {
		route := path.Base(r.URL.Path)
		dir := path.Dir(r.URL.Path)
		build_id := path.Base(dir)
		dir = path.Dir(dir)
		if dir == "/builds" && route == "wait" {
			h.waitBuild(build_id, w, r)
		} else if dir == "/builds" && route == "output" {
			h.getBuildOutput(build_id, w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func (h *buildsHandler) newBuild(w http.ResponseWriter, r *http.Request) {
	tk := token.GenSecure(16)

	work_dir, err := ioutil.TempDir(h.BuildsDir, "simple-builder")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var buildDescriptor struct {
		build.BuildDescriptor
		Callbacks []string `json:"callbacks"`
	}
	err = json.NewDecoder(r.Body).Decode(&buildDescriptor)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	buildDescriptor.WorkDir = work_dir
	b := build.NewBuild(h.ctx, buildDescriptor.BuildDescriptor)
	go h.waitBuildObject(tk, b, buildDescriptor.Callbacks)

	w.Header().Set("Location", "/builds/"+tk)

	json.NewEncoder(w).Encode(b)
	h.setBuildObject(tk, b)
}

func (h *buildsHandler) waitBuildObject(tk string, build *build.Build, callbacks []string) {
	<-build.Done()

	data, err := json.Marshal(build)
	if err != nil {
		for _, cb := range callbacks {
			res, err := http.Post(cb, "application/json", bytes.NewBuffer(data))
			_ = err
			_ = res
		}
	}

	h.deleteBuildObject(tk)
}

func (h *buildsHandler) setBuildObject(tk string, build *build.Build) {
	h.Lock.Lock()
	defer h.Lock.Unlock()
	h.Builds[tk] = build
}

func (h *buildsHandler) getBuildObject(tk string) *build.Build {
	h.Lock.Lock()
	defer h.Lock.Unlock()
	return h.Builds[tk]
}

func (h *buildsHandler) deleteBuildObject(tk string) *build.Build {
	h.Lock.Lock()
	defer h.Lock.Unlock()
	res := h.Builds[tk]
	delete(h.Builds, tk)
	return res
}

func (h *buildsHandler) getBuild(id string, w http.ResponseWriter, r *http.Request) {
	b := h.getBuildObject(id)
	if b == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(b)
}

func (h *buildsHandler) waitBuild(id string, w http.ResponseWriter, r *http.Request) {
	b := h.getBuildObject(id)
	if b == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	<-b.Done()
	json.NewEncoder(w).Encode(b)
}

func (h *buildsHandler) getBuildOutput(id string, w http.ResponseWriter, r *http.Request) {
	b := h.getBuildObject(id)
	if b == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	http.ServeFile(w, r, b.OutputFileName())
}

func BuildsHandler(ctx context.Context, builds_dir string) http.Handler {
	return &buildsHandler{
		ctx:       ctx,
		BuildsDir: builds_dir,
		Builds:    make(map[string]*build.Build),
	}
}

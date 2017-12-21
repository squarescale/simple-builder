package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"sync"

	"github.com/squarescale/simple-builder/build"
	"github.com/squarescale/simple-builder/util/token"
)

type BuildDescriptorWithCallback struct {
	build.BuildDescriptor
	Callbacks []string `json:"callbacks"`
}

type BuildsHandler interface {
	http.Handler
	CreateBuild(wg *sync.WaitGroup, descr BuildDescriptorWithCallback) (b *build.Build, tk string, err error)
}

type buildsHandler struct {
	SingleBuild bool
	Lock        sync.Mutex
	Builds      map[string]*build.Build
	BuildsDir   string
	ctx         context.Context
}

func (h *buildsHandler) uniqueBuildId() string {
	if !h.SingleBuild || len(h.Builds) > 1 {
		panic("more than one build in single build mode")
	}
	for k := range h.Builds {
		return k
	}
	return ""
}

func (h *buildsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.SingleBuild && path.Dir(r.URL.Path) == "/build" {
		build_id := h.uniqueBuildId()
		route := path.Base(r.URL.Path)
		if build_id == "" {
			w.WriteHeader(http.StatusNotFound)
		} else if route == "wait" {
			h.waitBuild(build_id, w, r)
		} else if route == "output" {
			h.getBuildOutput(build_id, w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	} else if (r.URL.Path == "/builds" || r.URL.Path == "/builds/") && r.Method == http.MethodPost {
		if h.SingleBuild {
			w.WriteHeader(http.StatusNotFound)
		} else {
			h.newBuild(w, r)
		}
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
	var buildDescriptor BuildDescriptorWithCallback
	err := json.NewDecoder(r.Body).Decode(&buildDescriptor)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	b, tk, err := h.CreateBuild(new(sync.WaitGroup), buildDescriptor)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Location", "/builds/"+tk)
	json.NewEncoder(w).Encode(b)
}

func (h *buildsHandler) CreateBuild(wg *sync.WaitGroup, descr BuildDescriptorWithCallback) (b *build.Build, tk string, err error) {
	wg.Add(1)
	tk = token.GenSecure(16)
	work_dir, err := ioutil.TempDir(h.BuildsDir, "simple-builder")
	if err != nil {
		return nil, tk, err
	}

	descr.WorkDir = work_dir
	descr.Token = tk
	log.Printf("[build %s] start", tk)
	log.Printf("[build %s] Git URL: %s", tk, descr.GitUrl)
	log.Printf("[build %s] Build script:\n%s", tk, descr.BuildScript)
	b = build.NewBuild(h.ctx, descr.BuildDescriptor)
	go h.waitBuildObject(wg, tk, b, descr.Callbacks)

	h.setBuildObject(tk, b)
	return b, tk, nil
}

func (h *buildsHandler) waitBuildObject(wg *sync.WaitGroup, tk string, build *build.Build, callbacks []string) {
	defer wg.Done()
	<-build.Done()

	if len(callbacks) > 0 {
		data, err := json.Marshal(build)
		if err != nil {
			log.Printf("[build %s] callback serialize error: %s", tk, err.Error())
		} else {
			for _, cb := range callbacks {
				log.Printf("[build %s] call %s", tk, cb)
				res, err := http.Post(cb, "application/json", bytes.NewBuffer(data))
				if err != nil {
					log.Printf("[build %s] callback error: %s", tk, err.Error())
				}
				_ = res
			}
		}
	}

	log.Printf("[build %s] done", tk)
	b := h.deleteBuildObject(tk)
	if len(b.Errors) == 0 {
		log.Printf("[build %s] Success", tk)
	} else {
		for _, e := range b.Errors {
			log.Printf("[build %s] Error: %v", tk, e)
		}
	}
	log.Printf("[build %s] Output:\n%s", tk, b.Output)
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

func NewBuildsHandler(ctx context.Context, builds_dir string, singleBuild bool) BuildsHandler {
	return &buildsHandler{
		SingleBuild: singleBuild,
		ctx:         ctx,
		BuildsDir:   builds_dir,
		Builds:      make(map[string]*build.Build),
	}
}

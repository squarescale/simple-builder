package scriptrunner

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

var (
	tmpDir string
)

func TestScriptRunner(t *testing.T) {
	testFuncs := map[string]func(*testing.T){
		"write build file": testWriteBuildFile,
		"run success":      testRunSuccess,
	}

	for desc, f := range testFuncs {
		setUp(t)
		t.Run(desc, f)
		tearDown(t)
	}
}

func setUp(t *testing.T) {
	d, err := ioutil.TempDir(
		"", "scriptrunnertestsuite",
	)

	require.Nil(t, err)

	tmpDir = d
}

func tearDown(t *testing.T) {
	err := os.RemoveAll(tmpDir)
	require.Nil(t, err)
}

func testWriteBuildFile(t *testing.T) {
	buildFile := filepath.Join(tmpDir, "build")
	ensureDoesNotExist(t, buildFile)

	r := New(
		context.TODO(),
		&Config{
			ScriptContents: "plop",
			ScriptFile:     buildFile,
		},
	)

	err := r.writeBuildFile()
	require.Nil(t, err)
	require.NotNil(t, r.Cfg.ScriptFile)

	require.Equal(t, r.Cfg.ScriptFile, buildFile)
	require.FileExists(t, r.Cfg.ScriptFile)

	info, err := os.Stat(r.Cfg.ScriptFile)
	require.Nil(t, err)
	require.Equal(t, info.Mode(), os.FileMode(0700))

	buff, err := ioutil.ReadFile(r.Cfg.ScriptFile)
	require.Nil(t, err)
	require.Equal(t, buff, []byte("plop"))
}

func testRunSuccess(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(
		context.Background(),
	)
	defer cancelFunc()

	logFile, err := os.OpenFile(
		filepath.Join(tmpDir, "all.log"),
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0600,
	)

	require.Nil(t, err)
	defer logFile.Close()

	c := New(ctx, &Config{
		ScriptContents: "#!/bin/bash\nls\nbasename \"$PWD\"\necho OK\nexit 0",
		ScriptFile:     filepath.Join(tmpDir, "build"),

		WorkDir:  tmpDir,
		Logger:   zerolog.New(logFile).With().Timestamp().Logger(),
		ExtraEnv: extraEnv(),
	})

	err = c.Run()
	require.Nil(t, err)

	info, err := os.Stat(logFile.Name())
	require.Nil(t, err)
	require.True(t, info.Size() >= 700)
}

func ensureDoesNotExist(t *testing.T, path string) {
	_, err := os.Stat(path)
	require.NotNil(t, err)
	require.True(t, os.IsNotExist(err))
}

func extraEnv() []string {
	env := map[string]string{
		"HOME":  tmpDir,
		"PATH":  os.Getenv("PATH"),
		"SHELL": os.Getenv("SHELL"),
		"USER":  os.Getenv("USER"),
	}

	// ---

	buff := []string{}

	for k, v := range env {
		buff = append(
			buff, fmt.Sprintf("%s=%s", k, v),
		)
	}

	return buff
}

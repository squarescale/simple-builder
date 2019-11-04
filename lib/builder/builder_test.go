package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFullBuild(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(
		context.Background(),
	)

	defer cancelFunc()

	b, err := New(
		ctx, "testdata/testfullbuild.json",
	)
	require.Nil(t, err)

	runPrechecks(t, b)

	defer b.Cleanup()

	prvKey, err := ioutil.ReadFile(
		"testdata/id",
	)
	require.Nil(t, err)

	b.cloner.Cfg.SSHKeyContents = string(prvKey)
	b.cloner.Cfg.ExtraEnv = append(
		b.cloner.Cfg.ExtraEnv,
		fmt.Sprintf(
			"SSH_AUTH_SOCK=%s", os.Getenv("SSH_AUTH_SOCK"),
		),
	)

	err = b.Run()
	require.Nil(t, err)

	runGitChecks(t, b)

	require.Empty(t, b.Errors)
	require.Nil(t, b.ProcessState)
	require.NotEmpty(t, b.Output)

	runScriptChecks(t, b)
}

func runPrechecks(t *testing.T, b *Builder) {
	require.NotNil(t, b)

	require.Empty(t, b.Output)
	require.Empty(t, b.Errors)
	require.Nil(t, b.ProcessState)

	require.NotNil(t, b.logger)

	isAbsolutePath(t, b.workDir)
	require.DirExists(t, b.workDir)

	checkLogFile(t, b)

	require.True(t, b.cloner.Cfg.Recursive)

	isAbsolutePath(t, b.cloner.Cfg.SSHKeyDir)

	isAbsolutePath(t, b.cloner.Cfg.CheckoutDir)
	ensureDoesNotExist(t, b.cloner.Cfg.CheckoutDir)

	isAbsolutePath(t, b.runner.Cfg.WorkDir)

	require.Equal(t,
		b.runner.Cfg.WorkDir,
		b.cloner.Cfg.CheckoutDir,
	)
}

func runGitChecks(t *testing.T, b *Builder) {
	kd := b.cloner.Cfg.SSHKeyDir

	require.DirExists(t, kd)
	require.FileExists(t,
		filepath.Join(
			kd, b.cloner.Cfg.SSHKeyFile,
		),
	)

	// ---

	cd := b.cloner.Cfg.CheckoutDir

	require.DirExists(t, cd)

	require.FileExists(t,
		filepath.Join(cd, "README.md"),
	)
}

func runScriptChecks(t *testing.T, b *Builder) {
	require.FileExists(t,
		b.runner.Cfg.ScriptFile,
	)

	messages := []string{
		fmt.Sprintf(
			"Cloning into '%s'...",
			b.cloner.Cfg.CheckoutDir,
		),

		"main.go",

		fmt.Sprintf(
			"PWD: %s",
			b.runner.Cfg.WorkDir,
		),
	}

	for _, m := range messages {
		checkOutputContains(t, b, m)
	}
}

func checkLogFile(t *testing.T, b *Builder) {
	require.FileExists(t, b.logFile.Name())

	info, err := os.Stat(b.logFile.Name())
	require.Nil(t, err)
	require.Equal(t, info.Mode(), os.FileMode(0600))
}

func isAbsolutePath(t *testing.T, p string) {
	require.True(t,
		path.IsAbs(p),
	)
}

func ensureDoesNotExist(t *testing.T, path string) {
	_, err := os.Stat(path)
	require.NotNil(t, err)
	require.True(t, os.IsNotExist(err))
}

func checkOutputContains(t *testing.T, b *Builder, msg string) {
	require.NotEmpty(t, b.Output)

	type Line struct {
		Time    string `json:"-"`
		Message string `json:"message"`
	}

	found := false

	for _, logLine := range strings.Split(b.Output, "\n") {
		if len(logLine) == 0 {
			continue
		}

		l := new(Line)

		err := json.Unmarshal(
			[]byte(logLine), &l,
		)

		require.Nil(t, err)

		if l.Message == msg {
			found = true
			break
		}
	}

	require.True(
		t,
		found,
		fmt.Sprintf("Expected to find %q", msg),
	)
}

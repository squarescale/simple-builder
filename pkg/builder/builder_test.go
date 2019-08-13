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

	"github.com/stretchr/testify/suite"
)

type BuilderTestSuite struct {
	suite.Suite
}

func (s *BuilderTestSuite) TestFullBuild() {
	ctx, cancelFunc := context.WithCancel(
		context.Background(),
	)

	defer cancelFunc()

	b, err := New(
		ctx, "testdata/testfullbuild.json",
	)
	s.Nil(err)

	s.runPrechecks(b)

	fmt.Println(b.workDir)
	//defer func() {
	//	err := os.RemoveAll(b.workDir)
	//	s.Nil(err)
	//}()

	prvKey, err := ioutil.ReadFile(
		"testdata/id",
	)
	s.Nil(err)

	b.cloner.Cfg.SSHKeyContents = string(prvKey)

	err = b.Run()
	s.Nil(err)

	s.runGitChecks(b)

	s.Empty(b.Errors)
	s.Nil(b.ProcessState)
	s.NotEmpty(b.Output)

	s.runScriptChecks(b)
}

func (s *BuilderTestSuite) runPrechecks(b *Builder) {
	s.NotNil(b)

	s.Empty(b.Output)
	s.Empty(b.Errors)
	s.Nil(b.ProcessState)

	s.NotNil(b.logger)

	s.isAbsolutePath(b.workDir)
	s.DirExists(b.workDir)

	s.checkLogFile(b)

	s.True(b.cloner.Cfg.Recursive)

	s.isAbsolutePath(b.cloner.Cfg.SSHKeyDir)

	s.isAbsolutePath(b.cloner.Cfg.CheckoutDir)
	s.ensureDoesNotExist(b.cloner.Cfg.CheckoutDir)

	s.isAbsolutePath(b.runner.Cfg.WorkDir)

	s.Equal(
		b.runner.Cfg.WorkDir,
		b.cloner.Cfg.CheckoutDir,
	)
}

func (s *BuilderTestSuite) runGitChecks(b *Builder) {
	kd := b.cloner.Cfg.SSHKeyDir

	s.DirExists(kd)
	s.FileExists(
		filepath.Join(
			kd, b.cloner.Cfg.SSHKeyFile,
		),
	)

	// ---

	cd := b.cloner.Cfg.CheckoutDir

	s.DirExists(cd)

	s.FileExists(
		filepath.Join(cd, "README.md"),
	)
}

func (s *BuilderTestSuite) runScriptChecks(b *Builder) {
	s.FileExists(
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
		s.checkOutputContains(b, m)
	}
}

func (s *BuilderTestSuite) checkLogFile(b *Builder) {
	s.FileExists(b.logFile.Name())

	info, err := os.Stat(b.logFile.Name())
	s.Nil(err)
	s.Equal(info.Mode(), os.FileMode(0600))
}

func (s *BuilderTestSuite) isAbsolutePath(p string) {
	s.True(
		path.IsAbs(p),
	)
}

func (s *BuilderTestSuite) ensureDoesNotExist(path string) {
	_, err := os.Stat(path)
	s.NotNil(err)
	s.True(os.IsNotExist(err))
}

func (s *BuilderTestSuite) checkOutputContains(b *Builder, msg string) {
	s.NotEmpty(b.Output)

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

		s.Nil(err)

		if l.Message == msg {
			found = true
			break
		}
	}

	s.True(
		found,
		fmt.Sprintf("Expected to find %q", msg),
	)
}

func TestBuilderTestSuite(t *testing.T) {
	suite.Run(t, new(BuilderTestSuite))
}

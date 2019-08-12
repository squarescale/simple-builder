package scriptrunner

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
)

type ScriptRunnerTestSuite struct {
	suite.Suite

	tmpDir string
}

func (s *ScriptRunnerTestSuite) SetupTest() {
	d, err := ioutil.TempDir(
		"", "scriptrunnertestsuite",
	)

	s.Nil(err)

	s.tmpDir = d
}

func (s *ScriptRunnerTestSuite) TearDownTest() {
	err := os.RemoveAll(s.tmpDir)
	s.Nil(err)
}

func (s *ScriptRunnerTestSuite) TestWriteBuildFile() {
	buildFile := filepath.Join(s.tmpDir, "build")
	s.EnsureDoesNotExist(buildFile)

	r := New(
		context.TODO(),
		&Config{
			Script:  "plop",
			WorkDir: s.tmpDir,
		},
	)

	bf, err := r.writeBuildFile()
	s.Nil(err)
	s.NotNil(bf)

	s.Equal(buildFile, bf)
	s.FileExists(bf)

	info, err := os.Stat(bf)
	s.Nil(err)
	s.Equal(info.Mode(), os.FileMode(0700))

	buff, err := ioutil.ReadFile(bf)
	s.Nil(err)
	s.Equal(buff, []byte("plop"))
}

func (s *ScriptRunnerTestSuite) EnsureDoesNotExist(path string) {
	_, err := os.Stat(path)
	s.NotNil(err)
	s.True(os.IsNotExist(err))
}

func (s *ScriptRunnerTestSuite) TestRunSuccess() {
	ctx, cancelFunc := context.WithCancel(
		context.Background(),
	)
	defer cancelFunc()

	logFile, err := os.OpenFile(
		filepath.Join(s.tmpDir, "all.log"),
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0600,
	)

	s.Nil(err)
	defer logFile.Close()

	c := New(ctx, &Config{
		Script: "#!/bin/bash\nls\nbasename \"$PWD\"\necho OK\nexit 0",

		WorkDir:  s.tmpDir,
		Logger:   zerolog.New(logFile).With().Timestamp().Logger(),
		ExtraEnv: s.extraEnv(),
	})

	err = c.Run()
	s.Nil(err)

	info, err := os.Stat(logFile.Name())
	s.Nil(err)
	s.True(info.Size() >= 700)
}

func (s *ScriptRunnerTestSuite) extraEnv() []string {
	env := map[string]string{
		"HOME":  s.tmpDir,
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

func TestScriptRunnerTestSuite(t *testing.T) {
	suite.Run(t, new(ScriptRunnerTestSuite))
}

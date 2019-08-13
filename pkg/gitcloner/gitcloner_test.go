package gitcloner

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

/*
	./testdata/id.pub must be installed as a deploy key here:

	- https://github.com/squarescale/simple-builder/settings/keys
*/

type GitClonerTestSuite struct {
	suite.Suite

	tmpDir string
}

func (s *GitClonerTestSuite) SetupTest() {
	d, err := ioutil.TempDir(
		"", "gitclonertestsuite",
	)

	s.Nil(err)

	s.tmpDir = d
}

func (s *GitClonerTestSuite) TearDownTest() {
	err := os.RemoveAll(s.tmpDir)
	s.Nil(err)
}

func (s *GitClonerTestSuite) TestWriteSSHSecretKey() {
	c := New(
		context.TODO(),
		&Config{
			SSHKeyContents: "aa",
		},
	)

	err := c.writeSSHSecretKey()
	s.NotNil(err)

	// ----

	sshDir := filepath.Join(s.tmpDir, ".ssh")
	s.EnsureDoesNotExist(sshDir)

	keyFile := filepath.Join(sshDir, "id")
	s.EnsureDoesNotExist(keyFile)

	c = New(
		context.TODO(),
		&Config{
			SSHKeyDir:      sshDir,
			SSHKeyFile:     "id",
			SSHKeyContents: "plop",
		},
	)

	err = c.writeSSHSecretKey()

	s.Nil(err)

	s.DirExists(sshDir)
	s.FileExists(keyFile)

	info, err := os.Stat(keyFile)
	s.Nil(err)
	s.Equal(info.Mode(), os.FileMode(0600))

	buff, err := ioutil.ReadFile(keyFile)
	s.Nil(err)
	s.Equal(buff, []byte("plop"))
}

func (s *GitClonerTestSuite) TestCmdArgs() {
	cfg := &Config{
		RepoURL:     "repo.url",
		CheckoutDir: "checkout/dir",
	}

	c := New(
		context.TODO(), cfg,
	)

	s.Equal(
		c.cmdArgs(),
		[]string{
			"clone",
			"--depth", "1",
			cfg.RepoURL,
			cfg.CheckoutDir,
		},
	)

	// ----

	cfg.FullClone = true
	cfg.Recursive = true
	cfg.Branch = "dummy"

	c.Cfg = cfg

	s.Equal(
		c.cmdArgs(),
		[]string{
			"clone",
			"--recursive",
			"-b", "dummy",
			cfg.RepoURL,
			cfg.CheckoutDir,
		},
	)
}

func (s *GitClonerTestSuite) EnsureDoesNotExist(path string) {
	_, err := os.Stat(path)
	s.NotNil(err)
	s.True(os.IsNotExist(err))
}

func (s *GitClonerTestSuite) TestRunSuccess() {
	ctx, cancelFunc := context.WithCancel(
		context.Background(),
	)
	defer cancelFunc()

	secretDeployKey, err := ioutil.ReadFile(
		"testdata/id",
	)
	s.Nil(err)
	s.NotNil(secretDeployKey)

	logFile, err := os.OpenFile(
		filepath.Join(s.tmpDir, "all.log"),
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0600,
	)

	s.Nil(err)
	defer logFile.Close()

	c := New(ctx, &Config{
		RepoURL: "git@github.com:squarescale/simple-builder.git",

		SSHKeyContents: string(secretDeployKey),
		SSHKeyFile:     "id",
		SSHKeyDir:      filepath.Join(s.tmpDir, ".ssh"),

		FullClone: true,
		Recursive: true,

		WorkDir:  s.tmpDir,
		Logger:   zerolog.New(logFile).With().Timestamp().Logger(),
		ExtraEnv: s.extraEnv(),
	})

	err = c.Run()
	s.Nil(err)

	info, err := os.Stat(logFile.Name())
	s.Nil(err)
	s.True(info.Size() >= 4000)
}

func (s *GitClonerTestSuite) extraEnv() []string {
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

func TestGitClonerTestSuite(t *testing.T) {
	suite.Run(t, new(GitClonerTestSuite))
}

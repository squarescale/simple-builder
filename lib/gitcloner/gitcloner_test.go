package gitcloner

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

/*
	./testdata/id.pub must be installed as a deploy key here:

	- https://github.com/squarescale/simple-builder/settings/keys
*/

/*
type GitClonerTestSuite struct {
	suite.Suite

	tmpDir string
}

func (s *GitClonerTestSuite) SetupTest() {
	d, err := ioutil.TempDir(
		"", "gitclonertestsuite",
	)

	require.Nil(t,err)

	s.tmpDir = d
}

func (s *GitClonerTestSuite) TearDownTest() {
	err := os.RemoveAll(s.tmpDir)
	require.Nil(t,err)
}

func (s *GitClonerTestSuite) TestWriteSSHSecretKey() {
	c := New(
		context.TODO(),
		&Config{
			SSHKeyContents: "aa",
		},
	)

	err := c.writeSSHSecretKey()
	require.NotNil(t,err)

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

	require.Nil(t,err)

	require.DirExists(t,sshDir)
	require.FileExists(t,keyFile)

	info, err := os.Stat(keyFile)
	require.Nil(t,err)
	require.Equal(t,info.Mode(), os.FileMode(0600))

	buff, err := ioutil.ReadFile(keyFile)
	require.Nil(t,err)
	require.Equal(t,buff, []byte("plop"))
}

func (s *GitClonerTestSuite) TestCmdArgs() {
	cfg := &Config{
		RepoURL:     "repo.url",
		CheckoutDir: "checkout/dir",
	}

	c := New(
		context.TODO(), cfg,
	)

	require.Equal(t,
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

	require.Equal(t,
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
	require.NotNil(t,err)
	require.True(t,os.IsNotExist(err))
}

func (s *GitClonerTestSuite) TestRunSuccess() {
	ctx, cancelFunc := context.WithCancel(
		context.Background(),
	)
	defer cancelFunc()

	secretDeployKey, err := ioutil.ReadFile(
		"testdata/id",
	)
	require.Nil(t,err)
	require.NotNil(t,secretDeployKey)

	logFile, err := os.OpenFile(
		filepath.Join(s.tmpDir, "all.log"),
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0600,
	)

	require.Nil(t,err)
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
	require.Nil(t,err)

	info, err := os.Stat(logFile.Name())
	require.Nil(t,err)
	require.True(t,info.Size() >= 4000)
}

func (s *GitClonerTestSuite) extraEnv() []string {
	env := map[string]string{
		"HOME":          s.tmpDir,
		"PATH":          os.Getenv("PATH"),
		"SHELL":         os.Getenv("SHELL"),
		"USER":          os.Getenv("USER"),
		"SSH_AUTH_SOCK": os.Getenv("SSH_AUTH_SOCK"),
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
*/

// ----

var (
	tmpDir string
)

func TestMain(m *testing.M) {
	os.Exit(
		m.Run(),
	)
}

func TestGitCloner(t *testing.T) {
	testFuncs := map[string]func(*testing.T){
		"write ssh secret key": testWriteSSHSecretKey,
		"cmd args":             testCmdArgs,
		"run success":          testRunSuccess,
	}

	for desc, f := range testFuncs {
		setUp(t)
		t.Run(desc, f)
		tearDown(t)
	}
}

func setUp(t *testing.T) {
	d, err := ioutil.TempDir(
		"", "gitclonertestsuite",
	)

	require.Nil(t, err)

	tmpDir = d
}

func tearDown(t *testing.T) {
	err := os.RemoveAll(tmpDir)
	require.Nil(t, err)
}

func testWriteSSHSecretKey(t *testing.T) {
	c := New(
		context.TODO(),
		&Config{
			SSHKeyContents: "aa",
		},
	)

	err := c.writeSSHSecretKey()
	require.NotNil(t, err)

	// ----

	sshDir := filepath.Join(tmpDir, ".ssh")
	ensureDoesNotExist(t, sshDir)

	keyFile := filepath.Join(sshDir, "id")
	ensureDoesNotExist(t, keyFile)

	c = New(
		context.TODO(),
		&Config{
			SSHKeyDir:      sshDir,
			SSHKeyFile:     "id",
			SSHKeyContents: "plop",
		},
	)

	err = c.writeSSHSecretKey()

	require.Nil(t, err)

	require.DirExists(t, sshDir)
	require.FileExists(t, keyFile)

	info, err := os.Stat(keyFile)
	require.Nil(t, err)
	require.Equal(t, info.Mode(), os.FileMode(0600))

	buff, err := ioutil.ReadFile(keyFile)
	require.Nil(t, err)
	require.Equal(t, buff, []byte("plop"))
}

func testCmdArgs(t *testing.T) {
	cfg := &Config{
		RepoURL:     "repo.url",
		CheckoutDir: "checkout/dir",
	}

	c := New(
		context.TODO(), cfg,
	)

	require.Equal(t,
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

	require.Equal(t,
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

func testRunSuccess(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(
		context.Background(),
	)
	defer cancelFunc()

	secretDeployKey, err := ioutil.ReadFile(
		"testdata/id",
	)
	require.Nil(t, err)
	require.NotNil(t, secretDeployKey)

	logFile, err := os.OpenFile(
		filepath.Join(tmpDir, "all.log"),
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0600,
	)

	require.Nil(t, err)
	defer logFile.Close()

	c := New(ctx, &Config{
		RepoURL: "git@github.com:squarescale/simple-builder.git",

		SSHKeyContents: string(secretDeployKey),
		SSHKeyFile:     "id",
		SSHKeyDir:      filepath.Join(tmpDir, ".ssh"),

		FullClone: true,
		Recursive: true,

		WorkDir:  tmpDir,
		Logger:   zerolog.New(logFile).With().Timestamp().Logger(),
		ExtraEnv: extraEnv(),
	})

	err = c.Run()
	require.Nil(t, err)

	info, err := os.Stat(logFile.Name())
	require.Nil(t, err)
	require.True(t, info.Size() >= 4000)
}

func ensureDoesNotExist(t *testing.T, path string) {
	_, err := os.Stat(path)
	require.NotNil(t, err)
	require.True(t, os.IsNotExist(err))
}

func extraEnv() []string {
	env := map[string]string{
		"HOME":          tmpDir,
		"PATH":          os.Getenv("PATH"),
		"SHELL":         os.Getenv("SHELL"),
		"USER":          os.Getenv("USER"),
		"SSH_AUTH_SOCK": os.Getenv("SSH_AUTH_SOCK"),
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

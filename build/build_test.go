package build

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

const SECRET_DEPLOY_KEY = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBdAHOwhOxFF3/kjC1JET9dWPdvh8PVt+gJ9ckmEXJlAQAAAJhtJ3SzbSd0
swAAAAtzc2gtZWQyNTUxOQAAACBdAHOwhOxFF3/kjC1JET9dWPdvh8PVt+gJ9ckmEXJlAQ
AAAECEgoZ07SQ9CJ5AaB2rtMXI08sMvtMxm9gJMzFfvWf3pl0Ac7CE7EUXf+SMLUkRP11Y
92+Hw9W36An1ySYRcmUBAAAAEG1pbGRyZWRAbW9pcmFpbmUBAgMEBQ==
-----END OPENSSH PRIVATE KEY-----
`

// must be installed as a deploy key here:
// https://github.com/squarescale/simple-builder/settings/keys
const PUBLIC_DEPLOY_KEY = `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIF0Ac7CE7EUXf+SMLUkRP11Y92+Hw9W36An1ySYRcmUB`

type BuildTestSuite struct {
	suite.Suite

	tmpDir string
}

func (s *BuildTestSuite) SetupTest() {
	d, err := ioutil.TempDir(
		"", "simple-builder-test-builds",
	)

	s.Nil(err)

	s.tmpDir = d
}

func (s *BuildTestSuite) TearDownTest() {
	err := os.RemoveAll(s.tmpDir)
	s.Nil(err)
}

func (s *BuildTestSuite) TestFullBuild() {
	ctx := context.Background()

	b := NewBuild(
		ctx,
		BuildDescriptor{
			WorkDir:      s.tmpDir,
			BuildScript:  "#!/bin/bash\nls main.go\nbasename \"$PWD\"\necho OK\nexit 0",
			GitUrl:       "git@github.com:squarescale/simple-builder.git",
			GitSecretKey: SECRET_DEPLOY_KEY,
		},
	)

	<-b.Done()

	log.Printf("Output:\n%s\n", string(b.Output))

	log.Printf("Build: %#v", *b)
	log.Printf("Process State: %#v", *b.ProcessState)
	log.Println()
	log.Printf("Actual Output:   %#v", string(b.Output))

	expectedOutput := "main.go\nsimple-builder\nOK\n"

	log.Printf("Expected Output: %#v", expectedOutput)

	s.Empty(b.Errors)

	s.Contains(
		string(b.Output),
		expectedOutput,
	)
}

func (s *BuildTestSuite) TestWriteSSHKey() {
	sshDir := filepath.Join(s.tmpDir, ".ssh")
	s.EnsureDoesNotExist(sshDir)

	keyFile := filepath.Join(sshDir, "id")
	s.EnsureDoesNotExist(keyFile)

	err := writeSSHKey(
		sshDir, "id", "plop",
	)

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

func TestBuildTestSuite(t *testing.T) {
	suite.Run(t, new(BuildTestSuite))
}

func (s *BuildTestSuite) EnsureDoesNotExist(path string) {
	_, err := os.Stat(path)
	s.NotNil(err)
	s.True(os.IsNotExist(err))
}

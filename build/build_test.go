package build

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
)

var secret_deploy_key = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBdAHOwhOxFF3/kjC1JET9dWPdvh8PVt+gJ9ckmEXJlAQAAAJhtJ3SzbSd0
swAAAAtzc2gtZWQyNTUxOQAAACBdAHOwhOxFF3/kjC1JET9dWPdvh8PVt+gJ9ckmEXJlAQ
AAAECEgoZ07SQ9CJ5AaB2rtMXI08sMvtMxm9gJMzFfvWf3pl0Ac7CE7EUXf+SMLUkRP11Y
92+Hw9W36An1ySYRcmUBAAAAEG1pbGRyZWRAbW9pcmFpbmUBAgMEBQ==
-----END OPENSSH PRIVATE KEY-----
`

var public_deploy_key string = `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIF0Ac7CE7EUXf+SMLUkRP11Y92+Hw9W36An1ySYRcmUB`

func TestBuild(t *testing.T) {
	ctx := context.Background()
	work_dir, err := ioutil.TempDir("", "simple-builder-test-build")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(work_dir)
		if err != nil {
			log.Print(err)
		}
	}()

	b := NewBuild(ctx, BuildDescriptor{
		WorkDir:      work_dir,
		BuildScript:  "#!/bin/bash\nls main.go\nbasename \"$PWD\"\necho OK\nexit 0",
		GitUrl:       "git@github.com:squarescale/simple-builder.git",
		GitSecretKey: secret_deploy_key,
	})

	<-b.Done()
	expected_output := "main.go\nsimple-builder\nOK\n"

	log.Printf("Output:\n%s\n", string(b.Output))

	log.Printf("Build: %#v", *b)
	log.Printf("Process State: %#v", *b.ProcessState)
	log.Println()
	log.Printf("Actual Output:   %#v", string(b.Output))
	log.Printf("Expected Output: %#v", expected_output)

	if len(b.Errors) > 0 || !strings.Contains(string(b.Output), expected_output) {
		t.Fail()
	}
}

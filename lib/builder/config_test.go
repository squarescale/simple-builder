package builder

import (
	"testing"

	"github.com/squarescale/simple-builder/lib/gitcloner"
	"github.com/squarescale/simple-builder/lib/scriptrunner"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
}

func (s *ConfigTestSuite) TestNewConfigFromFile() {
	c, err := NewConfigFromFile("not.found")
	s.Nil(c)
	s.NotNil(err)

	// ---

	c, err = NewConfigFromFile("testdata/config.json")
	s.Nil(err)

	s.Equal(c, &Config{
		GitCloner: &gitcloner.Config{
			RepoURL:     "a",
			Branch:      "b",
			CheckoutDir: "c",

			SSHKeyContents: "d",

			FullClone: true,
			Recursive: true,
		},

		ScriptRunner: &scriptrunner.Config{
			ScriptContents: "foo",
		},

		Callbacks: []string{"cb1", "cb2"},
	})
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

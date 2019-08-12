package builder

import (
	"testing"

	"github.com/squarescale/simple-builder/pkg/gitcloner"
	"github.com/squarescale/simple-builder/pkg/scriptrunner"
	"github.com/stretchr/testify/suite"
)

type BuilderTestSuite struct {
	suite.Suite
}

func (s *BuilderTestSuite) TestNewConfigFromFile() {
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
			Script: "foo",
		},

		Callbacks: []string{"cb1", "cb2"},
	})
}

func TestBuilderTestSuite(t *testing.T) {
	suite.Run(t, new(BuilderTestSuite))
}

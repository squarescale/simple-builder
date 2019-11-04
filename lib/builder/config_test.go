package builder

import (
	"testing"

	"github.com/squarescale/simple-builder/lib/gitcloner"
	"github.com/squarescale/simple-builder/lib/scriptrunner"
	"github.com/stretchr/testify/require"
)

func testNewConfigFromFile(t *testing.T) {
	c, err := NewConfigFromFile("not.found")
	require.Nil(t, c)
	require.NotNil(t, err)

	// ---

	c, err = NewConfigFromFile("testdata/config.json")
	require.Nil(t, err)

	require.Equal(t, c, &Config{
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

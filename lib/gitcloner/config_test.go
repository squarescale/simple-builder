package gitcloner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func testSetCheckoutDir(t *testing.T) {
	testCases := []struct {
		desc                string
		c                   *Config
		expectedCheckoutDir string
	}{
		{
			desc: "checkout dir specified",
			c: &Config{
				CheckoutDir: "a",
			},
			expectedCheckoutDir: "a",
		},
		{
			desc: "checkout dir base on repo URL with trailing .git",
			c: &Config{
				WorkDir: "/b",
				RepoURL: "foo.git",
			},
			expectedCheckoutDir: "/b/foo",
		},
		{
			desc: "checkout dir base on repo URL no trailing .git",
			c: &Config{
				WorkDir: "/c",
				RepoURL: "bar",
			},
			expectedCheckoutDir: "/c/bar",
		},
	}

	for _, tc := range testCases {
		tc.c.setCheckoutDir()

		require.Equal(t,
			tc.expectedCheckoutDir,
			tc.c.CheckoutDir,
			tc.desc,
		)
	}
}

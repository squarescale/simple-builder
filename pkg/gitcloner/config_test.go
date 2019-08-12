package gitcloner

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
}

func (s *ConfigTestSuite) TestSetCheckoutDir() {
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

		s.Equal(
			tc.expectedCheckoutDir,
			tc.c.CheckoutDir,
			tc.desc,
		)
	}
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

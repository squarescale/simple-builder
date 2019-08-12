package gitcloner

import (
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
)

type Config struct {
	RepoURL     string `json:"git_url"`
	Branch      string `json:"git_branch"`
	CheckoutDir string `json:"git_checkout_dir"`

	SSHKeyContents string `json:"git_secret_key"`
	SSHKeyFile     string `json:"-"`
	SSHKeyDir      string `json:"-"`

	FullClone bool `json:"git_full_clone"`
	Recursive bool `json:"git_recursive"`

	WorkDir  string   `json:"-"`
	ExtraEnv []string `json:"-"`

	Logger zerolog.Logger `json:"-"`
}

func (c *Config) setCheckoutDir() {
	if c.CheckoutDir != "" {
		return
	}

	d := filepath.Base(c.RepoURL)

	if strings.HasSuffix(d, ".git") {
		d = d[:len(d)-4]
	}

	c.CheckoutDir = filepath.Join(
		c.WorkDir, d,
	)
}

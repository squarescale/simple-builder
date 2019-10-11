package gitcloner

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Cloner struct {
	Cfg          *Config
	ProcessState *os.ProcessState

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func New(ctx context.Context, cfg *Config) *Cloner {
	ctx2, cancelFunc := context.WithCancel(ctx)

	cfg.setCheckoutDir()

	return &Cloner{
		Cfg: cfg,

		ctx:        ctx2,
		cancelFunc: cancelFunc,
	}
}

func (c *Cloner) Run() error {
	err := c.ctx.Err()
	if err != nil {
		return err
	}

	err = c.writeSSHSecretKey()
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"git", c.cmdArgs()...,
	)

	cmd.Dir = c.Cfg.WorkDir

	cmd.Stdout = c.Cfg.Logger
	cmd.Stderr = c.Cfg.Logger

	cmd.Env = append(
		cmd.Env, c.gitSSHCommand(),
	)

	cmd.Env = append(
		cmd.Env, c.Cfg.ExtraEnv...,
	)

	c.dumpCmd(cmd)

	err = c.ctx.Err()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Wait()
		c.ProcessState = cmd.ProcessState
	}()

	select {
	case <-c.ctx.Done():
		cmd.Process.Kill()

		c.Cfg.Logger.Error().Msg(
			"\nContext expired, command killed\n\n",
		)

		return c.ctx.Err()

	case err := <-errChan:
		if err != nil {
			c.Cfg.Logger.Error().Msgf(
				"\nFailed: %s\n\n", err.Error(),
			)
		}

		return err
	}
}

// ----

func (c *Cloner) writeSSHSecretKey() error {
	if len(c.Cfg.SSHKeyContents) == 0 {
		return nil
	}

	if c.Cfg.SSHKeyDir == "" {
		return errors.New("SSHKeyDir not provided")
	}

	if c.Cfg.SSHKeyFile == "" {
		return errors.New("SSHKeyFile not provided")
	}

	// ---

	err := os.MkdirAll(
		c.Cfg.SSHKeyDir, 0700,
	)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(
		filepath.Join(
			c.Cfg.SSHKeyDir,
			c.Cfg.SSHKeyFile,
		),
		[]byte(c.Cfg.SSHKeyContents),
		0600,
	)
}

func (c *Cloner) cmdArgs() []string {
	args := []string{
		"clone",
	}

	cfg := c.Cfg

	if !cfg.FullClone {
		args = append(
			args, "--depth", "1",
		)
	}

	if cfg.Recursive {
		args = append(
			args, "--recursive",
		)
	}

	if cfg.Branch != "" {
		args = append(
			args, "-b", cfg.Branch,
		)
	}

	args = append(
		args,
		cfg.RepoURL,
		cfg.CheckoutDir,
	)

	return args
}

func (c *Cloner) gitSSHCommand() string {
	return fmt.Sprintf(
		"%s=%s",
		"GIT_SSH_COMMAND",
		strings.Join(
			[]string{
				"ssh",
				"-v",
				"-o StrictHostKeyChecking=no",
				"-o UserKnownHostsFile=/dev/null",
				fmt.Sprintf("-i %s/id", c.Cfg.SSHKeyDir),
			},
			" ",
		),
	)
}

func (c *Cloner) dumpCmd(cmd *exec.Cmd) {
	l := c.Cfg.Logger

	l.Info().Msgf("WD: %s", c.Cfg.WorkDir)
	l.Info().Msgf("ARGS: %q", cmd.Args)
	l.Info().Msgf("ENV: %q", cmd.Env)
}

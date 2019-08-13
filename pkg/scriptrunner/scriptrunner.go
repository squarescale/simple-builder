package scriptrunner

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
)

type Runner struct {
	ProcessState *os.ProcessState

	Cfg *Config

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func New(ctx context.Context, c *Config) *Runner {
	ctx2, cancelFunc := context.WithCancel(ctx)

	return &Runner{
		Cfg: c,

		ctx:        ctx2,
		cancelFunc: cancelFunc,
	}
}

func (r *Runner) Run() error {
	err := r.ctx.Err()
	if err != nil {
		return err
	}

	err = r.writeBuildFile()
	if err != nil {
		return err
	}

	cmd := exec.Command(
		r.Cfg.ScriptFile,
	)

	cmd.Dir = r.Cfg.WorkDir

	cmd.Stdout = r.Cfg.Logger
	cmd.Stderr = r.Cfg.Logger

	cmd.Env = append(
		cmd.Env, r.Cfg.ExtraEnv...,
	)

	r.dumpCmd(cmd)

	err = r.ctx.Err()
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
		r.ProcessState = cmd.ProcessState
	}()

	select {
	case <-r.ctx.Done():
		cmd.Process.Kill()

		r.Cfg.Logger.Error().Msg(
			"\nContext expired, command killed\n\n",
		)

		return r.ctx.Err()

	case err := <-errChan:
		if err != nil {
			r.Cfg.Logger.Error().Msgf(
				"\nFailed: %s\n\n", err.Error(),
			)
		}

		return err
	}
}

// ----

func (r *Runner) writeBuildFile() error {
	return ioutil.WriteFile(
		r.Cfg.ScriptFile,
		[]byte(r.Cfg.ScriptContents),
		0700,
	)
}

func (r *Runner) dumpCmd(cmd *exec.Cmd) {
	l := r.Cfg.Logger

	l.Info().Msgf("WD: %s", r.Cfg.WorkDir)
	l.Info().Msgf("ARGS: %q", cmd.Args)
	l.Info().Msgf("ENV: %q", cmd.Env)
}

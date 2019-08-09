package build

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/hpcloud/tail"
)

type BuildDescriptor struct {
	WorkDir        string `json:"-"`
	BuildScript    string `json:"build_script"`
	GitUrl         string `json:"git_url"`
	GitSecretKey   string `json:"git_secret_key"`
	GitBranch      string `json:"git_branch"`
	GitFullClone   bool   `json:"git_full_clone"`
	GitRecursive   bool   `json:"git_recursive"`
	GitCheckoutDir string `json:"git_checkout_dir"`
}

type Build struct {
	BuildDescriptor
	ProcessState *os.ProcessState `json:"process_state"`
	cancelFunc   context.CancelFunc
	Errors       []*BuildError `json:"errors"`
	Output       string        `json:"output"`
	done         chan struct{}
}

func NewBuild(ctx context.Context, descr BuildDescriptor) *Build {
	ctx2, cancelFunc := context.WithCancel(ctx)

	b := &Build{
		BuildDescriptor: descr,
		cancelFunc:      cancelFunc,
		done:            make(chan struct{}, 0),
	}

	b.maybeSetGitCheckoutDir()

	go b.run(ctx2)

	return b
}

func (b *Build) OutputFileName() string {
	return filepath.Join(b.WorkDir, "output.log")
}

func (b *Build) Cancel() {
	b.cancelFunc()
}

func (b *Build) Done() <-chan struct{} {
	return b.done
}

func (b *Build) run(ctx context.Context) {
	defer close(b.done)

	defer b.prepareBuildOutput()

	go b.tailBuildOutput(ctx)

	err := b.gitClone(ctx)
	if err != nil {
		b.Errors = append(b.Errors, &BuildError{err})
		return
	}

	err = b.runBuildScript(ctx)
	if err != nil {
		b.Errors = append(b.Errors, &BuildError{err})
		return
	}
}

func (b *Build) gitClone(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return err
	}

	err = b.maybeWriteGitSecretKey()
	if err != nil {
		return err
	}

	out, err := createOutputFile(
		b.OutputFileName(),
	)

	if err != nil {
		return err
	}

	defer out.Close()

	cmd := exec.Command(
		"git",
		b.gitCloneArgs(b.checkoutDir())...,
	)

	cmd.Dir = b.WorkDir

	cmd.Stdout = out
	cmd.Stderr = out

	cmd.Env = append(
		cmd.Env, b.gitSSHCommand(),
	)

	cmd.Env = append(
		cmd.Env, b.commonEnv()...,
	)

	err = logCommand(out, cmd)
	if err != nil {
		return err
	}

	err = ctx.Err()
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
		b.ProcessState = cmd.ProcessState
	}()

	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		fmt.Fprintf(out, "\nContext expired, command killed\n\n")
		return ctx.Err()
	case err := <-errChan:
		if err == nil {
			fmt.Fprintf(out, "\nSuccess\n\n")
		} else {
			fmt.Fprintf(out, "\nFailed: %s\n\n", err.Error())
		}
		return err
	}
}

func (b *Build) runBuildScript(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return err
	}

	buildScript, err := b.writeBuildScript()
	if err != nil {
		return err
	}

	out, err := createOutputFile(
		b.OutputFileName(),
	)

	if err != nil {
		return err
	}

	defer out.Close()

	cmd := exec.Command(buildScript)

	cmd.Dir = b.checkoutDir()

	cmd.Stdout = out
	cmd.Stderr = out

	cmd.Env = append(
		cmd.Env, b.commonEnv()...,
	)

	err = logCommand(out, cmd)
	if err != nil {
		return err
	}

	err = ctx.Err()
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
		b.ProcessState = cmd.ProcessState
	}()

	select {
	case <-ctx.Done():
		_ = cmd.Process.Signal(syscall.SIGTERM)
		fmt.Fprintf(out, "\nContext expired, command terminated\n\n")
		return ctx.Err()
	case err := <-errChan:
		if err == nil {
			fmt.Fprintf(out, "\nSuccess\n\n")
		} else {
			fmt.Fprintf(out, "\nFailed: %s\n\n", err.Error())
		}
		return err
	}
}

func (b *Build) gitCloneArgs(checkoutDir string) []string {
	args := []string{
		"clone",
	}

	if !b.GitFullClone {
		args = append(
			args, "--depth", "1",
		)
	}

	// WTF ?!?!
	if !b.GitRecursive {
		args = append(
			args, "--recursive",
		)
	}

	if b.GitBranch != "" {
		args = append(
			args, "-b", b.GitBranch,
		)
	}

	args = append(
		args, b.GitUrl, checkoutDir,
	)

	return args
}

func (b *Build) commonEnv() []string {
	env := map[string]string{
		"HOME": b.WorkDir,
	}

	keys := []string{
		"PATH", "SHELL", "USER", "LOGNAME",
	}

	for _, k := range keys {
		env[k] = os.Getenv(k)
	}

	// ---

	buff := []string{}

	for k, v := range env {
		buff = append(
			buff, fmt.Sprintf("%s=%s", k, v),
		)
	}

	return buff
}

func (b *Build) maybeWriteGitSecretKey() error {
	if b.GitSecretKey == "" {
		return nil
	}

	sshDir := filepath.Join(
		b.WorkDir, ".ssh",
	)

	return writeSSHKey(
		sshDir, "id", b.GitSecretKey,
	)
}

func (b *Build) maybeSetGitCheckoutDir() {
	if b.GitCheckoutDir != "" {
		return
	}

	b.GitCheckoutDir = filepath.Base(b.GitUrl)

	if strings.HasSuffix(b.GitCheckoutDir, ".git") {
		b.GitCheckoutDir = b.GitCheckoutDir[:len(b.GitCheckoutDir)-4]
	}
}

func (b *Build) gitSSHCommand() string {
	return fmt.Sprintf(
		"%s=%s",
		"GIT_SSH_COMMAND",
		strings.Join(
			[]string{
				"ssh",
				"-v",
				"-o StrictHostKeyChecking=no",
				"-o UserKnownHostsFile=/dev/null",
				"-i .ssh/id",
			},
			" ",
		),
	)
}

func (b *Build) checkoutDir() string {
	return filepath.Join(
		b.WorkDir, "workspace", b.GitCheckoutDir,
	)
}

func (b *Build) prepareBuildOutput() {
	out := b.OutputFileName()

	bytes, err := ioutil.ReadFile(out)

	if err != nil {
		b.Errors = append(
			b.Errors, &BuildError{err},
		)
	}

	b.Output = string(bytes)
}

func (b *Build) tailBuildOutput(ctx context.Context) {
	t, err := tail.TailFile(
		b.OutputFileName(),
		tail.Config{Follow: true},
	)

	if err != nil {
		log.Printf(
			"[build output error] %s",
			err.Error(),
		)

		return
	}

	tailCtx, tailStop := context.WithCancel(
		context.Background(),
	)

	go func() {
		select {
		case <-ctx.Done():
			t.Stop()
		case <-tailCtx.Done():
		}
	}()

	defer t.Cleanup()

	for line := range t.Lines {
		log.Printf(
			"[build output] %s",
			line.Text,
		)
	}

	tailStop()
}

func (b *Build) writeBuildScript() (string, error) {
	buildFile := filepath.Join(
		b.WorkDir, "build",
	)

	err := ioutil.WriteFile(
		buildFile,
		[]byte(b.BuildScript),
		0700,
	)

	return buildFile, err
}

// ----

func logCommand(w io.Writer, cmd *exec.Cmd) error {
	if cmd.Dir != "" {
		_, err := fmt.Fprintf(w, "cd %s\n", cmd.Dir)
		if err != nil {
			return err
		}
	}
	for _, env := range cmd.Env {
		_, err := fmt.Fprintf(w, "export %s\n", env)
		if err != nil {
			return err
		}
	}
	var args string
	for _, arg := range cmd.Args {
		if strings.Trim(arg, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890-_=+/.@:") != "" {
			args += fmt.Sprintf(" '%s'", strings.Replace(arg, "'", "'\"'\"'", -1))
		} else {
			args += fmt.Sprintf(" %s", arg)
		}
	}
	_, err := fmt.Fprintf(w, "exec%s\n\n", args)

	return err
}

func writeSSHKey(dir string, file string, contents string) error {
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(
		filepath.Join(
			dir, file,
		),
		[]byte(contents),
		0600,
	)
}

func createOutputFile(path string) (*os.File, error) {
	return os.OpenFile(
		path,
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0600,
	)
}

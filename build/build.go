package build

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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
	Errors       []error `json:"errors"`
	Output       string  `json:"output"`
	done         chan struct{}
}

func NewBuild(ctx context.Context, descr BuildDescriptor) *Build {
	ctx2, cancelFunc := context.WithCancel(ctx)
	b := &Build{
		BuildDescriptor: descr,
		cancelFunc:      cancelFunc,
		done:            make(chan struct{}, 0),
	}
	if b.GitCheckoutDir == "" {
		b.GitCheckoutDir = filepath.Base(b.GitUrl)
		if strings.HasSuffix(b.GitCheckoutDir, ".git") {
			b.GitCheckoutDir = b.GitCheckoutDir[:len(b.GitCheckoutDir)-4]
		}
	}
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

	var out_file = filepath.Join(b.WorkDir, "output.log")
	defer func() {
		var err error
		bytes, err := ioutil.ReadFile(out_file)
		if err != nil {
			b.Errors = append(b.Errors, err)
		}
		b.Output = string(bytes)
	}()

	err := b.gitClone(ctx)
	if err != nil {
		b.Errors = append(b.Errors, err)
		return
	}

	err = b.runBuildScript(ctx)
	if err != nil {
		b.Errors = append(b.Errors, err)
		return
	}
}

func (b *Build) gitClone(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var checkout_dir = filepath.Join(b.WorkDir, "workspace", b.GitCheckoutDir)
	var ssh_dir = filepath.Join(b.WorkDir, ".ssh")
	var ssh_id = filepath.Join(ssh_dir, "id")
	var out_file = filepath.Join(b.WorkDir, "output.log")

	if b.GitSecretKey != "" {
		err := os.MkdirAll(ssh_dir, 0700)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(ssh_id, []byte(b.GitSecretKey), 0600)
		if err != nil {
			return err
		}
	}

	outf, err := os.OpenFile(out_file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer outf.Close()

	var args []string
	args = append(args, "clone")
	if !b.GitFullClone {
		args = append(args, "--depth", "1")
	}
	if !b.GitRecursive {
		args = append(args, "--recursive")
	}
	if b.GitBranch != "" {
		args = append(args, "-b", b.GitBranch)
	}
	args = append(args, b.GitUrl, checkout_dir)
	cmd := exec.Command("git", args...)
	cmd.Dir = b.WorkDir
	cmd.Stdout = outf
	cmd.Stderr = outf
	cmd.Env = append(cmd.Env,
		"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -i .ssh/id",
		"HOME="+b.WorkDir,
		"PATH="+os.Getenv("PATH"),
		"SHELL="+os.Getenv("SHELL"),
		"USER="+os.Getenv("USER"),
		"LOGNAME="+os.Getenv("LOGNAME"))

	err = logCommand(outf, cmd)
	if err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
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
		fmt.Fprintf(outf, "\nContext expired, command killed\n\n")
		return ctx.Err()
	case err := <-errChan:
		if err == nil {
			fmt.Fprintf(outf, "\nSuccess\n\n")
		} else {
			fmt.Fprintf(outf, "\nFailed: %s\n\n", err.Error())
		}
		return err
	}
}

func (b *Build) runBuildScript(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var checkout_dir = filepath.Join(b.WorkDir, "workspace", b.GitCheckoutDir)
	var build_script = filepath.Join(b.WorkDir, "build")
	var out_file = filepath.Join(b.WorkDir, "output.log")

	err := ioutil.WriteFile(build_script, []byte(b.BuildScript), 0700)
	if err != nil {
		return err
	}

	outf, err := os.OpenFile(out_file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer outf.Close()

	cmd := exec.Command(build_script)
	cmd.Dir = checkout_dir
	cmd.Stdout = outf
	cmd.Stderr = outf
	cmd.Env = append(cmd.Env,
		"HOME="+b.WorkDir,
		"PATH="+os.Getenv("PATH"),
		"SHELL="+os.Getenv("SHELL"),
		"USER="+os.Getenv("USER"),
		"LOGNAME="+os.Getenv("LOGNAME"))

	err = logCommand(outf, cmd)
	if err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
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
		fmt.Fprintf(outf, "\nContext expired, command terminated\n\n")
		return ctx.Err()
	case err := <-errChan:
		if err == nil {
			fmt.Fprintf(outf, "\nSuccess\n\n")
		} else {
			fmt.Fprintf(outf, "\nFailed: %s\n\n", err.Error())
		}
		return err
	}
}

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

package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/hpcloud/tail"
	"github.com/squarescale/simple-builder/pkg/gitcloner"
	"github.com/squarescale/simple-builder/pkg/scriptrunner"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/rs/zerolog"
)

type Builder struct {
	Cfg *Config

	Errors []*BuilderError `json:"errors"`
	Output string          `json:"output"`

	// XXX: there is no data available for JSON marshalling in os.ProcessState
	ProcessState *os.ProcessState `json:"-"`

	workDir string
	logFile *os.File
	logger  zerolog.Logger

	cloner *gitcloner.Cloner
	runner *scriptrunner.Runner

	ctx        context.Context
	cancelFunc context.CancelFunc

	wg *sync.WaitGroup
}

func New(ctx context.Context, cfgFile string) (*Builder, error) {
	cfg, err := NewConfigFromFile(cfgFile)
	if err != nil {
		return nil, err
	}

	ctx2, cancelFunc := context.WithCancel(ctx)

	wd, err := initWorkDir()
	if err != nil {
		return nil, err
	}

	lf, err := initLogFile(wd)
	if err != nil {
		return nil, err
	}

	logger := zerolog.New(lf).With().Timestamp().Logger()

	b := &Builder{
		Cfg: cfg,

		workDir: wd,
		logFile: lf,
		logger:  logger,

		ctx:        ctx2,
		cancelFunc: cancelFunc,
	}

	// ---

	b.initGitCloner()

	b.initScriptRunner()

	return b, nil
}

func (b *Builder) Run() error {
	defer func() {
		b.logFile.Close()
		b.fetchBuildOutput()
		b.notifyCallbacks()
	}()

	if b.isTerminal() {
		go b.tailBuildOutput(b.ctx)
	}

	err := b.cloner.Run()
	if err != nil {
		b.appendError(err)
		b.setProcessState(b.cloner.ProcessState)
		return err
	}

	err = b.runner.Run()
	if err != nil {
		b.appendError(err)
		b.setProcessState(b.runner.ProcessState)
	}

	return err
}

func (b *Builder) setProcessState(s *os.ProcessState) {
	b.ProcessState = s
}

func (b *Builder) initGitCloner() {
	// XXX: forced to keep buggy legacy behavior
	b.Cfg.GitCloner.Recursive = true

	cfg := b.Cfg.GitCloner

	b.cloner = gitcloner.New(b.ctx, &gitcloner.Config{
		RepoURL:     cfg.RepoURL,
		Branch:      cfg.Branch,
		CheckoutDir: cfg.CheckoutDir,

		SSHKeyContents: cfg.SSHKeyContents,
		SSHKeyFile:     "id",
		SSHKeyDir: filepath.Join(
			b.workDir, ".ssh",
		),

		FullClone: cfg.FullClone,
		Recursive: cfg.Recursive,

		WorkDir:  b.workDir,
		ExtraEnv: commonEnv(b.workDir),

		Logger: b.logger,
	})
}

func (b *Builder) initScriptRunner() {
	cfg := b.Cfg.ScriptRunner

	b.runner = scriptrunner.New(b.ctx, &scriptrunner.Config{
		Script:   cfg.Script,
		ExtraEnv: commonEnv(b.workDir),
		Logger:   b.logger,

		//XXX: because the script must be executed at the root of the git repo
		WorkDir: b.cloner.Cfg.CheckoutDir,
	})
}

func (b *Builder) fetchBuildOutput() {
	bytes, err := ioutil.ReadFile(
		b.logFile.Name(),
	)

	if err != nil {
		b.appendError(err)
	}

	b.Output = string(bytes)
}

func (b *Builder) notifyCallbacks() error {
	if len(b.Cfg.Callbacks) == 0 {
		return nil
	}

	data, err := json.Marshal(b)
	if err != nil {
		return err
	}

	for _, cb := range b.Cfg.Callbacks {
		_, err := http.Post(
			cb,
			"application/json",
			bytes.NewBuffer(data),
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Builder) appendError(e error) {
	if e == nil {
		return
	}

	b.Errors = append(
		b.Errors,
		&BuilderError{e},
	)
}

func (b *Builder) isTerminal() bool {
	return terminal.IsTerminal(
		int(os.Stdout.Fd()),
	)
}

func (b *Builder) tailBuildOutput(ctx context.Context) {
	t, err := tail.TailFile(
		b.logFile.Name(),
		tail.Config{Follow: true},
	)

	if err != nil {
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
		fmt.Println(line.Text)
	}

	tailStop()
}

// ---

func initWorkDir() (string, error) {
	tmp, err := ioutil.TempDir(
		"", "simple-builder",
	)

	if err != nil {
		return "", err
	}

	return tmp, nil
}

func initLogFile(root string) (*os.File, error) {
	f, err := os.OpenFile(
		filepath.Join(root, "build.log"),
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0600,
	)

	if err != nil {
		return nil, err
	}

	return f, nil
}

func commonEnv(home string) []string {
	env := map[string]string{
		"HOME": home,
	}

	keys := []string{
		"PATH",
		"SHELL",
		"USER",
		"LOGNAME",
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

package scriptrunner

import (
	"github.com/rs/zerolog"
)

type Config struct {
	ScriptContents string   `json:"build_script"`
	ScriptFile     string   `json:"-"`
	WorkDir        string   `json:"-"`
	ExtraEnv       []string `json:"-"`

	Logger zerolog.Logger `json:"-"`
}

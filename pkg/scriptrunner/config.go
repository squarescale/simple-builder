package scriptrunner

import (
	"github.com/rs/zerolog"
)

type Config struct {
	Script   string   `json:"build_script"`
	WorkDir  string   `json:"-"`
	ExtraEnv []string `json:"-"`

	Logger zerolog.Logger `json:"-"`
}

package builder

import (
	"encoding/json"
	"io/ioutil"

	"github.com/squarescale/simple-builder/lib/gitcloner"
	"github.com/squarescale/simple-builder/lib/scriptrunner"
)

type Config struct {
	Callbacks []string `json:"callbacks"`

	GitCloner    *gitcloner.Config
	ScriptRunner *scriptrunner.Config
}

func NewConfigFromFile(name string) (*Config, error) {
	buff, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	c := new(Config)

	err = json.Unmarshal(buff, c)
	if err != nil {
		return nil, err
	}

	clonerCfg := new(gitcloner.Config)
	err = json.Unmarshal(buff, clonerCfg)
	if err != nil {
		return nil, err
	}

	runnerCfg := new(scriptrunner.Config)
	err = json.Unmarshal(buff, runnerCfg)
	if err != nil {
		return nil, err
	}

	c.GitCloner = clonerCfg
	c.ScriptRunner = runnerCfg

	return c, nil
}

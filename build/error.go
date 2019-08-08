package build

import "encoding/json"

type BuildError struct {
	error
}

func (err *BuildError) MarshalJSON() ([]byte, error) {
	return json.Marshal(err.Error())
}

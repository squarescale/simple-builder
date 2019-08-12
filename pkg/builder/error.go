package builder

import "encoding/json"

type BuilderError struct {
	error
}

func (err *BuilderError) MarshalJSON() ([]byte, error) {
	return json.Marshal(err.Error())
}

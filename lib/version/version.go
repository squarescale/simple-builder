package version

import (
	"fmt"
)

var (
	GitBranch string
	GitCommit string
	BuildDate string
)

func String() string {
	return fmt.Sprintf(
		"%s/%s (%s)",
		GitBranch,
		GitCommit,
		BuildDate,
	)
}

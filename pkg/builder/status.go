package builder

type status uint8

const (
	STATUS_INIT = iota
	STATUS_GIT_CLONE
	STATUS_BUILD
	STATUS_FAILURE
	STATUS_SUCCESS
)

func (s status) String() string {
	switch s {
	case STATUS_INIT:
		return "init"
	case STATUS_GIT_CLONE:
		return "git_clone"
	case STATUS_BUILD:
		return "build"
	case STATUS_FAILURE:
		return "failure"
	case STATUS_SUCCESS:
		return "success"
	default:
		return "unknown"
	}
}

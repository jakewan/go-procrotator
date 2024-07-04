package watchdirs

type (
	WatcherEvent struct {
		Path string
		Ops  []WatcherEventOp
	}
	WatcherEventOp int
)

const (
	UKNOWN WatcherEventOp = iota
	CHMOD
	CREATE
	REMOVE
	RENAME
	WRITE
)

func (r WatcherEventOp) String() string {
	return [...]string{"UNKNOWN", "CHMOD", "CREATE", "REMOVE", "RENAME", "WRITE"}[r]
}

func (r WatcherEventOp) EnumIndex() int {
	return int(r)
}

package raft

// Store is implemented by the client representing the underlying data store
type Store interface {
	// Commit is called whenever a raft log is fully replicated and committed
	Commit(l *Log)
}

type Operation string

const (
	Set    Operation = "set"
	Delete Operation = "delete"
)

type Command struct {
	Op  Operation
	Key string
	Val []byte
}

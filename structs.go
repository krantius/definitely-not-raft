package raft

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

type LogEntry struct {
	Index uint64
	Term  int
	Data  []byte
}

type Election struct {
	term  int
	voted string
}

type Candidacy struct {
	term  int
	votes int
}

type State string

const (
	Follower  State = "follower"
	Candidate State = "candidate"
	Leader    State = "leader"
)

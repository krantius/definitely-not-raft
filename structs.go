package raft

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

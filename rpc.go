package raft

type RaftRPC interface {
	AppendEntries(args AppendEntriesArgs, res *AppendEntriesResponse) error
	RequestVote(args RequestVoteArgs, res *RequestVoteResponse) error
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     string
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesResponse struct {
	Term    int
	Success bool
}

type RequestVoteArgs struct {
	Term         int
	CadidateId   string
	LastLogIndex uint64
	LastLogTerm  int
}

type RequestVoteResponse struct {
	Term        int // IS THIS RIGHT??
	VoteGranted bool
}
